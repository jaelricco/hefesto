package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// Media is an uploaded asset of the athlete's.
type Media struct {
	ID         uuid.UUID
	Kind       string
	StorageKey string
	Mime       string
	Bytes      int64
	Width      *int
	Height     *int
	Status     string
	CreatedAt  time.Time
	ReadyAt    *time.Time
}

// MediaKey is where a user's asset lives in object storage. Every object of
// a user shares the prefix, so deleting an account can remove them together.
func MediaKey(userID, id uuid.UUID) string { return MediaPrefix(userID) + id.String() }

// MediaPrefix is the object-storage prefix of everything a user uploaded.
func MediaPrefix(userID uuid.UUID) string { return "u/" + userID.String() + "/" }

// ReserveMedia records a pending asset. Reserving the same pending asset
// again returns it, so a client can ask for a fresh upload URL; any other
// clash on the id is ErrAlreadyExists.
func (s *Store) ReserveMedia(ctx context.Context, userID uuid.UUID, m Media) (Media, error) {
	row, err := s.q.InsertMediaAsset(ctx, dbgen.InsertMediaAssetParams{
		ID: m.ID, OwnerUserID: &userID, Kind: m.Kind, StorageKey: MediaKey(userID, m.ID), Mime: m.Mime,
		Bytes: m.Bytes, Width: int32Ptr(m.Width), Height: int32Ptr(m.Height),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		prev, err := s.q.GetMediaAsset(ctx, dbgen.GetMediaAssetParams{ID: m.ID, OwnerUserID: &userID})
		if err == nil && prev.Status == "pending" && prev.Mime == m.Mime && prev.Bytes == m.Bytes && prev.Kind == m.Kind {
			return mediaFromRow(prev), nil
		}
		return Media{}, ErrAlreadyExists
	}
	if err != nil {
		return Media{}, translate(fmt.Errorf("reserving media: %w", err))
	}
	return mediaFromRow(row), nil
}

// GetMedia returns a live asset of the user's.
func (s *Store) GetMedia(ctx context.Context, userID, id uuid.UUID) (Media, error) {
	row, err := s.q.GetMediaAsset(ctx, dbgen.GetMediaAssetParams{ID: id, OwnerUserID: &userID})
	if err != nil {
		return Media{}, translate(err)
	}
	return mediaFromRow(row), nil
}

// MarkMediaReady marks a pending (or already ready) asset ready.
func (s *Store) MarkMediaReady(ctx context.Context, userID, id uuid.UUID) (Media, error) {
	row, err := s.q.MarkMediaReady(ctx, dbgen.MarkMediaReadyParams{ID: id, OwnerUserID: &userID})
	if err != nil {
		return Media{}, translate(err)
	}
	return mediaFromRow(row), nil
}

// MarkMediaFailed marks a pending asset failed.
func (s *Store) MarkMediaFailed(ctx context.Context, userID, id uuid.UUID) (Media, error) {
	row, err := s.q.MarkMediaFailed(ctx, dbgen.MarkMediaFailedParams{ID: id, OwnerUserID: &userID})
	if err != nil {
		return Media{}, translate(err)
	}
	return mediaFromRow(row), nil
}

// DeleteMedia tombstones an asset and detaches it from set elements, touching
// those elements so the detachment reaches other devices. It returns the
// asset so the caller can remove the object.
func (s *Store) DeleteMedia(ctx context.Context, userID, id uuid.UUID) (Media, error) {
	var out Media
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if err := q.TouchElementsWithMedia(ctx, dbgen.TouchElementsWithMediaParams{MediaID: id, UserID: userID}); err != nil {
			return fmt.Errorf("touching elements: %w", err)
		}
		if err := q.DetachMedia(ctx, dbgen.DetachMediaParams{MediaID: id, UserID: userID}); err != nil {
			return fmt.Errorf("detaching media: %w", err)
		}
		row, err := q.SoftDeleteMediaAsset(ctx, dbgen.SoftDeleteMediaAssetParams{ID: id, OwnerUserID: &userID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		out = mediaFromRow(row)
		return nil
	})
	return out, err
}

// ExpirePendingMedia marks uploads reserved before `before` and never
// completed as failed, and returns their object keys for removal.
func (s *Store) ExpirePendingMedia(ctx context.Context, before time.Time) ([]string, error) {
	keys, err := s.q.ExpirePendingMedia(ctx, before)
	if err != nil {
		return nil, fmt.Errorf("expiring pending media: %w", err)
	}
	return keys, nil
}

func mediaFromRow(r dbgen.MediaAsset) Media {
	return Media{
		ID: r.ID, Kind: r.Kind, StorageKey: r.StorageKey, Mime: r.Mime, Bytes: r.Bytes,
		Width: intPtr32(r.Width), Height: intPtr32(r.Height), Status: r.Status, CreatedAt: r.CreatedAt, ReadyAt: r.ReadyAt,
	}
}
