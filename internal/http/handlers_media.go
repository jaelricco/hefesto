package http

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/media"
	"github.com/jaelricco/hefesto/internal/store"
)

type mediaUploadIn struct {
	ID     uuid.UUID `json:"id"`
	Kind   string    `json:"kind"`
	Mime   string    `json:"mime"`
	Bytes  int64     `json:"bytes"`
	Width  *int      `json:"width"`
	Height *int      `json:"height"`
}

type mediaOut struct {
	ID           uuid.UUID  `json:"id"`
	Kind         string     `json:"kind"`
	Mime         string     `json:"mime"`
	Bytes        int64      `json:"bytes"`
	Width        *int       `json:"width"`
	Height       *int       `json:"height"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ReadyAt      *time.Time `json:"ready_at"`
	URL          *string    `json:"url"`
	URLExpiresAt *time.Time `json:"url_expires_at"`
}

type uploadOut struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type mediaUploadOut struct {
	Media  mediaOut  `json:"media"`
	Upload uploadOut `json:"upload"`
}

func (h *handlers) presignTTL() time.Duration {
	if h.PresignTTL > 0 {
		return h.PresignTTL
	}
	return 15 * time.Minute
}

// mediaFrom describes an asset, with a download URL once it is ready.
func (h *handlers) mediaFrom(r *http.Request, m store.Media) (mediaOut, error) {
	out := mediaOut{
		ID: m.ID, Kind: m.Kind, Mime: m.Mime, Bytes: m.Bytes, Width: m.Width, Height: m.Height, Status: m.Status,
		CreatedAt: utc(m.CreatedAt), ReadyAt: utcPtr(m.ReadyAt),
	}
	if m.Status == "ready" {
		p, err := h.Media.PresignGet(r.Context(), m.StorageKey, h.presignTTL())
		if err != nil {
			return out, err
		}
		exp := utc(p.ExpiresAt)
		out.URL, out.URLExpiresAt = &p.URL, &exp
	}
	return out, nil
}

func (h *handlers) createUpload(w http.ResponseWriter, r *http.Request) error {
	if h.Media == nil {
		return errMediaUnavailable
	}
	var in mediaUploadIn
	if err := h.body(w, r, "MediaUploadRequest", &in); err != nil {
		return err
	}
	m, err := h.Store.ReserveMedia(r.Context(), principalFrom(r.Context()).UserID, store.Media{
		ID: in.ID, Kind: in.Kind, Mime: in.Mime, Bytes: in.Bytes, Width: in.Width, Height: in.Height,
	})
	if err != nil {
		return err
	}
	p, err := h.Media.PresignPut(r.Context(), m.StorageKey, m.Mime, m.Bytes, h.presignTTL())
	if err != nil {
		return err
	}
	out, err := h.mediaFrom(r, m)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusCreated, mediaUploadOut{
		Media:  out,
		Upload: uploadOut{Method: http.MethodPut, URL: p.URL, Headers: p.Headers, ExpiresAt: utc(p.ExpiresAt)},
	})
	return nil
}

// completeUpload checks what arrived in the bucket against the reservation.
func (h *handlers) completeUpload(w http.ResponseWriter, r *http.Request) error {
	if h.Media == nil {
		return errMediaUnavailable
	}
	id, err := pathID(r, "mediaId")
	if err != nil {
		return err
	}
	userID := principalFrom(r.Context()).UserID
	m, err := h.Store.GetMedia(r.Context(), userID, id)
	if err != nil {
		return err
	}
	switch m.Status {
	case "ready":
	case "pending":
		obj, err := h.Media.Stat(r.Context(), m.StorageKey)
		if errors.Is(err, media.ErrNotFound) {
			return training.FieldErrors{"": "nothing has been uploaded for this asset yet"}.Err()
		}
		if err != nil {
			return err
		}
		if obj.Size != m.Bytes || obj.ContentType != m.Mime {
			if err := h.Media.Remove(r.Context(), m.StorageKey); err != nil {
				return err
			}
			if _, err := h.Store.MarkMediaFailed(r.Context(), userID, id); err != nil {
				return err
			}
			return training.FieldErrors{"": "the uploaded file does not match what was reserved; reserve a new id"}.Err()
		}
		if m, err = h.Store.MarkMediaReady(r.Context(), userID, id); err != nil {
			return err
		}
	default:
		return training.FieldErrors{"": "this upload " + m.Status + "; reserve a new id"}.Err()
	}
	out, err := h.mediaFrom(r, m)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) getMedia(w http.ResponseWriter, r *http.Request) error {
	if h.Media == nil {
		return errMediaUnavailable
	}
	id, err := pathID(r, "mediaId")
	if err != nil {
		return err
	}
	m, err := h.Store.GetMedia(r.Context(), principalFrom(r.Context()).UserID, id)
	if err != nil {
		return err
	}
	out, err := h.mediaFrom(r, m)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) deleteMedia(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "mediaId")
	if err != nil {
		return err
	}
	m, err := h.Store.DeleteMedia(r.Context(), principalFrom(r.Context()).UserID, id)
	if err != nil {
		return err
	}
	if h.Media != nil {
		// The row is already gone for the athlete; an object left behind is
		// removed with the account at the latest.
		if err := h.Media.Remove(r.Context(), m.StorageKey); err != nil {
			slog.WarnContext(r.Context(), "removing a deleted asset's object failed", "error", err,
				"request_id", RequestIDFrom(r.Context()))
		}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
