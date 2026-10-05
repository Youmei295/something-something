package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/youmei295/something-something/src/backend/internal/config"
)

// newTestAPI builds an API with only the fields the router needs. Handlers that
// require services are not exercised here; this test guards route wiring.
func newTestAPI() *API {
	return New(Options{
		Config: &config.Config{
			HTTP: config.HTTPConfig{AllowedOrigins: []string{"http://localhost:3000"}},
			Auth: config.AuthConfig{CookieName: "inbox_session"},
		},
		Logger: slog.Default(),
		Enable: Enable{Identity: true, Conversation: true},
	})
}

func TestRouter_Health(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newTestAPI().Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestRouter_ProtectedRoutesRequireAuth(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/conversations"},
		{http.MethodGet, "/api/v1/auth/me"},
	}
	for _, rt := range routes {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(rt.method, rt.path, nil)
		newTestAPI().Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s = %d, want 401", rt.method, rt.path, rec.Code)
		}
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	newTestAPI().Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown route = %d, want 404", rec.Code)
	}
}

// A service must expose only its own contexts. An identity-only router should
// answer auth routes but not conversation or submission routes.
func TestRouter_PartialEnable(t *testing.T) {
	api := New(Options{
		Config: &config.Config{
			HTTP: config.HTTPConfig{AllowedOrigins: []string{"http://localhost:3000"}},
			Auth: config.AuthConfig{CookieName: "inbox_session"},
		},
		Logger: slog.Default(),
		Enable: Enable{Identity: true},
	})
	handler := api.Router()

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodPost, "/api/v1/auth/login", http.StatusBadRequest}, // handler runs, decodes empty body
		{http.MethodGet, "/api/v1/conversations", http.StatusNotFound},
		{http.MethodPost, "/api/v1/messages", http.StatusNotFound},
		{http.MethodPost, "/api/v1/attachments/upload-url", http.StatusNotFound},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

func TestRouter_CORS(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/messages", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	newTestAPI().Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}
