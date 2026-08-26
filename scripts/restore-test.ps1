$ErrorActionPreference='Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
$dump = Join-Path $env:TEMP 'infragraph-restore-test.dump'
try {
  docker compose -f compose.test.yml up -d --wait postgres
  $env:INFRAGRAPH_TEST_DATABASE_URL = 'postgres://infragraph:test-only-password@localhost:55432/infragraph_test?sslmode=disable'
  go test -count=1 -run TestMigrationsConcurrencyAndTenantIsolation ./internal/database
  docker exec infragraph-test-postgres-1 pg_dump -U infragraph -d infragraph_test -Fc -f /tmp/infragraph.dump
  docker cp infragraph-test-postgres-1:/tmp/infragraph.dump $dump
  docker run -d --name infragraph-restore-postgres --label com.infragraph.managed=true --label com.infragraph.purpose=test -e POSTGRES_PASSWORD=restore -e POSTGRES_DB=restore postgres:17.6-alpine3.22 | Out-Null
  for($i=0;$i -lt 30;$i++){ docker exec infragraph-restore-postgres pg_isready -U postgres 2>$null; if($LASTEXITCODE -eq 0){break}; Start-Sleep -Seconds 1 }
  docker cp $dump infragraph-restore-postgres:/tmp/infragraph.dump
  docker exec infragraph-restore-postgres pg_restore --no-owner --no-privileges -U postgres -d restore /tmp/infragraph.dump
  docker exec infragraph-restore-postgres psql -v ON_ERROR_STOP=1 -U postgres -d restore -c "SELECT (SELECT count(*) FROM assets) AS assets, (SELECT count(*) FROM asset_relationships) AS relationships, (SELECT count(*) FROM users) AS users, (SELECT count(*) FROM reconciliation_policies) AS policies, (SELECT count(*) FROM audit_events) AS audit;"
} finally {
  docker rm -f infragraph-restore-postgres 2>$null | Out-Null
  docker compose -f compose.test.yml down -v --remove-orphans
  if (Test-Path -LiteralPath $dump) { Remove-Item -LiteralPath $dump }
  Remove-Item Env:INFRAGRAPH_TEST_DATABASE_URL -ErrorAction SilentlyContinue
  Pop-Location
}
