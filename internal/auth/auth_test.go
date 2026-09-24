package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var fastArgon = Argon2Params{MemoryKiB: 1024, Iterations: 1, Parallelism: 1}

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery", fastArgon)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=1024,t=1,p=1$") {
		t.Fatalf("unexpected encoding %q", h)
	}
	if ok, err := VerifyPassword("correct horse battery", h); err != nil || !ok {
		t.Fatalf("right password rejected: %v %v", ok, err)
	}
	if ok, _ := VerifyPassword("wrong horse battery", h); ok {
		t.Fatal("wrong password accepted")
	}
	h2, _ := HashPassword("correct horse battery", fastArgon)
	if h == h2 {
		t.Fatal("two hashes of one password share a salt")
	}
	for _, bad := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$a$b", "$argon2id$v=19$m=x$a$b"} {
		if _, err := VerifyPassword("x", bad); err == nil {
			t.Errorf("malformed hash %q accepted", bad)
		}
	}
}

func newIssuer(t *testing.T) *Issuer {
	t.Helper()
	i, err := NewIssuer([]byte(strings.Repeat("k", 32)), "hefesto", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestAccessTokens(t *testing.T) {
	if _, err := NewIssuer([]byte("short"), "hefesto", time.Minute); err == nil {
		t.Fatal("short key accepted")
	}
	iss := newIssuer(t)
	dev := uuid.New()
	p := Principal{UserID: uuid.New(), DeviceID: &dev}
	tok, err := iss.Issue(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := iss.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != p.UserID || got.DeviceID == nil || *got.DeviceID != dev {
		t.Fatalf("round trip lost data: %+v", got)
	}

	noDev, _ := iss.Issue(Principal{UserID: p.UserID})
	if got, err := iss.Verify(noDev); err != nil || got.DeviceID != nil {
		t.Fatalf("token without device: %+v %v", got, err)
	}

	// Expired.
	iss.now = func() time.Time { return time.Now().Add(-time.Hour) }
	old, _ := iss.Issue(p)
	iss.now = time.Now
	if _, err := iss.Verify(old); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token: %v", err)
	}

	// Signed with another key.
	other, _ := NewIssuer([]byte(strings.Repeat("x", 32)), "hefesto", time.Minute)
	forged, _ := other.Issue(p)
	if _, err := iss.Verify(forged); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("forged token: %v", err)
	}

	// alg=none must never pass.
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"iss": "hefesto", "sub": p.UserID.String(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := iss.Verify(none); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("alg=none token: %v", err)
	}

	// Wrong issuer.
	otherIss, _ := NewIssuer([]byte(strings.Repeat("k", 32)), "someone-else", time.Minute)
	wrong, _ := otherIss.Issue(p)
	if _, err := iss.Verify(wrong); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong issuer: %v", err)
	}
}

func TestRefreshTokens(t *testing.T) {
	a, ha, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := NewRefreshToken()
	if a == b {
		t.Fatal("two refresh tokens are equal")
	}
	if string(HashRefreshToken(a)) != string(ha) {
		t.Fatal("hash does not match the token")
	}
}

func TestAuthorize(t *testing.T) {
	me := uuid.New()
	if err := Authorize(Principal{UserID: me}, me); err != nil {
		t.Fatal(err)
	}
	if err := Authorize(Principal{UserID: me}, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatal("acting on another user's data was allowed")
	}
}

// ------------------------------------------------------------------ apple

type staticKeys map[string]*rsa.PublicKey

func (s staticKeys) Key(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := s[kid]; ok {
		return k, nil
	}
	return nil, errors.New("unknown kid")
}

func appleToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func nonceHash(raw string) string {
	s := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(s[:])
}

func TestAppleVerifier(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := NewAppleVerifier([]string{"fit.hefesto.ios"}, staticKeys{"k1": &key.PublicKey})

	base := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss": AppleIssuer, "aud": "fit.hefesto.ios", "sub": "001234.abcd",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
			"email": "a@privaterelay.appleid.com", "email_verified": "true", "is_private_email": true,
			"nonce": nonceHash("n0nce"),
		}
	}

	id, err := v.Verify(context.Background(), appleToken(t, key, "k1", base()), "n0nce")
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "001234.abcd" || id.Email == nil || !id.EmailVerified || !id.IsPrivateEmail {
		t.Fatalf("claims lost: %+v", id)
	}

	cases := map[string]struct {
		mutate func(jwt.MapClaims)
		key    *rsa.PrivateKey
		kid    string
		nonce  string
	}{
		"wrong audience": {mutate: func(c jwt.MapClaims) { c["aud"] = "com.someone.else" }},
		"wrong issuer":   {mutate: func(c jwt.MapClaims) { c["iss"] = "https://evil.example" }},
		"expired":        {mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }},
		"no subject":     {mutate: func(c jwt.MapClaims) { delete(c, "sub") }},
		"nonce mismatch": {nonce: "other"},
		"nonce missing":  {mutate: func(c jwt.MapClaims) { delete(c, "nonce") }},
		"other signer":   {key: other},
		"unknown key id": {kid: "k2"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := base()
			if tc.mutate != nil {
				tc.mutate(c)
			}
			k, kid, nonce := key, "k1", "n0nce"
			if tc.key != nil {
				k = tc.key
			}
			if tc.kid != "" {
				kid = tc.kid
			}
			if tc.nonce != "" {
				nonce = tc.nonce
			}
			if _, err := v.Verify(context.Background(), appleToken(t, k, kid, c), nonce); !errors.Is(err, ErrInvalidIdentityToken) {
				t.Fatalf("accepted or wrong error: %v", err)
			}
		})
	}

	// HS256 signed with the public key's bytes (algorithm confusion) must fail.
	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, base())
	hs.Header["kid"] = "k1"
	forged, _ := hs.SignedString([]byte("anything"))
	if _, err := v.Verify(context.Background(), forged, "n0nce"); err == nil {
		t.Fatal("HS256 token accepted")
	}

	unconfigured := NewAppleVerifier(nil, staticKeys{})
	if _, err := unconfigured.Verify(context.Background(), appleToken(t, key, "k1", base()), "n0nce"); !errors.Is(err, ErrInvalidIdentityToken) {
		t.Fatal("verifier without audiences accepted a token")
	}
}

func TestJWKSFetchesAndCaches(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer srv.Close()

	j := NewJWKS(srv.URL, srv.Client())
	got, err := j.Key(context.Background(), "k1")
	if err != nil {
		t.Fatal(err)
	}
	if got.N.Cmp(key.N) != 0 || got.E != key.E {
		t.Fatal("decoded key differs")
	}
	if _, err := j.Key(context.Background(), "k1"); err != nil || hits != 1 {
		t.Fatalf("cached key refetched: hits=%d err=%v", hits, err)
	}
	if _, err := j.Key(context.Background(), "nope"); err == nil || hits != 1 {
		t.Fatalf("unknown kid within a minute refetched: hits=%d err=%v", hits, err)
	}
}

func TestFlexBool(t *testing.T) {
	for in, want := range map[string]bool{`true`: true, `"true"`: true, `false`: false, `"false"`: false, `null`: false} {
		var b flexBool
		if err := json.Unmarshal([]byte(in), &b); err != nil || bool(b) != want {
			t.Errorf("%s: got %v %v", in, b, err)
		}
	}
	var b flexBool
	if err := json.Unmarshal([]byte(`"yes"`), &b); err == nil {
		t.Error(`"yes" accepted`)
	}
}

func TestNormaliseEmail(t *testing.T) {
	for in, ok := range map[string]bool{
		"a@example.test": true, "  a@example.test ": true, "A B <a@example.test>": false,
		"not-an-email": false, "": false,
	} {
		if _, err := normaliseEmail(in); (err == nil) != ok {
			t.Errorf("%q: err=%v", in, err)
		}
	}
}
