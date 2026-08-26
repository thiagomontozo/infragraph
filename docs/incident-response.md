# Incident response

Use detect → contain → preserve → eradicate → recover → review. Record UTC timeline, request IDs, audit checkpoints, affected organizations, snapshots/artifacts, versions, and actions. Do not paste secrets or real infrastructure data into public tickets.

- **Stolen collector credential/private key:** revoke collector, block egress, preserve host evidence, invalidate queued jobs, re-enroll on a clean host, compare signed snapshots and reconcile from last trusted success.
- **Stolen master key:** stop decrypting writes, restrict DB/object access, rotate every encrypted credential/TOTP seed as applicable, install a new key version, invalidate sessions, re-enroll sensitive connectors, and assess backup exposure.
- **Database compromise:** isolate API, revoke DB credentials/sessions, preserve logs, restore from trusted backup/PITR, rotate master/session/collector credentials, verify audit chains and tenant queries.
- **Object storage exposure:** block bucket access, inventory accessed keys/version logs, rotate S3 credentials, remove public policy, restore/replace artifacts, and assume raw optional artifacts were disclosed.
- **Incorrect merge or reconciliation release:** pause workers, capture policy/run evidence, roll back compatible application code, restore affected logical state from history/backup, never delete provenance, then replay trusted snapshots.
- **Collector compromise:** revoke it and its infrastructure credentials, isolate the host, inspect Docker/Kubernetes access, reimage/re-enroll, and treat observations since compromise as untrusted.

