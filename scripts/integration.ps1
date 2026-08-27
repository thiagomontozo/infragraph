$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
  docker compose -f compose.test.yml up -d --wait
  $env:INFRAGRAPH_TEST_DATABASE_URL = 'postgres://infragraph:test-only-password@localhost:55432/infragraph_test?sslmode=disable'
  $env:INFRAGRAPH_TEST_S3_ENDPOINT = 'localhost:59000'
  go test -count=1 ./internal/database ./internal/storage ./internal/reconcile ./internal/app ./internal/graph
} finally {
  Remove-Item Env:INFRAGRAPH_TEST_DATABASE_URL -ErrorAction SilentlyContinue
  Remove-Item Env:INFRAGRAPH_TEST_S3_ENDPOINT -ErrorAction SilentlyContinue
  docker compose -f compose.test.yml down -v --remove-orphans
  Pop-Location
}
