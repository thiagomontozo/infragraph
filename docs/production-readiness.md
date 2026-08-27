# Production readiness

InfraGraph remains **Production Candidate (`1.0.0-rc.1`)**. The supported candidate topology is one API replica behind HTTPS, managed PostgreSQL with TLS/PITR, private TLS-enabled S3-compatible storage, and separately isolated outbound collectors. A release must not be called Production Ready merely because unit tests pass.

## What the 2026-08-27 hardening closed

- Signed snapshots now reconcile assets, strong identities, claims, relationships, changes, missing state, runs, and audit records in one PostgreSQL transaction. Same-ID/same-hash retries are idempotent; conflicting reuse and non-monotonic sequences are rejected.
- Enrollment creates an explicitly collector-bound connector and returns its ID. The collector sends heartbeats, keeps a durable monotonic sequence, and retries a bounded on-disk snapshot spool.
- Graph traversal runs as bounded indexed PostgreSQL breadth-first queries instead of loading the tenant's full edge set into API memory.
- Session and CSRF token hashes are keyed by the external session secret. Forwarded client addresses are trusted only from configured CIDRs, rate-limit state is pruned, and production configuration rejects non-TLS PostgreSQL, local/non-TLS object storage, invalid origins, zero master keys, and unsafe concurrency.
- Readiness checks PostgreSQL and object storage. Audit appends serialize per organization. Integration tests exercise migrations, storage, collector enrollment, signed ingest, persistence, cross-source strong identity, conflict detection, absence handling, idempotency, and bounded graph traversal.

## Release gate matrix (53 gates)

The release owner records a link or artifact for every gate. “Not run” is not a pass. Code gates are automated where practical; environment gates require evidence from the target deployment.

### Source, build, and static validation

1. [ ] Working tree is clean and the release commit is reviewed.
2. [ ] `VERSION`, OpenAPI, package, image, changelog, and tag versions agree.
3. [ ] All Go files pass `gofmt` verification.
4. [ ] `go vet ./...` passes.
5. [ ] All three Go commands build with the supported Go version.
6. [ ] Frontend dependency installation is reproducible with `npm ci`.
7. [ ] Frontend lint passes with zero warnings.
8. [ ] Frontend TypeScript checking passes.
9. [ ] Frontend production build passes.
10. [ ] Docker Compose development, test, and production examples validate.

### Functional and recovery tests

11. [ ] Go unit/package suite passes uncached.
12. [ ] Authentication, CSRF, CORS, proxy trust, and security unit tests pass.
13. [ ] Snapshot signature mutation, wrong-key, binding, replay, and idempotency tests pass.
14. [ ] PostgreSQL migrations pass from an empty database.
15. [ ] Concurrent migration locking passes.
16. [ ] Collector enrollment-to-reconciliation integration passes.
17. [ ] Cross-source strong identity and conflict reconciliation passes.
18. [ ] Successful-absence and failed-collection missing-state invariants pass.
19. [ ] PostgreSQL tenant-isolation integration passes.
20. [ ] S3-compatible object-storage integration passes.
21. [ ] Synthetic labeled Docker discovery E2E passes.
22. [ ] Playwright browser smoke passes.
23. [ ] Backup and restore test passes against the release schema.
24. [ ] Bounded performance smoke passes with recorded inputs/results.

### Security and dependency gates

25. [ ] Go race detector passes on security/reconciliation/graph/import hot paths.
26. [ ] `govulncheck` reports no reachable known vulnerability at the configured threshold.
27. [ ] `npm audit --audit-level=high` passes.
28. [ ] Gitleaks history/worktree scan passes.
29. [ ] Trivy filesystem scan reports no unfixed critical finding.
30. [ ] Trivy scans API, collector, and web images with no unfixed critical finding.
31. [ ] CodeQL Go analysis passes.
32. [ ] CodeQL JavaScript/TypeScript analysis passes.
33. [ ] OpenSSF Scorecard workflow completes and findings are reviewed.
34. [ ] Threat model and security model reflect the release behavior.

### Container and supply-chain gates

35. [ ] API image builds for amd64 and arm64.
36. [ ] Collector image builds for amd64 and arm64.
37. [ ] Web image builds for amd64 and arm64.
38. [ ] Runtime images execute as non-root with dropped capabilities/read-only filesystems.
39. [ ] Release images are referenced by immutable digest in deployment configuration.
40. [ ] SPDX SBOMs and checksums are generated and retained.
41. [ ] Provenance attestations and keyless Cosign signatures verify for every image digest.

### Target-environment operational gates

42. [ ] Production configuration fail-closed test passes with secret-manager inputs.
43. [ ] TLS/DNS/reverse-proxy headers and trusted-proxy CIDRs are verified externally.
44. [ ] PostgreSQL TLS, least privilege, connection budget, PITR, and alerting are verified.
45. [ ] S3 TLS, encryption, private access, lifecycle, versioning, and least privilege are verified.
46. [ ] Metrics/log collection and request-ID correlation are verified without secret leakage.
47. [ ] Readiness/liveness behavior and rollback procedure are exercised.
48. [ ] Restore drill meets the approved RPO/RTO using production-equivalent backup data.
49. [ ] Capacity/soak test meets approved latency, error-rate, and resource budgets.

### Governance and release gates

50. [ ] Main branch rules require CI, CodeQL, and Security with review/no force-push policy.
51. [ ] All workflows for the exact release commit complete successfully on GitHub.
52. [ ] Known limitations, runbooks, incident contacts, and upgrade notes are approved by the operator.
53. [ ] Signed `v1.0.0` tag/release is created only after gates 1–52 have recorded evidence.

## Current blockers to `1.0.0`

- Gates 48–49 need target-environment recovery objectives and capacity budgets from the operator; the repository cannot invent these acceptance thresholds.
- Gate 50 requires repository ruleset/branch-protection administration and confirmation that required checks are enforced.
- The current UI is an operational read/preview surface. Administrative workflows such as user/MFA lifecycle, connector/policy management, import apply, export generation, and merge review are not complete. Either implement and test them before calling the broad product scope 1.0, or explicitly approve a narrower read-only collector/inventory production scope.
- Kubernetes discovery has a tested connector library but is not yet wired into the shipped collector executable. Docker discovery is the only executable collector path in this release candidate.
- Multi-replica API remains unsupported because admission/rate limiting is process-local. Use one API replica until shared coordination is implemented and tested.

Do not change `VERSION` or create a `v1.0.0` tag while any blocker above remains unresolved.
