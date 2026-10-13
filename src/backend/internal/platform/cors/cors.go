// Package cors provides one CORS implementation shared by the HTTP transport
// and the API gateway, so behavior (and the allow-list) is consistent.
package cors

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// IsLocalhost reports whether the origin's host is a loopback address. Local
// origins are always allowed, which keeps local development working across
// localhost/127.0.0.1 and arbitrary dev ports without weakening production
// (an attacker's page cannot be served from the user's loopback).
func IsLocalhost(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

// Allowed reports whether origin may be reflected in CORS responses.
func Allowed(origin string, allowed []string) bool {
	origin = strings.TrimRight(origin, "/")
	if origin == "" {
		return false
	}
	if IsLocalhost(origin) {
		return true
	}
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimRight(a, "/"), origin) {
			return true
		}
	}
	return false
}

// SetHeaders writes the CORS response headers for an allowed origin.
func SetHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
	w.Header().Set("Access-Control-Max-Age", "600")
}

// Middleware applies the CORS policy. Rejected origins are logged so the cause
// of a browser "CORS request did not succeed" is visible in the server logs.
func Middleware(allowed []string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			switch {
			case origin == "":
				// Same-origin or non-browser request: nothing to do.
			case Allowed(origin, allowed):
				SetHeaders(w, origin)
			default:
				if log != nil {
					log.Warn("cors: origin rejected",
						slog.String("origin", origin),
						slog.Any("allowed", allowed))
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
