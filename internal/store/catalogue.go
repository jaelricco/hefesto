package store

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// Exercise is the API's view of a catalogue exercise.
type Exercise struct {
	ID              uuid.UUID
	Slug            string
	Name            string
	Aka             []string
	Family          string
	DefaultMeasure  string
	LoadSemantics   string
	IsBodyweight    bool
	Unilateral      bool
	TempoApplicable bool
	Equipment       []string
	Summary         string
	Cues            []string
	CommonFaults    []string
	Status          string
}

// ExerciseFilter narrows the catalogue; nil fields do not filter.
type ExerciseFilter struct {
	Q, Family, Equipment *string
}

// ContentVersion returns the checksum (hex) of the content the database
// currently holds, or "" before the first seed.
func (s *Store) ContentVersion(ctx context.Context) (string, error) {
	v, err := s.q.GetLatestContentVersion(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading content version: %w", err)
	}
	return hex.EncodeToString(v.Checksum), nil
}

// ListExercises searches the non-retired catalogue.
func (s *Store) ListExercises(ctx context.Context, f ExerciseFilter) ([]Exercise, error) {
	rows, err := s.q.ListExercises(ctx, dbgen.ListExercisesParams{Q: f.Q, Family: f.Family, Equipment: f.Equipment})
	if err != nil {
		return nil, fmt.Errorf("listing exercises: %w", err)
	}
	out := make([]Exercise, len(rows))
	for i, r := range rows {
		out[i] = Exercise{
			ID: r.ID, Slug: r.Slug, Name: r.Name, Aka: r.Aka, Family: r.FamilySlug,
			DefaultMeasure: r.DefaultMeasure, LoadSemantics: r.LoadSemantics, IsBodyweight: r.IsBodyweight,
			Unilateral: r.Unilateral, TempoApplicable: r.TempoApplicable, Equipment: r.Equipment,
			Summary: r.Summary, Cues: r.Cues, CommonFaults: r.CommonFaults, Status: r.Status,
		}
	}
	return out, nil
}

// GetExercise returns one non-retired exercise by slug.
func (s *Store) GetExercise(ctx context.Context, slug string) (Exercise, error) {
	r, err := s.q.GetExerciseBySlug(ctx, slug)
	if err != nil {
		return Exercise{}, translate(err)
	}
	return Exercise{
		ID: r.ID, Slug: r.Slug, Name: r.Name, Aka: r.Aka, Family: r.FamilySlug,
		DefaultMeasure: r.DefaultMeasure, LoadSemantics: r.LoadSemantics, IsBodyweight: r.IsBodyweight,
		Unilateral: r.Unilateral, TempoApplicable: r.TempoApplicable, Equipment: r.Equipment,
		Summary: r.Summary, Cues: r.Cues, CommonFaults: r.CommonFaults, Status: r.Status,
	}, nil
}

// Band is a resistance band from the catalogue or a user's collection.
type Band struct {
	ID              uuid.UUID
	Mine            bool
	Brand           string
	ColourLabel     string
	ResistanceMinKg float64
	ResistanceMaxKg float64
	LengthCm        *float64
	ThicknessMm     *float64
}

// ListBands returns the catalogue plus the user's own live bands.
func (s *Store) ListBands(ctx context.Context, userID uuid.UUID) ([]Band, error) {
	rows, err := s.q.ListBandsForUser(ctx, &userID)
	if err != nil {
		return nil, fmt.Errorf("listing bands: %w", err)
	}
	out := make([]Band, len(rows))
	for i, r := range rows {
		out[i] = bandFromRow(r)
	}
	return out, nil
}

// CreateBand adds a band to a user's collection.
func (s *Store) CreateBand(ctx context.Context, userID uuid.UUID, b Band) (Band, error) {
	lo, err := numeric(b.ResistanceMinKg)
	if err != nil {
		return Band{}, err
	}
	hi, err := numeric(b.ResistanceMaxKg)
	if err != nil {
		return Band{}, err
	}
	length, err := numericPtr(b.LengthCm)
	if err != nil {
		return Band{}, err
	}
	thick, err := numericPtr(b.ThicknessMm)
	if err != nil {
		return Band{}, err
	}
	r, err := s.q.InsertUserBand(ctx, dbgen.InsertUserBandParams{
		ID: b.ID, OwnerUserID: &userID, Brand: b.Brand, ColourLabel: b.ColourLabel,
		ResistanceMinKg: lo, ResistanceMaxKg: hi, LengthCm: length, ThicknessMm: thick,
	})
	if err != nil {
		if isUnique(err, "bands_pkey") || isUnique(err, "bands_identity_uk") {
			return Band{}, ErrAlreadyExists
		}
		return Band{}, translate(err)
	}
	return bandFromRow(r), nil
}

// DeleteBand removes a band from a user's collection. Logged sets keep it.
func (s *Store) DeleteBand(ctx context.Context, userID, id uuid.UUID) error {
	n, err := s.q.SoftDeleteUserBand(ctx, dbgen.SoftDeleteUserBandParams{ID: id, OwnerUserID: &userID})
	if err != nil {
		return fmt.Errorf("deleting band: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func bandFromRow(r dbgen.Band) Band {
	return Band{
		ID: r.ID, Mine: r.OwnerUserID != nil, Brand: r.Brand, ColourLabel: r.ColourLabel,
		ResistanceMinKg: numericToFloat(r.ResistanceMinKg), ResistanceMaxKg: numericToFloat(r.ResistanceMaxKg),
		LengthCm: numericPtrToFloat(r.LengthCm), ThicknessMm: numericPtrToFloat(r.ThicknessMm),
	}
}

// ReapDeletedUsers hard-deletes accounts whose deletion grace period has
// passed. purge runs first for each account, to remove what lives outside the
// database (media objects); an account whose purge fails is kept for the next
// run. Safe to run from several processes: an advisory lock makes all but one
// skip.
func (s *Store) ReapDeletedUsers(ctx context.Context, grace time.Duration, purge func(context.Context, uuid.UUID) error) (int64, error) {
	var n int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var got bool
		if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", reaperLockKey).Scan(&got); err != nil {
			return fmt.Errorf("taking reaper lock: %w", err)
		}
		if !got {
			return nil
		}
		q := dbgen.New(tx)
		cutoff := time.Now().Add(-grace)
		ids, err := q.UsersDueForReaping(ctx, &cutoff)
		if err != nil {
			return fmt.Errorf("listing accounts to reap: %w", err)
		}
		for _, id := range ids {
			if purge != nil {
				if err := purge(ctx, id); err != nil {
					slog.ErrorContext(ctx, "purging a deleted account failed; keeping it for the next run", "user_id", id, "error", err)
					continue
				}
			}
			k, err := q.ReapUser(ctx, dbgen.ReapUserParams{ID: id, Cutoff: &cutoff})
			if err != nil {
				return fmt.Errorf("reaping user: %w", err)
			}
			n += k
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("reaping deleted accounts: %w", err)
	}
	return n, nil
}

const reaperLockKey = 0x72656170 // "reap"
