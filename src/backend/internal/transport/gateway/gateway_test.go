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

	gw, err := New(config.GatewayConfig{
		IdentityURL:     identity.URL,
		SubmissionURL:   submission.URL,
		ConversationURL: conversation.URL,
	}, slog.Default())
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
		{"/api/v1/conversations/123", "conversation"},
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
	gw, err := New(config.GatewayConfig{}, slog.Default())
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

	gw, err := New(config.GatewayConfig{IdentityURL: upstream.URL}, slog.Default())
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

func marker(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Upstream", name)
		w.WriteHeader(http.StatusOK)
	})
}
