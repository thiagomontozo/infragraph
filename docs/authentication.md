# Authentication

Local passwords use Argon2id with per-password random salt, 64 MiB memory, three iterations, and parallelism two. Sessions are opaque random tokens; only a SHA-256 token hash is stored in PostgreSQL. Cookies are HttpOnly, Secure in production, SameSite=Strict, path scoped, expiring, and revocable. Session tokens and JWTs are never stored in browser localStorage.

Mutations require a separate random CSRF value supplied through a cookie/header double-submit binding to the server-side session hash. Password change revokes other sessions. TOTP uses the interoperable 30-second SHA-1 algorithm with a one-step clock window; its secret is AES-GCM encrypted under an external master key. Recovery codes are random, one-use, and hashed at rest. Privileged roles may require MFA.

