# Imports and exports

Version 1.0 supports upload → bounded parse → validate → preview. A preview never changes effective inventory. Limits cover bytes, UTF-8, rows, column schema, and field length. JSON has byte/item/depth/schema limits and rejects unknown top-level fields. Terraform uses the stricter sanitizer described separately.

Transactional import apply and CSV/JSON/graph/report exports are not version 1.0 capabilities. They remain design requirements for post-1.0 work: idempotent organization-bound jobs, explicit confirmation, formula-injection protection, opaque organization-prefixed object keys, retention metadata, authorization checks, audit events, and tenant-isolation tests.

