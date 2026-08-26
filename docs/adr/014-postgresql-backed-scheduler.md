# ADR 014: PostgreSQL-backed scheduler

Status: Accepted (2026-08-25).

Schedules and jobs use PostgreSQL rows plus leases/advisory locks. This avoids Redis and duplicate work across API replicas while retaining durable job state. Workers are bounded and leases expire after crashes. LISTEN/NOTIFY may signal state changes but is not the durable queue.

