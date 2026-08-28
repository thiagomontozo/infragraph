//go:build integration

package kubernetes

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestKindDiscoveryWithServiceAccount(t *testing.T) {
	if os.Getenv("INFRAGRAPH_KUBERNETES_E2E") != "1" {
		t.Skip("set INFRAGRAPH_KUBERNETES_E2E=1 to run the kind integration")
	}
	ca, err := os.ReadFile(os.Getenv("INFRAGRAPH_KUBERNETES_E2E_CA_FILE"))
	if err != nil {
		t.Fatal(err)
	}
	connector, err := NewWithConfig(Config{
		BaseURL:     os.Getenv("INFRAGRAPH_KUBERNETES_E2E_URL"),
		Token:       os.Getenv("INFRAGRAPH_KUBERNETES_E2E_TOKEN"),
		CAPEM:       ca,
		ClusterName: "infragraph-e2e",
		Timeout:     30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	assets, relationships, err := connector.Discover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{"Deployment": false, "Pod": false, "Service": false}
	for _, asset := range assets {
		labels, _ := asset.Attributes["labels"].(map[string]string)
		if labels["infragraph.dev/e2e"] == "true" {
			if kind, _ := asset.Attributes["kind"].(string); kind != "" {
				found[kind] = true
			}
		}
	}
	for kind, present := range found {
		if !present {
			t.Errorf("kind fixture %s was not discovered", kind)
		}
	}
	wantedRelationships := map[string]bool{"MANAGED_BY": false, "RUNS_ON": false, "ROUTES_TO": false}
	for _, relationship := range relationships {
		if _, ok := wantedRelationships[relationship.Type]; ok {
			wantedRelationships[relationship.Type] = true
		}
	}
	for kind, present := range wantedRelationships {
		if !present {
			t.Errorf("relationship %s was not discovered", kind)
		}
	}
	raw, err := json.Marshal(assets)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "infragraph-secret-must-not-leak") {
		t.Fatal("Kubernetes Secret content was collected")
	}
}
