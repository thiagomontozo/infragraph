package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	dockerconnector "github.com/thiagomontozo/infragraph/internal/connectors/docker"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"github.com/thiagomontozo/infragraph/internal/security"
)

const version = "1.0.0-rc.1"

type identity struct {
	CollectorID    string `json:"collectorId"`
	OrganizationID string `json:"organizationId"`
	Credential     string `json:"credential"`
	PrivateKey     string `json:"privateKey"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	control := strings.TrimRight(os.Getenv("INFRAGRAPH_CONTROL_PLANE_URL"), "/")
	id, err := loadOrEnroll(ctx, control, env("INFRAGRAPH_COLLECTOR_DATA_DIR", "./data/collector"))
	if err != nil {
		slog.Error("collector enrollment failed", "error", err)
		os.Exit(1)
	}
	interval := 30 * time.Second
	if value, err := time.ParseDuration(os.Getenv("INFRAGRAPH_COLLECTOR_INTERVAL")); err == nil && value > 0 {
		interval = value
	}
	for {
		if err = runOnce(ctx, control, id); err != nil {
			slog.Error("discovery failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func loadOrEnroll(ctx context.Context, control, data string) (identity, error) {
	if control == "" {
		return identity{}, errors.New("INFRAGRAPH_CONTROL_PLANE_URL required")
	}
	if err := os.MkdirAll(data, 0700); err != nil {
		return identity{}, err
	}
	identityPath := filepath.Join(data, "identity.json")
	if raw, err := os.ReadFile(identityPath); err == nil {
		var id identity
		err = json.Unmarshal(raw, &id)
		return id, err
	}
	token := os.Getenv("INFRAGRAPH_ENROLLMENT_TOKEN")
	if token == "" {
		return identity{}, errors.New("enrollment token required for first start")
	}
	publicKey, privateKey, err := security.GenerateCollectorKey()
	if err != nil {
		return identity{}, err
	}
	body := map[string]string{"token": token, "name": env("INFRAGRAPH_COLLECTOR_NAME", "collector"), "publicKey": base64.StdEncoding.EncodeToString(publicKey), "collectorVersion": version, "protocolVersion": "1.0"}
	var enrolled struct {
		CollectorID    string `json:"collectorId"`
		OrganizationID string `json:"organizationId"`
		Credential     string `json:"credential"`
	}
	if err = post(ctx, control+"/collector/v1/enroll", "", body, &enrolled, 1<<20); err != nil {
		return identity{}, err
	}
	id := identity{enrolled.CollectorID, enrolled.OrganizationID, enrolled.Credential, base64.StdEncoding.EncodeToString(privateKey)}
	raw, _ := json.Marshal(id)
	if err = os.WriteFile(identityPath, raw, 0600); err != nil {
		return identity{}, err
	}
	return id, nil
}

func runOnce(ctx context.Context, control string, id identity) error {
	connector, err := dockerconnector.NewUnix(env("INFRAGRAPH_DOCKER_SOCKET", "/var/run/docker.sock"), env("INFRAGRAPH_DOCKER_LABEL_SCOPE", "com.infragraph.test=true"), 15*time.Second)
	if err != nil {
		return err
	}
	assets, relationships, err := connector.Discover(ctx)
	if err != nil {
		return err
	}
	sequence := time.Now().UnixNano()
	snapshot := domain.SnapshotEnvelope{ProtocolVersion: "1.0", SnapshotID: "snapshot-" + strconv.FormatInt(sequence, 10), OrganizationID: id.OrganizationID, CollectorID: id.CollectorID, ConnectorID: env("INFRAGRAPH_CONNECTOR_ID", "docker"), ConnectorType: "DOCKER", ConnectorVersion: version, StartedAt: time.Now().Add(-time.Second).UTC(), CompletedAt: time.Now().UTC(), Sequence: sequence, Assets: assets, Relationships: relationships, Statistics: map[string]int{"assets": len(assets), "relationships": len(relationships)}}
	key, err := base64.StdEncoding.DecodeString(id.PrivateKey)
	if err != nil {
		return err
	}
	snapshot, err = security.SignSnapshot(snapshot, ed25519.PrivateKey(key))
	if err != nil {
		return err
	}
	return post(ctx, control+"/collector/v1/snapshots", id.Credential, snapshot, nil, 60<<20)
}

func post(ctx context.Context, url, credential string, in, out any, max int64) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	res, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	response, err := io.ReadAll(io.LimitReader(res.Body, max))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("control plane returned %s", res.Status)
	}
	if out != nil {
		return json.Unmarshal(response, out)
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
