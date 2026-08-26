# Production deployment

Use public DNS, HTTPS, and a reverse proxy such as the Caddy example in `deploy/`. Terminate TLS with an automatically renewed certificate, forward only from known proxy addresses, preserve request IDs, cap bodies, and use longer timeouts only for signed snapshot ingest. Do not expose PostgreSQL, MinIO/S3 administration, metrics, or collector credentials to the public internet.

Run PostgreSQL with TLS, point-in-time recovery, tested backups, connection limits, and separate application credentials. Use an S3-compatible service with bucket encryption, versioning where appropriate, private networking, lifecycle rules, and a least-privilege bucket policy. Supply session and master keys through a secret manager; never an env file committed to Git. Production startup rejects weak secrets, insecure origins, and debug mode.

Run migrations as a controlled pre-deploy step; the application advisory lock prevents concurrent application. Take a backup first. Deploy web/API images by immutable digest, verify Cosign identity and SBOM, then observe readiness, error rate, DB pool, signature failures, reconciliation latency, and connector freshness. Roll back application code only when the migration is backward compatible. Collectors use outbound HTTPS and dedicated credentials; Docker socket access belongs only on a hardened collector host or restricted proxy.

