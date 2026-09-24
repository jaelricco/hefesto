//go:build integration

package media_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/media"
	"github.com/jaelricco/hefesto/internal/testutil/s3test"
)

func put(t *testing.T, p media.Presigned, body []byte, contentType string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, p.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestPresignedUploadAndDownload(t *testing.T) {
	b := s3test.New(t)
	ctx := context.Background()
	body := []byte("\x89PNG not really")

	up, err := b.Store.PresignPut(ctx, "u/x/one", "image/png", int64(len(body)), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if up.Headers["Content-Type"] != "image/png" || up.Headers["Content-Length"] != "15" {
		t.Fatalf("headers %v", up.Headers)
	}
	if s3test.IsMinIO() {
		// The signature covers the type and the size.
		if code := put(t, up, body, "image/jpeg"); code != http.StatusForbidden {
			t.Fatalf("upload with another content type: %d", code)
		}
		if code := put(t, up, append(body, '!'), "image/png"); code < 400 {
			t.Fatalf("upload of another size: %d", code)
		}
	}
	if code := put(t, up, body, "image/png"); code != http.StatusOK {
		t.Fatalf("upload: %d", code)
	}

	obj, err := b.Store.Stat(ctx, "u/x/one")
	if err != nil || obj.Size != int64(len(body)) || obj.ContentType != "image/png" {
		t.Fatalf("stat %+v %v", obj, err)
	}
	down, err := b.Store.PresignGet(ctx, "u/x/one", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(down.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Equal(got, body) {
		t.Fatalf("download %d %q", resp.StatusCode, got)
	}

	if _, err := b.Store.Stat(ctx, "u/x/missing"); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("missing object: %v", err)
	}
	if err := b.Store.Remove(ctx, "u/x/one"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Store.Stat(ctx, "u/x/one"); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("removed object: %v", err)
	}
	if err := b.Store.Remove(ctx, "u/x/one"); err != nil {
		t.Fatalf("removing twice: %v", err)
	}
}

func TestRemovePrefix(t *testing.T) {
	b := s3test.New(t)
	ctx := context.Background()
	for _, k := range []string{"u/a/1", "u/a/2", "u/b/1"} {
		up, err := b.Store.PresignPut(ctx, k, "image/png", 1, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if code := put(t, up, []byte("x"), "image/png"); code != http.StatusOK {
			t.Fatalf("upload %s: %d", k, code)
		}
	}
	if err := b.Store.RemovePrefix(ctx, "u/a/"); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]bool{"u/a/1": false, "u/a/2": false, "u/b/1": true} {
		_, err := b.Store.Stat(ctx, k)
		if exists := err == nil; exists != want {
			t.Fatalf("%s exists=%v, want %v (%v)", k, exists, want, err)
		}
	}
}

func TestConfig(t *testing.T) {
	if _, err := media.NewS3(media.Config{Endpoint: "minio:9000", Region: "r", Bucket: "b"}); err == nil ||
		!strings.Contains(err.Error(), "not an http(s) URL") {
		t.Fatalf("endpoint without scheme: %v", err)
	}
	if _, err := media.NewS3(media.Config{Endpoint: "http://minio:9000"}); err == nil {
		t.Fatal("missing bucket accepted")
	}
}
