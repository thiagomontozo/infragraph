# Version 1.0 product scope

InfraGraph 1.0 is a read-only infrastructure inventory and dependency-inspection product. “Read-only” means it does not mutate managed infrastructure and its supported web UI does not administer the control plane. Signed collectors can still write observations to InfraGraph because reconciliation is the product's ingestion boundary, not an infrastructure mutation.

## Supported

- Authenticate and inspect organization-scoped inventory.
- Search canonical assets and inspect effective state.
- Traverse bounded relationships and dependencies.
- Inspect changes, findings, connectors, collectors, and tamper-evident audit records.
- Validate and preview CSV, JSON, and allowlisted Terraform-state inputs without applying them.
- Enroll collectors through bootstrap-issued tokens and ingest signed snapshots.

## Explicitly excluded

- User, role, recovery-code, or MFA lifecycle administration in the web UI.
- Creating, editing, disabling, or rotating connectors and source-authority policies in the web UI.
- Applying import previews to effective inventory.
- Generating downloadable exports.
- Manual merge review, reversal, or conflict resolution.
- Starting, stopping, changing, or deleting infrastructure.
- Arbitrary commands, scripts, or remote execution.

Excluded functions are not hidden feature flags and are not supported through undocumented endpoints. They are post-1.0 roadmap work and require their own security and tenant-isolation acceptance criteria.

The normative decision and its consequences are recorded in [ADR 021](adr/021-v1-read-only-product-scope.md).
