// Package gateway is a thin reverse proxy that fronts the standalone services.
// It gives clients a single origin and routes by path prefix, which is the
// minimal viable API gateway for the microservice layout.
package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/youmei295/something-something/src/backend/internal/config"
)

type route struct {
	prefix string
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// Gateway routes incoming requests to services based on path prefixes.
type Gateway struct {
	routes []route
	log    *slog.Logger
	health []string
}

// New builds a gateway from the configured upstream URLs. A blank upstream
// disables that route, so a subset of services can be run during development.
func New(cfg config.GatewayConfig, log *slog.Logger) (*Gateway, error) {
	specs := []struct {
		prefix string
		rawURL string
	}{
		{"/api/v1/auth/", cfg.IdentityURL},
		{"/api/v1/messages", cfg.SubmissionURL},
		{"/api/v1/visitor/", cfg.SubmissionURL},
		{"/api/v1/conversations/", cfg.ConversationURL},
		{"/api/v1/attachments/", cfg.AttachmentURL},
	}

	g := &Gateway{log: log}
	seen := map[string]struct{}{}
	for _, spec := range specs {
		if strings.TrimSpace(spec.rawURL) == "" {
			continue
		}
		target, err := url.Parse(spec.rawURL)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("gateway upstream error",
				slog.String("prefix", spec.prefix),
				slog.String("error", err.Error()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
		}
		g.routes = append(g.routes, route{prefix: spec.prefix, target: target, proxy: proxy})

		if _, ok := seen[target.String()]; !ok {
			seen[target.String()] = struct{}{}
			g.health = append(g.health, target.String())
		}
	}
	return g, nil
}

// Handler returns the gateway's HTTP handler.
func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", g.handleHealth)
	mux.HandleFunc("/readyz", g.handleReady)
	mux.HandleFunc("/", g.dispatch)
	return g.withRequestID(mux)
}

func (g *Gateway) dispatch(w http.ResponseWriter, r *http.Request) {
	var matched *route
	for i := range g.routes {
		if strings.HasPrefix(r.URL.Path, g.routes[i].prefix) {
			if matched == nil || len(g.routes[i].prefix) > len(matched.prefix) {
				matched = &g.routes[i]
			}
		}
	}
	if matched == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	matched.proxy.ServeHTTP(w, r)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleReady probes each upstream's /healthz concurrently.
func (g *Gateway) handleReady(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 2 * time.Second}
	ready := true
	for _, base := range g.health {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/healthz", nil)
		resp, err := client.Do(req)
		cancel()
		if err != nil || resp.StatusCode != http.StatusOK {
			ready = false
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func (g *Gateway) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		r.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}
