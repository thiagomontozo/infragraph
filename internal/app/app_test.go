package app

import (
	"github.com/thiagomontozo/infragraph/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHeadersErrorsAndMetrics(t *testing.T) {
	a := New(config.Config{Environment: "development", MaxGraphDepth: 6, MaxGraphNodes: 500}, nil, nil)
	h := a.Handler()
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("X-Request-ID") == "" {
		t.Fatal("health hardening failed")
	}
	r = httptest.NewRequest("GET", "/api/v1/assets", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"requestId"`) {
		t.Fatal("error model failed")
	}
	r = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "http_requests_total") {
		t.Fatal("metrics unavailable")
	}
}
func TestCORSIsRestrictive(t *testing.T) {
	a := New(config.Config{AllowedOrigins: []string{"https://infra.example.com"}, MaxGraphDepth: 6, MaxGraphNodes: 500}, nil, nil)
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/assets", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	if w.Code != 403 || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("untrusted origin allowed")
	}
}
