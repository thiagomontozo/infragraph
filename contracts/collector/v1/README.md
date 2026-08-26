# InfraGraph Collector Protocol 1.0

Protocol 1.0 is an outbound-HTTPS, JSON protocol between a separately deployed read-only collector and the InfraGraph control plane. Enrollment uses a short-lived, single-use organization-scoped token. The collector generates an Ed25519 key pair locally, submits only its public key, receives a separate bearer credential, and signs every inventory snapshot. The control plane binds the credential, organization, collector ID, public key, monotonically increasing sequence, timestamp, snapshot ID, and content hash before staging a snapshot.

Compatibility follows the protocol major version: `1.x` is compatible, a supported but older collector may be `UPGRADE_RECOMMENDED`, another major version is `INCOMPATIBLE`, and unrecognized metadata is `UNKNOWN`. Collectors are not remote execution agents. Jobs select a compiled connector and bounded configuration; the protocol has no command, shell, script-upload, or executable-download primitive.

All schemas use JSON Schema 2020-12. Unknown fields are rejected by implementations at trust boundaries. Timestamps are UTC RFC 3339. Identifiers are opaque strings. Snapshot payloads are size limited and a replayed or stale envelope is rejected.

