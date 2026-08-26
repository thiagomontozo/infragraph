#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
gofmt -w ./cmd ./internal
go test ./cmd/... ./internal/...
(cd web && npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build)
