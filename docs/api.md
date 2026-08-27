# API

User APIs are under `/api/v1`; collector trust endpoints are under `/collector/v1`. JSON failures always use `{ "error": { "code", "message", "requestId" } }` without stacks or SQL. Lists are paginated and capped. Browser mutations are limited to login/logout and bounded import preview; they require the applicable session, permission, CSRF, and body-limit controls.

The version 1.0 browser API covers authentication, overview, assets/types, bounded relationships/graph, changes, findings, connector/collector inspection, import preview, and audit inspection. It does not expose user/MFA administration, connector/policy mutation, import apply, exports, or merge review. Collector endpoints use a collector credential, never a browser session, and snapshots additionally require signature/replay validation. The normative public contract is in `openapi.yaml`; implementation-only health endpoints remain simple operational contracts. See [version 1.0 product scope](product-scope.md).

