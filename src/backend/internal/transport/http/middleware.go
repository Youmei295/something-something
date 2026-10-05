package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/domain"
)

type ctxKey string

const (
	ctxUser      ctxKey = "user"
	ctxUserID    ctxKey = "user_id"
	ctxSessionID ctxKey = "session_id"
	requestIDKey ctxKey = "request_id"
)

// requestID injects a correlation ID into the context and response headers.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// logRequests emits one structured log line per request.
func logRequests(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", requestIDFrom(r.Context())),
			)
		})
	}
}

// recoverPanic prevents one bad request from taking down the process.
func recoverPanic(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						slog.Any("panic", rec),
						slog.String("path", r.URL.Path),
						slog.String("request_id", requestIDFrom(r.Context())),
					)
					writeError(w, errors.New("panic recovered"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// cors applies a strict allow-list of origins.
func cors(allowed []string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			if _, ok := allowedSet[origin]; ok && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireAuth resolves the session cookie and rejects unauthenticated requests.
func (a *API) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(a.cfg.Auth.CookieName)
		if err != nil || cookie.Value == "" {
			writeError(w, domain.ErrUnauthorized)
			return
		}
		sessionID, err := uuid.Parse(cookie.Value)
		if err != nil {
			writeError(w, domain.ErrUnauthorized)
			return
		}
		user, err := a.auth.Authenticate(r.Context(), sessionID)
		if err != nil {
			writeError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, user.ID)
		ctx = context.WithValue(ctx, ctxUser, user)
		ctx = context.WithValue(ctx, ctxSessionID, sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func sessionIDFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxSessionID).(uuid.UUID)
	return v, ok
}

func userFrom(ctx context.Context) (*domain.User, bool) {
	v, ok := ctx.Value(ctxUser).(*domain.User)
	return v, ok
}
