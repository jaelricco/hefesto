package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

const problemBase = "https://hefesto.fit/problems/"

func problem(slug, title string, status int, detail string) Problem {
	return Problem{Type: problemBase + slug, Title: title, Status: status, Detail: detail}
}

// writeError maps an error from any layer to its problem response. Unknown
// errors are logged with the request id and become a 500 that says nothing
// about the internals.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		bad errBadRequest
		ve  *training.ValidationError
	)
	var p Problem
	switch {
	case errors.As(err, &bad):
		p = problem("bad-request", "Bad request", http.StatusBadRequest, bad.msg)
	case errors.As(err, &ve):
		p = problem("validation", "Validation failed", http.StatusUnprocessableEntity, "")
		p.Errors = ve.Fields
	case errors.Is(err, store.ErrNotFound):
		p = problem("not-found", "Not found", http.StatusNotFound, "")
	case errors.Is(err, store.ErrAlreadyExists):
		p = problem("already-exists", "Already exists", http.StatusConflict, "a resource with this id already exists")
	case errors.Is(err, store.ErrEmailTaken):
		p = problem("email-taken", "Email already registered", http.StatusConflict, "")
	case errors.Is(err, store.ErrOrderConflict):
		p = problem("order-conflict", "Order conflict", http.StatusConflict,
			"another live sibling already has this order_index; use the reorder endpoint to move several at once")
	case errors.Is(err, auth.ErrInvalidCredentials):
		p = problem("invalid-credentials", "Invalid credentials", http.StatusUnauthorized, "")
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		p = problem("invalid-refresh-token", "Invalid refresh token", http.StatusUnauthorized, "sign in again")
		slog.InfoContext(r.Context(), "refresh refused", "reason", err, "request_id", RequestIDFrom(r.Context()))
	case errors.Is(err, auth.ErrInvalidIdentityToken):
		p = problem("invalid-identity-token", "Invalid identity token", http.StatusUnauthorized, "")
		slog.InfoContext(r.Context(), "apple sign-in refused", "reason", err, "request_id", RequestIDFrom(r.Context()))
	case errors.Is(err, auth.ErrInvalidToken):
		p = problem("unauthorized", "Unauthorized", http.StatusUnauthorized, "missing, malformed or expired access token")
	case errors.Is(err, auth.ErrAccountSuspended), errors.Is(err, auth.ErrForbidden):
		p = problem("forbidden", "Forbidden", http.StatusForbidden, "")
	default:
		slog.ErrorContext(r.Context(), "request failed", "error", err, "request_id", RequestIDFrom(r.Context()),
			"method", r.Method, "path", r.URL.Path)
		p = problem("internal", "Internal server error", http.StatusInternalServerError, "")
	}
	p.TraceID = RequestIDFrom(r.Context())
	WriteProblem(w, r, p)
}
