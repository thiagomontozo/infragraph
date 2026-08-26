# ADR 019: Tamper-evident audit chain

Status: Accepted (2026-08-25).

Each organization audit event hashes canonical content with the previous event hash. Verification detects many deletions/edits/reordering. It is explicitly not tamper-proof against an attacker controlling database and checkpoints; production exports checkpoints to separate immutable monitoring. Append occurs with the critical transaction.

