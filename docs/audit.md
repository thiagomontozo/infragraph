# Audit

Security and administrative actions produce organization-scoped events with actor, action, resource, request ID, payload, time, previous hash, and event hash. Event hash is SHA-256 over the previous hash and canonical event content. Appending is serialized in the critical transaction so a missing link is detectable.

Audit is tamper-evident rather than tamper-proof. Periodically verify each chain, export checkpoints to independent immutable storage/SIEM, alert on breaks, and retain critical audit longer than ordinary observations. Never place passwords, tokens, private keys, raw snapshots, or decrypted connector configuration in payloads.

