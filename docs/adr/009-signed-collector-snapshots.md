# ADR 009: Signed collector snapshots

Status: Accepted (2026-08-25).

Collectors generate Ed25519 keys locally and sign canonical envelopes. The control plane stores the public key/fingerprint and also requires a revocable collector credential. Sequence, timestamp, snapshot ID, hash, signature, and identity binding mitigate tampering and replay. Signatures establish collector-key origin, not truthfulness of a compromised source.

