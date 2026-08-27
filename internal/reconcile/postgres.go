package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"github.com/thiagomontozo/infragraph/internal/resolution"
)

// ApplyPostgres persists a verified successful snapshot and reconciles its
// canonical assets and relationships in the caller's transaction.
func ApplyPostgres(ctx context.Context, tx pgx.Tx, snapshot domain.SnapshotEnvelope, missingThreshold int) (Summary, error) {
	if snapshot.OrganizationID == "" || snapshot.ConnectorID == "" || snapshot.SnapshotID == "" {
		return Summary{}, errors.New("snapshot organization, connector, and ID are required")
	}
	if missingThreshold < 1 {
		missingThreshold = 2
	}
	now := snapshot.CompletedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	assets := make(map[string]string, len(snapshot.Assets))
	seenAssets := make(map[string]bool, len(snapshot.Assets))
	summary := Summary{}
	for _, observation := range snapshot.Assets {
		if observation.ExternalID == "" || len(observation.ExternalID) > 1024 {
			return Summary{}, errors.New("every observation requires a bounded external ID")
		}
		if _, ok := domain.AssetTypeLabels[observation.AssetType]; !ok {
			return Summary{}, fmt.Errorf("unknown asset type %q", observation.AssetType)
		}
		if seenAssets[observation.ExternalID] {
			return Summary{}, fmt.Errorf("duplicate asset external ID %q", observation.ExternalID)
		}
		seenAssets[observation.ExternalID] = true
		observedAt := observation.ObservedAt.UTC()
		if observedAt.IsZero() {
			observedAt = now
		}

		var assetID string
		err := tx.QueryRow(ctx, `SELECT asset_id FROM asset_source_identities WHERE organization_id=$1 AND connector_id=$2 AND external_id=$3`, snapshot.OrganizationID, snapshot.ConnectorID, observation.ExternalID).Scan(&assetID)
		created := false
		if errors.Is(err, pgx.ErrNoRows) {
			for key, value := range resolution.StrongIdentities(observation) {
				var resolved string
				lookupErr := tx.QueryRow(ctx, `SELECT asset_id FROM asset_strong_identities WHERE organization_id=$1 AND identity_key=$2 AND identity_value=$3`, snapshot.OrganizationID, key, value).Scan(&resolved)
				if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
					return Summary{}, lookupErr
				}
				if resolved != "" && assetID != "" && resolved != assetID {
					return Summary{}, errors.New("strong identifiers resolve to conflicting canonical assets")
				}
				if resolved != "" {
					assetID = resolved
				}
			}
			if assetID == "" {
				assetID = stableID("asset", snapshot.OrganizationID, snapshot.ConnectorID, observation.ExternalID)
				display := observationName(observation)
				environment := stringAttribute(observation.Attributes, "environment")
				_, err = tx.Exec(ctx, `INSERT INTO assets(id,organization_id,canonical_name,display_name,asset_type,status,environment,criticality,first_seen_at,last_seen_at) VALUES($1,$2,$3,$3,$4,'ACTIVE',nullif($5,''),'NORMAL',$6,$6)`, assetID, snapshot.OrganizationID, display, observation.AssetType, environment, observedAt)
				if err != nil {
					return Summary{}, err
				}
				created = true
				summary.Created++
			} else {
				if _, err = tx.Exec(ctx, `UPDATE assets SET status='ACTIVE',last_seen_at=GREATEST(last_seen_at,$1),updated_at=now() WHERE id=$2 AND organization_id=$3`, observedAt, assetID, snapshot.OrganizationID); err != nil {
					return Summary{}, err
				}
				summary.Unchanged++
			}
		} else if err != nil {
			return Summary{}, err
		} else {
			var prior string
			if err = tx.QueryRow(ctx, `SELECT status FROM assets WHERE id=$1 AND organization_id=$2 FOR UPDATE`, assetID, snapshot.OrganizationID).Scan(&prior); err != nil {
				return Summary{}, err
			}
			if _, err = tx.Exec(ctx, `UPDATE assets SET status='ACTIVE',last_seen_at=GREATEST(last_seen_at,$1),updated_at=now() WHERE id=$2 AND organization_id=$3`, observedAt, assetID, snapshot.OrganizationID); err != nil {
				return Summary{}, err
			}
			if prior == string(domain.AssetMissing) || prior == string(domain.AssetStale) {
				summary.Updated++
				if err = insertChange(ctx, tx, snapshot, assetID, "ASSET_RETURNED", prior, domain.AssetActive, "asset returned in a successful snapshot"); err != nil {
					return Summary{}, err
				}
				summary.Changes++
			} else {
				summary.Unchanged++
			}
		}

		identityID := stableID("identity", snapshot.OrganizationID, snapshot.ConnectorID, observation.ExternalID)
		_, err = tx.Exec(ctx, `INSERT INTO asset_source_identities(id,organization_id,asset_id,connector_id,external_id,external_type,first_seen_at,last_seen_at,status,consecutive_absences) VALUES($1,$2,$3,$4,$5,$6,$7,$7,'ACTIVE',0) ON CONFLICT(organization_id,connector_id,external_id) DO UPDATE SET last_seen_at=GREATEST(asset_source_identities.last_seen_at,EXCLUDED.last_seen_at),status='ACTIVE',consecutive_absences=0`, identityID, snapshot.OrganizationID, assetID, snapshot.ConnectorID, observation.ExternalID, observation.AssetType, observedAt)
		if err != nil {
			return Summary{}, err
		}
		for key, value := range resolution.StrongIdentities(observation) {
			var resolved string
			err = tx.QueryRow(ctx, `INSERT INTO asset_strong_identities(organization_id,identity_key,identity_value,asset_id,first_seen_at,last_seen_at) VALUES($1,$2,$3,$4,$5,$5) ON CONFLICT(organization_id,identity_key,identity_value) DO UPDATE SET last_seen_at=GREATEST(asset_strong_identities.last_seen_at,EXCLUDED.last_seen_at) RETURNING asset_id`, snapshot.OrganizationID, key, value, assetID, observedAt).Scan(&resolved)
			if err != nil {
				return Summary{}, err
			}
			if resolved != assetID {
				return Summary{}, errors.New("strong identifier is already bound to another canonical asset")
			}
		}
		attributes, err := json.Marshal(observation.Attributes)
		if err != nil {
			return Summary{}, err
		}
		hints, err := json.Marshal(observation.IdentityHints)
		if err != nil {
			return Summary{}, err
		}
		fingerprint := observation.Fingerprint
		if fingerprint == "" {
			sum := sha256.Sum256(attributes)
			fingerprint = hex.EncodeToString(sum[:])
		}
		observationID := stableID("observation", snapshot.SnapshotID, observation.ExternalID)
		_, err = tx.Exec(ctx, `INSERT INTO asset_observations(id,organization_id,snapshot_id,connector_id,collector_id,external_id,asset_type,observed_at,attributes,identity_hints,fingerprint,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'OBSERVED')`, observationID, snapshot.OrganizationID, snapshot.SnapshotID, snapshot.ConnectorID, snapshot.CollectorID, observation.ExternalID, observation.AssetType, observedAt, attributes, hints, fingerprint)
		if err != nil {
			return Summary{}, err
		}
		for key, value := range observation.Attributes {
			if len(key) > 200 {
				return Summary{}, errors.New("attribute key exceeds 200 characters")
			}
			if _, err = tx.Exec(ctx, `UPDATE asset_attribute_claims SET active=false WHERE organization_id=$1 AND asset_id=$2 AND connector_id=$3 AND attribute_key=$4 AND active`, snapshot.OrganizationID, assetID, snapshot.ConnectorID, key); err != nil {
				return Summary{}, err
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				return Summary{}, err
			}
			var authority string
			err = tx.QueryRow(ctx, `SELECT coalesce((SELECT authority FROM reconciliation_policies WHERE organization_id=$1 AND (connector_id IS NULL OR connector_id=$2) AND (asset_type IS NULL OR asset_type=$3) AND (attribute_key IS NULL OR attribute_key=$4) ORDER BY (connector_id IS NOT NULL)::int DESC,(asset_type IS NOT NULL)::int DESC,(attribute_key IS NOT NULL)::int DESC,precedence DESC LIMIT 1),'OBSERVED')`, snapshot.OrganizationID, snapshot.ConnectorID, observation.AssetType, key).Scan(&authority)
			if err != nil {
				return Summary{}, err
			}
			claimID := stableID("claim", snapshot.SnapshotID, observation.ExternalID, key)
			if _, err = tx.Exec(ctx, `INSERT INTO asset_attribute_claims(id,organization_id,asset_id,attribute_key,value,connector_id,snapshot_id,observed_at,authority,confidence,active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'OBSERVED',true)`, claimID, snapshot.OrganizationID, assetID, key, encoded, snapshot.ConnectorID, snapshot.SnapshotID, observedAt, authority); err != nil {
				return Summary{}, err
			}
		}
		conflict, updateErr := updateEffectiveAsset(ctx, tx, snapshot.OrganizationID, assetID)
		if updateErr != nil {
			return Summary{}, updateErr
		}
		if conflict {
			summary.Conflicting++
			if err = insertChange(ctx, tx, snapshot, assetID, "SOURCE_CONFLICT", nil, nil, "active source claims conflict"); err != nil {
				return Summary{}, err
			}
			summary.Changes++
		}
		if created {
			if err = insertChange(ctx, tx, snapshot, assetID, "ASSET_DISCOVERED", nil, map[string]any{"externalId": observation.ExternalID, "assetType": observation.AssetType}, "asset discovered"); err != nil {
				return Summary{}, err
			}
			summary.Changes++
		}
		assets[observation.ExternalID] = assetID
	}

	for _, observed := range snapshot.Relationships {
		from, fromOK := assets[observed.ExternalFromID]
		to, toOK := assets[observed.ExternalToID]
		if !fromOK || !toOK || observed.Type == "" || len(observed.Type) > 160 {
			return Summary{}, errors.New("relationship endpoints and bounded type must reference assets in the snapshot")
		}
		observedAt := observed.ObservedAt.UTC()
		if observedAt.IsZero() {
			observedAt = now
		}
		relationshipID := stableID("relationship", snapshot.OrganizationID, from, to, observed.Type)
		var existing bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asset_relationships WHERE organization_id=$1 AND from_asset_id=$2 AND to_asset_id=$3 AND type=$4)`, snapshot.OrganizationID, from, to, observed.Type).Scan(&existing); err != nil {
			return Summary{}, err
		}
		_, err := tx.Exec(ctx, `INSERT INTO asset_relationships(id,organization_id,from_asset_id,to_asset_id,type,status,first_seen_at,last_seen_at) VALUES($1,$2,$3,$4,$5,'ACTIVE',$6,$6) ON CONFLICT(organization_id,from_asset_id,to_asset_id,type) DO UPDATE SET status='ACTIVE',last_seen_at=GREATEST(asset_relationships.last_seen_at,EXCLUDED.last_seen_at),updated_at=now()`, relationshipID, snapshot.OrganizationID, from, to, observed.Type, observedAt)
		if err != nil {
			return Summary{}, err
		}
		if !existing {
			summary.RelationshipsCreated++
			if err = insertChange(ctx, tx, snapshot, from, "RELATIONSHIP_ADDED", nil, map[string]any{"relationshipId": relationshipID, "toAssetId": to, "type": observed.Type}, "observed relationship added"); err != nil {
				return Summary{}, err
			}
			summary.Changes++
		}
		sourceID := stableID("relationship-source", snapshot.OrganizationID, snapshot.ConnectorID, observed.ExternalFromID, observed.ExternalToID, observed.Type)
		_, err = tx.Exec(ctx, `INSERT INTO relationship_source_identities(id,organization_id,connector_id,relationship_id,external_from_id,external_to_id,type,status,consecutive_absences,first_seen_at,last_seen_at) VALUES($1,$2,$3,$4,$5,$6,$7,'ACTIVE',0,$8,$8) ON CONFLICT(organization_id,connector_id,external_from_id,external_to_id,type) DO UPDATE SET relationship_id=EXCLUDED.relationship_id,status='ACTIVE',consecutive_absences=0,last_seen_at=GREATEST(relationship_source_identities.last_seen_at,EXCLUDED.last_seen_at)`, sourceID, snapshot.OrganizationID, snapshot.ConnectorID, relationshipID, observed.ExternalFromID, observed.ExternalToID, observed.Type, observedAt)
		if err != nil {
			return Summary{}, err
		}
		attrs, _ := json.Marshal(observed.Attributes)
		observationID := stableID("relationship-observation", snapshot.SnapshotID, observed.ExternalFromID, observed.ExternalToID, observed.Type)
		if _, err = tx.Exec(ctx, `INSERT INTO relationship_observations(id,organization_id,relationship_id,snapshot_id,connector_id,external_from_id,external_to_id,type,observed_at,attributes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, observationID, snapshot.OrganizationID, relationshipID, snapshot.SnapshotID, snapshot.ConnectorID, observed.ExternalFromID, observed.ExternalToID, observed.Type, observedAt, attrs); err != nil {
			return Summary{}, err
		}
	}

	if err := markMissing(ctx, tx, snapshot, missingThreshold, &summary); err != nil {
		return Summary{}, err
	}
	runID := stableID("reconciliation", snapshot.SnapshotID)
	runSummary, _ := json.Marshal(summary)
	_, err := tx.Exec(ctx, `INSERT INTO reconciliation_runs(id,organization_id,connector_id,snapshot_id,status,started_at,completed_at,assets_created,assets_updated,assets_unchanged,assets_missing,assets_conflicting,relationships_created,relationships_removed,change_events_created,summary) VALUES($1,$2,$3,$4,'SUCCEEDED',$5,now(),$6,$7,$8,$9,$10,$11,$12,$13,$14)`, runID, snapshot.OrganizationID, snapshot.ConnectorID, snapshot.SnapshotID, snapshot.StartedAt, summary.Created, summary.Updated, summary.Unchanged, summary.Missing, summary.Conflicting, summary.RelationshipsCreated, summary.RelationshipsRemoved, summary.Changes, runSummary)
	return summary, err
}

func markMissing(ctx context.Context, tx pgx.Tx, snapshot domain.SnapshotEnvelope, threshold int, summary *Summary) error {
	_, err := tx.Exec(ctx, `UPDATE asset_source_identities SET consecutive_absences=consecutive_absences+1 WHERE organization_id=$1 AND connector_id=$2 AND NOT EXISTS (SELECT 1 FROM asset_observations o WHERE o.snapshot_id=$3 AND o.external_id=asset_source_identities.external_id)`, snapshot.OrganizationID, snapshot.ConnectorID, snapshot.SnapshotID)
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `UPDATE asset_source_identities SET status='MISSING' WHERE organization_id=$1 AND connector_id=$2 AND status='ACTIVE' AND consecutive_absences >= $3 RETURNING asset_id`, snapshot.OrganizationID, snapshot.ConnectorID, threshold)
	if err != nil {
		return err
	}
	var missing []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		missing = append(missing, id)
	}
	rows.Close()
	for _, assetID := range missing {
		if _, err = tx.Exec(ctx, `UPDATE asset_attribute_claims SET active=false WHERE organization_id=$1 AND connector_id=$2 AND asset_id=$3 AND active AND NOT EXISTS (SELECT 1 FROM asset_source_identities i WHERE i.organization_id=$1 AND i.connector_id=$2 AND i.asset_id=$3 AND i.status='ACTIVE')`, snapshot.OrganizationID, snapshot.ConnectorID, assetID); err != nil {
			return err
		}
		if _, err = updateEffectiveAsset(ctx, tx, snapshot.OrganizationID, assetID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `UPDATE assets SET status='MISSING',updated_at=now() WHERE id=$1 AND organization_id=$2 AND status NOT IN ('MISSING','RETIRED') AND NOT EXISTS (SELECT 1 FROM asset_source_identities i WHERE i.asset_id=$1 AND i.status='ACTIVE')`, assetID, snapshot.OrganizationID)
		if err != nil {
			return err
		}
		if command.RowsAffected() > 0 {
			summary.Missing++
			summary.Changes++
			if err = insertChange(ctx, tx, snapshot, assetID, "ASSET_MISSING", domain.AssetActive, domain.AssetMissing, "absent from sufficient successful snapshots"); err != nil {
				return err
			}
		}
	}

	_, err = tx.Exec(ctx, `UPDATE relationship_source_identities SET consecutive_absences=consecutive_absences+1 WHERE organization_id=$1 AND connector_id=$2 AND NOT EXISTS (SELECT 1 FROM relationship_observations o WHERE o.snapshot_id=$3 AND o.external_from_id=relationship_source_identities.external_from_id AND o.external_to_id=relationship_source_identities.external_to_id AND o.type=relationship_source_identities.type)`, snapshot.OrganizationID, snapshot.ConnectorID, snapshot.SnapshotID)
	if err != nil {
		return err
	}
	relRows, err := tx.Query(ctx, `UPDATE relationship_source_identities SET status='REMOVED' WHERE organization_id=$1 AND connector_id=$2 AND status='ACTIVE' AND consecutive_absences >= $3 RETURNING relationship_id`, snapshot.OrganizationID, snapshot.ConnectorID, threshold)
	if err != nil {
		return err
	}
	var relationships []string
	for relRows.Next() {
		var id string
		if err = relRows.Scan(&id); err != nil {
			relRows.Close()
			return err
		}
		relationships = append(relationships, id)
	}
	relRows.Close()
	for _, relationshipID := range relationships {
		command, err := tx.Exec(ctx, `UPDATE asset_relationships SET status='REMOVED',updated_at=now() WHERE id=$1 AND organization_id=$2 AND status='ACTIVE' AND NOT EXISTS (SELECT 1 FROM relationship_source_identities i WHERE i.relationship_id=$1 AND i.status='ACTIVE')`, relationshipID, snapshot.OrganizationID)
		if err != nil {
			return err
		}
		if command.RowsAffected() > 0 {
			summary.RelationshipsRemoved++
		}
	}
	return nil
}

func updateEffectiveAsset(ctx context.Context, tx pgx.Tx, organizationID, assetID string) (bool, error) {
	var environment, display *string
	_ = tx.QueryRow(ctx, `SELECT value #>> '{}' FROM asset_attribute_claims WHERE organization_id=$1 AND asset_id=$2 AND attribute_key='environment' AND active ORDER BY CASE authority WHEN 'AUTHORITATIVE' THEN 3 WHEN 'OBSERVED' THEN 2 ELSE 1 END DESC,observed_at DESC,id DESC LIMIT 1`, organizationID, assetID).Scan(&environment)
	_ = tx.QueryRow(ctx, `SELECT value #>> '{}' FROM asset_attribute_claims WHERE organization_id=$1 AND asset_id=$2 AND attribute_key IN ('name','hostname','display_name') AND active ORDER BY CASE authority WHEN 'AUTHORITATIVE' THEN 3 WHEN 'OBSERVED' THEN 2 ELSE 1 END DESC,observed_at DESC,id DESC LIMIT 1`, organizationID, assetID).Scan(&display)
	var conflict bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asset_attribute_claims WHERE organization_id=$1 AND asset_id=$2 AND active GROUP BY attribute_key HAVING count(DISTINCT value::text)>1)`, organizationID, assetID).Scan(&conflict); err != nil {
		return false, err
	}
	status := "ACTIVE"
	if conflict {
		status = "CONFLICTING"
	}
	_, err := tx.Exec(ctx, `UPDATE assets SET environment=coalesce($1,environment),display_name=coalesce(nullif($2,''),display_name),canonical_name=coalesce(nullif($2,''),canonical_name),status=$3,updated_at=now() WHERE id=$4 AND organization_id=$5`, environment, display, status, assetID, organizationID)
	return conflict, err
}

func insertChange(ctx context.Context, tx pgx.Tx, snapshot domain.SnapshotEnvelope, assetID, changeType string, before, after any, reason string) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	logical := strings.Join([]string{snapshot.SnapshotID, assetID, changeType}, "|")
	id := stableID("change", logical)
	_, err := tx.Exec(ctx, `INSERT INTO infrastructure_changes(id,organization_id,asset_id,change_type,before_value,after_value,source,detected_at,summary,confidence,logical_identity) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'DETERMINISTIC',$10) ON CONFLICT(organization_id,logical_identity) DO NOTHING`, id, snapshot.OrganizationID, assetID, changeType, beforeJSON, afterJSON, snapshot.ConnectorID, snapshot.CompletedAt, reason, logical)
	return err
}

func stableID(prefix string, values ...string) string {
	h := sha256.New()
	for _, value := range values {
		h.Write([]byte(value))
		h.Write([]byte{0})
	}
	return prefix + "-" + hex.EncodeToString(h.Sum(nil))[:32]
}

func observationName(observation domain.Observation) string {
	for _, key := range []string{"name", "hostname", "display_name"} {
		if value := stringAttribute(observation.Attributes, key); value != "" {
			return value
		}
	}
	return observation.ExternalID
}

func stringAttribute(attributes map[string]any, key string) string {
	if attributes == nil {
		return ""
	}
	value, ok := attributes[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
