package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type DB struct{ Pool *pgxpool.Pool }

func Open(ctx context.Context, url string) (*DB, error) {
	if url == "" {
		return nil, errors.New("database URL required")
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		return nil, e
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return nil, e
	}
	if e = pool.Ping(ctx); e != nil {
		pool.Close()
		return nil, e
	}
	return &DB{pool}, nil
}
func (d *DB) Close() { d.Pool.Close() }
func (d *DB) Migrate(ctx context.Context) error {
	conn, e := d.Pool.Acquire(ctx)
	if e != nil {
		return e
	}
	defer conn.Release()
	if _, e = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", int64(0x494E465241)); e != nil {
		return e
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", int64(0x494E465241))
	if _, e = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); e != nil {
		return e
	}
	entries, e := migrationFS.ReadDir("migrations")
	if e != nil {
		return e
	}
	var names []string
	for _, v := range entries {
		if strings.HasSuffix(v.Name(), ".up.sql") {
			names = append(names, v.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.Split(name, "_")[0]
		var exists bool
		if e = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists); e != nil {
			return e
		}
		if exists {
			continue
		}
		sql, e := migrationFS.ReadFile("migrations/" + name)
		if e != nil {
			return e
		}
		tx, e := conn.Begin(ctx)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(sql)); e == nil {
			_, e = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", version)
		}
		if e != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, e)
		}
		if e = tx.Commit(ctx); e != nil {
			return e
		}
	}
	return nil
}
func (d *DB) Ready(ctx context.Context) error { return d.Pool.Ping(ctx) }
