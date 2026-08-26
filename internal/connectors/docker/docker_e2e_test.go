package docker

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSyntheticDockerTopology(t *testing.T) {
	if os.Getenv("INFRAGRAPH_DOCKER_E2E") != "1" {
		t.Skip("synthetic Docker E2E is opt-in")
	}
	c, e := NewUnix(env("INFRAGRAPH_DOCKER_SOCKET", "/var/run/docker.sock"), "com.infragraph.test=true", 20*time.Second)
	if e != nil {
		t.Fatal(e)
	}
	assets, rels, e := c.Discover(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	names := map[string]string{}
	for _, a := range assets {
		if n, ok := a.Attributes["name"].(string); ok {
			names[n] = a.ExternalID
		}
	}
	for _, n := range []string{"infragraph-test-web", "infragraph-test-api", "infragraph-test-db"} {
		if names[n] == "" {
			t.Fatalf("synthetic container %s not discovered", n)
		}
	}
	expected := map[string]bool{"infragraph-test-web>infragraph-test-api": false, "infragraph-test-api>infragraph-test-db": false}
	for _, r := range rels {
		for k := range expected {
			parts := split(k)
			if r.ExternalFromID == names[parts[0]] && r.ExternalToID == names[parts[1]] && r.Type == "CONNECTED_TO" {
				expected[k] = true
			}
		}
	}
	for k, v := range expected {
		if !v {
			t.Fatalf("expected dependency not discovered: %s", k)
		}
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func split(v string) [2]string {
	for i := range v {
		if v[i] == '>' {
			return [2]string{v[:i], v[i+1:]}
		}
	}
	return [2]string{v, ""}
}
