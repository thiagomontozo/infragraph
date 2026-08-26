# Asset model

An asset is the canonical representation of something believed to exist. Its lifecycle is `ACTIVE`, `STALE`, `MISSING`, `CONFLICTING`, `RETIRED`, or `UNKNOWN`. `STALE` follows connector-specific freshness; `MISSING` requires policy-sufficient successful observations without the external identity; `RETIRED` is explicit. A failed or partial snapshot cannot create missing state.

Types come from one registry, including hosts, VMs, containers/images, Kubernetes resources, applications, services, data systems, storage/network identities, certificates, volumes, cloud resources, and unknown. Sites, owners, tags, criticality, and constrained custom fields add governed context. `AssetSourceIdentity` maps `(organization, connector, externalId)` to the canonical asset while observations remain append-only evidence.

