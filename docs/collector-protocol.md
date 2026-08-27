# Collector protocol

Versioned JSON Schema contracts live in `contracts/collector/v1`. Enrollment accepts a short-lived single-use token and Ed25519 public key, creates a connector bound to that collector, and returns the connector ID plus a separate credential once. Heartbeats report version, protocol, OS/architecture, capabilities, job count, and a non-secret health summary.

Snapshots bind organization, collector, connector, type, times, monotonically increasing sequence, observations, warnings/statistics, SHA-256 content hash, algorithm, and Ed25519 signature. The gateway rejects wrong binding/type, revoked credentials, invalid/mutated signatures, stale/future timestamps, replayed sequence, duplicate IDs with different content, contract-limit violations, and oversized bodies. Repeating the same snapshot ID and content hash returns the persisted status without applying it twice.

Verified snapshots are reconciled synchronously in one database transaction in the supported single-API topology. Assets, strong identities, observations, claims, relationships, changes, missing counters, reconciliation summary, snapshot status, and audit event commit together or roll back together.

