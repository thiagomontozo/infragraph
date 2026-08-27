package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/database"
	"github.com/thiagomontozo/infragraph/internal/security"
	"os"
	"strings"
	"time"
)

func main() {
	var dbURL, org, name, email, password string
	var createCollectorToken bool
	var collectorTokenTTL time.Duration
	flag.StringVar(&dbURL, "database-url", os.Getenv("INFRAGRAPH_DATABASE_URL"), "PostgreSQL URL")
	flag.StringVar(&org, "organization", "", "organization name")
	flag.StringVar(&name, "name", "", "owner display name")
	flag.StringVar(&email, "email", "", "owner email")
	flag.StringVar(&password, "password", "", "owner password (prefer prompt/secret injection)")
	flag.BoolVar(&createCollectorToken, "create-collector-token", false, "print one short-lived collector enrollment token")
	flag.DurationVar(&collectorTokenTTL, "collector-token-ttl", 15*time.Minute, "collector enrollment token lifetime")
	flag.Parse()
	if dbURL == "" || org == "" || name == "" || email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "all flags are required")
		os.Exit(2)
	}
	if createCollectorToken && (collectorTokenTTL < time.Minute || collectorTokenTTL > 24*time.Hour) {
		fmt.Fprintln(os.Stderr, "collector token TTL must be between 1 minute and 24 hours")
		os.Exit(2)
	}
	db, e := database.Open(context.Background(), dbURL)
	if e != nil {
		fatal(e)
	}
	defer db.Close()
	if e = db.Migrate(context.Background()); e != nil {
		fatal(e)
	}
	hash, e := security.HashPassword(password)
	if e != nil {
		fatal(e)
	}
	oid := "org-" + security.TokenHash(strings.ToLower(org))[:16]
	uid := "user-" + security.TokenHash(strings.ToLower(email))[:16]
	tx, e := db.Pool.Begin(context.Background())
	if e != nil {
		fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(context.Background(), "INSERT INTO organizations(id,name) VALUES($1,$2)", oid, org); e == nil {
		_, e = tx.Exec(context.Background(), "INSERT INTO users(id,organization_id,email,password_hash,display_name) VALUES($1,$2,$3,$4,$5)", uid, oid, email, hash, name)
	}
	if e == nil {
		_, e = tx.Exec(context.Background(), "INSERT INTO user_roles(user_id,role_id) VALUES($1,'role-owner')", uid)
	}
	var collectorToken string
	if e == nil && createCollectorToken {
		collectorToken, e = security.RandomToken(32)
		if e == nil {
			_, e = tx.Exec(context.Background(), "INSERT INTO collector_enrollment_tokens(id,organization_id,token_hash,created_by,expires_at) VALUES($1,$2,$3,$4,$5)", "enrollment-"+security.TokenHash(collectorToken)[:24], oid, security.TokenHash(collectorToken), uid, time.Now().Add(collectorTokenTTL))
		}
	}
	if e != nil {
		fatal(e)
	}
	if e = tx.Commit(context.Background()); e != nil {
		fatal(e)
	}
	fmt.Println("bootstrap owner created for organization", oid)
	if collectorToken != "" {
		fmt.Println("collector enrollment token (shown once):", collectorToken)
		fmt.Println("collector enrollment token expires at:", time.Now().Add(collectorTokenTTL).UTC().Format(time.RFC3339))
	}
}
func fatal(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
