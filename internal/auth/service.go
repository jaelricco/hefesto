package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

// Errors the service returns for the HTTP layer to map.
var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrAccountSuspended    = errors.New("account suspended")
)

// Password bounds. The upper bound stops a multi-megabyte "password" from
// becoming a denial of service through argon2.
const (
	MinPasswordLen = 10
	MaxPasswordLen = 256
)

// Config tunes the service.
type Config struct {
	RefreshTTL time.Duration
	Argon2     Argon2Params
}

// Service runs the account and token flows.
type Service struct {
	store  *store.Store
	tokens *Issuer
	apple  *AppleVerifier
	cfg    Config
	now    func() time.Time
	// dummyHash is verified against when an email is unknown, so a login
	// attempt costs the same whether or not the account exists.
	dummyHash string
}

// NewService builds the service. apple may be nil when Sign in with Apple is
// not configured.
func NewService(st *store.Store, tokens *Issuer, apple *AppleVerifier, cfg Config) (*Service, error) {
	dummy, err := HashPassword("not a real password", cfg.Argon2)
	if err != nil {
		return nil, err
	}
	return &Service{store: st, tokens: tokens, apple: apple, cfg: cfg, now: time.Now, dummyHash: dummy}, nil
}

// Client describes where a sign-in comes from.
type Client struct {
	Device    *store.Device
	UserAgent string
	IP        *netip.Addr
}

// Session is what a successful sign-in or refresh returns.
type Session struct {
	AccessToken       string
	ExpiresIn         time.Duration
	RefreshToken      string
	RefreshExpiresAt  time.Time
	User              store.User
	DeletionCancelled bool
}

// Registration is a new email/password account.
type Registration struct {
	Email, Password, DisplayName, Locale, Timezone string
}

// Register creates an account and signs it in.
func (s *Service) Register(ctx context.Context, r Registration, c Client) (Session, error) {
	errs := training.FieldErrors{}
	email, err := normaliseEmail(r.Email)
	if err != nil {
		errs["/email"] = "not a valid email address"
	}
	if n := utf8.RuneCountInString(r.Password); n < MinPasswordLen || n > MaxPasswordLen {
		errs["/password"] = fmt.Sprintf("between %d and %d characters", MinPasswordLen, MaxPasswordLen)
	}
	if err := errs.Err(); err != nil {
		return Session{}, err
	}
	hash, err := HashPassword(r.Password, s.cfg.Argon2)
	if err != nil {
		return Session{}, err
	}
	u, err := s.store.CreateUser(ctx, store.NewUser{
		ID: uuid.Must(uuid.NewV7()), Email: &email, PasswordHash: &hash,
		DisplayName: r.DisplayName, Locale: orDefault(r.Locale, "de-CH"), Timezone: orDefault(r.Timezone, "Europe/Zurich"),
	})
	if err != nil {
		return Session{}, fmt.Errorf("creating account: %w", err)
	}
	return s.startSession(ctx, u, c)
}

// Login checks an email and password and signs the account in.
func (s *Service) Login(ctx context.Context, email, password string, c Client) (Session, error) {
	u, err := s.store.UserByEmail(ctx, strings.TrimSpace(email))
	switch {
	case errors.Is(err, store.ErrNotFound):
		_, _ = VerifyPassword(password, s.dummyHash)
		return Session{}, ErrInvalidCredentials
	case err != nil:
		return Session{}, fmt.Errorf("reading account: %w", err)
	case u.PasswordHash == nil:
		_, _ = VerifyPassword(password, s.dummyHash)
		return Session{}, ErrInvalidCredentials
	}
	ok, err := VerifyPassword(password, *u.PasswordHash)
	if err != nil {
		return Session{}, fmt.Errorf("verifying password: %w", err)
	}
	if !ok {
		return Session{}, ErrInvalidCredentials
	}
	return s.startSession(ctx, u, c)
}

// AppleSignIn is a Sign in with Apple request.
type AppleSignIn struct {
	IdentityToken, Nonce, DisplayName, Locale, Timezone string
}

// SignInWithApple verifies an Apple identity token and signs its account in,
// creating or linking it on first use. It reports whether it created one.
func (s *Service) SignInWithApple(ctx context.Context, a AppleSignIn, c Client) (Session, bool, error) {
	if s.apple == nil {
		return Session{}, false, fmt.Errorf("%w: sign in with apple is not configured", ErrInvalidIdentityToken)
	}
	id, err := s.apple.Verify(ctx, a.IdentityToken, a.Nonce)
	if err != nil {
		return Session{}, false, err
	}
	u, created, err := s.store.UpsertAppleUser(ctx, store.AppleSignIn{
		Subject: id.Subject, Email: id.Email, EmailVerified: id.EmailVerified, IsPrivateEmail: id.IsPrivateEmail,
		DisplayName: a.DisplayName, Locale: orDefault(a.Locale, "de-CH"), Timezone: orDefault(a.Timezone, "Europe/Zurich"),
	})
	if err != nil {
		return Session{}, false, fmt.Errorf("resolving apple account: %w", err)
	}
	sess, err := s.startSession(ctx, u, c)
	return sess, created, err
}

// Refresh spends a refresh token and returns a new session.
func (s *Service) Refresh(ctx context.Context, token string, c Client) (Session, error) {
	plain, hash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	next := store.RefreshToken{
		ID: uuid.Must(uuid.NewV7()), Hash: hash, ExpiresAt: s.now().Add(s.cfg.RefreshTTL),
		UserAgent: nonEmpty(c.UserAgent), IP: c.IP,
	}
	next, err = s.store.RotateRefreshToken(ctx, HashRefreshToken(token), next, s.now())
	switch {
	case errors.Is(err, store.ErrRefreshUnknown), errors.Is(err, store.ErrRefreshExpired),
		errors.Is(err, store.ErrRefreshRevoked), errors.Is(err, store.ErrRefreshReused):
		return Session{}, fmt.Errorf("%w: %w", ErrInvalidRefreshToken, err)
	case err != nil:
		return Session{}, fmt.Errorf("rotating refresh token: %w", err)
	}

	u, err := s.store.UserByID(ctx, next.UserID)
	if err != nil {
		return Session{}, fmt.Errorf("reading account: %w", err)
	}
	if u.Status == "suspended" {
		return Session{}, ErrAccountSuspended
	}
	access, err := s.tokens.Issue(Principal{UserID: u.ID, DeviceID: next.DeviceID})
	if err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken: access, ExpiresIn: s.tokens.TTL(), RefreshToken: plain,
		RefreshExpiresAt: next.ExpiresAt, User: u,
	}, nil
}

// Logout revokes the presented refresh token's family.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.store.RevokeRefreshToken(ctx, HashRefreshToken(token)); err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}
	return nil
}

// Authenticate turns an access token into a principal.
func (s *Service) Authenticate(token string) (Principal, error) {
	return s.tokens.Verify(token)
}

// startSession finishes every sign-in: refuses suspended accounts, cancels a
// pending deletion, binds the device, and issues a new token family.
func (s *Service) startSession(ctx context.Context, u store.User, c Client) (Session, error) {
	if u.Status == "suspended" {
		return Session{}, ErrAccountSuspended
	}
	cancelled := false
	if u.Status == "deletion_pending" {
		var err error
		if cancelled, err = s.store.CancelDeletion(ctx, u.ID); err != nil {
			return Session{}, err
		}
		if u, err = s.store.UserByID(ctx, u.ID); err != nil {
			return Session{}, fmt.Errorf("reading account: %w", err)
		}
	}

	var deviceID *uuid.UUID
	if c.Device != nil {
		if err := s.store.RegisterDevice(ctx, u.ID, *c.Device); err != nil {
			if errors.Is(err, store.ErrDeviceTaken) {
				return Session{}, training.FieldErrors{
					"/device/id": "this device id belongs to another account; generate one per account",
				}.Err()
			}
			return Session{}, err
		}
		deviceID = &c.Device.ID
	}

	plain, hash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	rt := store.RefreshToken{
		ID: uuid.Must(uuid.NewV7()), UserID: u.ID, FamilyID: uuid.Must(uuid.NewV7()), Hash: hash,
		DeviceID: deviceID, ExpiresAt: s.now().Add(s.cfg.RefreshTTL), UserAgent: nonEmpty(c.UserAgent), IP: c.IP,
	}
	if err := s.store.InsertRefreshToken(ctx, rt); err != nil {
		return Session{}, err
	}
	access, err := s.tokens.Issue(Principal{UserID: u.ID, DeviceID: deviceID})
	if err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken: access, ExpiresIn: s.tokens.TTL(), RefreshToken: plain,
		RefreshExpiresAt: rt.ExpiresAt, User: u, DeletionCancelled: cancelled,
	}, nil
}

func normaliseEmail(raw string) (string, error) {
	e := strings.TrimSpace(raw)
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || a.Name != "" {
		return "", errors.New("invalid email")
	}
	return e, nil
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	if len(s) > 512 {
		s = s[:512]
	}
	return &s
}
