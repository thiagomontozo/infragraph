package imports

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"io"
	"strings"
	"unicode/utf8"
)

type Limits struct {
	MaxBytes                    int64
	MaxItems, MaxDepth, MaxRows int
}
type Preview struct {
	Assets        []domain.Observation             `json:"assets"`
	Relationships []domain.RelationshipObservation `json:"relationships"`
	Errors        []string                         `json:"errors"`
}

func CSV(r io.Reader, l Limits) (Preview, error) {
	if l.MaxBytes <= 0 {
		l.MaxBytes = 10 << 20
	}
	if l.MaxRows <= 0 {
		l.MaxRows = 10000
	}
	raw, e := io.ReadAll(io.LimitReader(r, l.MaxBytes+1))
	if e != nil {
		return Preview{}, e
	}
	if int64(len(raw)) > l.MaxBytes {
		return Preview{}, errors.New("CSV exceeds byte limit")
	}
	if !utf8.Valid(raw) {
		return Preview{}, errors.New("CSV must be UTF-8")
	}
	cr := csv.NewReader(bytes.NewReader(raw))
	cr.ReuseRecord = true
	header, e := cr.Read()
	if e != nil {
		return Preview{}, e
	}
	indexes := map[string]int{}
	for i, h := range header {
		indexes[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, required := range []string{"external_id", "asset_type", "name"} {
		if _, ok := indexes[required]; !ok {
			return Preview{}, fmt.Errorf("missing required column %s", required)
		}
	}
	p := Preview{}
	for row := 1; ; row++ {
		record, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			p.Errors = append(p.Errors, fmt.Sprintf("row %d: %v", row+1, e))
			continue
		}
		if row > l.MaxRows {
			return Preview{}, errors.New("CSV exceeds row limit")
		}
		get := func(k string) string {
			i := indexes[k]
			if i >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[i])
		}
		typ := domain.AssetType(strings.ToUpper(get("asset_type")))
		if _, ok := domain.AssetTypeLabels[typ]; !ok {
			p.Errors = append(p.Errors, fmt.Sprintf("row %d: unknown asset type", row+1))
			continue
		}
		attrs := map[string]any{"name": get("name")}
		if i, ok := indexes["environment"]; ok && i < len(record) {
			attrs["environment"] = strings.ToUpper(strings.TrimSpace(record[i]))
		}
		p.Assets = append(p.Assets, domain.Observation{ExternalID: get("external_id"), AssetType: typ, Attributes: attrs, IdentityHints: map[string]any{}})
	}
	return p, nil
}

type JSONDocument struct {
	Assets        []domain.Observation             `json:"assets"`
	Relationships []domain.RelationshipObservation `json:"relationships"`
}

func JSON(r io.Reader, l Limits) (Preview, error) {
	if l.MaxBytes <= 0 {
		l.MaxBytes = 10 << 20
	}
	if l.MaxItems <= 0 {
		l.MaxItems = 10000
	}
	if l.MaxDepth <= 0 {
		l.MaxDepth = 16
	}
	raw, e := io.ReadAll(io.LimitReader(r, l.MaxBytes+1))
	if e != nil {
		return Preview{}, e
	}
	if int64(len(raw)) > l.MaxBytes {
		return Preview{}, errors.New("JSON exceeds byte limit")
	}
	if depth(raw) > l.MaxDepth {
		return Preview{}, errors.New("JSON exceeds depth limit")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var d JSONDocument
	if e = dec.Decode(&d); e != nil {
		return Preview{}, fmt.Errorf("invalid inventory JSON: %w", e)
	}
	if len(d.Assets)+len(d.Relationships) > l.MaxItems {
		return Preview{}, errors.New("JSON exceeds item limit")
	}
	for _, a := range d.Assets {
		if a.ExternalID == "" {
			return Preview{}, errors.New("asset externalId required")
		}
		if _, ok := domain.AssetTypeLabels[a.AssetType]; !ok {
			return Preview{}, errors.New("unknown asset type")
		}
	}
	return Preview{Assets: d.Assets, Relationships: d.Relationships}, nil
}
func depth(raw []byte) int {
	d, max := 0, 0
	in, esc := false, false
	for _, c := range raw {
		if in {
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				in = false
			}
			continue
		}
		if c == '"' {
			in = true
		} else if c == '{' || c == '[' {
			d++
			if d > max {
				max = d
			}
		} else if c == '}' || c == ']' {
			d--
		}
	}
	return max
}

func SafeCSVCell(v string) string {
	if v == "" {
		return v
	}
	if strings.ContainsRune("=+-@", rune(v[0])) {
		return "'" + v
	}
	return v
}

type terraformState struct {
	Resources []struct {
		Mode, Type, Name, Provider string
		Instances                  []struct {
			Attributes map[string]json.RawMessage `json:"attributes"`
		} `json:"instances"`
	} `json:"resources"`
	Outputs map[string]struct {
		Sensitive bool            `json:"sensitive"`
		Value     json.RawMessage `json:"value"`
	} `json:"outputs"`
}

var terraformAllow = map[string]bool{"id": true, "name": true, "arn": true, "resource_group_name": true, "location": true, "region": true, "availability_zone": true, "instance_id": true, "cluster_name": true, "namespace": true, "tags": true, "labels": true}

func Terraform(r io.Reader, maxBytes int64) (Preview, error) {
	if maxBytes <= 0 {
		maxBytes = 50 << 20
	}
	raw, e := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if e != nil {
		return Preview{}, e
	}
	if int64(len(raw)) > maxBytes {
		return Preview{}, errors.New("Terraform state exceeds byte limit")
	}
	var s terraformState
	if e = json.Unmarshal(raw, &s); e != nil {
		return Preview{}, errors.New("invalid Terraform state")
	}
	p := Preview{}
	for _, resource := range s.Resources {
		if resource.Mode != "managed" {
			continue
		}
		for i, instance := range resource.Instances {
			attrs := map[string]any{"terraform_address": fmt.Sprintf("%s.%s", resource.Type, resource.Name), "terraform_type": resource.Type, "provider": resource.Provider}
			hints := map[string]any{"terraform_resource_id": fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, i)}
			for k, v := range instance.Attributes {
				if !terraformAllow[k] || sensitiveName(k) {
					continue
				}
				var value any
				if json.Unmarshal(v, &value) == nil {
					attrs[k] = value
					if k == "id" {
						hints["terraform_resource_id"] = fmt.Sprint(value)
					}
				}
			}
			p.Assets = append(p.Assets, domain.Observation{ExternalID: fmt.Sprint(hints["terraform_resource_id"]), AssetType: domain.CloudResource, Attributes: attrs, IdentityHints: hints})
		}
	}
	return p, nil
}
func sensitiveName(k string) bool {
	k = strings.ToLower(k)
	for _, bad := range []string{"password", "secret", "token", "private_key", "connection_string", "user_data", "credential"} {
		if strings.Contains(k, bad) {
			return true
		}
	}
	return false
}

func ReadLines(r io.Reader, max int) ([]string, error) {
	s := bufio.NewScanner(r)
	var out []string
	for s.Scan() {
		if len(out) >= max {
			return nil, errors.New("line limit exceeded")
		}
		out = append(out, s.Text())
	}
	return out, s.Err()
}
