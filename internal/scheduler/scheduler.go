package scheduler

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Lease struct {
	pool     *pgxpool.Pool
	owner    string
	duration time.Duration
}

func New(pool *pgxpool.Pool, owner string, duration time.Duration) *Lease {
	return &Lease{pool, owner, duration}
}
func (l *Lease) Acquire(ctx context.Context, name string) (bool, error) {
	var ok bool
	e := l.pool.QueryRow(ctx, `INSERT INTO scheduler_leases(name,owner_id,leased_until) VALUES($1,$2,now()+$3::interval) ON CONFLICT(name) DO UPDATE SET owner_id=EXCLUDED.owner_id,leased_until=EXCLUDED.leased_until,updated_at=now() WHERE scheduler_leases.leased_until<now() OR scheduler_leases.owner_id=EXCLUDED.owner_id RETURNING true`, name, l.owner, l.duration.String()).Scan(&ok)
	return ok, e
}
