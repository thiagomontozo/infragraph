# Operations runbook

Start with `/health`, `/startup`, and `/ready`; readiness failure means database, migrations, object storage, or workers are unavailable. Check request-ID-correlated structured logs, PostgreSQL pool/locks/storage, object-store reachability, scheduler leases, and recent deploys. External connector outages do not make API readiness fail.

For collector offline, confirm expected heartbeat policy, collector process/disk spool, outbound DNS/TLS, credential revocation, protocol compatibility, and connector-specific API access. For reconciliation failure, freeze retirement/missing actions, retain the snapshot/error, compare the last successful run, and retry only after fixing the deterministic cause. For signature spikes, revoke suspect collector credentials and preserve evidence.

For DB pool exhaustion, reduce admission, find long transactions/graph/import requests, inspect limits and total replica pools, and avoid blind restarts. For object storage errors, pause new artifact jobs while inventory reads continue. Page on readiness down, sustained reconciliation failures, signature failures, stale critical collectors, and backup/restore failures; create a timestamped incident and use the incident playbooks.

