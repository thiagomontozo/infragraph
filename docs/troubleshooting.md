# Troubleshooting

**API will not start:** read the structured configuration error; production rejects missing/default session/master keys, insecure origins, and debug. Confirm database URL/TLS and migration permissions.

**Ready is 503:** health may still be 200. Check database connectivity, migration table/lock, object store bucket and worker startup. Do not make readiness depend on external connectors.

**Snapshot rejected:** compare collector ID/organization/credential, revocation, protocol major, Ed25519 public fingerprint, completed timestamp, sequence, content hash/signature, connector existence, and body size. Never bypass verification to recover.

**Asset not marked missing:** confirm the latest relevant runs succeeded completely and the connector-specific threshold was reached. Failed/partial runs intentionally do not count.

**Graph rejected:** lower depth/nodes or start from a narrower asset. A cycle is handled automatically; a timeout or cap is a safety result, not data corruption.

**Docker discovery empty:** confirm the collector—not API—can reach Docker and the synthetic/authorized containers carry the configured scope label. Do not remove the label filter to “fix” production discovery.

