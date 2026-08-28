# Collectors

The collector is a separate Go process that enrolls, maintains local identity, sends heartbeats, executes one explicitly selected `DOCKER` or `KUBERNETES` connector, normalizes evidence, signs snapshots, and retries from a bounded local spool. Control-plane communication is outbound HTTPS; there is no administrative inbound channel. Connector type is persisted at enrollment and cannot be changed by reusing the identity directory. See [Kubernetes connector](kubernetes-connector.md) for the in-cluster deployment.

Collector identity, sequence, and spool files use `0600` on Unix. On Windows, deploy under a dedicated service account and restrict the data-directory ACL to that identity; verify ACLs during installation. `INFRAGRAPH_COLLECTOR_MAX_SPOOL_BYTES` defaults to 512 MiB and `INFRAGRAPH_COLLECTOR_MAX_SPOOL_AGE` defaults to seven days. Reaching either bound stops new discovery without silently deleting evidence; restore connectivity or perform an operator-reviewed recovery. Revocation blocks future heartbeats and snapshot ingest.

