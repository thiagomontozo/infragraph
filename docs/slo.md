# Suggested service objectives

These are operator-configurable targets, not an observed SLA: API successful availability 99.9% monthly excluding planned maintenance; 95% of ordinary reads under 500 ms; 95% of bounded depth-3 graph requests under 2 s; successful snapshot-to-effective-state reconciliation within 10 minutes; critical collector heartbeat freshness within twice its configured interval; recovery point objective 15 minutes and recovery time objective 4 hours when the selected PostgreSQL/S3 platform supports them.

Measure at the user/API boundary and reconciliation timestamps. Define error-budget policy, exclusions, paging rules, and dependencies before adopting targets. Revisit after representative load and recovery exercises.

