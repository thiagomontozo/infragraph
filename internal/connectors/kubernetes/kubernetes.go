package kubernetes

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thiagomontozo/infragraph/internal/domain"
)

const (
	defaultPageSize     = 500
	defaultMaxResources = 100000
	maxResponseBytes    = 10 << 20
)

type Config struct {
	BaseURL, Token, ClusterID, ClusterName string
	CAPEM                                  []byte
	Timeout                                time.Duration
	PageSize, MaxResources                 int
}

type Connector struct {
	base, token, clusterID, clusterName string
	client                              *http.Client
	pageSize, maxResources              int
}

func New(base, token string, timeout time.Duration) (*Connector, error) {
	return NewWithConfig(Config{BaseURL: base, Token: token, Timeout: timeout})
}

func NewWithConfig(config Config) (*Connector, error) {
	u, err := url.Parse(config.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid Kubernetes API URL")
	}
	loopback := u.Hostname() == "localhost"
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		loopback = ip.IsLoopback()
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return nil, errors.New("Kubernetes API must use HTTPS except loopback tests")
	}
	if strings.TrimSpace(config.Token) == "" {
		return nil, errors.New("Kubernetes bearer token is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.PageSize <= 0 || config.PageSize > 1000 {
		config.PageSize = defaultPageSize
	}
	if config.MaxResources <= 0 || config.MaxResources > defaultMaxResources {
		config.MaxResources = defaultMaxResources
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if len(config.CAPEM) > 0 {
		roots, rootErr := x509.SystemCertPool()
		if rootErr != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(config.CAPEM) {
			return nil, errors.New("Kubernetes CA file contains no valid certificate")
		}
		transport.TLSClientConfig.RootCAs = roots
	}
	return &Connector{
		base:         strings.TrimRight(config.BaseURL, "/"),
		token:        strings.TrimSpace(config.Token),
		clusterID:    strings.TrimSpace(config.ClusterID),
		clusterName:  strings.TrimSpace(config.ClusterName),
		client:       &http.Client{Transport: transport, Timeout: config.Timeout},
		pageSize:     config.PageSize,
		maxResources: config.MaxResources,
	}, nil
}

type ownerReference struct {
	Kind, Name, UID string
}

type backend struct {
	Service struct{ Name string }
}

type item struct {
	Metadata struct {
		Name, Namespace, UID string
		Labels               map[string]string
		OwnerReferences      []ownerReference
	}
	Spec struct {
		NodeName, ServiceName, VolumeName string
		Selector                          json.RawMessage
		DefaultBackend                    backend
		Rules                             []struct {
			HTTP struct {
				Paths []struct{ Backend backend }
			}
		}
	}
}

type list struct {
	Metadata struct{ Continue string }
	Items    []item
}

type resource struct {
	path, kind string
	typ        domain.AssetType
}

var resources = []resource{
	{"/api/v1/namespaces", "Namespace", domain.KubernetesNamespace},
	{"/api/v1/nodes", "Node", domain.KubernetesNode},
	{"/api/v1/pods", "Pod", domain.KubernetesPod},
	{"/api/v1/services", "Service", domain.Service},
	{"/apis/apps/v1/deployments", "Deployment", domain.KubernetesWorkload},
	{"/apis/apps/v1/statefulsets", "StatefulSet", domain.KubernetesWorkload},
	{"/apis/apps/v1/daemonsets", "DaemonSet", domain.KubernetesWorkload},
	{"/apis/apps/v1/replicasets", "ReplicaSet", domain.KubernetesWorkload},
	{"/apis/networking.k8s.io/v1/ingresses", "Ingress", domain.LoadBalancer},
	{"/api/v1/persistentvolumes", "PersistentVolume", domain.Storage},
	{"/api/v1/persistentvolumeclaims", "PersistentVolumeClaim", domain.Volume},
}

type discovered struct {
	resource resource
	item     item
}

func (c *Connector) Discover(ctx context.Context) ([]domain.Observation, []domain.RelationshipObservation, error) {
	items := make([]discovered, 0)
	for _, r := range resources {
		listed, err := c.list(ctx, r.path)
		if err != nil {
			return nil, nil, err
		}
		for _, value := range listed {
			if value.Metadata.UID == "" || value.Metadata.Name == "" {
				return nil, nil, fmt.Errorf("Kubernetes API %s returned resource without name or UID", r.path)
			}
			items = append(items, discovered{resource: r, item: value})
			if len(items) > c.maxResources {
				return nil, nil, fmt.Errorf("Kubernetes discovery exceeded resource limit %d", c.maxResources)
			}
		}
	}

	clusterIdentity := c.clusterID
	for _, value := range items {
		if value.resource.kind == "Namespace" && value.item.Metadata.Name == "kube-system" && clusterIdentity == "" {
			clusterIdentity = value.item.Metadata.UID
			break
		}
	}
	if clusterIdentity == "" {
		return nil, nil, errors.New("cannot determine cluster identity: set INFRAGRAPH_KUBERNETES_CLUSTER_ID or allow kube-system namespace listing")
	}
	clusterExternalID := "cluster:" + clusterIdentity
	clusterName := c.clusterName
	if clusterName == "" {
		clusterName = clusterIdentity
	}
	now := time.Now().UTC()
	assets := []domain.Observation{{ExternalID: clusterExternalID, AssetType: domain.KubernetesCluster, ObservedAt: now, Attributes: map[string]any{"name": clusterName}, IdentityHints: map[string]any{"kubernetes_cluster_id": clusterIdentity}}}
	relationships := make([]domain.RelationshipObservation, 0)
	byNamespace := map[string]string{}
	byNode := map[string]string{}
	byName := map[string]string{}
	assetIDs := map[string]bool{clusterExternalID: true}
	pods := make([]discovered, 0)

	for _, value := range items {
		metadata := value.item.Metadata
		attributes := map[string]any{"name": metadata.Name, "kind": value.resource.kind}
		if metadata.Namespace != "" {
			attributes["namespace"] = metadata.Namespace
		}
		if len(metadata.Labels) > 0 {
			attributes["labels"] = metadata.Labels
		}
		if value.item.Spec.NodeName != "" {
			attributes["nodeName"] = value.item.Spec.NodeName
		}
		selector := selectorLabels(value.item.Spec.Selector)
		if len(selector) > 0 {
			attributes["selector"] = selector
		}
		if value.item.Spec.ServiceName != "" {
			attributes["serviceName"] = value.item.Spec.ServiceName
		}
		if value.item.Spec.VolumeName != "" {
			attributes["volumeName"] = value.item.Spec.VolumeName
		}
		assets = append(assets, domain.Observation{ExternalID: metadata.UID, AssetType: value.resource.typ, ObservedAt: now, Attributes: attributes, IdentityHints: map[string]any{"kubernetes_uid": metadata.UID}})
		assetIDs[metadata.UID] = true
		byName[namespacedName(metadata.Namespace, value.resource.kind, metadata.Name)] = metadata.UID
		switch value.resource.kind {
		case "Namespace":
			byNamespace[metadata.Name] = metadata.UID
		case "Node":
			byNode[metadata.Name] = metadata.UID
		case "Pod":
			pods = append(pods, value)
		}
	}

	for _, value := range items {
		metadata := value.item.Metadata
		parent := clusterExternalID
		if metadata.Namespace != "" {
			if namespaceID := byNamespace[metadata.Namespace]; namespaceID != "" {
				parent = namespaceID
			}
		}
		relationships = append(relationships, relation(parent, metadata.UID, "CONTAINS", now))
		for _, owner := range metadata.OwnerReferences {
			if assetIDs[owner.UID] {
				relationships = append(relationships, relation(metadata.UID, owner.UID, "MANAGED_BY", now))
			}
		}
		if nodeID := byNode[value.item.Spec.NodeName]; nodeID != "" {
			relationships = append(relationships, relation(metadata.UID, nodeID, "RUNS_ON", now))
		}
		if value.item.Spec.ServiceName != "" {
			if serviceID := byName[namespacedName(metadata.Namespace, "Service", value.item.Spec.ServiceName)]; serviceID != "" {
				relationships = append(relationships, relation(metadata.UID, serviceID, "USES_SERVICE", now))
			}
		}
		if value.item.Spec.VolumeName != "" {
			if volumeID := byName[namespacedName("", "PersistentVolume", value.item.Spec.VolumeName)]; volumeID != "" {
				relationships = append(relationships, relation(metadata.UID, volumeID, "BOUND_TO", now))
			}
		}
		selector := selectorLabels(value.item.Spec.Selector)
		if value.resource.kind == "Service" && len(selector) > 0 {
			for _, pod := range pods {
				if pod.item.Metadata.Namespace == metadata.Namespace && labelsMatch(selector, pod.item.Metadata.Labels) {
					relationships = append(relationships, relation(metadata.UID, pod.item.Metadata.UID, "ROUTES_TO", now))
				}
			}
		}
		if value.resource.kind == "Ingress" {
			for _, serviceName := range ingressServices(value.item) {
				if serviceID := byName[namespacedName(metadata.Namespace, "Service", serviceName)]; serviceID != "" {
					relationships = append(relationships, relation(metadata.UID, serviceID, "ROUTES_TO", now))
				}
			}
		}
	}
	return assets, relationships, nil
}

func (c *Connector) list(ctx context.Context, path string) ([]item, error) {
	result := make([]item, 0)
	continuation := ""
	for page := 0; ; page++ {
		if page > c.maxResources/c.pageSize+1 {
			return nil, errors.New("Kubernetes pagination exceeded safe page limit")
		}
		query := url.Values{"limit": {fmt.Sprintf("%d", c.pageSize)}}
		if continuation != "" {
			query.Set("continue", continuation)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path+"?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", "infragraph-collector/1.0")
		res, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Kubernetes API %s: %w", path, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(body) > maxResponseBytes {
			return nil, fmt.Errorf("Kubernetes API %s response exceeds %d bytes", path, maxResponseBytes)
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Kubernetes API %s returned %s", path, res.Status)
		}
		var listed list
		if err = json.Unmarshal(body, &listed); err != nil {
			return nil, fmt.Errorf("decode Kubernetes API %s: %w", path, err)
		}
		result = append(result, listed.Items...)
		if len(result) > c.maxResources {
			return nil, fmt.Errorf("Kubernetes API %s exceeded resource limit %d", path, c.maxResources)
		}
		if listed.Metadata.Continue == "" {
			return result, nil
		}
		continuation = listed.Metadata.Continue
	}
}

func namespacedName(namespace, kind, name string) string {
	return namespace + "\x00" + kind + "\x00" + name
}

func labelsMatch(selector, labels map[string]string) bool {
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return len(selector) > 0
}

func selectorLabels(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	direct := map[string]string{}
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct
	}
	structured := struct {
		MatchLabels map[string]string `json:"matchLabels"`
	}{}
	if err := json.Unmarshal(raw, &structured); err == nil {
		return structured.MatchLabels
	}
	return nil
}

func ingressServices(value item) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	add(value.Spec.DefaultBackend.Service.Name)
	for _, rule := range value.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			add(path.Backend.Service.Name)
		}
	}
	return out
}

func relation(from, to, kind string, observedAt time.Time) domain.RelationshipObservation {
	return domain.RelationshipObservation{ExternalFromID: from, ExternalToID: to, Type: kind, ObservedAt: observedAt, Attributes: map[string]any{}}
}
