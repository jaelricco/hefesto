//go:build integration

// Package s3test gives integration tests real S3-compatible storage.
//
// By default it starts the MinIO image Compose uses, once per test binary.
// HEFESTO_TEST_S3_ENDPOINT (with HEFESTO_TEST_S3_ACCESS_KEY and
// HEFESTO_TEST_S3_SECRET_KEY) points at an existing server instead, for
// environments without Docker. Each call to New gets its own bucket.
package s3test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/jaelricco/hefesto/internal/media"
)

// Image matches docker-compose.yml.
const Image = "minio/minio:RELEASE.2025-04-22T22-12-26Z"

const region = "us-east-1"

var (
	once     sync.Once
	endpoint string
	access   string
	secret   string
	setupErr error
)

func setup() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	endpoint = os.Getenv("HEFESTO_TEST_S3_ENDPOINT")
	access, secret = os.Getenv("HEFESTO_TEST_S3_ACCESS_KEY"), os.Getenv("HEFESTO_TEST_S3_SECRET_KEY")
	if endpoint != "" {
		return
	}
	c, err := tcminio.Run(ctx, Image, tcminio.WithUsername("hefesto"), tcminio.WithPassword("hefesto-test"))
	if err != nil {
		setupErr = fmt.Errorf("starting %s (set HEFESTO_TEST_S3_ENDPOINT to use an existing server): %w", Image, err)
		return
	}
	// The container lives as long as the test binary; Ryuk reaps it.
	host, err := c.ConnectionString(ctx)
	if err != nil {
		setupErr = fmt.Errorf("minio address: %w", err)
		return
	}
	endpoint, access, secret = "http://"+host, c.Username, c.Password
}

// IsMinIO reports whether the tests run against MinIO itself, which checks
// signatures; a stand-in given by HEFESTO_TEST_S3_ENDPOINT may not.
func IsMinIO() bool { return os.Getenv("HEFESTO_TEST_S3_ENDPOINT") == "" }

// Bucket is a fresh bucket and a Store on it.
type Bucket struct {
	Store *media.S3
	Name  string
}

// New creates a bucket that lives for the test.
func New(t testing.TB) Bucket {
	t.Helper()
	once.Do(setup)
	if setupErr != nil {
		t.Fatalf("s3test setup: %v", setupErr)
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
