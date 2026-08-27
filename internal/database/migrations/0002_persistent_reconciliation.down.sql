DROP TABLE IF EXISTS asset_strong_identities;
DROP TABLE IF EXISTS relationship_source_identities;
ALTER TABLE asset_source_identities DROP COLUMN IF EXISTS consecutive_absences;
