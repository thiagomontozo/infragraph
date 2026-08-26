package audit

import (
	"encoding/json"
	"github.com/thiagomontozo/infragraph/internal/security"
	"testing"
	"time"
)

func TestTamperEvidentChain(t *testing.T) {
	at := time.Unix(1, 0).UTC()
	events := []Event{{ID: "1", OrganizationID: "o", Action: "asset.updated", Payload: map[string]any{"x": 1}, OccurredAt: at}, {ID: "2", OrganizationID: "o", Action: "asset.retired", Payload: map[string]any{}, OccurredAt: at}}
	prev := ""
	for i := range events {
		b, _ := json.Marshal(map[string]any{"id": events[i].ID, "organizationId": events[i].OrganizationID, "actorId": "", "action": events[i].Action, "resourceType": "", "resourceId": "", "requestId": "", "payload": events[i].Payload, "occurredAt": at.Format(time.RFC3339Nano)})
		events[i].PreviousHash = prev
		events[i].EventHash = security.AuditHash(prev, b)
		prev = events[i].EventHash
	}
	if !Verify(events) {
		t.Fatal("valid chain rejected")
	}
	events[0].Action = "tampered"
	if Verify(events) {
		t.Fatal("tampering not detected")
	}
}
