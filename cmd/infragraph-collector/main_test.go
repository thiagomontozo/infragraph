package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/thiagomontozo/infragraph/internal/domain"
)

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
