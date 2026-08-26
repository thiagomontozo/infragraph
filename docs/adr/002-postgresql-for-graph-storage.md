# ADR 002: PostgreSQL for graph storage

Status: Accepted (2026-08-25).

Assets and typed edges remain in PostgreSQL. Recursive CTEs, directional indexes, cycle checks, and bounded traversal satisfy initial graph needs while keeping transactions, backup, tenancy, and operations in one database. Neo4j is not a V1 dependency. GraphService isolates traversal semantics so a second engine can be evaluated when measured scale justifies it.

