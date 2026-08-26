# ADR 013: Object storage for artifacts

Status: Accepted (2026-08-25).

Imports, exports, evidence bundles, and optional sanitized raw snapshots belong in local/S3-compatible object storage, not PostgreSQL byte columns. PostgreSQL stores hashes and lifecycle metadata. A pending/final compensation protocol handles the absence of distributed ACID. Raw snapshots are disabled by default and short-retained when enabled.

