# Migrations

Migrations are ordered immutable `.up.sql`/`.down.sql` files embedded in the API. `schema_migrations` records versions and a PostgreSQL advisory lock serializes runners across replicas. Each new migration is transactional where PostgreSQL permits it and is never edited after release.

Migration 0002 adds persistent absence counters, relationship source identities, and cross-connector strong identities. It is additive; its down migration removes those new reconciliation structures and must not be used after production data has entered them.

Before applying in production: validate backup, inspect locks/disk, test against a restored copy, stop incompatible writers, run once, verify readiness/schema, then deploy. Down migration 0001 removes all InfraGraph data and is for disposable environments only. Irreversible data transforms must be labeled and provide a forward repair procedure rather than a misleading rollback.

