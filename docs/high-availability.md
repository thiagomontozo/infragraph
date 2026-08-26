# High availability

API replicas are stateless behind a health-aware load balancer. PostgreSQL provides durable sessions, leases, inventory, and coordination; S3-compatible storage provides artifacts. Run database and object storage with their vendor-supported HA, backups, and failure-domain separation. Collectors retry outbound delivery through a bounded spool.

Scheduler acquisition uses PostgreSQL leases/advisory coordination so one replica schedules a connector occurrence. Lightweight PostgreSQL LISTEN/NOTIFY may fan out SSE notifications but is not a durable queue; clients reconnect and read authoritative job state. Deploy rolling updates only across schema/protocol-compatible versions.

