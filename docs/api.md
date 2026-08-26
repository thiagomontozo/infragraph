# API

User APIs are under `/api/v1`; collector trust endpoints are under `/collector/v1`. JSON failures always use `{ "error": { "code", "message", "requestId" } }` without stacks or SQL. Lists are paginated and capped. Mutations require cookie session, permission, CSRF, body limit, and where applicable an idempotency key.

Core domains are auth, organizations/users/roles, assets/types/relationships/graph, sites/owners/tags, connectors/collectors/snapshots/reconciliation, changes/findings/policies, imports/exports, audit, and settings. Graph reads accept bounded depth/nodes. Collector endpoints use a collector credential, never a browser session, and snapshots additionally require signature/replay validation. The normative public subset is in `openapi.yaml`; implementation-only health endpoints remain simple operational contracts.

