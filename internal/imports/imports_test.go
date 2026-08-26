package imports

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCSVPreviewAndLimits(t *testing.T) {
	p, err := CSV(strings.NewReader("external_id,asset_type,name,environment\na,APPLICATION,ERP,staging\n"), Limits{MaxBytes: 1024, MaxRows: 10})
	if err != nil || len(p.Assets) != 1 || p.Assets[0].Attributes["environment"] != "STAGING" {
		t.Fatalf("bad preview %#v %v", p, err)
	}
	if _, err = CSV(strings.NewReader(strings.Repeat("x", 100)), Limits{MaxBytes: 10}); err == nil {
		t.Fatal("oversized CSV accepted")
	}
	if SafeCSVCell("=cmd") != "'=cmd" {
		t.Fatal("formula not neutralized")
	}
}
func TestJSONDepthAndUnknownFields(t *testing.T) {
	if _, err := JSON(strings.NewReader(`{"assets":[],"relationships":[],"unknown":true}`), Limits{MaxBytes: 1024}); err == nil {
		t.Fatal("unknown field accepted")
	}
	if _, err := JSON(strings.NewReader(`{"assets":[[[[]]]],"relationships":[]}`), Limits{MaxBytes: 1024, MaxDepth: 3}); err == nil {
		t.Fatal("deep JSON accepted")
	}
}
func TestTerraformDataMinimization(t *testing.T) {
	state := `{"outputs":{"pw":{"sensitive":true,"value":"leak"}},"resources":[{"mode":"managed","type":"aws_instance","name":"app","provider":"aws","instances":[{"attributes":{"id":"i-1","name":"app","password":"leak","user_data":"leak","tags":{"env":"prod"},"arbitrary":"drop"}}]}]}`
	p, err := Terraform(strings.NewReader(state), 10240)
	if err != nil || len(p.Assets) != 1 {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	for _, bad := range []string{"password", "user_data", "arbitrary", "leak"} {
		if bytes.Contains(raw, []byte(bad)) {
			t.Fatalf("sensitive/unapproved field retained: %s", bad)
		}
	}
}
