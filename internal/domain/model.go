package domain

import (
	"encoding/json"
	"time"
)

type AssetStatus string

const (
	AssetActive      AssetStatus = "ACTIVE"
	AssetStale       AssetStatus = "STALE"
	AssetMissing     AssetStatus = "MISSING"
	AssetConflicting AssetStatus = "CONFLICTING"
	AssetRetired     AssetStatus = "RETIRED"
	AssetUnknown     AssetStatus = "UNKNOWN"
)

type AssetType string

const (
	PhysicalHost        AssetType = "PHYSICAL_HOST"
	VirtualMachine      AssetType = "VIRTUAL_MACHINE"
	Container           AssetType = "CONTAINER"
	ContainerImage      AssetType = "CONTAINER_IMAGE"
	KubernetesCluster   AssetType = "KUBERNETES_CLUSTER"
	KubernetesNode      AssetType = "KUBERNETES_NODE"
	KubernetesNamespace AssetType = "KUBERNETES_NAMESPACE"
	KubernetesWorkload  AssetType = "KUBERNETES_WORKLOAD"
	KubernetesPod       AssetType = "KUBERNETES_POD"
	Application         AssetType = "APPLICATION"
	Service             AssetType = "SERVICE"
	Database            AssetType = "DATABASE"
	Cache               AssetType = "CACHE"
	MessageBroker       AssetType = "MESSAGE_BROKER"
	Storage             AssetType = "STORAGE"
	Network             AssetType = "NETWORK"
	IPAddress           AssetType = "IP_ADDRESS"
	DNSName             AssetType = "DNS_NAME"
	LoadBalancer        AssetType = "LOAD_BALANCER"
	Certificate         AssetType = "CERTIFICATE"
	Volume              AssetType = "VOLUME"
	CloudResource       AssetType = "CLOUD_RESOURCE"
	Unknown             AssetType = "UNKNOWN"
)

var AssetTypeLabels = map[AssetType]string{PhysicalHost: "Physical host", VirtualMachine: "Virtual machine", Container: "Container", ContainerImage: "Container image", KubernetesCluster: "Kubernetes cluster", KubernetesNode: "Kubernetes node", KubernetesNamespace: "Kubernetes namespace", KubernetesWorkload: "Kubernetes workload", KubernetesPod: "Kubernetes pod", Application: "Application", Service: "Service", Database: "Database", Cache: "Cache", MessageBroker: "Message broker", Storage: "Storage", Network: "Network", IPAddress: "IP address", DNSName: "DNS name", LoadBalancer: "Load balancer", Certificate: "Certificate", Volume: "Volume", CloudResource: "Cloud resource", Unknown: "Unknown"}

type Asset struct {
	ID, OrganizationID, CanonicalName, DisplayName string
	Type                                           AssetType
	Status                                         AssetStatus
	Environment, SiteID, OwnerID, Criticality      string
	FirstSeenAt, LastSeenAt, CreatedAt, UpdatedAt  time.Time
	RetiredAt                                      *time.Time
}
type Observation struct {
	ID, OrganizationID, SnapshotID, ConnectorID, CollectorID, ExternalID string
	AssetType                                                            AssetType
	ObservedAt                                                           time.Time
	Attributes, IdentityHints                                            map[string]any
	Fingerprint, Status                                                  string
}
type AttributeClaim struct {
	AssetID, AttributeKey   string
	Value                   any
	ConnectorID, SnapshotID string
	ObservedAt              time.Time
	Authority, Confidence   string
	Active                  bool
}
type Relationship struct {
	ID, OrganizationID, FromAssetID, ToAssetID, Type, Status string
	FirstSeenAt, LastSeenAt, CreatedAt, UpdatedAt            time.Time
}
type RelationshipObservation struct {
	RelationshipID, SnapshotID, ConnectorID, ExternalFromID, ExternalToID, Type string
	ObservedAt                                                                  time.Time
	Attributes                                                                  map[string]any
}

type SnapshotStatus string

const (
	SnapshotQueued    SnapshotStatus = "QUEUED"
	SnapshotRunning   SnapshotStatus = "RUNNING"
	SnapshotSucceeded SnapshotStatus = "SUCCEEDED"
	SnapshotPartial   SnapshotStatus = "PARTIAL"
	SnapshotFailed    SnapshotStatus = "FAILED"
	SnapshotCancelled SnapshotStatus = "CANCELLED"
	SnapshotRejected  SnapshotStatus = "REJECTED"
)

type SnapshotEnvelope struct {
	ProtocolVersion    string                    `json:"protocolVersion"`
	SnapshotID         string                    `json:"snapshotId"`
	OrganizationID     string                    `json:"organizationId"`
	CollectorID        string                    `json:"collectorId"`
	ConnectorID        string                    `json:"connectorId"`
	ConnectorType      string                    `json:"connectorType"`
	ConnectorVersion   string                    `json:"connectorVersion"`
	StartedAt          time.Time                 `json:"startedAt"`
	CompletedAt        time.Time                 `json:"completedAt"`
	Sequence           int64                     `json:"sequence"`
	Assets             []Observation             `json:"assets"`
	Relationships      []RelationshipObservation `json:"relationships"`
	Warnings           []string                  `json:"warnings"`
	Statistics         map[string]int            `json:"statistics"`
	ContentHash        string                    `json:"contentHash"`
	SignatureAlgorithm string                    `json:"signatureAlgorithm"`
	Signature          string                    `json:"signature"`
}

func (s SnapshotEnvelope) SigningBytes() ([]byte, error) { s.Signature = ""; return json.Marshal(s) }

type EffectiveAttribute struct {
	Value    any              `json:"value"`
	Conflict bool             `json:"conflict"`
	Reason   string           `json:"reason"`
	Claims   []AttributeClaim `json:"claims"`
}
type Finding struct {
	ID, OrganizationID, AssetID, Type, Status, Priority, Explanation string
	CreatedAt                                                        time.Time
}
type Change struct {
	ID, OrganizationID, AssetID, RelationshipID, Type string
	BeforeValue, AfterValue                           json.RawMessage
	Source                                            string
	DetectedAt                                        time.Time
	Summary, Confidence, LogicalKey                   string
}
