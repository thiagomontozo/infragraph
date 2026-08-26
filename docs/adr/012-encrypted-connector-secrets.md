# ADR 012: Encrypted connector secrets

Status: Accepted (2026-08-25).

Connector/TOTP secrets use AES-256-GCM envelopes with random nonce, authenticated key version, and external master key. The database never stores plaintext or the master key. The interface anticipates Vault and cloud secret managers. Rotation complexity is accepted to prevent a database-only compromise from exposing credentials.

