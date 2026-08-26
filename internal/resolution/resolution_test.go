package resolution

import (
	"github.com/thiagomontozo/infragraph/internal/domain"
	"testing"
)

func TestStrongAndWeakResolution(t *testing.T) {
	o := domain.Observation{IdentityHints: map[string]any{"docker_container_id": "abc"}}
	r := Resolve(o, nil, map[string]string{"docker_container_id:abc": "asset-1"})
	if r.Decision != Match || r.AssetID != "asset-1" {
		t.Fatal("strong ID did not match")
	}
	o = domain.Observation{IdentityHints: map[string]any{"hostname": "app"}, Attributes: map[string]any{"environment": "staging"}}
	r = Resolve(o, []domain.Asset{{ID: "a", CanonicalName: "app", Environment: "production"}}, nil)
	if r.Decision != Candidate {
		t.Fatal("weak conflicting hostname auto-merged")
	}
}
