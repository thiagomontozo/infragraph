# Threat model

## Assets and attackers

Protected assets include inventory evidence, connector and user credentials, signing keys, master keys, tenant boundaries, effective state, history, artifacts, and audit continuity. Attackers may be external, a malicious tenant user, compromised admin, compromised collector/host, dependency maintainer, or operator with database/object-storage access.

## Threats and controls

- **Compromised admin or stolen session:** least privilege, privileged MFA, Secure/HttpOnly/SameSite cookies, CSRF, short expiry/revocation, audit, and rapid global session invalidation. Admin authority still permits harmful policy/merge changes; review audit externally.
- **Enrollment token/collector credential/private-key theft:** short-lived single-use hashed tokens, credential scope, key `0600`/restricted ACL, revocation, key fingerprint, rotation, rate limits, and alerts. Re-enroll after compromise.
- **Fake, mutated, replayed, stale, or oversized snapshot:** credential-to-org/collector binding, Ed25519 signature, SHA-256 content hash, unique ID/hash/sequence, timestamp window, max body/items/depth, bounded decoder, and transactional staging.
- **Malicious connector or Docker socket compromise:** compiled read-only adapters, no raw verb/path/command surface, output schema/size validation, separate host, restricted proxy, egress control. Socket read-only mount is explicitly not a security boundary.
- **Kubernetes credential theft:** minimal get/list/watch RBAC without Secret, ConfigMap data, exec/logs, token request, mutation, or cluster-admin; dedicated account and rotation.
- **Terraform leakage:** no raw state by default, output exclusion, positive attribute allowlist, sensitive-name denylist, preview, short artifact retention when explicitly enabled.
- **Cross-organization disclosure:** principal-derived organization, predicates and composite uniqueness, authorization tests across every resource domain, opaque object prefixes, no tenant ID trust from browser.
- **SSRF:** typed connector configuration, HTTPS requirements, allowed/private endpoint policy at deployment, no arbitrary URL proxy; infrastructure endpoints are resolved only inside collectors.
- **Malicious CSV/JSON or decompression bomb:** byte/row/item/depth limits, UTF-8/schema validation, unknown-field rejection, no automatic apply, formula-safe export, no compressed snapshot ingest by default.
- **Graph/query DoS:** configured depth/nodes, cycle set, timeouts, pagination, rate limits, statement deadline, no whole-graph initial render.
- **Credential/database/object-storage theft:** TLS, network isolation, least privilege, AES-GCM connector secrets, external keys, private buckets, backup encryption, rotation and incident playbooks.
- **Audit tampering:** SHA-256 previous/event chain, verification tooling, restricted writes, external log export. This detects many edits but is not tamper-proof against an attacker controlling all storage.
- **Dependency poisoning/supply-chain compromise:** lockfiles, Dependabot without automerge, CodeQL, govulncheck/npm audit/Trivy/gitleaks, minimal permissions, SBOM, provenance, digest pinning, keyless Cosign.
- **Incorrect merge or stale data:** no weak auto-merge, candidate review/history, deterministic authority, connector freshness, inconclusive status on failed runs, restore/repair runbook.
- **Vulnerable image:** multi-stage non-root images, dropped capabilities, read-only roots, critical Trivy gate, signed release images, rebuild cadence.

Residual risks include powerful admins, Docker-equivalent collector host control, process-local rate limits in multi-replica deployments, per-replica SSE fanout, and V1 merge reversal complexity. Operators must accept or mitigate these before production.

