package gateway

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/youmei295/something-something/src/backend/internal/config"
)

func TestGateway_RoutesByPrefix(t *testing.T) {
	identity := httptest.NewServer(marker("identity"))
	defer identity.Close()
	submission := httptest.NewServer(marker("submission"))
	defer submission.Close()
	conversation := httptest.NewServer(marker("conversation"))
	defer conversation.Close()
	attachment := httptest.NewServer(marker("attachment"))
	defer attachment.Close()

	gw, err := newGateway(config.GatewayConfig{
		IdentityURL:     identity.URL,
		SubmissionURL:   submission.URL,
		ConversationURL: conversation.URL,
		AttachmentURL:   attachment.URL,
	}, []string{"http://localhost:3000"}, slog.Default())
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	handler := gw.Handler()

	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/auth/login", "identity"},
		{"/api/v1/messages", "submission"},
		{"/api/v1/visitor/threads/abc", "submission"},
		{"/api/v1/conversations", "conversation"}, // collection endpoint
		{"/api/v1/conversations/123", "conversation"},
		{"/api/v1/conversations/123/replies", "conversation"},
		{"/api/v1/attachments/upload-url", "attachment"},
		{"/api/v1/attachments/123/download", "attachment"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Header().Get("X-Upstream") != tc.want {
			t.Fatalf("%s routed to %q, want %q", tc.path, rec.Header().Get("X-Upstream"), tc.want)
		}
	}
}

func TestGateway_UnknownRoute(t *testing.T) {
	gw, err := newGateway(config.GatewayConfig{}, nil, slog.Default())
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	gw.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown route = %d, want 404", rec.Code)
	}
}

func TestGateway_ReadyProbesUpstreams(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	gw, err := newGateway(config.GatewayConfig{IdentityURL: upstream.URL}, nil, slog.Default())
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	gw.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz = %d, want 200", rec.Code)
	}
}

func TestGateway_PreflightHandledLocally(t *testing.T) {
	gw, err := newGateway(config.GatewayConfig{}, []string{"http://localhost:3000"}, slog.Default())
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3000") // localhost reflection
	req.Header.Set("Access-Control-Request-Method", "POST")
	gw.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:3000" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected Allow-Credentials: true")
	}
}

func TestGateway_UpstreamErrorIncludesCORS(t *testing.T) {
	// Point at a closed port so the proxy errors.
	gw, err := newGateway(config.GatewayConfig{IdentityURL: "http://127.0.0.1:1"}, []string{"http://localhost:3000"}, slog.Default())
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	gw.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("upstream error = %d, want 502", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("error response missing CORS header, got %q", got)
	}
}

func marker(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Upstream", name)
		w.WriteHeader(http.StatusOK)
	})
}
