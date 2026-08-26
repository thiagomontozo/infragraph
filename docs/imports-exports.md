# Imports and exports

CSV follows upload → stream parse → preview → validate → differences → confirm → transactional apply. Limits cover bytes, UTF-8, rows, column schema, and field length. JSON has byte/item/depth/schema limits and rejects unknown top-level fields. Terraform uses the stricter sanitizer described separately. Import jobs are idempotent and organization bound.

Exports support CSV, JSON, bounded graph snapshots, and inventory reports. CSV cells beginning with `=`, `+`, `-`, or `@` are prefixed with an apostrophe to reduce spreadsheet formula injection. Artifacts use opaque organization-prefixed object keys, expiration/retention metadata, authorization checks, and no cross-tenant bulk query.

