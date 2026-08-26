#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
dump=$(mktemp)
cleanup(){ docker rm -f infragraph-restore-postgres >/dev/null 2>&1 || true; docker compose -f compose.test.yml down -v --remove-orphans; rm -f "$dump"; }
trap cleanup EXIT
docker compose -f compose.test.yml up -d --wait postgres
INFRAGRAPH_TEST_DATABASE_URL='postgres://infragraph:test-only-password@localhost:55432/infragraph_test?sslmode=disable' go test -count=1 -run TestMigrationsConcurrencyAndTenantIsolation ./internal/database
docker exec infragraph-test-postgres-1 pg_dump -U infragraph -d infragraph_test -Fc -f /tmp/db.dump
docker cp infragraph-test-postgres-1:/tmp/db.dump "$dump"
docker run -d --name infragraph-restore-postgres --label com.infragraph.managed=true --label com.infragraph.purpose=test -e POSTGRES_PASSWORD=restore -e POSTGRES_DB=restore postgres:17.6-alpine3.22 >/dev/null
until docker exec infragraph-restore-postgres pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
docker cp "$dump" infragraph-restore-postgres:/tmp/db.dump
docker exec infragraph-restore-postgres pg_restore --no-owner --no-privileges -U postgres -d restore /tmp/db.dump
docker exec infragraph-restore-postgres psql -v ON_ERROR_STOP=1 -U postgres -d restore -c 'SELECT (SELECT count(*) FROM assets) AS assets, (SELECT count(*) FROM asset_relationships) AS relationships, (SELECT count(*) FROM users) AS users, (SELECT count(*) FROM reconciliation_policies) AS policies, (SELECT count(*) FROM audit_events) AS audit;'
