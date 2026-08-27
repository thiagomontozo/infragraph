# Production deployment

Use public DNS, HTTPS, and a reverse proxy such as the Caddy example in `deploy/`. Start from `deploy/production.env.example`, keep the resulting `.env.production` outside version control, and replace image placeholders with verified immutable digests. Terminate TLS with an automatically renewed certificate, forward only from the explicitly configured proxy CIDR, preserve request IDs, cap bodies, and use longer timeouts only for signed snapshot ingest. Do not expose PostgreSQL, MinIO/S3 administration, metrics, or collector credentials to the public internet.

Run PostgreSQL with TLS, point-in-time recovery, tested backups, connection limits, and separate application credentials. Use an S3-compatible service with bucket encryption, versioning where appropriate, private networking, lifecycle rules, and a least-privilege bucket policy. Supply session and master keys through a secret manager; never an env file committed to Git. Production startup rejects weak secrets, insecure origins, and debug mode.

Run migrations as a controlled pre-deploy step; the application advisory lock prevents concurrent application. Take a backup first. Migration 0002 is additive, but its down migration deletes newly accumulated source-identity state and is not a routine rollback. Deploy web/API images by immutable digest, verify Cosign identity and SBOM, then observe readiness, error rate, DB pool, signature failures, reconciliation latency, spool age, and connector freshness. Roll back application code only when the migration is backward compatible. Collectors use outbound HTTPS and dedicated credentials; Docker socket access belongs only on a hardened collector host or restricted proxy.

This release candidate supports one API replica. Before promoting 1.0, record every item in the [production-readiness matrix](production-readiness.md), including a restore drill and capacity/soak results against the intended PostgreSQL and object-storage services.

