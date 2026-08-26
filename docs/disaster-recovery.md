# Disaster recovery

Declare disaster scope and freeze writes. Provision clean PostgreSQL/object storage, restore the selected database point and matching object versions, inject escrowed configuration/key versions, run migrations, verify core counts/tenant isolation/audit chains, sample evidence hashes, revoke all sessions, and admit one API replica. Resume schedulers only after inventory consistency checks.

Collectors whose credentials remain trusted reconnect outbound and replay bounded spools subject to sequence rules. After master-key, database, or collector-host compromise, re-enroll collectors and rotate infrastructure credentials instead. Validate DNS/TLS, readiness, metrics, reconciliation, exports, and a synthetic collector before user traffic. Record achieved RPO/RTO and deviations from operator targets.

