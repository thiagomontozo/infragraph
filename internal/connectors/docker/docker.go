package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"net"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"
	"time"
)

type Connector struct {
	client         *http.Client
	baseURL, label string
}

func NewUnix(socket, label string, timeout time.Duration) (*Connector, error) {
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	if label == "" {
		return nil, errors.New("a label scope is required")
	}
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		if runtime.GOOS == "windows" {
			return nil, errors.New("Unix Docker socket is unavailable on Windows; run collector in Linux container")
		}
		return (&net.Dialer{Timeout: timeout}).DialContext(ctx, "unix", socket)
	}}
	return &Connector{&http.Client{Transport: tr, Timeout: timeout}, "http://docker", label}, nil
}
func NewHTTP(base, label string, timeout time.Duration) (*Connector, error) {
	u, e := url.Parse(base)
	if e != nil || u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("invalid Docker API URL")
	}
	if label == "" {
		return nil, errors.New("a label scope is required")
	}
	return &Connector{&http.Client{Timeout: timeout}, strings.TrimRight(base, "/"), label}, nil
}

type item struct {
	ID              string
	Names           []string
	Image, ImageID  string
	Name            string
	Labels          map[string]string
	NetworkSettings struct{ Networks map[string]any }
	Mounts          []struct{ Name, Type string }
}

func (c *Connector) get(ctx context.Context, p string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+p, nil)
	res, e := c.client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Docker API returned %s", res.Status)
	}
	return json.NewDecoder(http.MaxBytesReader(nil, res.Body, 10<<20)).Decode(out)
}
func (c *Connector) Discover(ctx context.Context) ([]domain.Observation, []domain.RelationshipObservation, error) {
	filter := url.QueryEscape(fmt.Sprintf(`{"label":["%s"]}`, c.label))
	var containers []item
	if e := c.get(ctx, "/containers/json?all=1&filters="+filter, &containers); e != nil {
		return nil, nil, e
	}
	now := time.Now().UTC()
	host := domain.Observation{ExternalID: "docker-host", AssetType: domain.PhysicalHost, ObservedAt: now, Attributes: map[string]any{"name": "Docker Host"}, IdentityHints: map[string]any{}}
	assets := []domain.Observation{host}
	rels := []domain.RelationshipObservation{}
	seen := map[string]bool{"docker-host": true}
	byName := map[string]string{}
	for _, v := range containers {
		byName[strings.TrimPrefix(first(v.Names), "/")] = v.ID
	}
	add := func(id string, t domain.AssetType, name string, hints map[string]any) {
		if seen[id] {
			return
		}
		seen[id] = true
		assets = append(assets, domain.Observation{ExternalID: id, AssetType: t, ObservedAt: now, Attributes: map[string]any{"name": name}, IdentityHints: hints})
	}
	for _, v := range containers {
		name := strings.TrimPrefix(first(v.Names), "/")
		add(v.ID, domain.Container, name, map[string]any{"docker_container_id": v.ID})
		add(v.ImageID, domain.ContainerImage, v.Image, map[string]any{"docker_image_id": v.ImageID})
		rels = append(rels, relation(v.ID, "docker-host", "RUNS_ON", now), relation(v.ID, v.ImageID, "USES_IMAGE", now))
		for n := range v.NetworkSettings.Networks {
			id := "network:" + n
			add(id, domain.Network, n, nil)
			rels = append(rels, relation(v.ID, id, "CONNECTED_TO", now))
		}
		for _, m := range v.Mounts {
			if m.Type == "volume" && m.Name != "" {
				id := "volume:" + m.Name
				add(id, domain.Volume, m.Name, nil)
				rels = append(rels, relation(v.ID, id, "USES_VOLUME", now))
			}
		}
		if target := byName[v.Labels["com.infragraph.depends-on"]]; target != "" {
			rels = append(rels, relation(v.ID, target, "CONNECTED_TO", now))
		}
	}
	return assets, rels, nil
}
func relation(a, b, t string, at time.Time) domain.RelationshipObservation {
	return domain.RelationshipObservation{ExternalFromID: a, ExternalToID: b, Type: t, ObservedAt: at, Attributes: map[string]any{}}
}
func first(v []string) string {
	for _, s := range v {
		if s != "" {
			return path.Base(s)
		}
	}
	return ""
}
