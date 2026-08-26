# NetScope integration

InfraGraph accepts future normalized NetScope observations without modifying or reaching into the NetScope repository. Supported concepts are host, service, IP address, DNS name, and network relationship. The integration contract uses an InfraGraph-issued API credential bound to one organization and `ingest` scope.

The credential is hashed at rest, rate limited, and cannot administer users, assets, collectors, or policies. Each submission carries an idempotency key, source timestamp, bounded item count, and normalized external identifiers. Authentication determines the organization; an organization ID in payload data is not trusted.

