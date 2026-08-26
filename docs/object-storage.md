# Object storage

`ObjectStorage` provides Put/Get/Delete/Ready implementations for local filesystem and S3-compatible services. Local keys reject absolute paths and traversal and use pending files plus atomic rename. S3 objects are private, hashed with SHA-256 while streaming, and referenced from PostgreSQL by metadata rather than storing large bytes in tables.

Object storage is not ACID with PostgreSQL. Writers upload a pending object, verify size/hash, commit artifact metadata in a database transaction, then finalize; failures remove or tombstone the pending object. A periodic janitor may delete expired unreferenced pending objects. MinIO is development/test only; production uses a managed or operated S3-compatible service.

