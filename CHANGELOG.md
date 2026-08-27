# Changelog

All notable changes are documented here using [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and Semantic Versioning.

## [Unreleased]

### Added

- Initial InfraGraph production-candidate platform: living inventory, source claims, bounded graph, deterministic reconciliation, secure collector protocol, read-only Docker/Kubernetes connectors, minimized imports, operational UI, PostgreSQL/Object Storage foundations, tests, documentation, and CI/CD.
- Persistent transactional snapshot reconciliation with source observations, strong identities, claims, relationships, changes, missing-state handling, reconciliation runs, and audit events.
- Collector heartbeats, collector-bound connector enrollment, durable monotonic sequences, and a bounded retry spool.
- Integration coverage for signed ingest, retry idempotency, cross-source identity/conflicts, absence handling, and PostgreSQL graph traversal.
- ADR 021 and a normative version 1.0 product-scope document defining the supported read-only inventory surface.

### Changed

- Graph traversal now uses a bounded recursive PostgreSQL query instead of loading all tenant relationships into API memory.
- Production configuration now requires TLS-enabled PostgreSQL and S3-compatible storage and validates trusted proxy CIDRs and concurrency bounds.
- Session and CSRF token hashes are keyed with the configured session secret; upgrading invalidates existing pre-hardening sessions.
- Readiness now includes object-storage availability, and audit-chain appends serialize per organization.
- Release documentation now contains the explicit 53-gate matrix and precise remaining production blockers.
- The web navigation and Asset 360 tabs now expose only implemented API-backed views; unsupported administrative placeholders, fake source claims, and inactive mutation buttons were removed.

[Unreleased]: https://github.com/thiagomontozo/infragraph/compare/v1.0.0-rc.1...HEAD

