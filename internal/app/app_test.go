package app

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/thiagomontozo/infragraph/internal/config"
	"github.com/thiagomontozo/infragraph/internal/domain"
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

func TestSnapshotContractLimits(t *testing.T) {
	now := time.Now().UTC()
	snapshot := domain.SnapshotEnvelope{ProtocolVersion: "1.0", SnapshotID: "snapshot", OrganizationID: "org", CollectorID: "collector", ConnectorID: "connector", Sequence: 1, StartedAt: now.Add(-time.Second), CompletedAt: now, Assets: []domain.Observation{{ExternalID: "asset", AssetType: domain.Application, ObservedAt: now, Attributes: map[string]any{}, IdentityHints: map[string]any{}, Status: "OBSERVED"}}, Relationships: []domain.RelationshipObservation{}, Warnings: []string{}, Statistics: map[string]int{}}
	if !validSnapshotContract(snapshot) {
		t.Fatal("valid snapshot contract rejected")
	}
	snapshot.Assets[0].Status = ""
	if validSnapshotContract(snapshot) {
		t.Fatal("invalid observation status accepted")
	}
}

func TestClientIPTrustsForwardingOnlyFromConfiguredProxy(t *testing.T) {
	a := New(config.Config{TrustedProxyCIDRs: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, MaxGraphDepth: 6, MaxGraphNodes: 500}, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.RemoteAddr = "10.0.0.4:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.3")
	if got := a.clientIP(r); got != "203.0.113.5" {
		t.Fatalf("trusted proxy client IP=%q", got)
	}
	r.RemoteAddr = "192.0.2.10:1234"
	if got := a.clientIP(r); got != "192.0.2.10" {
		t.Fatalf("untrusted forwarding header was accepted: %q", got)
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
