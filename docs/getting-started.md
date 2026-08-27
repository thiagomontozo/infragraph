# Getting started

Install Docker and Compose v2. Clone the repository, copy `.env.example` only if running binaries outside Compose, then run `docker compose up --build`. The development stack exposes the web UI on port 5173 and API on 8080. Its passwords are public development defaults and must never be promoted.

Create the first organization and owner with `cmd/infragraph-bootstrap`; the command requires all values explicitly and hashes the password with Argon2id. Pass `-create-collector-token` to print one 15-minute enrollment token (or set `-collector-token-ttl` between one minute and 24 hours). The token is shown once, stored only as a hash, and should be injected into the collector as `INFRAGRAPH_ENROLLMENT_TOKEN` on its first start. Enrollment creates and returns the collector-bound connector ID.

Run `scripts/test`, `scripts/integration`, and `scripts/e2e` before changes are submitted. Cleanup scripts only target resources labeled `com.infragraph.managed=true`.

