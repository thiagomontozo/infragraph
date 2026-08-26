package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment, HTTPAddr, DatabaseURL, SessionSecret, MasterKey string
	AllowedOrigins                                               []string
	ObjectStorageType, ObjectStoragePath                         string
	S3Endpoint, S3Bucket, S3AccessKey, S3SecretKey               string
	S3UseTLS, OTELEnabled, Debug                                 bool
	OTELEndpoint                                                 string
	MaxGraphDepth, MaxGraphNodes                                 int
	MaxImportBytes, MaxSnapshotBytes                             int64
	MaxConcurrentReconciliations                                 int
	CollectorHeartbeatInterval                                   time.Duration
}

func Load() (Config, error) {
	c := Config{
		Environment: env("INFRAGRAPH_ENV", "development"), HTTPAddr: env("INFRAGRAPH_HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("INFRAGRAPH_DATABASE_URL"), SessionSecret: os.Getenv("INFRAGRAPH_SESSION_SECRET"), MasterKey: os.Getenv("INFRAGRAPH_MASTER_KEY"),
		AllowedOrigins: split(os.Getenv("INFRAGRAPH_ALLOWED_ORIGINS")), ObjectStorageType: env("INFRAGRAPH_OBJECT_STORAGE_TYPE", "local"), ObjectStoragePath: env("INFRAGRAPH_OBJECT_STORAGE_PATH", "./data/objects"),
		S3Endpoint: os.Getenv("INFRAGRAPH_S3_ENDPOINT"), S3Bucket: os.Getenv("INFRAGRAPH_S3_BUCKET"), S3AccessKey: os.Getenv("INFRAGRAPH_S3_ACCESS_KEY"), S3SecretKey: os.Getenv("INFRAGRAPH_S3_SECRET_KEY"),
		S3UseTLS: boolean("INFRAGRAPH_S3_USE_TLS"), OTELEnabled: boolean("INFRAGRAPH_OTEL_ENABLED"), OTELEndpoint: os.Getenv("INFRAGRAPH_OTEL_ENDPOINT"), Debug: boolean("INFRAGRAPH_DEBUG"),
		MaxGraphDepth: integer("INFRAGRAPH_MAX_GRAPH_DEPTH", 6), MaxGraphNodes: integer("INFRAGRAPH_MAX_GRAPH_NODES", 500), MaxImportBytes: int64(integer("INFRAGRAPH_MAX_IMPORT_BYTES", 10<<20)), MaxSnapshotBytes: int64(integer("INFRAGRAPH_MAX_SNAPSHOT_BYTES", 50<<20)), MaxConcurrentReconciliations: integer("INFRAGRAPH_MAX_CONCURRENT_RECONCILIATIONS", 4),
		CollectorHeartbeatInterval: duration("INFRAGRAPH_COLLECTOR_HEARTBEAT_INTERVAL", 30*time.Second),
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if c.MaxGraphDepth < 1 || c.MaxGraphDepth > 20 || c.MaxGraphNodes < 1 || c.MaxGraphNodes > 5000 {
		return errors.New("graph limits are outside safe bounds")
	}
	if c.MaxImportBytes < 1024 || c.MaxSnapshotBytes < 1024 {
		return errors.New("payload limits are invalid")
	}
	if c.Environment != "production" {
		return nil
	}
	var problems []string
	if len(c.SessionSecret) < 32 || strings.Contains(strings.ToLower(c.SessionSecret), "development") || strings.Contains(strings.ToLower(c.SessionSecret), "change-me") {
		problems = append(problems, "strong session secret required")
	}
	key, err := base64.StdEncoding.DecodeString(c.MasterKey)
	if err != nil || len(key) != 32 {
		problems = append(problems, "master key must be base64-encoded 32 bytes")
	}
	if len(c.AllowedOrigins) == 0 {
		problems = append(problems, "allowed origins required")
	}
	for _, o := range c.AllowedOrigins {
		if o == "*" || strings.HasPrefix(o, "http://") {
			problems = append(problems, "production origins must be explicit HTTPS origins")
		}
	}
	if c.Debug {
		problems = append(problems, "debug must be disabled")
	}
	if len(problems) > 0 {
		return fmt.Errorf("unsafe production configuration: %s", strings.Join(problems, "; "))
	}
	return nil
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func split(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func integer(k string, d int) int {
	v, err := strconv.Atoi(os.Getenv(k))
	if err != nil || v == 0 {
		return d
	}
	return v
}
func boolean(k string) bool { v, _ := strconv.ParseBool(os.Getenv(k)); return v }
func duration(k string, d time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(k))
	if err != nil || v <= 0 {
		return d
	}
	return v
}
