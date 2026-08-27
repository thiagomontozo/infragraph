package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	dockerconnector "github.com/thiagomontozo/infragraph/internal/connectors/docker"
	kubernetesconnector "github.com/thiagomontozo/infragraph/internal/connectors/kubernetes"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"github.com/thiagomontozo/infragraph/internal/security"
)

const version = "1.0.0-rc.1"

type identity struct {
	CollectorID    string `json:"collectorId"`
	ConnectorID    string `json:"connectorId"`
	OrganizationID string `json:"organizationId"`
	Credential     string `json:"credential"`
	PrivateKey     string `json:"privateKey"`
	ConnectorType  string `json:"connectorType"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	control := strings.TrimRight(os.Getenv("INFRAGRAPH_CONTROL_PLANE_URL"), "/")
	connectorType, err := configuredConnectorType()
	if err != nil {
		slog.Error("collector configuration failed", "error", err)
		os.Exit(1)
	}
	id, err := loadOrEnroll(ctx, control, env("INFRAGRAPH_COLLECTOR_DATA_DIR", "./data/collector"), connectorType)
	if err != nil {
		slog.Error("collector enrollment failed", "error", err)
		os.Exit(1)
	}
	interval := 30 * time.Second
	if value, err := time.ParseDuration(os.Getenv("INFRAGRAPH_COLLECTOR_INTERVAL")); err == nil && value > 0 {
		interval = value
	}
	data := env("INFRAGRAPH_COLLECTOR_DATA_DIR", "./data/collector")
	for {
		if err = heartbeat(ctx, control, id, "RUNNING"); err != nil {
			slog.Warn("collector heartbeat failed", "error", err)
		}
		if err = flushSpool(ctx, control, id, data); err == nil {
			err = runOnce(ctx, control, id, data)
		}
		if err != nil {
			slog.Error("discovery failed", "error", err)
			_ = heartbeat(ctx, control, id, "DISCOVERY_FAILED")
		} else {
			_ = heartbeat(ctx, control, id, "HEALTHY")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func loadOrEnroll(ctx context.Context, control, data, connectorType string) (identity, error) {
	if control == "" {
		return identity{}, errors.New("INFRAGRAPH_CONTROL_PLANE_URL required")
	}
	if err := os.MkdirAll(data, 0700); err != nil {
		return identity{}, err
	}
	identityPath := filepath.Join(data, "identity.json")
	if raw, err := os.ReadFile(identityPath); err == nil {
		var id identity
		if err = json.Unmarshal(raw, &id); err != nil {
			return identity{}, err
		}
		if id.CollectorID == "" || id.OrganizationID == "" || id.Credential == "" || id.PrivateKey == "" {
			return identity{}, errors.New("collector identity file is incomplete")
		}
		if id.ConnectorType == "" {
			id.ConnectorType = "DOCKER"
		}
		if id.ConnectorType != connectorType {
			return identity{}, fmt.Errorf("configured connector type %s does not match enrolled identity type %s; use a separate data directory and enrollment", connectorType, id.ConnectorType)
		}
		return id, nil
	}
	token := os.Getenv("INFRAGRAPH_ENROLLMENT_TOKEN")
	if token == "" {
		return identity{}, errors.New("enrollment token required for first start")
	}
	publicKey, privateKey, err := security.GenerateCollectorKey()
	if err != nil {
		return identity{}, err
	}
	defaultConnectorName := "Docker discovery"
	if connectorType == "KUBERNETES" {
		defaultConnectorName = "Kubernetes discovery"
	}
	body := map[string]string{"token": token, "name": env("INFRAGRAPH_COLLECTOR_NAME", "collector"), "publicKey": base64.StdEncoding.EncodeToString(publicKey), "collectorVersion": version, "protocolVersion": "1.0", "connectorName": env("INFRAGRAPH_CONNECTOR_NAME", defaultConnectorName), "connectorType": connectorType}
	var enrolled struct {
		CollectorID    string `json:"collectorId"`
		ConnectorID    string `json:"connectorId"`
		OrganizationID string `json:"organizationId"`
		Credential     string `json:"credential"`
	}
	if err = post(ctx, control+"/collector/v1/enroll", "", body, &enrolled, 1<<20); err != nil {
		return identity{}, err
	}
	id := identity{CollectorID: enrolled.CollectorID, ConnectorID: enrolled.ConnectorID, OrganizationID: enrolled.OrganizationID, Credential: enrolled.Credential, PrivateKey: base64.StdEncoding.EncodeToString(privateKey), ConnectorType: connectorType}
	raw, _ := json.Marshal(id)
	if err = os.WriteFile(identityPath, raw, 0600); err != nil {
		return identity{}, err
	}
	return id, nil
}

func runOnce(ctx context.Context, control string, id identity, data string) error {
	startedAt := time.Now().UTC()
	assets, relationships, err := discover(ctx, id.ConnectorType)
	if err != nil {
		return err
	}
	for i := range assets {
		if assets[i].Attributes == nil {
			assets[i].Attributes = map[string]any{}
		}
		if assets[i].IdentityHints == nil {
			assets[i].IdentityHints = map[string]any{}
		}
		assets[i].Status = "OBSERVED"
		fingerprint, marshalErr := json.Marshal(map[string]any{"externalId": assets[i].ExternalID, "assetType": assets[i].AssetType, "attributes": assets[i].Attributes, "identityHints": assets[i].IdentityHints})
		if marshalErr != nil {
			return marshalErr
		}
		hash := sha256.Sum256(fingerprint)
		assets[i].Fingerprint = hex.EncodeToString(hash[:])
	}
	for i := range relationships {
		if relationships[i].Attributes == nil {
			relationships[i].Attributes = map[string]any{}
		}
	}
	sequence, err := nextSequence(data)
	if err != nil {
		return err
	}
	connectorID := id.ConnectorID
	if connectorID == "" {
		connectorID = os.Getenv("INFRAGRAPH_CONNECTOR_ID")
	}
	if connectorID == "" {
		return errors.New("collector identity does not contain a connector ID; re-enroll this pre-1.0 identity")
	}
	snapshot := domain.SnapshotEnvelope{ProtocolVersion: "1.0", SnapshotID: "snapshot-" + strconv.FormatInt(sequence, 10), OrganizationID: id.OrganizationID, CollectorID: id.CollectorID, ConnectorID: connectorID, ConnectorType: id.ConnectorType, ConnectorVersion: version, StartedAt: startedAt, CompletedAt: time.Now().UTC(), Sequence: sequence, Assets: assets, Relationships: relationships, Warnings: []string{}, Statistics: map[string]int{"assets": len(assets), "relationships": len(relationships)}}
	key, err := base64.StdEncoding.DecodeString(id.PrivateKey)
	if err != nil {
		return err
	}
	snapshot, err = security.SignSnapshot(snapshot, ed25519.PrivateKey(key))
	if err != nil {
		return err
	}
	if err = queueSnapshot(data, snapshot); err != nil {
		return err
	}
	return flushSpool(ctx, control, id, data)
}

func discover(ctx context.Context, connectorType string) ([]domain.Observation, []domain.RelationshipObservation, error) {
	switch connectorType {
	case "DOCKER":
		connector, err := dockerconnector.NewUnix(env("INFRAGRAPH_DOCKER_SOCKET", "/var/run/docker.sock"), env("INFRAGRAPH_DOCKER_LABEL_SCOPE", "com.infragraph.test=true"), 15*time.Second)
		if err != nil {
			return nil, nil, err
		}
		return connector.Discover(ctx)
	case "KUBERNETES":
		token, err := readBoundedFile(env("INFRAGRAPH_KUBERNETES_TOKEN_FILE", "/var/run/secrets/kubernetes.io/serviceaccount/token"), 1<<20)
		if err != nil {
			return nil, nil, fmt.Errorf("read Kubernetes token: %w", err)
		}
		ca, err := readBoundedFile(env("INFRAGRAPH_KUBERNETES_CA_FILE", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"), 1<<20)
		if err != nil {
			return nil, nil, fmt.Errorf("read Kubernetes CA: %w", err)
		}
		connector, err := kubernetesconnector.NewWithConfig(kubernetesconnector.Config{
			BaseURL:      env("INFRAGRAPH_KUBERNETES_API_URL", "https://kubernetes.default.svc"),
			Token:        string(token),
			CAPEM:        ca,
			ClusterID:    os.Getenv("INFRAGRAPH_KUBERNETES_CLUSTER_ID"),
			ClusterName:  os.Getenv("INFRAGRAPH_KUBERNETES_CLUSTER_NAME"),
			Timeout:      envDuration("INFRAGRAPH_KUBERNETES_TIMEOUT", 30*time.Second),
			PageSize:     envInt("INFRAGRAPH_KUBERNETES_PAGE_SIZE", 500),
			MaxResources: envInt("INFRAGRAPH_KUBERNETES_MAX_RESOURCES", 100000),
		})
		if err != nil {
			return nil, nil, err
		}
		return connector.Discover(ctx)
	default:
		return nil, nil, fmt.Errorf("unsupported connector type %q", connectorType)
	}
}

func heartbeat(ctx context.Context, control string, id identity, health string) error {
	body := map[string]any{"collectorVersion": version, "protocolVersion": "1.0", "os": runtime.GOOS, "architecture": runtime.GOARCH, "capabilities": []string{"DISCOVER_ASSETS", "DISCOVER_RELATIONSHIPS", "DISCOVER_SERVICES", "DISCOVER_NETWORKS", "DISCOVER_STORAGE", "DISCOVER_METADATA"}, "runningJobs": 0, "healthSummary": health}
	return post(ctx, control+"/collector/v1/heartbeat", id.Credential, body, nil, 1<<20)
}

func post(ctx context.Context, url, credential string, in, out any, max int64) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return postBytes(ctx, url, credential, raw, out, max)
}

func postBytes(ctx context.Context, url, credential string, raw []byte, out any, max int64) error {
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

func nextSequence(data string) (int64, error) {
	path := filepath.Join(data, "sequence")
	var previous int64
	if raw, err := os.ReadFile(path); err == nil {
		previous, _ = strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	}
	sequence := time.Now().UnixNano()
	if sequence <= previous {
		sequence = previous + 1
	}
	temporary := path + ".pending"
	if err := os.WriteFile(temporary, []byte(strconv.FormatInt(sequence, 10)), 0600); err != nil {
		return 0, err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return 0, err
	}
	return sequence, nil
}

func queueSnapshot(data string, snapshot domain.SnapshotEnvelope) error {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	spool := filepath.Join(data, "spool")
	if err = os.MkdirAll(spool, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(spool)
	if err != nil {
		return err
	}
	maxBytes := envInt64("INFRAGRAPH_COLLECTOR_MAX_SPOOL_BYTES", 512<<20)
	var used int64
	for _, entry := range entries {
		if info, infoErr := entry.Info(); infoErr == nil && !entry.IsDir() {
			used += info.Size()
		}
	}
	if used+int64(len(raw)) > maxBytes {
		return fmt.Errorf("collector spool limit exceeded: used=%d max=%d", used, maxBytes)
	}
	path := filepath.Join(spool, fmt.Sprintf("%020d-%s.json", snapshot.Sequence, snapshot.SnapshotID))
	temporary := path + ".pending"
	if err = os.WriteFile(temporary, raw, 0600); err != nil {
		return err
	}
	if err = os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func flushSpool(ctx context.Context, control string, id identity, data string) error {
	spool := filepath.Join(data, "spool")
	entries, err := os.ReadDir(spool)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	maxAge := envDuration("INFRAGRAPH_COLLECTOR_MAX_SPOOL_AGE", 7*24*time.Hour)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if time.Since(info.ModTime()) > maxAge {
			return fmt.Errorf("collector spool contains snapshot older than %s: %s", maxAge, entry.Name())
		}
		path := filepath.Join(spool, entry.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err = postBytes(ctx, control+"/collector/v1/snapshots", id.Credential, raw, nil, 60<<20); err != nil {
			return err
		}
		if err = os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func configuredConnectorType() (string, error) {
	connectorType := strings.ToUpper(strings.TrimSpace(env("INFRAGRAPH_CONNECTOR_TYPE", "DOCKER")))
	if connectorType != "DOCKER" && connectorType != "KUBERNETES" {
		return "", fmt.Errorf("INFRAGRAPH_CONNECTOR_TYPE must be DOCKER or KUBERNETES, got %q", connectorType)
	}
	return connectorType, nil
}

func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("file is empty")
	}
	return raw, nil
}

func envInt64(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
