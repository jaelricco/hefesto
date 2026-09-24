//go:build integration

package http_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/content"
	lhttp "github.com/jaelricco/hefesto/internal/http"
	"github.com/jaelricco/hefesto/internal/store"
	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
	"github.com/jaelricco/hefesto/internal/testutil/s3test"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Run(m)) }

const appleAudience = "fit.hefesto.ios"

// api is a running server on a fresh database seeded with the repository's
// content, plus what a test needs to poke at it.
type api struct {
	t        *testing.T
	srv      *httptest.Server
	pool     *pgxpool.Pool
	store    *store.Store
	schemas  *lhttp.Schemas
	appleKey *rsa.PrivateKey
	bucket   *s3test.Bucket // set by withMedia
}

// option adjusts a test server before it starts.
type option func(t *testing.T, a *api, deps *lhttp.RouterDeps)

// withMedia gives the server object storage: a fresh bucket of its own.
func withMedia() option {
	return func(t *testing.T, a *api, deps *lhttp.RouterDeps) {
		b := s3test.New(t)
		a.bucket = &b
		deps.Media, deps.PresignTTL = b.Store, 5*time.Minute
	}
}

type staticKeys map[string]*rsa.PublicKey

func (s staticKeys) Key(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := s[kid]; ok {
		return k, nil
	}
	return nil, errors.New("unknown kid")
}

func newAPI(t *testing.T, opts ...option) *api {
	t.Helper()
	pool := pgtest.New(t)
	st := store.New(pool)

	tree, issues, err := content.Load(filepath.Join(pgtest.RepoRoot(), "content"))
	if err != nil || content.HasErrors(issues, false) {
		t.Fatalf("loading content: %v %v", err, issues)
	}
	if _, err := store.SeedContent(context.Background(), pool, tree, store.SeedOptions{AppliedBy: "test"}); err != nil {
		t.Fatal(err)
	}

	issuer, err := auth.NewIssuer([]byte(strings.Repeat("s", 32)), "hefesto", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	appleKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	apple := auth.NewAppleVerifier([]string{appleAudience}, staticKeys{"test": &appleKey.PublicKey})
	svc, err := auth.NewService(st, issuer, apple, auth.Config{
		RefreshTTL: 24 * time.Hour,
		Argon2:     auth.Argon2Params{MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := lhttp.LoadSchemas()
	if err != nil {
		t.Fatal(err)
	}
	a := &api{t: t, pool: pool, store: st, schemas: schemas, appleKey: appleKey}
	deps := lhttp.RouterDeps{
		DB: pool, Auth: svc, Store: st, Schemas: schemas,
		DeletionGrace: 30 * 24 * time.Hour, AuthPerMinute: 10000,
	}
	for _, o := range opts {
		o(t, a, &deps)
	}
	a.srv = httptest.NewServer(lhttp.NewRouter(deps))
	t.Cleanup(a.srv.Close)
	return a
}

// res is a response, already read.
type res struct {
	t       *testing.T
	status  int
	header  http.Header
	body    []byte
	schemas *lhttp.Schemas
	desc    string
}

// call sends a request. body may be nil, a []byte, or anything JSON-encodable.
func (a *api) call(method, path, token string, body any) res {
	a.t.Helper()
	return a.do(method, path, token, body, nil)
}

// callWith sends a request with extra headers.
func (a *api) callWith(method, path, token string, headers map[string]string, body ...any) res {
	a.t.Helper()
	var b any
	if len(body) > 0 {
		b = body[0]
	}
	return a.do(method, path, token, b, headers)
}

func (a *api) do(method, path, token string, body any, headers map[string]string) res {
	a.t.Helper()
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		rd = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			a.t.Fatal(err)
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, a.srv.URL+path, rd)
	if err != nil {
		a.t.Fatal(err)
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.srv.Client().Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		a.t.Fatal(err)
	}
	return res{t: a.t, status: resp.StatusCode, header: resp.Header, body: raw, schemas: a.schemas, desc: method + " " + path}
}

// ok asserts the status and that the body matches the named schema of the
// OpenAPI document ("" for no body). It returns the body decoded as a map.
func (r res) ok(status int, schema string) map[string]any {
	r.t.Helper()
	if r.status != status {
		r.t.Fatalf("%s: status %d, want %d; body %s", r.desc, r.status, status, r.body)
	}
	if schema == "" {
		if len(bytes.TrimSpace(r.body)) != 0 {
			r.t.Fatalf("%s: expected no body, got %s", r.desc, r.body)
		}
		return nil
	}
	sch, err := r.schemas.Schema(schema)
	if err != nil {
		r.t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(r.body, &v); err != nil {
		r.t.Fatalf("%s: body is not JSON: %s", r.desc, r.body)
	}
	if err := sch.Validate(v); err != nil {
		r.t.Fatalf("%s: response does not match %s:\n%v\nbody: %s", r.desc, schema, err, r.body)
	}
	m, _ := v.(map[string]any)
	return m
}

// problem asserts a problem response of the given type slug and returns its
// field errors.
func (r res) problem(status int, slug string) map[string]any {
	r.t.Helper()
	if ct := r.header.Get("Content-Type"); ct != "application/problem+json" {
		r.t.Fatalf("%s: content type %q; body %s", r.desc, ct, r.body)
	}
	p := r.ok(status, "Problem")
	if got := p["type"]; got != "https://hefesto.fit/problems/"+slug {
		r.t.Fatalf("%s: problem type %v, want %s; body %s", r.desc, got, slug, r.body)
	}
	errs, _ := p["errors"].(map[string]any)
	return errs
}

func hasField(t *testing.T, errs map[string]any, field string) {
	t.Helper()
	if _, ok := errs[field]; !ok {
		t.Fatalf("expected a field error at %q, got %v", field, errs)
	}
}

// user is a registered test user.
type user struct {
	id      string
	access  string
	refresh string
	device  string
}

func (a *api) register(email string) user {
	a.t.Helper()
	device := uuid.Must(uuid.NewV7()).String()
	m := a.call("POST", "/v1/auth/register", "", map[string]any{
		"email": email, "password": "correct horse battery", "timezone": "Europe/Zurich",
		"device": map[string]any{"id": device, "platform": "ios"},
	}).ok(201, "AuthResponse")
	return user{
		id: m["user"].(map[string]any)["id"].(string), access: m["access_token"].(string),
		refresh: m["refresh_token"].(string), device: device,
	}
}

func (a *api) appleToken(claims jwt.MapClaims) string {
	a.t.Helper()
	base := jwt.MapClaims{
		"iss": auth.AppleIssuer, "aud": appleAudience, "exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range claims {
		base[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, base)
	tok.Header["kid"] = "test"
	s, err := tok.SignedString(a.appleKey)
	if err != nil {
		a.t.Fatal(err)
	}
	return s
}

func nonceHash(raw string) string {
	s := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(s[:])
}

func (a *api) exerciseID(slug string) string {
	a.t.Helper()
	var id uuid.UUID
	if err := a.pool.QueryRow(context.Background(), "SELECT id FROM exercises WHERE slug = $1", slug).Scan(&id); err != nil {
		a.t.Fatalf("exercise %s: %v", slug, err)
	}
	return id.String()
}

func newID() string { return uuid.Must(uuid.NewV7()).String() }

func (a *api) exec(sql string, args ...any) {
	a.t.Helper()
	if _, err := a.pool.Exec(context.Background(), sql, args...); err != nil {
		a.t.Fatalf("%s: %v", sql, err)
	}
}
