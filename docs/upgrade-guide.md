# Upgrade guide

Read release notes and migration compatibility, back up PostgreSQL/object storage/config references, verify master-key versions, test on a restored copy, then apply migrations once. Roll API replicas while monitoring readiness; the same collector protocol major remains accepted and the UI marks older supported collectors `UPGRADE_RECOMMENDED`.

Rollback application binaries only if the new migration is backward compatible. For irreversible migrations, restore the pre-upgrade backup into an isolated environment and follow the release-specific forward repair. Revoke sessions only when authentication semantics or keys changed; re-enroll collectors only after credential/private-key compromise.

