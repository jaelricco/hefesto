package store

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// Refresh-token failures. Each one means "sign in again"; they are distinct
// so the reason can be logged.
var (
	ErrRefreshUnknown = errors.New("refresh token unknown")
	ErrRefreshExpired = errors.New("refresh token expired")
	ErrRefreshRevoked = errors.New("refresh token revoked")
	// ErrRefreshReused means a spent token was presented again: its whole
	// family has now been revoked.
	ErrRefreshReused = errors.New("refresh token reused; family revoked")
)

// User is an account.
type User struct {
	ID                  uuid.UUID
	Email               *string
	EmailVerified       bool
	PasswordHash        *string
	DisplayName         string
	Locale              string
	UnitSystem          string
	WeekStart           int
	Timezone            string
	Status              string
	DeletionRequestedAt *time.Time
	CreatedAt           time.Time
}

func userFromRow(r dbgen.User) User {
	return User{
		ID: r.ID, Email: r.Email, EmailVerified: r.EmailVerifiedAt != nil, PasswordHash: r.PasswordHash,
		DisplayName: r.DisplayName, Locale: r.Locale, UnitSystem: r.UnitSystem, WeekStart: int(r.WeekStart),
		Timezone: r.Timezone, Status: r.Status, DeletionRequestedAt: r.DeletionRequestedAt, CreatedAt: r.CreatedAt,
	}
}

// NewUser is what creating an account needs.
type NewUser struct {
	ID              uuid.UUID
	Email           *string
	PasswordHash    *string
	DisplayName     string
	Locale          string
	Timezone        string
	EmailVerifiedAt *time.Time
}

// CreateUser inserts an account.
func (s *Store) CreateUser(ctx context.Context, u NewUser) (User, error) {
	return createUser(ctx, s.q, u)
}

func createUser(ctx context.Context, q *dbgen.Queries, u NewUser) (User, error) {
	r, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		ID: u.ID, Email: u.Email, PasswordHash: u.PasswordHash, DisplayName: u.DisplayName,
		Locale: u.Locale, Timezone: u.Timezone, EmailVerifiedAt: u.EmailVerifiedAt,
	})
	if err != nil {
		return User{}, translate(err)
	}
	return userFromRow(r), nil
}

// UserByID returns an account.
func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	r, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return User{}, translate(err)
	}
	return userFromRow(r), nil
}

// UserByEmail returns an account by email, case-insensitively.
func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	r, err := s.q.GetUserByEmail(ctx, &email)
	if err != nil {
		return User{}, translate(err)
	}
	return userFromRow(r), nil
}

// ProfileUpdate changes the fields that are non-nil.
type ProfileUpdate struct {
	DisplayName, Locale, UnitSystem, Timezone *string
	WeekStart                                 *int
}

// UpdateProfile applies a profile update.
func (s *Store) UpdateProfile(ctx context.Context, id uuid.UUID, p ProfileUpdate) (User, error) {
	r, err := s.q.UpdateUserProfile(ctx, dbgen.UpdateUserProfileParams{
		ID: id, DisplayName: p.DisplayName, Locale: p.Locale, UnitSystem: p.UnitSystem,
		Timezone: p.Timezone, WeekStart: int16Ptr(p.WeekStart),
	})
	if err != nil {
		return User{}, translate(err)
	}
	return userFromRow(r), nil
}

// RequestDeletion starts the deletion grace period and revokes every refresh
// token, signing the account out everywhere.
func (s *Store) RequestDeletion(ctx context.Context, id uuid.UUID) (User, error) {
	var out User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		r, err := q.RequestUserDeletion(ctx, id)
		if err != nil {
			return err //nolint:wrapcheck // translated below
		}
		if _, err := q.RevokeUserRefreshTokens(ctx, dbgen.RevokeUserRefreshTokensParams{
			UserID: id, Reason: ptr("logout"),
		}); err != nil {
			return fmt.Errorf("revoking tokens: %w", err)
		}
		out = userFromRow(r)
		return nil
	})
	return out, translate(err)
}

// CancelDeletion returns a pending account to active, reporting whether it
// was pending.
func (s *Store) CancelDeletion(ctx context.Context, id uuid.UUID) (bool, error) {
	n, err := s.q.CancelUserDeletion(ctx, id)
	if err != nil {
		return false, fmt.Errorf("cancelling deletion: %w", err)
	}
	return n > 0, nil
}

// Device is a client installation.
type Device struct {
	ID                           uuid.UUID
	Platform                     string
	Model, OSVersion, AppVersion *string
}

// RegisterDevice records a device for a user. A device id already claimed by
// another account is refused with ErrDeviceTaken.
func (s *Store) RegisterDevice(ctx context.Context, userID uuid.UUID, d Device) error {
	_, err := s.q.UpsertDevice(ctx, dbgen.UpsertDeviceParams{
		ID: d.ID, UserID: userID, Platform: d.Platform, Model: d.Model, OsVersion: d.OSVersion, AppVersion: d.AppVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrDeviceTaken
	}
	if err != nil {
		return fmt.Errorf("registering device: %w", err)
	}
	return nil
}

// RefreshToken is a stored refresh token. Only the hash is ever stored.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	Hash      []byte
	DeviceID  *uuid.UUID
	ExpiresAt time.Time
	UserAgent *string
	IP        *netip.Addr
}

// InsertRefreshToken stores the first token of a new family.
func (s *Store) InsertRefreshToken(ctx context.Context, t RefreshToken) error {
	return insertRefreshToken(ctx, s.q, t)
}

func insertRefreshToken(ctx context.Context, q *dbgen.Queries, t RefreshToken) error {
	if err := q.InsertRefreshToken(ctx, dbgen.InsertRefreshTokenParams{
		ID: t.ID, UserID: t.UserID, FamilyID: t.FamilyID, TokenHash: t.Hash, DeviceID: t.DeviceID,
		ExpiresAt: t.ExpiresAt, UserAgent: t.UserAgent, Ip: t.IP,
	}); err != nil {
		return fmt.Errorf("storing refresh token: %w", err)
	}
	return nil
}

// RotateRefreshToken spends the token with the given hash and stores next in
// its place, in the same family. next's UserID, FamilyID and DeviceID are
// filled in from the spent token and returned.
//
// Presenting a token that was already rotated revokes the whole family and
// returns ErrRefreshReused: the only way that happens legitimately is never,
// so it is treated as a stolen token.
func (s *Store) RotateRefreshToken(ctx context.Context, hash []byte, next RefreshToken, now time.Time) (RefreshToken, error) {
	var reused bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		cur, err := q.GetRefreshTokenForUpdate(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRefreshUnknown
		}
		if err != nil {
			return fmt.Errorf("reading refresh token: %w", err)
		}
		switch {
		case cur.RevokedAt != nil && cur.RevokedReason != nil && *cur.RevokedReason == "rotated":
			if _, err := q.RevokeRefreshFamily(ctx, dbgen.RevokeRefreshFamilyParams{
				FamilyID: cur.FamilyID, Reason: ptr("reuse_detected"),
			}); err != nil {
				return fmt.Errorf("revoking family: %w", err)
			}
			reused = true
			return nil // commit the revocation
		case cur.RevokedAt != nil:
			return ErrRefreshRevoked
		case !now.Before(cur.ExpiresAt):
			return ErrRefreshExpired
		}

		next.UserID, next.FamilyID, next.DeviceID = cur.UserID, cur.FamilyID, cur.DeviceID
		if err := insertRefreshToken(ctx, q, next); err != nil {
			return err
		}
		if err := q.MarkRefreshTokenRotated(ctx, dbgen.MarkRefreshTokenRotatedParams{ID: cur.ID, ReplacedBy: &next.ID}); err != nil {
			return fmt.Errorf("spending refresh token: %w", err)
		}
		return nil
	})
	if err != nil {
		return RefreshToken{}, fmt.Errorf("rotating refresh token: %w", err)
	}
	if reused {
		return RefreshToken{}, ErrRefreshReused
	}
	return next, nil
}

// RevokeRefreshToken signs one device out: the token's whole family is
// revoked. Unknown tokens are ignored.
func (s *Store) RevokeRefreshToken(ctx context.Context, hash []byte) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		cur, err := q.GetRefreshTokenForUpdate(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading refresh token: %w", err)
		}
		if _, err := q.RevokeRefreshFamily(ctx, dbgen.RevokeRefreshFamilyParams{FamilyID: cur.FamilyID, Reason: ptr("logout")}); err != nil {
			return fmt.Errorf("revoking family: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}
	return nil
}

// AppleSignIn is a verified Apple identity.
type AppleSignIn struct {
	Subject        string
	Email          *string
	EmailVerified  bool
	IsPrivateEmail bool
	DisplayName    string
	Locale         string
	Timezone       string
}

// UpsertAppleUser resolves an Apple identity to an account, in one
// transaction:
//
//   - a known subject signs in to its account;
//   - otherwise a verified email that matches an account links to it;
//   - otherwise a new account is created.
//
// It reports whether an account was created.
func (s *Store) UpsertAppleUser(ctx context.Context, a AppleSignIn) (User, bool, error) {
	var (
		out     User
		created bool
	)
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		ident, err := q.GetAppleIdentity(ctx, a.Subject)
		switch {
		case err == nil:
			r, err := q.GetUserByID(ctx, ident.UserID)
			if err != nil {
				return fmt.Errorf("reading user: %w", err)
			}
			out = userFromRow(r)
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("reading apple identity: %w", err)
		}

		if a.Email != nil && a.EmailVerified {
			r, err := q.GetUserByEmail(ctx, a.Email)
			switch {
			case err == nil:
				out = userFromRow(r)
			case !errors.Is(err, pgx.ErrNoRows):
				return fmt.Errorf("reading user by email: %w", err)
			}
		}
		if out.ID == uuid.Nil {
			// An unverified email is not recorded: it could claim someone
			// else's address.
			var email *string
			var verifiedAt *time.Time
			if a.Email != nil && a.EmailVerified {
				now := time.Now()
				email, verifiedAt = a.Email, &now
			}
			out, err = createUser(ctx, q, NewUser{
				ID: uuid.Must(uuid.NewV7()), Email: email, DisplayName: a.DisplayName,
				Locale: a.Locale, Timezone: a.Timezone, EmailVerifiedAt: verifiedAt,
			})
			if err != nil {
				return err
			}
			created = true
		}
		if err := q.InsertAppleIdentity(ctx, dbgen.InsertAppleIdentityParams{
			UserID: out.ID, AppleSub: a.Subject, IsPrivateEmail: a.IsPrivateEmail,
		}); err != nil {
			if isUnique(err, "apple_identities_user_id_key") {
				// The account with this email is already linked to a
				// different Apple ID.
				return ErrAlreadyExists
			}
			return fmt.Errorf("linking apple identity: %w", err)
		}
		return nil
	})
	if err != nil {
		return User{}, false, translate(err)
	}
	return out, created, nil
}

func ptr[T any](v T) *T { return &v }
