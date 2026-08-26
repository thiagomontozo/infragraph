# Secrets

Connector credentials and TOTP seeds use an envelope containing ciphertext, nonce, key version, and non-secret metadata. AES-256-GCM provides authenticated encryption; the 32-byte master key is external and versioned. Plaintext exists only in bounded memory while needed and is never logged, returned by list endpoints, or stored in audit payloads.

Production should inject keys from Vault or a cloud secret manager and rotate through a dual-read/new-write procedure: install a new version, re-encrypt records transactionally in batches, verify, then retire the old key after backups age out. Database theft without the master key should not reveal stored connector secrets; compromise of both requires credential rotation.

