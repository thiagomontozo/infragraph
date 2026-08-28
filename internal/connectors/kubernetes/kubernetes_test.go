package kubernetes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetadataPaginationAndRelationships(t *testing.T) {
	paths := []string{}
	fixtures := map[string]string{
		"/api/v1/nodes":                        `{"items":[{"metadata":{"name":"worker-1","uid":"node-uid"}}]}`,
		"/api/v1/pods":                         `{"items":[{"metadata":{"name":"api-1","namespace":"apps","uid":"pod-uid","labels":{"app":"api"},"ownerReferences":[{"kind":"ReplicaSet","name":"api-rs","uid":"rs-uid"}]},"spec":{"nodeName":"worker-1"},"data":{"password":"must-not-appear"}}]}`,
		"/api/v1/services":                     `{"items":[{"metadata":{"name":"api","namespace":"apps","uid":"service-uid"},"spec":{"selector":{"app":"api"}}}]}`,
		"/apis/apps/v1/deployments":            `{"items":[{"metadata":{"name":"api","namespace":"apps","uid":"deployment-uid"},"spec":{"selector":{"matchLabels":{"app":"api"}}}}]}`,
		"/apis/apps/v1/statefulsets":           `{"items":[]}`,
		"/apis/apps/v1/daemonsets":             `{"items":[]}`,
		"/apis/apps/v1/replicasets":            `{"items":[{"metadata":{"name":"api-rs","namespace":"apps","uid":"rs-uid","ownerReferences":[{"kind":"Deployment","name":"api","uid":"deployment-uid"}]}}]}`,
		"/apis/networking.k8s.io/v1/ingresses": `{"items":[{"metadata":{"name":"api","namespace":"apps","uid":"ingress-uid"},"spec":{"rules":[{"http":{"paths":[{"backend":{"service":{"name":"api"}}}]}}]}}]}`,
		"/api/v1/persistentvolumes":            `{"items":[{"metadata":{"name":"data-pv","uid":"pv-uid"}}]}`,
		"/api/v1/persistentvolumeclaims":       `{"items":[{"metadata":{"name":"data","namespace":"apps","uid":"pvc-uid"},"spec":{"volumeName":"data-pv"}}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("authorization header missing: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Errorf("page limit missing: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/namespaces" {
			if r.URL.Query().Get("continue") == "" {
				_, _ = w.Write([]byte(`{"metadata":{"continue":"page two"},"items":[{"metadata":{"name":"kube-system","uid":"system-uid"}}]}`))
			} else {
				_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"apps","uid":"apps-uid"},"stringData":{"token":"must-not-appear"}}]}`))
			}
			return
		}
		fixture, ok := fixtures[r.URL.Path]
		if !ok {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()
	connector, err := NewWithConfig(Config{BaseURL: server.URL, Token: "test-token", ClusterName: "test-cluster", Timeout: 2 * time.Second, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	assets, relationships, err := connector.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 11 {
		t.Fatalf("expected cluster plus 10 resources, got %d", len(assets))
	}
	if len(relationships) != 16 {
		t.Fatalf("expected 16 relationships, got %d", len(relationships))
	}
	raw, err := json.Marshal(assets)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "must-not-appear") || strings.Contains(string(raw), "password") || strings.Contains(string(raw), "token") {
		t.Fatalf("sensitive or non-allowlisted data escaped metadata projection: %s", raw)
	}
	for _, path := range paths {
		if strings.Contains(path, "secret") || strings.Contains(path, "configmap") || strings.Contains(path, "serviceaccount") {
			t.Fatalf("sensitive endpoint requested: %s", path)
		}
	}
	wanted := map[string]bool{"CONTAINS": false, "MANAGED_BY": false, "RUNS_ON": false, "ROUTES_TO": false, "BOUND_TO": false}
	for _, relationship := range relationships {
		if _, ok := wanted[relationship.Type]; ok {
			wanted[relationship.Type] = true
		}
	}
	for kind, found := range wanted {
		if !found {
			t.Errorf("relationship %s was not discovered", kind)
		}
	}
}

func TestRejectsUnsafeConfiguration(t *testing.T) {
	for name, config := range map[string]Config{
		"plaintext remote API": {BaseURL: "http://kubernetes.example.test", Token: "token"},
		"missing token":        {BaseURL: "https://kubernetes.example.test"},
		"invalid CA":           {BaseURL: "https://kubernetes.example.test", Token: "token", CAPEM: []byte("not a certificate")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewWithConfig(config); err == nil {
				t.Fatal("unsafe configuration was accepted")
			}
		})
	}
}
