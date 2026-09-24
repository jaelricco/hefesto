package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/store"
)

// Health reports whether a dependency is usable. Returning an error makes
// /readyz fail; /healthz never consults it.
type Health interface {
	Ping(ctx context.Context) error
}

// RouterDeps is everything the transport layer needs from the outside world.
// When Auth or Store is nil only the health endpoints are served.
type RouterDeps struct {
	DB      Health
	Version string
	Commit  string

	Auth    *auth.Service
	Store   *store.Store
	Schemas *Schemas

	// TrustProxy takes the client address from X-Forwarded-For.
	TrustProxy bool
	// DeletionGrace is how long a deleted account can still be recovered.
	DeletionGrace time.Duration
	// AuthPerMinute limits credential attempts per client address.
	AuthPerMinute float64
}

// NewRouter builds the HTTP handler.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID, RequestLogger, Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": deps.Version,
			"commit":  deps.Commit,
		})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			WriteProblem(w, r, problem("not-ready", "Not ready", http.StatusServiceUnavailable, "database is not configured"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := deps.DB.Ping(ctx); err != nil {
			slog.ErrorContext(ctx, "readiness check failed", "error", err)
			WriteProblem(w, r, problem("not-ready", "Not ready", http.StatusServiceUnavailable, "database unreachable"))
			return
		}
		WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ready"})
	})

	if deps.Auth != nil && deps.Store != nil && deps.Schemas != nil {
		h := &handlers{RouterDeps: deps}
		perMinute := deps.AuthPerMinute
		if perMinute <= 0 {
			perMinute = 10
		}
		limiter := newIPLimiter(perMinute, int(perMinute), deps.TrustProxy)

		r.Route("/v1", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(limiter.middleware)
				r.Post("/auth/register", h.wrap(h.register))
				r.Post("/auth/login", h.wrap(h.login))
				r.Post("/auth/apple", h.wrap(h.apple))
				r.Post("/auth/refresh", h.wrap(h.refresh))
			})
			r.Post("/auth/logout", h.wrap(h.logout))

			r.Group(func(r chi.Router) {
				r.Use(requireAuth(deps.Auth))

				r.Get("/me", h.wrap(h.getMe))
				r.Patch("/me", h.wrap(h.updateMe))
				r.Delete("/me", h.wrap(h.deleteMe))
				r.Get("/me/exercises/{exerciseId}/last-set", h.wrap(h.lastSet))
				r.Post("/me/bands", h.wrap(h.createBand))
				r.Delete("/me/bands/{bandId}", h.wrap(h.deleteBand))

				r.Get("/exercises", h.wrap(h.listExercises))
				r.Get("/exercises/{slug}", h.wrap(h.getExercise))
				r.Get("/bands", h.wrap(h.listBands))

				r.Post("/sessions", h.wrap(h.createSession))
				r.Get("/sessions", h.wrap(h.listSessions))
				r.Get("/sessions/{sessionId}", h.wrap(h.getSession))
				r.Patch("/sessions/{sessionId}", h.wrap(h.updateSession))
				r.Delete("/sessions/{sessionId}", h.wrap(h.deleteSession))
				r.Put("/sessions/{sessionId}/blocks/{blockId}", h.wrap(h.putBlock))
				r.Delete("/sessions/{sessionId}/blocks/{blockId}", h.wrap(h.deleteBlock))
				r.Put("/sessions/{sessionId}/sets/{setId}", h.wrap(h.putSet))
				r.Delete("/sessions/{sessionId}/sets/{setId}", h.wrap(h.deleteSet))
				r.Post("/sessions/{sessionId}/reorder", h.wrap(h.reorder))
				r.Post("/sessions/{sessionId}/complete", h.wrap(h.completeSession))

				r.Get("/skills", h.wrap(h.getSkillGraph))
				r.Get("/skills/{slug}", h.wrap(h.getSkill))
				r.Get("/me/skill-map", h.wrap(h.getMySkillMap))
				r.Post("/me/skills/{levelId}/attest", h.wrap(h.attestLevel))
				r.Get("/me/progress", h.wrap(h.getMyProgress))
			})
		})
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, problem("not-found", "Not found", http.StatusNotFound, "no route matches "+r.Method+" "+r.URL.Path))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, problem("method-not-allowed", "Method not allowed", http.StatusMethodNotAllowed, ""))
	})
	return r
}

type handlers struct {
	RouterDeps
}

// wrap adapts a handler that returns an error. Every error goes through
// writeError, so no handler writes a problem response by hand.
func (h *handlers) wrap(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			writeError(w, r, err)
		}
	}
}

// body reads the request body, validates it against the named schema of the
// OpenAPI document, and decodes it into v.
func (h *handlers) body(w http.ResponseWriter, r *http.Request, schema string, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return errBadRequest{"body too large or unreadable"}
	}
	if len(raw) == 0 {
		return errBadRequest{"empty body"}
	}
	if err := h.Schemas.Validate(schema, raw); err != nil {
		return err
	}
	r.Body = io.NopCloser(bytesReader(raw))
	return decode(w, r, v)
}

// pathID parses a UUID path parameter.
func pathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return uuid.Nil, errBadRequest{name + " is not a UUID"}
	}
	return id, nil
}

// writer builds the store's Writer for this request. at is the client's
// clock from the body, when it sent one.
func writer(r *http.Request, at *time.Time) store.Writer {
	p := principalFrom(r.Context())
	w := store.Writer{UserID: p.UserID, DeviceID: p.DeviceID, At: time.Now()}
	if at != nil {
		w.At = *at
	}
	return w
}

func (h *handlers) client(r *http.Request, d *deviceIn) auth.Client {
	return auth.Client{Device: d.toStore(), UserAgent: r.UserAgent(), IP: clientIP(r, h.TrustProxy)}
}
