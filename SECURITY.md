# Security policy

## Supported versions

Until 1.0.0, security fixes target the current `1.0.0-rc.x` line. After general availability, the latest minor release is supported; older lines receive fixes only when explicitly announced in release notes.

## Reporting a vulnerability

Use the repository **Security** tab and open a private GitHub Security Advisory. Do not disclose a suspected vulnerability in a public issue, discussion, pull request, log excerpt, or fixture. Include affected version, prerequisites, impact, a minimal synthetic reproduction, and suggested remediation if known. Never attach real collector credentials, private keys, database URLs, snapshots, or infrastructure data.

Maintainers will coordinate acknowledgment, validation, remediation, advisory publication, and credit. This document intentionally does not invent an email address or response-time guarantee.

## Security boundaries

The API must not receive a Docker socket. Collectors are read-only and are not remote shells. Organization isolation is enforced from the authenticated session, not browser input. A valid admin remains powerful; use MFA, least privilege, separate collector hosts, restricted Docker proxies, network isolation, encrypted backups, and external secret management.

