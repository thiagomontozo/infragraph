package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/thiagomontozo/infragraph/internal/config"
	"github.com/thiagomontozo/infragraph/internal/database"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"github.com/thiagomontozo/infragraph/internal/security"
)

func TestCollectorEnrollmentAndSnapshotReconcileEndToEnd(t *testing.T) {
	url := os.Getenv("INFRAGRAPH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PostgreSQL integration is opt-in")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	org := "org-app-" + suffix
	user := "user-app-" + suffix
	token := "enrollment-token-" + suffix
	if _, err = db.Pool.Exec(ctx, `INSERT INTO organizations(id,name) VALUES($1,'App integration')`, org); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,organization_id,email,password_hash,display_name) VALUES($1,$2,$3,'test','Test')`, user, org, suffix+"@example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO collector_enrollment_tokens(id,organization_id,token_hash,created_by,expires_at) VALUES($1,$2,$3,$4,now()+interval '5 minutes')`, "token-app-"+suffix, org, security.TokenHash(token), user); err != nil {
		t.Fatal(err)
	}

	publicKey, privateKey, err := security.GenerateCollectorKey()
	if err != nil {
		t.Fatal(err)
	}
	handler := New(config.Config{SessionSecret: "integration-secret", MaxGraphDepth: 6, MaxGraphNodes: 500, MaxImportBytes: 1 << 20, MaxSnapshotBytes: 10 << 20}, db, nil).Handler()
	enrollment := map[string]string{"token": token, "name": "integration collector", "publicKey": base64.StdEncoding.EncodeToString(publicKey), "collectorVersion": "test", "protocolVersion": "1.0", "connectorName": "integration docker", "connectorType": "DOCKER"}
	response := performJSON(handler, http.MethodPost, "/collector/v1/enroll", "", enrollment)
	if response.Code != http.StatusCreated {
		t.Fatalf("enrollment status=%d body=%s", response.Code, response.Body.String())
	}
	var enrolled struct {
		CollectorID, ConnectorID, OrganizationID, Credential string
	}
	if err = json.Unmarshal(response.Body.Bytes(), &enrolled); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	snapshot := domain.SnapshotEnvelope{ProtocolVersion: "1.0", SnapshotID: "snapshot-app-" + suffix, OrganizationID: org, CollectorID: enrolled.CollectorID, ConnectorID: enrolled.ConnectorID, ConnectorType: "DOCKER", ConnectorVersion: "test", StartedAt: now.Add(-time.Second), CompletedAt: now, Sequence: 1, Assets: []domain.Observation{{ExternalID: "container-1", AssetType: domain.Container, ObservedAt: now, Attributes: map[string]any{"name": "web"}, IdentityHints: map[string]any{"docker_container_id": "container-1"}, Status: "OBSERVED"}}, Relationships: []domain.RelationshipObservation{}, Warnings: []string{}, Statistics: map[string]int{"assets": 1}}
	snapshot, err = security.SignSnapshot(snapshot, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	response = performJSON(handler, http.MethodPost, "/collector/v1/snapshots", enrolled.Credential, snapshot)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"status":"SUCCEEDED"`)) {
		t.Fatalf("snapshot status=%d body=%s", response.Code, response.Body.String())
	}
	response = performJSON(handler, http.MethodPost, "/collector/v1/snapshots", enrolled.Credential, snapshot)
	if response.Code != http.StatusAccepted || !bytes.Contains(response.Body.Bytes(), []byte(`"idempotent":true`)) {
		t.Fatalf("idempotent retry status=%d body=%s", response.Code, response.Body.String())
	}
	var assets, audits int
	if err = db.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM assets WHERE organization_id=$1),(SELECT count(*) FROM audit_events WHERE organization_id=$1 AND action='collector.snapshot.reconciled')`, org).Scan(&assets, &audits); err != nil {
		t.Fatal(err)
	}
	if assets != 1 || audits != 1 {
		t.Fatalf("reconciliation assets=%d audits=%d", assets, audits)
	}
}

func performJSON(handler http.Handler, method, path, credential string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
