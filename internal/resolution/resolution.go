package resolution

import (
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"strings"
)

type Decision string

const (
	Match     Decision = "MATCH"
	NewAsset  Decision = "NEW_ASSET"
	Candidate Decision = "CANDIDATE"
)

type Result struct {
	Decision   Decision
	AssetID    string
	Confidence string
	Reasons    []string
}

var strongKeys = []string{"cloud_id", "kubernetes_uid", "docker_container_id", "vm_uuid", "hardware_uuid", "terraform_resource_id", "external_cmdb_id"}

func Resolve(observation domain.Observation, assets []domain.Asset, identities map[string]string) Result {
	for _, k := range strongKeys {
		v := strings.TrimSpace(fmt.Sprint(observation.IdentityHints[k]))
		if v == "" {
			continue
		}
		if id := identities[k+":"+v]; id != "" {
			return Result{Match, id, "HIGH", []string{"matching immutable identifier: " + k}}
		}
	}
	for _, a := range assets {
		if !strings.EqualFold(a.CanonicalName, strings.TrimSpace(fmt.Sprint(observation.IdentityHints["hostname"]))) {
			continue
		}
		oe := strings.TrimSpace(fmt.Sprint(observation.Attributes["environment"]))
		if oe != "" && a.Environment != "" && !strings.EqualFold(oe, a.Environment) {
			return Result{Candidate, a.ID, "LOW", []string{"hostname matches but environments differ"}}
		}
		return Result{Candidate, a.ID, "LOW", []string{"hostname is a weak identifier"}}
	}
	return Result{Decision: NewAsset, Confidence: "UNKNOWN", Reasons: []string{"no known identity matched"}}
}
