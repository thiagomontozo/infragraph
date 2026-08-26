# Source of truth

InfraGraph stores each source claim before calculating effective state. A structured reconciliation policy assigns `AUTHORITATIVE`, `OBSERVED`, or `DECLARED` precedence by connector and optionally asset type, attribute, or relationship type. Selection is deterministic: highest configured authority, then freshness; equal values from multiple active sources produce an agreement explanation.

Disagreement is not hidden. Effective state includes the selected value, all claims, conflict flag, and reason such as “Terraform is authoritative for environment.” There is no opaque global accuracy percentage or arbitrary weighted score. Confidence is shown as freshness, identifier strength, agreement, and conflicts.

