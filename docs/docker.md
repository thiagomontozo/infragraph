# Docker deployments

Dockerfiles use separate build/runtime stages, pinned stable base versions, non-root runtime identities, no shell in Go runtime images, and read-only-root compatibility. Compose drops all capabilities, enables `no-new-privileges`, avoids privileged/host network/PID/filesystem access, caps test resources, and labels managed resources for targeted cleanup.

The optional collector alone mounts the Docker socket. This conveys Docker-equivalent host risk despite a read-only mount flag; use a restricted API proxy or dedicated host. `compose.yml` is development-only. `deploy/compose.production.example.yml` assumes external managed PostgreSQL/S3 and Caddy TLS; inject secrets outside Compose source.

