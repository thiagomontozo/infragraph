# Contributing to InfraGraph

InfraGraph is a Go modular monolith plus a React application and separate Go collector. Read `docs/architecture.md`, the relevant ADRs, and `docs/security-model.md` before changing a trust boundary.

Use Go 1.26.6+, Node 22+, PostgreSQL 17, Docker, and Compose v2. Copy `.env.example` only for local development. Run unit tests first, then `scripts/integration`, labeled Docker E2E, frontend/Playwright, recovery, and performance smoke. Format Go with `gofmt`; run lint, typecheck, `go vet`, race tests, dependency audits, and `git diff --check`.

Connectors must be compiled, bounded, read-only adapters. They may not accept arbitrary verbs, paths, commands, scripts, or executables. Minimize fields, exclude secrets, document required permissions, use synthetic fixtures, and preserve provenance. New identifiers must state whether they are strong or weak and why.

Use conventional, focused commits such as `feat:`, `fix:`, `test:`, `docs:`, or `ci:`. Pull requests must explain behavior, migrations, backward compatibility, tenant impact, data collection changes, threat-model changes, tests, and documentation. Never commit credentials, private keys, raw production snapshots, or real Terraform state.
