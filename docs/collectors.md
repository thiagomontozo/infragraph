# Collectors

The collector is a separate Go process that enrolls, maintains local identity, sends heartbeats, executes only compiled connector types, normalizes evidence, signs snapshots, and retries from a bounded local spool. Control-plane communication is outbound HTTPS; there is no administrative inbound channel.

Collector identity files use `0600` on Unix. On Windows, deploy under a dedicated service account and restrict the file ACL to that identity; verify ACLs during installation. Configure maximum spool bytes/age and concurrent connector runs so outages cannot exhaust disk or goroutines. Revocation blocks future jobs and snapshot ingest and emits an audit event.

