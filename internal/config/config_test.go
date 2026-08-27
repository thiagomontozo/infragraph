package config

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestProductionValidation(t *testing.T) {
	t.Setenv("INFRAGRAPH_ENV", "production")
	t.Setenv("INFRAGRAPH_DATABASE_URL", "postgres://infragraph@db.example.test/infragraph?sslmode=verify-full")
	t.Setenv("INFRAGRAPH_SESSION_SECRET", "development-only-change-me")
	t.Setenv("INFRAGRAPH_ALLOWED_ORIGINS", "*")
	if _, err := Load(); err == nil {
		t.Fatal("expected unsafe production configuration rejection")
	}
	t.Setenv("INFRAGRAPH_SESSION_SECRET", strings.Repeat("s", 40))
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INFRAGRAPH_MASTER_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("INFRAGRAPH_ALLOWED_ORIGINS", "https://infra.example.com")
	t.Setenv("INFRAGRAPH_OBJECT_STORAGE_TYPE", "s3")
	t.Setenv("INFRAGRAPH_S3_ENDPOINT", "s3.example.com")
	t.Setenv("INFRAGRAPH_S3_BUCKET", "infragraph")
	t.Setenv("INFRAGRAPH_S3_ACCESS_KEY", "test-access")
	t.Setenv("INFRAGRAPH_S3_SECRET_KEY", "test-secret")
	t.Setenv("INFRAGRAPH_S3_USE_TLS", "true")
	if _, err := Load(); err != nil {
		t.Fatalf("valid production configuration rejected: %v", err)
	}
}

func TestConfigurationRejectsInvalidRuntimeAndProxy(t *testing.T) {
	t.Setenv("INFRAGRAPH_DATABASE_URL", "postgres://localhost/test")
	t.Setenv("INFRAGRAPH_TRUSTED_PROXY_CIDRS", "not-a-cidr")
	if _, err := Load(); err == nil {
		t.Fatal("invalid trusted proxy CIDR accepted")
	}
	t.Setenv("INFRAGRAPH_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	t.Setenv("INFRAGRAPH_MAX_CONCURRENT_RECONCILIATIONS", "1000")
	if _, err := Load(); err == nil {
		t.Fatal("unsafe reconciliation concurrency accepted")
	}
}
