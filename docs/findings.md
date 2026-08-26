# Findings

Findings surface inventory-quality conditions: undocumented assets, declared-but-not-observed, stale assets/connectors, source conflicts, orphan relationships, unowned critical assets, and duplicate candidates. Status is `OPEN`, `ACKNOWLEDGED`, `RESOLVED`, `ACCEPTED`, or `INCONCLUSIVE`; priority is informational through high, never automatically “critical”.

Every finding explains why it exists and cites the observations/policy that caused it. For example, a declared asset may be absent from three successful Docker runs. An unsuccessful run produces an inconclusive source condition, not a missing finding.

