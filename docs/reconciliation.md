# Reconciliation

The pipeline is receive → size/schema/signature validation → immutable staging → identity resolution → claims/relationship correlation → policy evaluation → change/finding generation → one logical database commit → successful status. Raw artifact upload, when enabled, uses a pending key and hash; failure triggers cleanup/tombstone because PostgreSQL and object storage are not jointly transactional.

`snapshotId`, connector sequence, content hash, unique identities, reconciliation run, and logical change identities make retries idempotent. Batches avoid per-row round trips. Missing counters advance only after a successful complete snapshot for the relevant connector and policy. Partial, failed, cancelled, and rejected snapshots never create absence conclusions.

