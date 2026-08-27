package graph

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/thiagomontozo/infragraph/internal/database"
)

func TestTraversePostgresIsBoundedAndDirectional(t *testing.T) {
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
	suffix := fmt.Sprint(time.Now().UnixNano())
	org := "org-graph-" + suffix
	a, b, c := "asset-a-"+suffix, "asset-b-"+suffix, "asset-c-"+suffix
	if _, err = db.Pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'Graph integration')`, org); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO assets(id,organization_id,canonical_name,display_name,asset_type,status,first_seen_at,last_seen_at) VALUES($1,$4,'a','a','APPLICATION','ACTIVE',$5,$5),($2,$4,'b','b','APPLICATION','ACTIVE',$5,$5),($3,$4,'c','c','DATABASE','ACTIVE',$5,$5)`, a, b, c, org, now); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO asset_relationships(id,organization_id,from_asset_id,to_asset_id,type,status,first_seen_at,last_seen_at) VALUES($1,$2,$3,$4,'DEPENDS_ON','ACTIVE',$7,$7),($5,$2,$4,$6,'USES_DATABASE','ACTIVE',$7,$7)`, "rel-ab-"+suffix, org, a, b, "rel-bc-"+suffix, c, now); err != nil {
		t.Fatal(err)
	}
	result, err := TraversePostgres(ctx, db.Pool, org, a, 2, 3, false)
	if err != nil || len(result.Nodes) != 3 || len(result.Relationships) != 2 || result.Depth != 2 {
		t.Fatalf("forward traversal=%+v err=%v", result, err)
	}
	reverse, err := TraversePostgres(ctx, db.Pool, org, c, 2, 3, true)
	if err != nil || len(reverse.Nodes) != 3 {
		t.Fatalf("reverse traversal=%+v err=%v", reverse, err)
	}
	if _, err = TraversePostgres(ctx, db.Pool, org, a, 2, 2, false); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("node limit was not enforced: %v", err)
	}
}
