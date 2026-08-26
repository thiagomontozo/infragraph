package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
	"strings"
	"testing"
	"time"
)

func TestS3CompatibleRoundtrip(t *testing.T) {
	endpoint := os.Getenv("INFRAGRAPH_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("MinIO integration is opt-in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	access, secret, bucket := "testaccess", "testsecret-change-me", "infragraph-test"
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(access, secret, "")})
	if err != nil {
		t.Fatal(err)
	}
	if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		exists, _ := client.BucketExists(ctx, bucket)
		if !exists {
			t.Fatal(err)
		}
	}
	defer client.RemoveBucket(context.Background(), bucket)
	s, err := NewS3(endpoint, bucket, access, secret, false)
	if err != nil {
		t.Fatal(err)
	}
	o, err := s.Put(ctx, "org/snapshot.json", strings.NewReader("{}"), 2, "application/json")
	if err != nil || o.Size != 2 {
		t.Fatal(err)
	}
	if err = s.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, o.Key); err != nil {
		t.Fatal(err)
	}
}
