package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidIdentityToken covers every way an Apple identity token can fail.
var ErrInvalidIdentityToken = errors.New("invalid apple identity token")

// AppleIssuer is the `iss` of every Apple identity token.
const AppleIssuer = "https://appleid.apple.com"

// AppleKeysURL is where Apple publishes its signing keys.
const AppleKeysURL = "https://appleid.apple.com/auth/keys"

// AppleIdentity is what a verified identity token tells us.
type AppleIdentity struct {
	Subject        string
	Email          *string
	EmailVerified  bool
	IsPrivateEmail bool
}

// KeySource resolves a key id to Apple's RSA public key.
type KeySource interface {
	Key(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// AppleVerifier checks identity tokens: RS256 signature by a current Apple
// key, issuer, audience (our bundle / service ids), expiry, and the nonce.
type AppleVerifier struct {
	audiences []string
	keys      KeySource
	now       func() time.Time
}

// NewAppleVerifier builds a verifier for the given client ids.
func NewAppleVerifier(audiences []string, keys KeySource) *AppleVerifier {
	return &AppleVerifier{audiences: audiences, keys: keys, now: time.Now}
}

type appleClaims struct {
	jwt.RegisteredClaims
	Email          string   `json:"email"`
	EmailVerified  flexBool `json:"email_verified"`
	IsPrivateEmail flexBool `json:"is_private_email"`
	Nonce          string   `json:"nonce"`
}

// Verify checks token. rawNonce is the nonce the app generated; Apple holds
// its SHA-256 hex. An empty rawNonce skips the check only if the token
// carries no nonce either.
func (v *AppleVerifier) Verify(ctx context.Context, token, rawNonce string) (AppleIdentity, error) {
	if len(v.audiences) == 0 {
		return AppleIdentity{}, fmt.Errorf("%w: sign in with apple is not configured", ErrInvalidIdentityToken)
	}
	var c appleClaims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("no kid")
		}
		return v.keys.Key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(AppleIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(v.now),
	)
	if err != nil {
		return AppleIdentity{}, fmt.Errorf("%w: %w", ErrInvalidIdentityToken, err)
	}
	if !audienceOK(c.Audience, v.audiences) {
		return AppleIdentity{}, fmt.Errorf("%w: audience %v", ErrInvalidIdentityToken, c.Audience)
	}
	if c.Subject == "" {
		return AppleIdentity{}, fmt.Errorf("%w: no subject", ErrInvalidIdentityToken)
	}
	if c.Nonce != "" || rawNonce != "" {
		sum := sha256.Sum256([]byte(rawNonce))
		if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(c.Nonce)) != 1 {
			return AppleIdentity{}, fmt.Errorf("%w: nonce mismatch", ErrInvalidIdentityToken)
		}
	}
	id := AppleIdentity{Subject: c.Subject, EmailVerified: bool(c.EmailVerified), IsPrivateEmail: bool(c.IsPrivateEmail)}
	if c.Email != "" {
		id.Email = &c.Email
	}
	return id, nil
}

func audienceOK(got jwt.ClaimStrings, want []string) bool {
	for _, g := range got {
		for _, w := range want {
			if g == w {
				return true
			}
		}
	}
	return false
}

// flexBool accepts Apple's booleans, which arrive as either true or "true".
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case `true`, `"true"`:
		*b = true
	case `false`, `"false"`, `null`:
		*b = false
	default:
		return fmt.Errorf("not a boolean: %s", data)
	}
	return nil
}

// JWKS fetches and caches Apple's signing keys. Keys rotate rarely; an
// unknown kid triggers a refetch, at most once a minute.
type JWKS struct {
	url    string
	client *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// NewJWKS builds a key source for url.
func NewJWKS(url string, client *http.Client) *JWKS {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &JWKS{url: url, client: client}
}

// Key returns the key with id kid.
func (j *JWKS) Key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if k, ok := j.keys[kid]; ok && time.Since(j.fetchedAt) < 24*time.Hour {
		return k, nil
	}
	if time.Since(j.fetchedAt) < time.Minute && j.keys != nil {
		return nil, fmt.Errorf("unknown key id %q", kid)
	}
	keys, err := j.fetch(ctx)
	if err != nil {
		return nil, err
	}
	j.keys, j.fetchedAt = keys, time.Now()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown key id %q", kid)
}

func (j *JWKS) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.url, nil)
	if err != nil {
		return nil, fmt.Errorf("building jwks request: %w", err)
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching apple keys: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching apple keys: status %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decoding apple keys: %w", err)
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("key %s modulus: %w", k.Kid, err)
		}
		e, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("key %s exponent: %w", k.Kid, err)
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	return out, nil
}
