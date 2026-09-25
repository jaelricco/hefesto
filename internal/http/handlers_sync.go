package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

const (
	maxSyncBodyBytes = 4 << 20 // two hundred ops, each at most a twenty-element set
	maxSyncOps       = 200
)

var idempotencyKeyRE = regexp.MustCompile(`^[\x21-\x7E]{8,255}$`)

func (h *handlers) pullChanges(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	cursor, err := strconv.ParseInt(q.Get("cursor"), 10, 64)
	if err != nil || cursor < 0 {
		return errBadRequest{"cursor must be a non-negative integer; start from 0"}
	}
	limit := 200
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 || limit > 1000 {
			return errBadRequest{"limit must be between 1 and 1000"}
		}
	}
	p := principalFrom(r.Context())
	page, err := h.Store.Pull(r.Context(), p.UserID, p.DeviceID, cursor, limit)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, syncPageFrom(page))
	return nil
}

// pushChanges applies a batch of offline writes behind an Idempotency-Key.
// The first complete response is stored and replayed for a repeat of the same
// request; a failure the client cannot fix by changing the request (a server
// error) gives the key back so a retry runs again.
func (h *handlers) pushChanges(w http.ResponseWriter, r *http.Request) error {
	key := r.Header.Get("Idempotency-Key")
	if !idempotencyKeyRE.MatchString(key) {
		return errBadRequest{"Idempotency-Key is required: 8 to 255 printable ASCII characters"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSyncBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return errBadRequest{fmt.Sprintf("body larger than %d bytes or unreadable", maxSyncBodyBytes)}
	}
	fp := sha256.Sum256(append([]byte(r.Method+" "+r.URL.Path+"\n"), raw...))

	userID := principalFrom(r.Context()).UserID
	stored, err := h.Store.ClaimIdempotencyKey(r.Context(), userID, key, fp[:])
	if err != nil {
		return err
	}
	if stored != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Idempotent-Replayed", "true")
		w.WriteHeader(stored.Status)
		_, _ = w.Write(stored.Body)
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

	ops, err := parseSyncOps(raw)
	if err != nil {
		return err // nothing applied; the key is released and the request may be fixed
	}
	out := syncPushOut{Results: make([]syncOpResultOut, len(ops))}
	for i, op := range ops {
		res, err := h.applySyncOp(r, op)
		if err != nil {
			return err
		}
		res.Index = i
		out.Results[i] = res
	}

	body, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("encoding sync results: %w", err)
	}
	if err := h.Store.FinishIdempotencyKey(r.Context(), userID, key, store.StoredResponse{Status: http.StatusOK, Body: body}); err != nil {
		return err
	}
	finished = true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return nil
}

// parseSyncOps checks the envelope. Each op's own validity is judged when it
// is applied, so one malformed op does not hold back the rest.
func parseSyncOps(raw []byte) ([]json.RawMessage, error) {
	var body struct {
		Ops []json.RawMessage `json:"ops"`
	}
	if err := decodeJSON(bytes.NewReader(raw), &body); err != nil {
		return nil, err
	}
	if len(body.Ops) == 0 || len(body.Ops) > maxSyncOps {
		return nil, training.FieldErrors{"/ops": fmt.Sprintf("between 1 and %d ops", maxSyncOps)}.Err()
	}
	return body.Ops, nil
}

// applySyncOp validates and runs one op. Its error is for failures that are
// not the op's fault; everything the op got wrong is in the result.
func (h *handlers) applySyncOp(r *http.Request, raw json.RawMessage) (syncOpResultOut, error) {
	op, err := h.decodeSyncOp(raw)
	if err == nil {
		err = h.runSyncOp(r, op)
	}
	var (
		completion *completionOut
		c          completed
	)
	if errors.As(err, &c) {
		completion, err = &c.out, nil
	}
	switch {
	case err == nil:
		return syncOpResultOut{Status: "applied", Completion: completion}, nil
	case errors.Is(err, store.ErrDeleted) && op.Op == "delete":
		return syncOpResultOut{Status: "applied"}, nil
	case errors.Is(err, store.ErrDeleted), errors.Is(err, store.ErrStaleWrite):
		p, _ := problemFor(r, err)
		p.Instance = ""
		return syncOpResultOut{Status: "superseded", Problem: &p}, nil
	}
	p, known := problemFor(r, err)
	if !known {
		return syncOpResultOut{}, err
	}
	p.Instance = ""
	return syncOpResultOut{Status: "rejected", Problem: &p}, nil
}

// completed carries a completion's outcome out of runSyncOp.
type completed struct{ out completionOut }

func (completed) Error() string { return "completed" }

// decodeSyncOp validates an op against the SyncOp schema and its data against
// the schema of that op. Pointers in errors are relative to the op.
func (h *handlers) decodeSyncOp(raw json.RawMessage) (syncOpIn, error) {
	var op syncOpIn
	// The envelope first, without data: data is judged against its own schema.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return op, errBadRequest{"an op is not a JSON object"}
	}
	data, hasData := fields["data"]
	delete(fields, "data")
	envelope, err := json.Marshal(fields)
	if err != nil {
		return op, fmt.Errorf("re-encoding op: %w", err)
	}
	if err := h.Schemas.Validate("SyncOp", envelope); err != nil {
		return op, err
	}
	if err := json.Unmarshal(envelope, &op); err != nil {
		return op, errBadRequest{err.Error()}
	}

	schema, needsSession := syncOpSchema(op.Entity, op.Op)
	switch {
	case schema == "-":
		return op, training.FieldErrors{"/op": op.Op + " is not an op on " + op.Entity}.Err()
	case needsSession && op.SessionID == nil:
		return op, training.FieldErrors{"/session_id": "required for a " + op.Entity}.Err()
	case !needsSession && op.SessionID != nil:
		return op, training.FieldErrors{"/session_id": "only blocks and sets name a session"}.Err()
	case schema == "" && hasData:
		return op, training.FieldErrors{"/data": "a " + op.Op + " op carries no data"}.Err()
	case schema != "" && !hasData:
		return op, training.FieldErrors{"/data": "required, a " + schema}.Err()
	}
	if schema != "" {
		if err := h.Schemas.Validate(schema, data); err != nil {
			return op, prefixFields(err, "/data")
		}
		op.Data = data
	}
	return op, nil
}

// syncOpSchema names the data schema of an op ("" for none, "-" for an op
// that does not exist) and whether it names a session.
func syncOpSchema(entity, op string) (schema string, needsSession bool) {
	switch entity + "." + op {
	case "session.put":
		return "SessionPut", false
	case "session.complete":
		return "SessionComplete", false
	case "session.delete", "bodyweight.delete":
		return "", false
	case "block.put":
		return "BlockWrite", true
	case "set.put":
		return "SetEntryWrite", true
	case "block.delete", "set.delete":
		return "", true
	case "bodyweight.put":
		return "BodyweightWrite", false
	}
	return "-", false
}

func prefixFields(err error, prefix string) error {
	var ve *training.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	out := training.FieldErrors{}
	for k, v := range ve.Fields {
		out[prefix+k] = v
	}
	return out.Err()
}

// runSyncOp runs an op through the same code as its REST endpoint.
func (h *handlers) runSyncOp(r *http.Request, op syncOpIn) error {
	ctx := r.Context()
	decode := func(v any) error { return prefixFields(decodeJSON(bytes.NewReader(op.Data), v), "/data") }

	switch op.Entity + "." + op.Op {
	case "session.put":
		var in sessionPutIn
		if err := decode(&in); err != nil {
			return err
		}
		return h.putSessionState(ctx, writer(r, in.UpdatedAt), op.ID, in)
	case "session.complete":
		var in completeIn
		if err := decode(&in); err != nil {
			return err
		}
		c, err := h.Store.CompleteSession(ctx, writer(r, in.UpdatedAt), op.ID, in.toStore())
		if err != nil {
			return err
		}
		return completed{completionFrom(c)}
	case "session.delete":
		return h.Store.DeleteSession(ctx, writer(r, nil), op.ID)
	case "block.put":
		var in blockIn
		if err := decode(&in); err != nil {
			return err
		}
		_, _, err := h.applyPutBlock(ctx, writer(r, in.UpdatedAt), *op.SessionID, op.ID, in)
		return err
	case "block.delete":
		return h.Store.DeleteBlock(ctx, writer(r, nil), *op.SessionID, op.ID)
	case "set.put":
		var in setIn
		if err := decode(&in); err != nil {
			return err
		}
		_, _, err := h.applyPutSet(ctx, writer(r, in.UpdatedAt), *op.SessionID, op.ID, in)
		return prefixFields(err, "/data")
	case "set.delete":
		return h.Store.DeleteSet(ctx, writer(r, nil), *op.SessionID, op.ID)
	case "bodyweight.put":
		var in bodyweightIn
		if err := decode(&in); err != nil {
			return err
		}
		return h.putBodyweight(ctx, writer(r, in.UpdatedAt), op.ID, in)
	case "bodyweight.delete":
		return h.Store.DeleteBodyweight(ctx, writer(r, nil), op.ID)
	}
	return training.FieldErrors{"/op": "unknown op"}.Err()
}

// putSessionState creates a session from its full state, or updates the
// fields sent, through the REST paths. A new session is created, then given
// the fields a create does not take. Completion is final: the status of a
// completed session is left alone.
func (h *handlers) putSessionState(ctx context.Context, wr store.Writer, id uuid.UUID, in sessionPutIn) error {
	cur, err := h.Store.GetSession(ctx, wr.UserID, id)
	if err == nil {
		upd := sessionUpdateIn{
			StartedAt: Optional[time.Time]{Set: true, Value: in.StartedAt}, Timezone: Optional[string]{Set: true, Value: in.Timezone},
			EndedAt: in.EndedAt, Title: in.Title, Notes: in.Notes, PerceivedFatigue: in.PerceivedFatigue,
			BodyweightKg: in.BodyweightKg, IsRestDay: in.IsRestDay, Status: in.Status,
		}
		if cur.Status == training.StatusCompleted {
			upd.Status = Optional[string]{}
		}
		_, err = h.applyUpdateSession(ctx, wr, id, upd)
		return prefixFields(err, "/data")
	}
	if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	create := sessionCreateIn{
		ID: id, StartedAt: in.StartedAt, Timezone: in.Timezone, Title: in.Title.Value, Notes: in.Notes.Value,
		IsRestDay: in.IsRestDay.Value, TemplateID: in.TemplateID,
	}
	if in.BodyweightKg.Set {
		create.BodyweightKg = in.BodyweightKg.Ptr()
	}
	if _, err := h.applyCreateSession(ctx, wr, create); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			if gone, e := h.Store.SessionTombstoned(ctx, wr.UserID, id); e == nil && gone {
				return store.ErrDeleted
			}
		}
		return prefixFields(err, "/data")
	}
	if !in.EndedAt.Set && !in.PerceivedFatigue.Set && !in.Status.Set {
		return nil
	}
	_, err = h.applyUpdateSession(ctx, wr, id, sessionUpdateIn{
		EndedAt: in.EndedAt, PerceivedFatigue: in.PerceivedFatigue, Status: in.Status,
	})
	return prefixFields(err, "/data")
}

func (h *handlers) putBodyweight(ctx context.Context, wr store.Writer, id uuid.UUID, in bodyweightIn) error {
	loc, err := time.LoadLocation(in.Timezone)
	if err != nil || in.Timezone == "Local" {
		return training.FieldErrors{"/data/timezone": "not a known IANA time zone"}.Err()
	}
	_, err = h.Store.PutBodyweight(ctx, wr, store.Bodyweight{
		ID: id, MeasuredAt: in.MeasuredAt, LocalDate: training.LocalDate(in.MeasuredAt, loc),
		BodyweightKg: in.BodyweightKg, Note: in.Note,
	})
	return err
}
