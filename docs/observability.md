# Observability

Structured `slog` records carry request ID and, when applicable, organization/user/asset/connector/collector/snapshot/reconciliation IDs. Secrets and raw bodies are excluded. Optional OpenTelemetry traces are disabled by default and must not prevent startup when no collector is configured. Pprof, if added later, remains disabled or loopback/admin only.

Alert on readiness down, exhausted DB pool, collector heartbeat overdue, repeated connector or reconciliation failures, snapshot signature failures, graph limit rejection spikes, object-storage errors, and stale critical connectors. Link logs, traces, metrics, audit, and SSE progress through request ID without using high-cardinality Prometheus labels.

