//go:build integration

package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/media"
	"github.com/jaelricco/hefesto/internal/store"
)

// upload plays the client's part: it sends the bytes to the presigned URL
// with exactly the headers the API returned.
func upload(t *testing.T, up map[string]any, body []byte) int {
	t.Helper()
	req, err := http.NewRequest(up["method"].(string), up["url"].(string), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range up["headers"].(map[string]any) {
		if k != "Content-Length" { // net/http sets it from the body
			req.Header.Set(k, v.(string))
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func reserve(a *api, u user, id string, size int) map[string]any {
	a.t.Helper()
	return a.call("POST", "/v1/media/uploads", u.access, map[string]any{
		"id": id, "kind": "image", "mime": "image/jpeg", "bytes": size, "width": 1080, "height": 1920,
	}).ok(201, "MediaUpload")
}

func TestMediaUploadAttachAndDelete(t *testing.T) {
	a := newAPI(t, withMedia())
	u := a.register("media@example.test")
	photo := []byte("\xff\xd8\xff a form check, honest")
	id := newID()

	m := reserve(a, u, id, len(photo))
	if m["media"].(map[string]any)["status"] != "pending" || m["media"].(map[string]any)["url"] != nil {
		t.Fatalf("reserved %v", m)
	}
	// Images only, for now (ADR 0002 §4).
	a.call("POST", "/v1/media/uploads", u.access, map[string]any{
		"id": newID(), "kind": "video", "mime": "video/mp4", "bytes": 10,
	}).problem(422, "validation")
	// Nothing uploaded yet.
	a.call("POST", "/v1/media/"+id+"/complete", u.access, nil).problem(422, "validation")
	// Reserving again gives a fresh URL for the same asset; another shape is a clash.
	up := reserve(a, u, id, len(photo))["upload"].(map[string]any)
	a.call("POST", "/v1/media/uploads", u.access, map[string]any{
		"id": id, "kind": "image", "mime": "image/png", "bytes": len(photo),
	}).problem(409, "already-exists")

	if code := upload(t, up, photo); code != http.StatusOK {
		t.Fatalf("upload: %d", code)
	}
	ready := a.call("POST", "/v1/media/"+id+"/complete", u.access, nil).ok(200, "Media")
	if ready["status"] != "ready" || ready["url"] == nil {
		t.Fatalf("completed %v", ready)
	}
	// Idempotent.
	a.call("POST", "/v1/media/"+id+"/complete", u.access, nil).ok(200, "Media")
	resp, err := http.Get(ready["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !bytes.Equal(got, photo) {
		t.Fatalf("downloaded %q", got)
	}

	// Attached to an element through the one set path.
	l := logger{a: a, u: u, session: newID(), block: newID()}
	a.call("POST", "/v1/sessions", u.access, map[string]any{
		"id": l.session, "started_at": "2026-09-23T10:00:00Z", "timezone": "Europe/Zurich",
	}).ok(201, "Session")
	a.call("PUT", l.path("blocks", l.block), u.access, map[string]any{"order_index": 0}).ok(201, "Block")
	setID := newID()
	el := reps(a.exerciseID("pull-up"), 5)
	el["media_ids"] = []string{id}
	if ids := elementsOf(l.putSet(setID, 0, el).ok(201, "SetEntry"))[0]["media_ids"].([]any); len(ids) != 1 || ids[0] != id {
		t.Fatalf("attached %v", ids)
	}
	_, cursor, _ := a.pullAll(u, 0, 100)

	// Only a ready image of the athlete's own can be attached.
	pending := newID()
	reserve(a, u, pending, 10)
	other := a.register("media-other@example.test")
	theirs := newID()
	reserve(a, other, theirs, 10)
	el["media_ids"] = []string{pending, theirs}
	errs := l.putSet(setID, 0, el).problem(422, "validation")
	hasField(t, errs, "/elements/0/media_ids/0")
	hasField(t, errs, "/elements/0/media_ids/1")
	a.call("GET", "/v1/media/"+id, other.access, nil).problem(404, "not-found")

	// Deleting the asset detaches it, and the detachment syncs.
	a.call("DELETE", "/v1/media/"+id, u.access, nil).ok(204, "")
	a.call("GET", "/v1/media/"+id, u.access, nil).problem(404, "not-found")
	s := a.call("GET", l.path(), u.access, nil).ok(200, "Session")
	if ids := s["blocks"].([]any)[0].(map[string]any)["sets"].([]any)[0].(map[string]any)["elements"].([]any)[0].(map[string]any)["media_ids"].([]any); len(ids) != 0 {
		t.Fatalf("still attached %v", ids)
	}
	seen, _, _ := a.pullAll(u, cursor, 100)
	if st, ok := seen[setID]; !ok || len(st["elements"].([]any)[0].(map[string]any)["media_ids"].([]any)) != 0 {
		t.Fatalf("the detachment did not sync: %v", seen)
	}
	if _, err := a.bucket.Store.Stat(context.Background(), store.MediaKey(uuid.MustParse(u.id), uuid.MustParse(id))); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("object after delete: %v", err)
	}
}

func TestMediaUploadThatDoesNotMatch(t *testing.T) {
	a := newAPI(t, withMedia())
	u := a.register("media-mismatch@example.test")
	id := newID()
	up := reserve(a, u, id, 100)["upload"].(map[string]any)
	// A stand-in for the bucket may not enforce the signed size; the API checks
	// what arrived either way.
	if code := upload(t, up, []byte("short")); code >= 300 && code != http.StatusForbidden && code != http.StatusBadRequest {
		t.Fatalf("upload: %d", code)
	}
	a.call("POST", "/v1/media/"+id+"/complete", u.access, nil).problem(422, "validation")
	if m := a.call("GET", "/v1/media/"+id, u.access, nil).ok(200, "Media"); m["status"] == "ready" {
		t.Fatalf("a mismatched upload became ready: %v", m)
	}
}

func TestMediaWithoutStorage(t *testing.T) {
	a := newAPI(t)
	u := a.register("no-media@example.test")
	a.call("POST", "/v1/media/uploads", u.access, map[string]any{
		"id": newID(), "kind": "image", "mime": "image/jpeg", "bytes": 10,
	}).problem(503, "not-ready")
}

func TestReaperRemovesAnAccountsObjects(t *testing.T) {
	a := newAPI(t, withMedia())
	u := a.register("media-reaped@example.test")
	id := newID()
	photo := []byte("bytes")
	if code := upload(t, reserve(a, u, id, len(photo))["upload"].(map[string]any), photo); code != http.StatusOK {
		t.Fatalf("upload: %d", code)
	}
	a.call("POST", "/v1/media/"+id+"/complete", u.access, nil).ok(200, "Media")
	a.call("DELETE", "/v1/me", u.access, nil).ok(202, "DeletionScheduled")
	a.exec("UPDATE users SET deletion_requested_at = now() - interval '31 days' WHERE id = $1", u.id)

	ctx := context.Background()
	n, err := a.store.ReapDeletedUsers(ctx, 30*24*time.Hour, func(ctx context.Context, userID uuid.UUID) error {
		return a.bucket.Store.RemovePrefix(ctx, store.MediaPrefix(userID))
	})
	if err != nil || n != 1 {
		t.Fatalf("reaped %d: %v", n, err)
	}
	if _, err := a.bucket.Store.Stat(ctx, store.MediaKey(uuid.MustParse(u.id), uuid.MustParse(id))); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("object after reaping: %v", err)
	}
}
