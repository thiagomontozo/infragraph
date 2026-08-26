# ADR 011: Argon2id authentication

Status: Accepted (2026-08-25).

Local passwords use Argon2id with random salts and explicit memory/time/parallelism parameters. It provides memory-hard resistance appropriate for stored passwords. Parameters are encoded for future migration; successful login can rehash when policy increases. Passwords never enter logs/audit.

