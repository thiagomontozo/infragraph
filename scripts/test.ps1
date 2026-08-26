$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
  gofmt -w ./cmd ./internal
  go test ./cmd/... ./internal/...
  docker run --rm --label com.infragraph.managed=true --label com.infragraph.purpose=test -v "${PWD}/web:/app" -w /app node:22.19.0-alpine3.22 sh -lc 'npm ci && npm run lint && npm run typecheck && npm run test:unit && npm run build'
} finally { Pop-Location }
