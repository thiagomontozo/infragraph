package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalStorageRoundtripAndTraversal(t *testing.T) {
	s, e := NewLocal(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	o, e := s.Put(context.Background(), "org/evidence.txt", strings.NewReader("evidence"), 8, "text/plain")
	if e != nil || len(o.ContentHash) != 64 {
		t.Fatal(e)
	}
	r, e := s.Get(context.Background(), o.Key)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	b, _ := io.ReadAll(r)
	if string(b) != "evidence" {
		t.Fatal("content mismatch")
	}
	if _, e = s.Put(context.Background(), "../escape", strings.NewReader("x"), 1, "text/plain"); e == nil {
		t.Fatal("path traversal accepted")
	}
}
