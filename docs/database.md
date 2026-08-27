# Database

PostgreSQL is the transactional source for tenancy, authentication, canonical inventory, provenance, graph edges, reconciliation, findings, changes, jobs, policies, and audit. `pg_trgm` indexes names and external IDs; ordinary B-tree indexes cover organization/status/type and both relationship directions. Large artifact bytes remain in object storage.

Every application query must bind `organization_id`; identifiers alone are insufficient. Snapshot reconciliation and its audit append use one transaction. Unique constraints enforce external and strong source identity, collector fingerprint, snapshot idempotency, connector sequence/hash, relationship cardinality, change logical identity, and job idempotency. The current reconciliation path uses bounded per-observation statements and must be capacity-tested with representative snapshot sizes before operators raise protocol limits.

