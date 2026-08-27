ALTER TABLE asset_source_identities
  ADD COLUMN consecutive_absences integer NOT NULL DEFAULT 0;

CREATE TABLE relationship_source_identities (
  id text PRIMARY KEY,
  organization_id text NOT NULL REFERENCES organizations(id),
  connector_id text NOT NULL REFERENCES infrastructure_connectors(id),
  relationship_id text NOT NULL REFERENCES asset_relationships(id) ON DELETE CASCADE,
  external_from_id text NOT NULL,
  external_to_id text NOT NULL,
  type text NOT NULL,
  status text NOT NULL DEFAULT 'ACTIVE',
  consecutive_absences integer NOT NULL DEFAULT 0,
  first_seen_at timestamptz NOT NULL,
  last_seen_at timestamptz NOT NULL,
  UNIQUE (organization_id, connector_id, external_from_id, external_to_id, type)
);

CREATE INDEX relationship_source_connector_status_idx
  ON relationship_source_identities(organization_id, connector_id, status);

CREATE TABLE asset_strong_identities (
  organization_id text NOT NULL REFERENCES organizations(id),
  identity_key text NOT NULL,
  identity_value text NOT NULL,
  asset_id text NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  first_seen_at timestamptz NOT NULL,
  last_seen_at timestamptz NOT NULL,
  PRIMARY KEY (organization_id, identity_key, identity_value)
);

CREATE INDEX asset_strong_identity_asset_idx
  ON asset_strong_identities(organization_id, asset_id);
