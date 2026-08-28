# Configuration

| Variable | Secret | Required/default | Production recommendation |
|---|---:|---|---|
| `INFRAGRAPH_ENV` | No | `development` | Set `production`. |
| `INFRAGRAPH_HTTP_ADDR` | No | `:8080` | Bind private interface behind TLS proxy. |
| `INFRAGRAPH_DATABASE_URL` | Yes | Required | TLS, least-privilege app role, secret manager. |
| `INFRAGRAPH_SESSION_SECRET` | Yes | Required in production | Random ≥32 characters; rotate with session invalidation. |
| `INFRAGRAPH_MASTER_KEY` | Yes | Required in production | Base64 32 bytes, external and versioned. |
| `INFRAGRAPH_ALLOWED_ORIGINS` | No | Empty | Explicit HTTPS origins; never `*`. |
| `INFRAGRAPH_OBJECT_STORAGE_TYPE` | No | `local` | `s3` for production. |
| `INFRAGRAPH_OBJECT_STORAGE_PATH` | No | `./data/objects` | Local development only. |
| `INFRAGRAPH_S3_ENDPOINT/BUCKET` | No | None | Private TLS service/bucket. |
| `INFRAGRAPH_S3_ACCESS_KEY/SECRET_KEY` | Yes | None | Least-privilege bucket credential. |
| `INFRAGRAPH_S3_USE_TLS` | No | `false` | `true`. |
| `INFRAGRAPH_MAX_GRAPH_DEPTH` | No | `6` | Lower only after workload analysis. |
| `INFRAGRAPH_MAX_GRAPH_NODES` | No | `500` | Keep bounded. |
| `INFRAGRAPH_MAX_IMPORT_BYTES` | No | 10 MiB | Align reverse proxy limit. |
| `INFRAGRAPH_MAX_SNAPSHOT_BYTES` | No | 50 MiB | Align gateway/proxy; avoid compression bombs. |
| `INFRAGRAPH_MAX_CONCURRENT_RECONCILIATIONS` | No | `4` | Tune with DB pool/CPU. |
| `INFRAGRAPH_COLLECTOR_HEARTBEAT_INTERVAL` | No | `30s` | Match alert freshness policy. |
| `INFRAGRAPH_OTEL_ENABLED/ENDPOINT` | Endpoint may be sensitive | `false`/empty | TLS endpoint; no secrets in attributes. |
| `INFRAGRAPH_DEBUG` | No | `false` | Must remain false. |

Production validation also requires PostgreSQL TLS (`sslmode` must not be `disable`), complete TLS-enabled S3 settings, a non-zero random master key, safe reconciliation concurrency (1–64), and syntactically valid trusted proxy CIDRs. `INFRAGRAPH_TRUSTED_PROXY_CIDRS` is a comma-separated allowlist; forwarded client addresses are ignored unless the direct peer belongs to it.

Collector-only variables include `INFRAGRAPH_CONTROL_PLANE_URL`, the one-use `INFRAGRAPH_ENROLLMENT_TOKEN`, `INFRAGRAPH_COLLECTOR_DATA_DIR`, `INFRAGRAPH_COLLECTOR_NAME`, `INFRAGRAPH_CONNECTOR_NAME`, `INFRAGRAPH_CONNECTOR_TYPE`, `INFRAGRAPH_COLLECTOR_INTERVAL`, `INFRAGRAPH_COLLECTOR_MAX_SPOOL_BYTES`, and `INFRAGRAPH_COLLECTOR_MAX_SPOOL_AGE`. Docker uses `INFRAGRAPH_DOCKER_SOCKET` and `INFRAGRAPH_DOCKER_LABEL_SCOPE`. Kubernetes uses the API URL, token/CA file, cluster identity/name, timeout, page size, and resource cap documented in [Kubernetes connector](kubernetes-connector.md).

