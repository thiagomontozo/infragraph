# ADR 021: Version 1.0 has a read-only inventory product scope

## Status

Accepted on 2026-08-27.

## Context

The control plane already provides authenticated, organization-scoped inventory queries, bounded graph traversal, operational records, collector-driven reconciliation, and safe import previews. It does not provide complete mutation APIs for user/MFA lifecycle, connector or policy administration, import apply, export generation, or merge review.

Presenting placeholder navigation for those workflows made the release candidate appear broader than its tested API contract. Building every administrative subsystem would also combine several independently security-sensitive projects into the 1.0 release.

## Decision

InfraGraph 1.0 is an inventory, evidence, and dependency-inspection product. The supported web surface is:

- overview and canonical asset search;
- effective asset details and bounded relationship/dependency traversal;
- changes, findings, connectors, collectors, and audit inspection;
- CSV, JSON, and Terraform-state validation/preview without apply;
- collector-driven ingestion and reconciliation through the signed collector protocol.

Version 1.0 does not include control-plane user/MFA administration, connector or authority-policy mutation, import apply, export generation, merge review/reversal, infrastructure mutation, or arbitrary remote execution. Those capabilities must be proposed, permissioned, audited, and tested independently after 1.0.

Bootstrap remains the supported way to create the initial owner and collector enrollment token. Connector lifecycle after enrollment is operational configuration, not a web administration workflow.

## Consequences

- The web application exposes only routes backed by supported API endpoints.
- Buttons, tabs, sample claims, and generic routes that implied unsupported mutations or data were removed.
- Import pages say "preview only" and cannot modify effective inventory.
- The narrower scope can be promoted independently of future administrative workflows, provided all remaining production gates pass.
- Adding an administrative capability requires an API contract, authorization model, CSRF/audit coverage, tenant-isolation tests, UI states, operator documentation, and a separate ADR when the trust boundary changes.
