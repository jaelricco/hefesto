package http

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"time"

	"github.com/jaelricco/hefesto/internal/store"
)

// withIdempotencyKey runs fn once per Idempotency-Key and user. The first
// complete response is stored and replayed for a repeat of the same request
// (method, path and body); the key with another request is
// idempotency-key-reuse. When fn fails, the key is given back so a retry
// runs again.
func (h *handlers) withIdempotencyKey(w http.ResponseWriter, r *http.Request, key string, raw []byte,
	fn func() (status int, body []byte, err error)) error {
	fp := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), raw...))
	userID := principalFrom(r.Context()).UserID
	stored, err := h.Store.ClaimIdempotencyKey(r.Context(), userID, key, fp[:])
	if err != nil {
		return err
	}
	if stored != nil {
		w.Header().Set("Idempotent-Replayed", "true")
		writeRaw(w, stored.Status, stored.Body)
		return nil
	}

	finished := false
	defer func() {
		if !finished {
			// Detached from the request: a cancelled request still frees its key.
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
			defer cancel()
			if err := h.Store.ReleaseIdempotencyKey(ctx, userID, key); err != nil {
				slog.ErrorContext(ctx, "releasing an idempotency key failed", "error", err,
					"request_id", RequestIDFrom(r.Context()))
			}
		}
	}()

	status, body, err := fn()
	if err != nil {
		return err
	}
	if err := h.Store.FinishIdempotencyKey(r.Context(), userID, key, store.StoredResponse{Status: status, Body: body}); err != nil {
		return err
	}
	finished = true
	writeRaw(w, status, body)
	return nil
}

// writeRaw writes an encoded JSON response.
func writeRaw(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
