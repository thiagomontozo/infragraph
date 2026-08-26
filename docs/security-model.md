# Security model

InfraGraph assumes the browser, collector host, infrastructure API, network, database, and object storage can fail independently. Defense is layered: HTTPS, server-side sessions, MFA/RBAC/CSRF, organization predicates, limited payloads, deterministic parsers, encrypted credentials, signed snapshots, replay checks, bounded traversal/workers, audit chaining, and minimal containers.

Data minimization is a product invariant. Docker env/logs/secrets, Kubernetes secrets/tokens/ConfigMap data, Terraform raw state/arbitrary attributes, and real infrastructure fixtures are excluded. Read-only connectors do not guarantee a harmless credential—especially Docker socket access—so collectors require host isolation and least privilege. Audit is tamper-evident, not tamper-proof; export chains to separate immutable monitoring for stronger assurance.

