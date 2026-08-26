package reconcile

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"reflect"
	"sort"
	"strings"
	"time"
)

type Authority string

const (
	Authoritative Authority = "AUTHORITATIVE"
	Observed      Authority = "OBSERVED"
	Declared      Authority = "DECLARED"
)

type Policy struct {
	AttributeAuthority         map[string]map[string]Authority
	MissingAfterSuccessfulRuns int
}
type State struct {
	Assets             map[string]domain.Asset
	ExternalToAsset    map[string]string
	Claims             map[string][]domain.AttributeClaim
	Relationships      map[string]domain.Relationship
	Changes            map[string]domain.Change
	SuccessfulAbsences map[string]int
	ProcessedSnapshots map[string]string
}
type Summary struct{ Created, Updated, Unchanged, Missing, Conflicting, RelationshipsCreated, RelationshipsRemoved, Changes int }

func NewState() *State {
	return &State{map[string]domain.Asset{}, map[string]string{}, map[string][]domain.AttributeClaim{}, map[string]domain.Relationship{}, map[string]domain.Change{}, map[string]int{}, map[string]string{}}
}

func Effective(claims []domain.AttributeClaim, attribute string, policy Policy) domain.EffectiveAttribute {
	active := []domain.AttributeClaim{}
	for _, c := range claims {
		if c.Active && c.AttributeKey == attribute {
			active = append(active, c)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		ai := policy.AttributeAuthority[active[i].ConnectorID][attribute]
		aj := policy.AttributeAuthority[active[j].ConnectorID][attribute]
		rank := func(a Authority) int {
			if a == Authoritative {
				return 3
			}
			if a == Observed {
				return 2
			}
			return 1
		}
		if rank(ai) != rank(aj) {
			return rank(ai) > rank(aj)
		}
		return active[i].ObservedAt.After(active[j].ObservedAt)
	})
	if len(active) == 0 {
		return domain.EffectiveAttribute{Reason: "no active claims"}
	}
	value := active[0].Value
	distinct := map[string]bool{}
	for _, c := range active {
		b, _ := json.Marshal(c.Value)
		distinct[string(b)] = true
	}
	reason := "most recent claim at highest configured authority"
	if len(distinct) == 1 && len(active) > 1 {
		reason = "multiple active sources agree"
	}
	return domain.EffectiveAttribute{Value: value, Conflict: len(distinct) > 1, Reason: reason, Claims: active}
}

func Apply(st *State, snapshot domain.SnapshotEnvelope, status domain.SnapshotStatus, policy Policy) (Summary, error) {
	if st == nil {
		return Summary{}, errors.New("state required")
	}
	if prior, ok := st.ProcessedSnapshots[snapshot.SnapshotID]; ok {
		if prior == snapshot.ContentHash {
			return Summary{}, nil
		}
		return Summary{}, errors.New("snapshot ID reused with different content hash")
	}
	if status != domain.SnapshotSucceeded {
		st.ProcessedSnapshots[snapshot.SnapshotID] = snapshot.ContentHash
		return Summary{}, nil
	}
	if policy.MissingAfterSuccessfulRuns < 1 {
		policy.MissingAfterSuccessfulRuns = 1
	}
	summary := Summary{}
	seen := map[string]bool{}
	now := snapshot.CompletedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, o := range snapshot.Assets {
		key := snapshot.OrganizationID + "|" + snapshot.ConnectorID + "|" + o.ExternalID
		seen[key] = true
		assetID := st.ExternalToAsset[key]
		if assetID == "" {
			assetID = "asset-" + safe(o.ExternalID)
			st.ExternalToAsset[key] = assetID
			a := domain.Asset{ID: assetID, OrganizationID: snapshot.OrganizationID, CanonicalName: name(o), DisplayName: name(o), Type: o.AssetType, Status: domain.AssetActive, FirstSeenAt: now, LastSeenAt: now, CreatedAt: now, UpdatedAt: now}
			st.Assets[assetID] = a
			summary.Created++
			addChange(st, snapshot, assetID, "ASSET_DISCOVERED", nil, a, "asset discovered", &summary)
		} else {
			a := st.Assets[assetID]
			before := a.Status
			a.LastSeenAt = now
			a.UpdatedAt = now
			a.Status = domain.AssetActive
			st.Assets[assetID] = a
			if before == domain.AssetMissing {
				addChange(st, snapshot, assetID, "ASSET_RETURNED", before, a.Status, "asset returned", &summary)
				summary.Updated++
			} else {
				summary.Unchanged++
			}
		}
		st.SuccessfulAbsences[key] = 0
		for k, v := range o.Attributes {
			claim := domain.AttributeClaim{AssetID: assetID, AttributeKey: k, Value: v, ConnectorID: snapshot.ConnectorID, SnapshotID: snapshot.SnapshotID, ObservedAt: o.ObservedAt, Authority: string(policy.AttributeAuthority[snapshot.ConnectorID][k]), Confidence: "OBSERVED", Active: true}
			ck := assetID + "|" + k
			for i := range st.Claims[ck] {
				if st.Claims[ck][i].ConnectorID == snapshot.ConnectorID {
					st.Claims[ck][i].Active = false
				}
			}
			st.Claims[ck] = append(st.Claims[ck], claim)
			eff := Effective(st.Claims[ck], k, policy)
			if eff.Conflict {
				a := st.Assets[assetID]
				a.Status = domain.AssetConflicting
				st.Assets[assetID] = a
				summary.Conflicting++
				addChange(st, snapshot, assetID, "SOURCE_CONFLICT", nil, eff.Value, "source claims conflict", &summary)
			}
		}
	}
	seenRelationships := map[string]bool{}
	for _, observed := range snapshot.Relationships {
		from := st.ExternalToAsset[snapshot.OrganizationID+"|"+snapshot.ConnectorID+"|"+observed.ExternalFromID]
		to := st.ExternalToAsset[snapshot.OrganizationID+"|"+snapshot.ConnectorID+"|"+observed.ExternalToID]
		if from == "" || to == "" {
			continue
		}
		key := snapshot.OrganizationID + "|" + snapshot.ConnectorID + "|" + from + "|" + to + "|" + observed.Type
		seenRelationships[key] = true
		relationship, exists := st.Relationships[key]
		if !exists {
			relationship = domain.Relationship{ID: "relationship-" + safe(key), OrganizationID: snapshot.OrganizationID, FromAssetID: from, ToAssetID: to, Type: observed.Type, Status: "ACTIVE", FirstSeenAt: now, LastSeenAt: now, CreatedAt: now, UpdatedAt: now}
			st.Relationships[key] = relationship
			summary.RelationshipsCreated++
			addChange(st, snapshot, from, "RELATIONSHIP_ADDED", nil, relationship, "observed relationship added", &summary)
		} else {
			relationship.Status = "ACTIVE"
			relationship.LastSeenAt = now
			relationship.UpdatedAt = now
			st.Relationships[key] = relationship
		}
	}
	for key, relationship := range st.Relationships {
		prefix := snapshot.OrganizationID + "|" + snapshot.ConnectorID + "|"
		if !strings.HasPrefix(key, prefix) || seenRelationships[key] || relationship.Status != "ACTIVE" {
			continue
		}
		before := relationship.Status
		relationship.Status = "REMOVED"
		relationship.UpdatedAt = now
		st.Relationships[key] = relationship
		summary.RelationshipsRemoved++
		addChange(st, snapshot, relationship.FromAssetID, "RELATIONSHIP_REMOVED", before, relationship.Status, "absent from a successful source snapshot", &summary)
	}
	for key, id := range st.ExternalToAsset {
		if !strings.HasPrefix(key, snapshot.OrganizationID+"|"+snapshot.ConnectorID+"|") || seen[key] {
			continue
		}
		st.SuccessfulAbsences[key]++
		if st.SuccessfulAbsences[key] >= policy.MissingAfterSuccessfulRuns {
			a := st.Assets[id]
			if a.Status != domain.AssetMissing && a.Status != domain.AssetRetired {
				before := a.Status
				a.Status = domain.AssetMissing
				a.UpdatedAt = now
				st.Assets[id] = a
				summary.Missing++
				addChange(st, snapshot, id, "ASSET_MISSING", before, a.Status, "absent from sufficient successful snapshots", &summary)
			}
		}
	}
	st.ProcessedSnapshots[snapshot.SnapshotID] = snapshot.ContentHash
	return summary, nil
}
func addChange(st *State, s domain.SnapshotEnvelope, assetID, typ string, before, after any, reason string, summary *Summary) {
	key := s.SnapshotID + "|" + assetID + "|" + typ
	if _, ok := st.Changes[key]; ok {
		return
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	st.Changes[key] = domain.Change{ID: "change-" + safe(key), OrganizationID: s.OrganizationID, AssetID: assetID, Type: typ, BeforeValue: b, AfterValue: a, Source: s.ConnectorID, DetectedAt: s.CompletedAt, Summary: reason, Confidence: "DETERMINISTIC", LogicalKey: key}
	summary.Changes++
}
func name(o domain.Observation) string {
	for _, k := range []string{"name", "hostname", "display_name"} {
		if v := strings.TrimSpace(fmt.Sprint(o.Attributes[k])); v != "" && v != "<nil>" {
			return v
		}
	}
	return o.ExternalID
}
func safe(v string) string {
	v = strings.ToLower(v)
	var b strings.Builder
	for _, r := range v {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
func Equal(a, b any) bool { return reflect.DeepEqual(a, b) }
