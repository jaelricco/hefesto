package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken covers every way an access token can be unusable.
var ErrInvalidToken = errors.New("invalid access token")

// Principal is who a request acts as.
type Principal struct {
	UserID   uuid.UUID
	DeviceID *uuid.UUID
}

// Issuer mints and checks access tokens: short-lived HS256 JWTs. They are
// verified by this service alone, so a shared secret is enough.
type Issuer struct {
	key    []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

// NewIssuer builds an issuer. The key must be at least 32 bytes.
func NewIssuer(key []byte, issuer string, ttl time.Duration) (*Issuer, error) {
	if len(key) < 32 {
		return nil, errors.New("access-token signing key must be at least 32 bytes")
	}
	return &Issuer{key: key, issuer: issuer, ttl: ttl, now: time.Now}, nil
}

// TTL is the access-token lifetime.
func (i *Issuer) TTL() time.Duration { return i.ttl }

type claims struct {
	jwt.RegisteredClaims
	DeviceID string `json:"did,omitempty"`
}

// Issue signs an access token for p.
func (i *Issuer) Issue(p Principal) (string, error) {
	now := i.now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   p.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
			ID:        uuid.NewString(),
		},
	}
	if p.DeviceID != nil {
		c.DeviceID = p.DeviceID.String()
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(i.key)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return s, nil
}

// Verify checks signature, algorithm, issuer and lifetime.
func (i *Issuer) Verify(token string) (Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return i.key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	uid, err := uuid.Parse(c.Subject)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: subject: %w", ErrInvalidToken, err)
	}
	p := Principal{UserID: uid}
	if c.DeviceID != "" {
		did, err := uuid.Parse(c.DeviceID)
		if err != nil {
			return Principal{}, fmt.Errorf("%w: device: %w", ErrInvalidToken, err)
		}
		p.DeviceID = &did
	}
	return p, nil
}

// NewRefreshToken returns an opaque token for the client and the hash to
// store. The token itself is never stored.
func NewRefreshToken() (token string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("reading random bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashRefreshToken(token), nil
}

// HashRefreshToken is how a presented refresh token is looked up.
func HashRefreshToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// ErrForbidden means the actor may not act on the subject.
var ErrForbidden = errors.New("forbidden")

// Authorize is the single authorisation decision (ADR 0002): may actor act
// on subject's data? In v1 only on their own.
func Authorize(actor Principal, subject uuid.UUID) error {
	if actor.UserID != subject {
		return ErrForbidden
	}
	return nil
}
