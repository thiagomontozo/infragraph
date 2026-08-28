# Testing

Run in increasing cost: Go unit/package tests, PostgreSQL/MinIO integration, protocol/signature tests, labeled Docker discovery E2E, pinned `kind` Kubernetes discovery E2E, reconciliation/import invariants, frontend unit/build, Playwright, restore, bounded performance, then the full suite/race/security scanners. Tests use PostgreSQL—not SQLite—and synthetic data only.

On Windows networks that perform TLS inspection, set `INFRAGRAPH_NPM_CA_FILE` to the inspecting organization's PEM CA bundle before `scripts/test.ps1`. Image builds accept the same bundle through BuildKit secret `--secret id=npm_ca,src=...`. Never disable npm TLS verification or commit the private-network CA bundle.

Critical cases include successful versus failed missing detection, strong/weak identity behavior, conflict authority/claim retention, relationship changes, idempotency, tenant isolation, signature mutation/wrong/revoked/replayed collectors, graph cycles/limits, import bombs, Terraform minimization, CSV formula defense, and local object path traversal. All Docker scripts use `finally`/`trap` and targeted labels.

Kubernetes E2E downloads the pinned official `kind` binary with SHA-256 verification and uses a digest-pinned node image. It applies the production RBAC, discovers a real Deployment/ReplicaSet/Pod/Service topology through the TLS API, and creates a Secret marker that must never appear in observations. Run `scripts/kubernetes-e2e.sh` on Linux or `scripts/kubernetes-e2e.ps1` on Windows with Docker and `kubectl` available.

