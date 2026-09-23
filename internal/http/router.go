package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Health reports whether a dependency is usable. Returning an error makes
// /readyz fail; /healthz never consults it.
type Health interface {
	Ping(ctx context.Context) error
}

// RouterDeps is everything the transport layer needs from the outside world.
//
// Phase 0 wires only health. Phase 2 adds the stores and the auth service.
type RouterDeps struct {
	DB      Health
	Version string
	Commit  string
}

// NewRouter builds the HTTP handler.
//
// Phase 0 uses net/http's pattern router. Phase 1 replaces this with chi so
// that subrouters, middleware groups and URL parameters stay readable; the
// handler signatures do not change when it does.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": deps.Version,
			"commit":  deps.Commit,
		})
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			WriteProblem(w, r, Problem{
				Type:   "https://hefesto.fit/problems/not-ready",
				Title:  "Not ready",
				Status: http.StatusServiceUnavailable,
				Detail: "database is not configured",
			})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := deps.DB.Ping(ctx); err != nil {
			slog.ErrorContext(ctx, "readiness check failed", "error", err)
			WriteProblem(w, r, Problem{
				Type:   "https://hefesto.fit/problems/not-ready",
				Title:  "Not ready",
				Status: http.StatusServiceUnavailable,
				Detail: "database is unreachable",
			})
			return
		}
		WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, Problem{
			Type:   "https://hefesto.fit/problems/not-found",
			Title:  "Not found",
			Status: http.StatusNotFound,
			Detail: "no route matches " + r.Method + " " + r.URL.Path,
		})
	})

	return RequestID(RequestLogger(Recoverer(mux)))
}
