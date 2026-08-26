# Backup and restore

Back up PostgreSQL with encrypted `pg_dump -Fc` for logical portability and platform-native physical/PITR backups for operational RPO. Back up the private S3 bucket with versioning/replication or provider snapshots. Preserve configuration descriptions and secret **references** separately; master keys need protected escrow because database ciphertext is unrecoverable without them. Collector private keys live on collectors and may be re-enrolled.

Restore procedure: isolate the target, provision the same or newer supported PostgreSQL, restore roles/schema/data, restore object versions/bucket policy, inject matching key versions, start one API replica, run migrations, verify organization/asset/relationship/user/policy/audit counts and audit chains, test artifact reads, revoke sessions, then admit traffic. `scripts/restore-test` performs a synthetic dump into a second temporary PostgreSQL and verifies the migration table; production exercises must also sample core entity counts and artifacts.

