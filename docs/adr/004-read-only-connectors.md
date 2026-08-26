# ADR 004: Read-only connectors

Status: Accepted (2026-08-25).

V1 observes and reconciles; it does not mutate infrastructure. Connector interfaces expose typed discovery capabilities only, not arbitrary requests or commands. Read-only credentials still carry risk, especially Docker, so least privilege and host isolation remain necessary. Remediation automation would require a separate future threat model and product boundary.

