package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"net/http"
	"strings"
	"time"
)

type Connector struct {
	base, token string
	client      *http.Client
}

func New(base, token string, timeout time.Duration) (*Connector, error) {
	if !strings.HasPrefix(base, "https://") && !strings.HasPrefix(base, "http://127.0.0.1") && !strings.HasPrefix(base, "http://localhost") {
		return nil, errors.New("Kubernetes API must use HTTPS except loopback tests")
	}
	return &Connector{strings.TrimRight(base, "/"), token, &http.Client{Timeout: timeout}}, nil
}

type list struct {
	Items []struct {
		Metadata struct {
			Name, Namespace, UID string
			Labels               map[string]string
		}
		Spec struct {
			NodeName, ServiceName string
			Selector              map[string]string
		}
	}
}

var resources = []struct {
	path string
	typ  domain.AssetType
}{{"/api/v1/namespaces", domain.KubernetesNamespace}, {"/api/v1/nodes", domain.KubernetesNode}, {"/api/v1/pods", domain.KubernetesPod}, {"/api/v1/services", domain.Service}, {"/apis/apps/v1/deployments", domain.KubernetesWorkload}, {"/apis/apps/v1/statefulsets", domain.KubernetesWorkload}, {"/apis/apps/v1/daemonsets", domain.KubernetesWorkload}, {"/apis/networking.k8s.io/v1/ingresses", domain.LoadBalancer}, {"/api/v1/persistentvolumes", domain.Storage}, {"/api/v1/persistentvolumeclaims", domain.Volume}}

func (c *Connector) Discover(ctx context.Context) ([]domain.Observation, error) {
	out := []domain.Observation{}
	for _, r := range resources {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+r.path, nil)
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		res, e := c.client.Do(req)
		if e != nil {
			return nil, e
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			return nil, fmt.Errorf("Kubernetes API %s returned %s", r.path, res.Status)
		}
		var v list
		e = json.NewDecoder(http.MaxBytesReader(nil, res.Body, 10<<20)).Decode(&v)
		res.Body.Close()
		if e != nil {
			return nil, e
		}
		for _, i := range v.Items {
			attrs := map[string]any{"name": i.Metadata.Name, "namespace": i.Metadata.Namespace, "labels": i.Metadata.Labels}
			out = append(out, domain.Observation{ExternalID: i.Metadata.UID, AssetType: r.typ, ObservedAt: time.Now().UTC(), Attributes: attrs, IdentityHints: map[string]any{"kubernetes_uid": i.Metadata.UID}})
		}
	}
	return out, nil
}
