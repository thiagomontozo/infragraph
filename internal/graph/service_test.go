package graph

import (
	"context"
	"errors"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestTraversalCycleAndTenantBoundary(t *testing.T) {
	s := New(Limits{6, 10, time.Second})
	e := []domain.Relationship{{OrganizationID: "a", FromAssetID: "1", ToAssetID: "2", Status: "ACTIVE"}, {OrganizationID: "a", FromAssetID: "2", ToAssetID: "1", Status: "ACTIVE"}, {OrganizationID: "b", FromAssetID: "2", ToAssetID: "secret", Status: "ACTIVE"}}
	r, err := s.Traverse(context.Background(), "a", "1", 6, 10, false, e)
	if err != nil || len(r.Nodes) != 2 {
		t.Fatalf("cycle/tenant traversal failed: %#v %v", r, err)
	}
}

func TestPerformanceSmoke(t *testing.T) {
	if os.Getenv("INFRAGRAPH_PERF") != "1" {
		t.Skip("performance smoke is opt-in")
	}
	assets, _ := strconv.Atoi(os.Getenv("INFRAGRAPH_PERF_ASSETS"))
	if assets < 1 {
		assets = 2000
	}
	relationships, _ := strconv.Atoi(os.Getenv("INFRAGRAPH_PERF_RELATIONSHIPS"))
	if relationships < 1 {
		relationships = 5000
	}
	edges := make([]domain.Relationship, 0, relationships)
	for i := 0; i < relationships; i++ {
		edges = append(edges, domain.Relationship{ID: fmt.Sprint(i), OrganizationID: "perf", FromAssetID: fmt.Sprint(i % assets), ToAssetID: fmt.Sprint((i*7 + 1) % assets), Status: "ACTIVE"})
	}
	s := New(Limits{MaxDepth: 6, MaxNodes: 500, Timeout: 3 * time.Second})
	start := time.Now()
	_, err := s.Traverse(context.Background(), "perf", "0", 3, 500, false, edges)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("bounded graph smoke exceeded 3s: %v", elapsed)
	}
	t.Logf("dataset assets=%d relationships=%d elapsed=%v", assets, relationships, elapsed)
}
func TestTraversalRejectsLimits(t *testing.T) {
	s := New(Limits{3, 2, time.Second})
	_, err := s.Traverse(context.Background(), "a", "1", 4, 2, false, nil)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatal("depth limit not rejected")
	}
	e := []domain.Relationship{{OrganizationID: "a", FromAssetID: "1", ToAssetID: "2", Status: "ACTIVE"}, {OrganizationID: "a", FromAssetID: "1", ToAssetID: "3", Status: "ACTIVE"}}
	_, err = s.Traverse(context.Background(), "a", "1", 1, 2, false, e)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatal("node limit not rejected")
	}
}
