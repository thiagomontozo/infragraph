# Collector protocol

Versioned JSON Schema contracts live in `contracts/collector/v1`. Enrollment accepts a short-lived single-use token and Ed25519 public key and returns a separate credential once. Heartbeats report version, protocol, OS/architecture, capabilities, job count, last success, and a non-secret health summary.

Snapshots bind organization, collector, connector, times, monotonically increasing sequence, observations, warnings/statistics, SHA-256 content hash, algorithm, and Ed25519 signature. The gateway rejects wrong binding, revoked credentials, invalid/mutated signatures, stale/future timestamps, replayed sequence/snapshot identity, duplicate IDs with different content, and oversized bodies. Same major protocol is compatible; UI may recommend upgrades for older supported versions.

