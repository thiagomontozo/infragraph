package database

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestMigrationsConcurrencyAndTenantIsolation(t *testing.T) {
	url := os.Getenv("INFRAGRAPH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PostgreSQL integration is opt-in")
	}
	db, err := Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			errs <- db.Migrate(ctx)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	_, _ = db.Pool.Exec(ctx, "INSERT INTO organizations(id,name) VALUES('org-a','A'),('org-b','B') ON CONFLICT DO NOTHING")
	now := time.Now()
	_, _ = db.Pool.Exec(ctx, "INSERT INTO assets(id,organization_id,canonical_name,display_name,asset_type,status,first_seen_at,last_seen_at) VALUES('tenant-a-asset','org-a','a','a','APPLICATION','ACTIVE',$1,$1),('tenant-b-asset','org-b','b','b','APPLICATION','ACTIVE',$1,$1) ON CONFLICT DO NOTHING", now)
	_, _ = db.Pool.Exec(ctx, "INSERT INTO assets(id,organization_id,canonical_name,display_name,asset_type,status,first_seen_at,last_seen_at) VALUES('tenant-a-db','org-a','db','db','DATABASE','ACTIVE',$1,$1) ON CONFLICT DO NOTHING", now)
	_, _ = db.Pool.Exec(ctx, "INSERT INTO asset_relationships(id,organization_id,from_asset_id,to_asset_id,type,status,first_seen_at,last_seen_at) VALUES('tenant-a-rel','org-a','tenant-a-asset','tenant-a-db','USES_DATABASE','ACTIVE',$1,$1) ON CONFLICT DO NOTHING", now)
	_, _ = db.Pool.Exec(ctx, "INSERT INTO users(id,organization_id,email,password_hash,display_name) VALUES('tenant-a-user','org-a','owner@example.test','synthetic-not-a-real-password-hash','Owner') ON CONFLICT DO NOTHING")
	_, _ = db.Pool.Exec(ctx, "INSERT INTO reconciliation_policies(id,organization_id,attribute_key,authority,precedence) VALUES('tenant-a-policy','org-a','environment','AUTHORITATIVE',100) ON CONFLICT DO NOTHING")
	_, _ = db.Pool.Exec(ctx, "INSERT INTO audit_events(id,organization_id,action,payload,previous_hash,event_hash) VALUES('tenant-a-audit','org-a','test.seed','{}','','synthetic-test-hash') ON CONFLICT DO NOTHING")
	var count int
	if err = db.Pool.QueryRow(ctx, "SELECT count(*) FROM assets WHERE organization_id=$1", "org-a").Scan(&count); err != nil || count != 2 {
		t.Fatalf("tenant scoped query failed count=%d err=%v", count, err)
	}
	var leaked int
	_ = db.Pool.QueryRow(ctx, "SELECT count(*) FROM assets WHERE organization_id=$1 AND id='tenant-b-asset'", "org-a").Scan(&leaked)
	if leaked != 0 {
		t.Fatal("cross-organization asset leaked")
	}
}
