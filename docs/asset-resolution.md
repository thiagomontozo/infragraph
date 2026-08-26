# Asset resolution

Resolution prioritizes immutable identifiers: provider IDs, Kubernetes UIDs, Docker container IDs, VM/hardware UUIDs, Terraform provider IDs, and known external CMDB IDs. A matching strong identifier resolves to an existing asset unless a contradictory namespace or organization boundary invalidates the comparison.

Hostname, IP, and contextual MAC data are weak signals. A hostname match—especially across environments—creates `AssetMergeCandidate` rather than an automatic merge. Privileged review records reasons, reviewer, status, and timestamps. Manual merge is transactional and preserves identities, claims, relationships, history, and an audit event. Full automated reversal is intentionally limited in V1; operators use recorded merge history and a controlled administrative repair procedure.

