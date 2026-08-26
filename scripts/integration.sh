#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
trap 'docker compose -f compose.test.yml down -v --remove-orphans' EXIT
docker compose -f compose.test.yml up -d --wait
INFRAGRAPH_TEST_DATABASE_URL='postgres://infragraph:test-only-password@localhost:55432/infragraph_test?sslmode=disable' INFRAGRAPH_TEST_S3_ENDPOINT='localhost:59000' go test -count=1 ./internal/database ./internal/storage
