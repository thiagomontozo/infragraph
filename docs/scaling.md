# Scaling

Scale reads with API replicas and PostgreSQL tuning; keep writes transactional and batch observations/claims. Index organization plus common filters, inspect query plans, cap pool connections across replicas, and enforce graph/import/snapshot limits. Artifact traffic streams to object storage instead of memory/database.

V1 rate limiting is process-local and SSE fanout is per replica. That is acceptable for a controlled single-instance production deployment. Horizontal deployments must add PostgreSQL-backed counters/advisory coordination or an ingress limiter and use LISTEN/NOTIFY plus state replay for SSE. Worker queues remain bounded; PostgreSQL job rows are durable and leases prevent duplicate ownership.

