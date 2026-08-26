# Graph engine

GraphService supports neighbors, dependencies, dependents, path, impact, and subgraph operations. Every request carries the authenticated organization, `maxDepth`, `maxNodes`, a context deadline, and direction. The traversal records visited nodes to handle cycles and rejects limits above configured ceilings. Default ceilings are depth 6 and 500 nodes.

PostgreSQL recursive CTEs may be used for database-side traversal, but the same organization predicate, depth counter, path/cycle check, node limit, and statement timeout remain mandatory. The web client never loads the organization graph by default; it expands a selected subgraph lazily and displays an equivalent relationship table.

