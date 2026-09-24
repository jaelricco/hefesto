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

// errMediaUnavailable is a media request on a server without object storage.
var errMediaUnavailable = errors.New("object storage is not configured")

// writeError maps an error from any layer to its problem response. Unknown
// errors are logged with the request id and become a 500 that says nothing
// about the internals.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	p, known := problemFor(r, err)
	if !known {
		slog.ErrorContext(r.Context(), "request failed", "error", err, "request_id", RequestIDFrom(r.Context()),
			"method", r.Method, "path", r.URL.Path)
	}
	p.TraceID = RequestIDFrom(r.Context())
	WriteProblem(w, r, p)
}

// problemFor maps an error to its problem. known is false for an error no
// layer expected: a bug, reported as a bare 500.
func problemFor(r *http.Request, err error) (p Problem, known bool) {
	var (
		bad errBadRequest
		ve  *training.ValidationError
	)
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
	case errors.Is(err, store.ErrPrerequisitesUnmet):
		p = problem("prerequisites-unmet", "Prerequisites not unlocked", http.StatusConflict,
			"unlock this level's prerequisites first")
	case errors.Is(err, store.ErrStaleWrite):
		p = problem("stale-write", "Stale write", http.StatusConflict,
			"the server's copy of this set is newer, by the athlete's clock, than this write")
	case errors.Is(err, store.ErrIdempotencyInProgress):
		p = problem("idempotency-in-progress", "Request in progress", http.StatusConflict,
			"a request with this Idempotency-Key is still being processed; retry shortly")
	case errors.Is(err, store.ErrIdempotencyKeyReuse):
		p = problem("idempotency-key-reuse", "Idempotency key reused", http.StatusUnprocessableEntity,
			"this Idempotency-Key was used with a different request")
	case errors.Is(err, errMediaUnavailable):
		p = problem("not-ready", "Not ready", http.StatusServiceUnavailable, "media storage is not available")
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
		return problem("internal", "Internal server error", http.StatusInternalServerError, ""), false
	}
	return p, true
}
