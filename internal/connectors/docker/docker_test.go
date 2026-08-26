package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscoveryIsScopedAndDoesNotRequestSecrets(t *testing.T) {
	var requests []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"Id":"c1","Names":["/test-api"],"Image":"alpine","ImageID":"sha256:i","Labels":{"com.infragraph.test":"true"},"NetworkSettings":{"Networks":{"testnet":{}}},"Mounts":[{"Type":"volume","Name":"data"}]}]`))
	}))
	defer s.Close()
	c, e := NewHTTP(s.URL, "com.infragraph.test=true", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	a, r, e := c.Discover(context.Background())
	if e != nil || len(a) != 5 || len(r) != 4 {
		t.Fatalf("unexpected discovery %d/%d %v", len(a), len(r), e)
	}
	joined := strings.Join(requests, " ")
	if !strings.Contains(joined, "filters=") || strings.Contains(joined, "inspect") || strings.Contains(joined, "logs") {
		t.Fatalf("unsafe/unscoped calls: %s", joined)
	}
}
