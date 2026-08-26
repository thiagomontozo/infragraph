package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestProductionValidation(t *testing.T) {
	t.Setenv("INFRAGRAPH_ENV", "production")
	t.Setenv("INFRAGRAPH_SESSION_SECRET", "development-only-change-me")
	t.Setenv("INFRAGRAPH_ALLOWED_ORIGINS", "*")
	if _, err := Load(); err == nil {
		t.Fatal("expected unsafe production configuration rejection")
	}
	t.Setenv("INFRAGRAPH_SESSION_SECRET", strings.Repeat("s", 40))
	t.Setenv("INFRAGRAPH_MASTER_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("INFRAGRAPH_ALLOWED_ORIGINS", "https://infra.example.com")
	if _, err := Load(); err != nil {
		t.Fatalf("valid production configuration rejected: %v", err)
	}
}
