# Authorization

Roles are Owner, Admin, Infra Admin, Asset Manager, Operator, Auditor, and Viewer. Permissions are explicit (`asset.read/manage/merge`, connector/collector/policy/finding operations, imports/exports, audit, users, and settings). Handlers check permission after authenticating the server-side session.

Organization is the tenant boundary. SQL predicates use the principal organization derived from session or collector credential. Resource lookup combines resource ID and organization; a browser payload cannot change the boundary. Cross-organization integration tests cover assets and establish the pattern required for relationships, connectors, collectors, snapshots, findings, changes, and exports.

