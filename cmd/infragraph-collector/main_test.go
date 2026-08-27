package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/thiagomontozo/infragraph/internal/domain"
)

func TestKubernetesEnrollmentPersistsConnectorType(t *testing.T) {
	data := t.TempDir()
	t.Setenv("INFRAGRAPH_ENROLLMENT_TOKEN", "enrollment-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request["connectorType"] != "KUBERNETES" || request["connectorName"] != "Kubernetes discovery" {
			t.Errorf("unexpected enrollment: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"collectorId":"collector","connectorId":"connector","organizationId":"organization","credential":"credential"}`))
	}))
	defer server.Close()
	id, err := loadOrEnroll(context.Background(), server.URL, data, "KUBERNETES")
	if err != nil {
		t.Fatal(err)
	}
	if id.ConnectorType != "KUBERNETES" || id.ConnectorID != "connector" {
		t.Fatalf("connector identity was not persisted: %#v", id)
	}
	if _, err = loadOrEnroll(context.Background(), server.URL, data, "DOCKER"); err == nil {
		t.Fatal("connector identity was reused with a different type")
	}
}

func TestConfiguredConnectorType(t *testing.T) {
	t.Setenv("INFRAGRAPH_CONNECTOR_TYPE", "kubernetes")
	if connectorType, err := configuredConnectorType(); err != nil || connectorType != "KUBERNETES" {
		t.Fatalf("unexpected type=%q err=%v", connectorType, err)
	}
	t.Setenv("INFRAGRAPH_CONNECTOR_TYPE", "shell")
	if _, err := configuredConnectorType(); err == nil {
		t.Fatal("unsupported connector type was accepted")
	}
}

func TestEnvIntRejectsInvalidAndOverflowingValues(t *testing.T) {
	const key = "INFRAGRAPH_TEST_POSITIVE_INT"
	for _, value := range []string{"", "0", "-1", "999999999999999999999999999999"} {
		t.Setenv(key, value)
		if got := envInt(key, 42); got != 42 {
			t.Fatalf("envInt(%q)=%d, want fallback", value, got)
		}
	}
	t.Setenv(key, "750")
	if got := envInt(key, 42); got != 750 {
		t.Fatalf("envInt returned %d, want 750", got)
	}
}

func TestSequenceAndSpoolAreDurable(t *testing.T) {
	data := t.TempDir()
	first, err := nextSequence(data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := nextSequence(data)
	if err != nil || second <= first {
		t.Fatalf("sequence is not monotonic: first=%d second=%d err=%v", first, second, err)
	}
	snapshot := domain.SnapshotEnvelope{SnapshotID: "snapshot-test", Sequence: second}
	if err = queueSnapshot(data, snapshot); err != nil {
		t.Fatal(err)
	}
	spool := filepath.Join(data, "spool")
	entries, err := os.ReadDir(spool)
	if err != nil || len(entries) != 1 {
		t.Fatalf("snapshot was not queued: entries=%d err=%v", len(entries), err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer credential" {
			t.Error("collector credential missing")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	if err = flushSpool(context.Background(), server.URL, identity{Credential: "credential"}, data); err != nil {
		t.Fatal(err)
	}
	entries, err = os.ReadDir(spool)
	if err != nil || len(entries) != 0 {
		t.Fatalf("delivered snapshot remained queued: entries=%d err=%v", len(entries), err)
	}
}
