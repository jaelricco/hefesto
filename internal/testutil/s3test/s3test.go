//go:build integration

// Package s3test gives integration tests real S3-compatible storage.
//
// The server is named by HEFESTO_TEST_S3_ENDPOINT, HEFESTO_TEST_S3_ACCESS_KEY
// and HEFESTO_TEST_S3_SECRET_KEY. `make test-integration` and CI set them by
// running the tests under scripts/with-minio.sh, which builds and starts
// MinIO: its images are no longer published, so testcontainers cannot fetch
// one. Each call to New gets its own bucket.
package s3test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/jaelricco/hefesto/internal/media"
)

const region = "us-east-1"

// ChecksSignatures reports whether the server refuses a presigned request
// that does not match its signature, as MinIO and S3 do. A stand-in that does
// not is declared with HEFESTO_TEST_S3_SKIP_SIGNATURE_CHECKS=true.
func ChecksSignatures() bool { return os.Getenv("HEFESTO_TEST_S3_SKIP_SIGNATURE_CHECKS") != "true" }

// Bucket is a fresh bucket and a Store on it.
type Bucket struct {
	Store *media.S3
	Name  string
}

// New creates a bucket that lives for the test.
func New(t testing.TB) Bucket {
	t.Helper()
	endpoint := os.Getenv("HEFESTO_TEST_S3_ENDPOINT")
	access, secret := os.Getenv("HEFESTO_TEST_S3_ACCESS_KEY"), os.Getenv("HEFESTO_TEST_S3_SECRET_KEY")
	if endpoint == "" {
		t.Fatal("s3test: HEFESTO_TEST_S3_ENDPOINT is not set; run the tests with `make test-integration` " +
			"or under scripts/with-minio.sh")
	}
	name := "t-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
	host := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
	cl, err := minio.New(host, &minio.Options{
		Creds: credentials.NewStaticV4(access, secret, ""), Secure: strings.HasPrefix(endpoint, "https://"),
		Region: region, BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := cl.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: region}); err != nil {
		t.Fatalf("creating bucket: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		for o := range cl.ListObjects(ctx, name, minio.ListObjectsOptions{Recursive: true}) {
			_ = cl.RemoveObject(ctx, name, o.Key, minio.RemoveObjectOptions{})
		}
		_ = cl.RemoveBucket(ctx, name)
	})
	st, err := media.NewS3(media.Config{
		Endpoint: endpoint, Region: region, Bucket: name, AccessKey: access, SecretKey: secret, PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return Bucket{Store: st, Name: name}
}
