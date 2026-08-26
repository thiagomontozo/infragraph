# Testing

Run in increasing cost: Go unit/package tests, PostgreSQL/MinIO integration, protocol/signature tests, labeled Docker discovery E2E, reconciliation/import invariants, frontend unit/build, Playwright, restore, bounded performance, then the full suite/race/security scanners. Tests use PostgreSQL—not SQLite—and synthetic data only.

Critical cases include successful versus failed missing detection, strong/weak identity behavior, conflict authority/claim retention, relationship changes, idempotency, tenant isolation, signature mutation/wrong/revoked/replayed collectors, graph cycles/limits, import bombs, Terraform minimization, CSV formula defense, and local object path traversal. All Docker scripts use `finally`/`trap` and targeted labels.

