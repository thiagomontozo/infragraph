# Database

PostgreSQL is the transactional source for tenancy, authentication, canonical inventory, provenance, graph edges, reconciliation, findings, changes, jobs, policies, and audit. `pg_trgm` indexes names and external IDs; ordinary B-tree indexes cover organization/status/type and both relationship directions. Large artifact bytes remain in object storage.

Every application query must bind `organization_id`; identifiers alone are insufficient. Snapshot reconciliation, merge, policy update, retirement, change/finding persistence, and critical audit use transactions. Unique constraints enforce external source identity, collector fingerprint, snapshot idempotency, connector sequence/hash, relationship cardinality, change logical identity, and job idempotency. Batch APIs are used for high-volume reconciliation.

