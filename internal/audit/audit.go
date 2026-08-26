package audit

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/infragraph/internal/security"
	"time"
)

type Event struct {
	ID, OrganizationID, ActorID, Action, ResourceType, ResourceID, RequestID string
	Payload                                                                  map[string]any
	OccurredAt                                                               time.Time
	PreviousHash, EventHash                                                  string
}

func Append(ctx context.Context, tx pgx.Tx, e Event) error {
	if e.OrganizationID == "" || e.Action == "" {
		return errors.New("organization and action required")
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	var previous string
	err := tx.QueryRow(ctx, "SELECT event_hash FROM audit_events WHERE organization_id=$1 ORDER BY occurred_at DESC,id DESC LIMIT 1 FOR UPDATE", e.OrganizationID).Scan(&previous)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	canonical, _ := json.Marshal(map[string]any{"id": e.ID, "organizationId": e.OrganizationID, "actorId": e.ActorID, "action": e.Action, "resourceType": e.ResourceType, "resourceId": e.ResourceID, "requestId": e.RequestID, "payload": e.Payload, "occurredAt": e.OccurredAt.UTC().Format(time.RFC3339Nano)})
	hash := security.AuditHash(previous, canonical)
	payload, _ := json.Marshal(e.Payload)
	_, err = tx.Exec(ctx, "INSERT INTO audit_events(id,organization_id,actor_id,action,resource_type,resource_id,request_id,payload,previous_hash,event_hash,occurred_at) VALUES($1,$2,nullif($3,''),$4,nullif($5,''),nullif($6,''),$7,$8,$9,$10,$11)", e.ID, e.OrganizationID, e.ActorID, e.Action, e.ResourceType, e.ResourceID, e.RequestID, payload, previous, hash, e.OccurredAt)
	return err
}
func Verify(events []Event) bool {
	previous := ""
	for _, e := range events {
		canonical, _ := json.Marshal(map[string]any{"id": e.ID, "organizationId": e.OrganizationID, "actorId": e.ActorID, "action": e.Action, "resourceType": e.ResourceType, "resourceId": e.ResourceID, "requestId": e.RequestID, "payload": e.Payload, "occurredAt": e.OccurredAt.UTC().Format(time.RFC3339Nano)})
		if e.PreviousHash != previous || e.EventHash != security.AuditHash(previous, canonical) {
			return false
		}
		previous = e.EventHash
	}
	return true
}
