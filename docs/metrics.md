# Metrics

`/metrics` exposes Prometheus text. Counters cover HTTP requests/failures, collector signature failures, and graph limit rejections; planned production collectors add DB pool gauges, asset/relationship totals, reconciliation results, sync durations, heartbeats, and graph latency histograms.

Do not use asset names, hostnames, emails, external IDs, request IDs, or other high-cardinality/sensitive values as labels. Protect metrics at the reverse proxy or private network. Alert ratios over windows rather than individual failures and correlate with readiness and logs.

