package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/database"
	"github.com/thiagomontozo/infragraph/internal/security"
	"os"
	"strings"
)

func main() {
	var dbURL, org, name, email, password string
	flag.StringVar(&dbURL, "database-url", os.Getenv("INFRAGRAPH_DATABASE_URL"), "PostgreSQL URL")
	flag.StringVar(&org, "organization", "", "organization name")
	flag.StringVar(&name, "name", "", "owner display name")
	flag.StringVar(&email, "email", "", "owner email")
	flag.StringVar(&password, "password", "", "owner password (prefer prompt/secret injection)")
	flag.Parse()
	if dbURL == "" || org == "" || name == "" || email == "" || password == "" {
		fmt.Fprintln(os.Stderr, "all flags are required")
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
	if e != nil {
		fatal(e)
	}
	if e = tx.Commit(context.Background()); e != nil {
		fatal(e)
	}
	fmt.Println("bootstrap owner created for organization", oid)
}
func fatal(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
