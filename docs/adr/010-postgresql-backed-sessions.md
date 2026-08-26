# ADR 010: PostgreSQL-backed sessions

Status: Accepted (2026-08-25).

Opaque user sessions are hashed and stored in PostgreSQL with expiry/revocation and organization binding. This avoids browser JWT persistence and supports immediate revocation across replicas without Redis. Database availability is required for authenticated traffic, consistent with the inventory control plane's readiness dependency.

