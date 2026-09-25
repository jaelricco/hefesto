// Package media keeps uploaded files in S3-compatible object storage: MinIO
// in development, Hetzner Object Storage in production. Clients upload and
// download straight to and from the bucket through presigned URLs; the API
// only signs them and checks what arrived.
package media

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is an object that does not exist.
var ErrNotFound = errors.New("object not found")

// Store is the object storage the API needs.
type Store interface {
	// PresignPut signs an upload of exactly size bytes of contentType.
	PresignPut(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (Presigned, error)
	// PresignGet signs a download.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (Presigned, error)
	// Stat describes a stored object, or returns ErrNotFound.
	Stat(ctx context.Context, key string) (Object, error)
	// Remove deletes an object; a missing one is not an error.
	Remove(ctx context.Context, key string) error
	// RemovePrefix deletes every object under prefix.
	RemovePrefix(ctx context.Context, prefix string) error
}

// Presigned is a signed request a client makes directly to the bucket.
type Presigned struct {
	URL       string
	Headers   map[string]string // headers the client must send, part of the signature
	ExpiresAt time.Time
}

// Object is what storage knows about a stored object.
type Object struct {
	Size        int64
	ContentType string
}

// Config locates the bucket. PublicEndpoint is the address clients reach,
// when it differs from the one the API uses (in Compose the API talks to
// http://minio:9000, a phone to the host's address).
type Config struct {
	Endpoint       string // e.g. http://minio:9000
	PublicEndpoint string // defaults to Endpoint
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	PathStyle      bool
}

// S3 is a Store on an S3-compatible service.
type S3 struct {
	api    *minio.Client // requests the API itself makes
	signer *minio.Client // signs URLs for clients; makes no requests
	bucket string
}

// NewS3 builds the client. It makes no network call.
func NewS3(c Config) (*S3, error) {
	if c.Bucket == "" || c.Region == "" {
		return nil, errors.New("media: bucket and region are required")
	}
	public := c.PublicEndpoint
	if public == "" {
		public = c.Endpoint
	}
	api, err := client(c.Endpoint, c)
	if err != nil {
		return nil, err
	}
	signer, err := client(public, c)
	if err != nil {
		return nil, err
	}
	return &S3{api: api, signer: signer, bucket: c.Bucket}, nil
}

func client(endpoint string, c Config) (*minio.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("media: endpoint %q is not an http(s) URL", endpoint)
	}
	lookup := minio.BucketLookupDNS
	if c.PathStyle {
		lookup = minio.BucketLookupPath
	}
	// With the region given, presigning never asks the server for it.
	cl, err := minio.New(u.Host, &minio.Options{
		Creds:        credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure:       u.Scheme == "https",
		Region:       c.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("media: client for %s: %w", endpoint, err)
	}
	return cl, nil
}

// PresignPut implements Store. Content-Type and Content-Length are signed,
// so the bucket refuses any other file.
func (s *S3) PresignPut(ctx context.Context, key, contentType string, size int64, ttl time.Duration) (Presigned, error) {
	h := http.Header{}
	h.Set("Content-Type", contentType)
	h.Set("Content-Length", strconv.FormatInt(size, 10))
	u, err := s.signer.PresignHeader(ctx, http.MethodPut, s.bucket, key, ttl, nil, h)
	if err != nil {
		return Presigned{}, fmt.Errorf("presigning upload: %w", err)
	}
	return Presigned{
		URL:       u.String(),
		Headers:   map[string]string{"Content-Type": contentType, "Content-Length": strconv.FormatInt(size, 10)},
		ExpiresAt: time.Now().Add(ttl),
	}, nil
}

// PresignGet implements Store.
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (Presigned, error) {
	u, err := s.signer.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return Presigned{}, fmt.Errorf("presigning download: %w", err)
	}
	return Presigned{URL: u.String(), Headers: map[string]string{}, ExpiresAt: time.Now().Add(ttl)}, nil
}

// Stat implements Store.
func (s *S3) Stat(ctx context.Context, key string) (Object, error) {
	info, err := s.api.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if code := minio.ToErrorResponse(err).Code; code == "NoSuchKey" || code == "NotFound" {
			return Object{}, ErrNotFound
		}
		return Object{}, fmt.Errorf("reading object: %w", err)
	}
	return Object{Size: info.Size, ContentType: info.ContentType}, nil
}

// Remove implements Store.
func (s *S3) Remove(ctx context.Context, key string) error {
	if err := s.api.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("removing object: %w", err)
	}
	return nil
}

// RemovePrefix implements Store.
func (s *S3) RemovePrefix(ctx context.Context, prefix string) error {
	objects := s.api.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	for e := range s.api.RemoveObjects(ctx, s.bucket, objects, minio.RemoveObjectsOptions{}) {
		if e.Err != nil {
			return fmt.Errorf("removing %s: %w", e.ObjectName, e.Err)
		}
	}
	return nil
}

// Ping checks that the bucket is reachable, for readiness.
func (s *S3) Ping(ctx context.Context) error {
	ok, err := s.api.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("reaching bucket: %w", err)
	}
	if !ok {
		return fmt.Errorf("bucket %s does not exist", s.bucket)
	}
	return nil
}
