package reconcile

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/thiagomontozo/infragraph/internal/database"
	"github.com/thiagomontozo/infragraph/internal/domain"
)

func TestApplyPostgresPersistsAndMarksMissing(t *testing.T) {
	url := os.Getenv("INFRAGRAPH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PostgreSQL integration is opt-in")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	suffix := fmt.Sprint(time.Now().UnixNano())
	org := "org-reconcile-" + suffix
	collector := "collector-reconcile-" + suffix
	connector := "connector-reconcile-" + suffix
	if _, err = tx.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'Reconcile integration')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO collectors(id,organization_id,name,public_key,fingerprint,status) VALUES($1,$2,'test','x','test','ACTIVE')`, collector, org); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO infrastructure_connectors(id,organization_id,collector_id,name,type,authoritative_level) VALUES($1,$2,$3,'test','DOCKER','OBSERVED')`, connector, org, collector); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	first := domain.SnapshotEnvelope{
		SnapshotID:     "snapshot-first-" + suffix,
		OrganizationID: org,
		CollectorID:    collector,
		ConnectorID:    connector,
		StartedAt:      now.Add(-time.Second),
		CompletedAt:    now,
		ContentHash:    "first",
		Assets: []domain.Observation{
			{ExternalID: "app", AssetType: domain.Application, ObservedAt: now, Attributes: map[string]any{"name": "web", "environment": "production"}, IdentityHints: map[string]any{"external_cmdb_id": "shared-app"}},
			{ExternalID: "db", AssetType: domain.Database, ObservedAt: now, Attributes: map[string]any{"name": "postgres"}},
		},
		Relationships: []domain.RelationshipObservation{{ExternalFromID: "app", ExternalToID: "db", Type: "USES_DATABASE", ObservedAt: now, Attributes: map[string]any{}}},
	}
	insertSnapshot := func(snapshot domain.SnapshotEnvelope, sequence int64) {
		t.Helper()
		_, insertErr := tx.Exec(ctx, `INSERT INTO source_snapshots(id,organization_id,connector_id,collector_id,status,sequence,started_at,completed_at,content_hash) VALUES($1,$2,$3,$4,'RUNNING',$5,$6,$7,$8)`, snapshot.SnapshotID, org, connector, collector, sequence, snapshot.StartedAt, snapshot.CompletedAt, snapshot.ContentHash)
		if insertErr != nil {
			t.Fatal(insertErr)
		}
	}
	insertSnapshot(first, 1)
	summary, err := ApplyPostgres(ctx, tx, first, 1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Created != 2 || summary.RelationshipsCreated != 1 {
		t.Fatalf("unexpected first summary: %+v", summary)
	}
	var assets, observations, relationships, claims, runs int
	if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM assets WHERE organization_id=$1),(SELECT count(*) FROM asset_observations WHERE organization_id=$1),(SELECT count(*) FROM asset_relationships WHERE organization_id=$1),(SELECT count(*) FROM asset_attribute_claims WHERE organization_id=$1),(SELECT count(*) FROM reconciliation_runs WHERE organization_id=$1)`, org).Scan(&assets, &observations, &relationships, &claims, &runs); err != nil {
		t.Fatal(err)
	}
	if assets != 2 || observations != 2 || relationships != 1 || claims != 3 || runs != 1 {
		t.Fatalf("snapshot was not fully persisted: assets=%d observations=%d relationships=%d claims=%d runs=%d", assets, observations, relationships, claims, runs)
	}
	connectorTwo := "connector-reconcile-two-" + suffix
	if _, err = tx.Exec(ctx, `INSERT INTO infrastructure_connectors(id,organization_id,collector_id,name,type,authoritative_level) VALUES($1,$2,$3,'test two','DOCKER','DECLARED')`, connectorTwo, org, collector); err != nil {
		t.Fatal(err)
	}
	otherSource := domain.SnapshotEnvelope{SnapshotID: "snapshot-other-" + suffix, OrganizationID: org, CollectorID: collector, ConnectorID: connectorTwo, StartedAt: now, CompletedAt: now.Add(30 * time.Second), ContentHash: "other", Assets: []domain.Observation{{ExternalID: "other-app", AssetType: domain.Application, ObservedAt: now, Attributes: map[string]any{"name": "web", "environment": "staging"}, IdentityHints: map[string]any{"external_cmdb_id": "shared-app"}}}}
	if _, err = tx.Exec(ctx, `INSERT INTO source_snapshots(id,organization_id,connector_id,collector_id,status,sequence,started_at,completed_at,content_hash) VALUES($1,$2,$3,$4,'RUNNING',1,$5,$6,$7)`, otherSource.SnapshotID, org, connectorTwo, collector, otherSource.StartedAt, otherSource.CompletedAt, otherSource.ContentHash); err != nil {
		t.Fatal(err)
	}
	if summary, err = ApplyPostgres(ctx, tx, otherSource, 1); err != nil {
		t.Fatal(err)
	}
	if summary.Created != 0 || summary.Conflicting != 1 {
		t.Fatalf("strong identity was not reconciled across sources: %+v", summary)
	}
	var conflicting int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM assets WHERE organization_id=$1 AND status='CONFLICTING'`, org).Scan(&conflicting); err != nil || conflicting != 1 {
		t.Fatalf("cross-source conflict count=%d err=%v", conflicting, err)
	}

	second := domain.SnapshotEnvelope{SnapshotID: "snapshot-empty-" + suffix, OrganizationID: org, CollectorID: collector, ConnectorID: connector, StartedAt: now, CompletedAt: now.Add(time.Minute), ContentHash: "second"}
	insertSnapshot(second, 2)
	summary, err = ApplyPostgres(ctx, tx, second, 1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Missing != 1 || summary.RelationshipsRemoved != 1 {
		t.Fatalf("successful absence was not reconciled: %+v", summary)
	}
	var missing int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM assets WHERE organization_id=$1 AND status='MISSING'`, org).Scan(&missing); err != nil || missing != 1 {
		t.Fatalf("missing assets=%d err=%v", missing, err)
	}
}
