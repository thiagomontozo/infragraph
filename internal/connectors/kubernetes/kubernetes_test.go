package kubernetes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetadataOnlyAndNeverSecrets(t *testing.T) {
	var paths []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Write([]byte(`{"items":[{"metadata":{"name":"x","namespace":"n","uid":"u","labels":{"app":"x"}},"data":{"password":"leak"},"stringData":{"token":"leak"}}]}`))
	}))
	defer s.Close()
	c, _ := New(s.URL, "token", time.Second)
	items, e := c.Discover(context.Background())
	if e != nil || len(items) != len(resources) {
		t.Fatal(e)
	}
	for _, p := range paths {
		if strings.Contains(p, "secret") || strings.Contains(p, "configmap") {
			t.Fatalf("sensitive endpoint requested: %s", p)
		}
	}
}
