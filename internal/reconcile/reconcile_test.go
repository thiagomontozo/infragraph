package reconcile

import (
	"github.com/thiagomontozo/infragraph/internal/domain"
	"testing"
	"time"
)

func snap(id, hash string, assets []domain.Observation) domain.SnapshotEnvelope {
	return domain.SnapshotEnvelope{SnapshotID: id, ContentHash: hash, OrganizationID: "org", ConnectorID: "docker", CompletedAt: time.Now(), Assets: assets}
}
func TestMissingOnlyAfterSuccessfulSnapshotAndIdempotency(t *testing.T) {
	st := NewState()
	p := Policy{MissingAfterSuccessfulRuns: 1}
	o := domain.Observation{ExternalID: "c1", AssetType: domain.Container, Attributes: map[string]any{"name": "web"}}
	if s, e := Apply(st, snap("1", "a", []domain.Observation{o}), domain.SnapshotSucceeded, p); e != nil || s.Created != 1 {
		t.Fatal(e)
	}
	if _, e := Apply(st, snap("failed", "b", nil), domain.SnapshotFailed, p); e != nil {
		t.Fatal(e)
	}
	if st.Assets["asset-c1"].Status == domain.AssetMissing {
		t.Fatal("failed snapshot created false MISSING")
	}
	s, e := Apply(st, snap("2", "c", nil), domain.SnapshotSucceeded, p)
	if e != nil || s.Missing != 1 {
		t.Fatalf("missing not detected: %+v %v", s, e)
	}
	before := len(st.Changes)
	s, e = Apply(st, snap("2", "c", nil), domain.SnapshotSucceeded, p)
	if e != nil || len(st.Changes) != before || s.Changes != 0 {
		t.Fatal("retry was not idempotent")
	}
}
func TestConflictPreservesClaimsAndAuthority(t *testing.T) {
	claims := []domain.AttributeClaim{{AttributeKey: "environment", Value: "staging", ConnectorID: "csv", Active: true}, {AttributeKey: "environment", Value: "production", ConnectorID: "tf", Active: true}}
	e := Effective(claims, "environment", Policy{AttributeAuthority: map[string]map[string]Authority{"tf": {"environment": Authoritative}, "csv": {"environment": Declared}}})
	if e.Value != "production" || !e.Conflict || len(e.Claims) != 2 {
		t.Fatalf("bad effective claim: %#v", e)
	}
}

func TestRelationshipChangeOnlyOnSuccessfulSnapshots(t *testing.T) {
	st := NewState()
	p := Policy{MissingAfterSuccessfulRuns: 2}
	assets := []domain.Observation{{ExternalID: "app", AssetType: domain.Application, Attributes: map[string]any{"name": "app"}}, {ExternalID: "db", AssetType: domain.Database, Attributes: map[string]any{"name": "db"}}}
	first := snap("r1", "rh1", assets)
	first.Relationships = []domain.RelationshipObservation{{ExternalFromID: "app", ExternalToID: "db", Type: "USES_DATABASE"}}
	summary, err := Apply(st, first, domain.SnapshotSucceeded, p)
	if err != nil || summary.RelationshipsCreated != 1 {
		t.Fatalf("relationship not created: %+v %v", summary, err)
	}
	failed := snap("r2", "rh2", assets)
	summary, err = Apply(st, failed, domain.SnapshotFailed, p)
	if err != nil || summary.RelationshipsRemoved != 0 {
		t.Fatal("failed snapshot removed relationship")
	}
	third := snap("r3", "rh3", assets)
	summary, err = Apply(st, third, domain.SnapshotSucceeded, p)
	if err != nil || summary.RelationshipsRemoved != 1 {
		t.Fatalf("successful absence not reconciled: %+v %v", summary, err)
	}
	for _, relationship := range st.Relationships {
		if relationship.Status != "REMOVED" {
			t.Fatal("relationship status not updated")
		}
	}
}
