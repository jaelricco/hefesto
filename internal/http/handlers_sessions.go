package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

func (h *handlers) createSession(w http.ResponseWriter, r *http.Request) error {
	var in sessionCreateIn
	if err := h.body(w, r, "SessionCreate", &in); err != nil {
		return err
	}
	loc, err := time.LoadLocation(in.Timezone)
	if err != nil || in.Timezone == "Local" {
		return training.FieldErrors{"/timezone": "not a known IANA time zone"}.Err()
	}
	s := training.Session{
		ID: in.ID, StartedAt: in.StartedAt, Timezone: in.Timezone, LocalDate: training.LocalDate(in.StartedAt, loc),
		Title: in.Title, Notes: in.Notes, BodyweightKg: in.BodyweightKg, IsRestDay: in.IsRestDay,
		TemplateID: in.TemplateID, Status: training.StatusDraft,
	}
	if err := training.ValidateSession(s); err != nil {
		return err
	}
	out, err := h.Store.CreateSession(r.Context(), writer(r, in.UpdatedAt), s)
	if err != nil {
		return err
	}
	w.Header().Set("Location", "/v1/sessions/"+out.ID.String())
	WriteJSON(w, r, http.StatusCreated, sessionFrom(out))
	return nil
}

func (h *handlers) getSession(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	s, err := h.Store.GetSession(r.Context(), principalFrom(r.Context()).UserID, id)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, sessionFrom(s))
	return nil
}

func (h *handlers) updateSession(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	var in sessionUpdateIn
	if err := h.body(w, r, "SessionUpdate", &in); err != nil {
		return err
	}
	out, err := h.Store.UpdateSession(r.Context(), writer(r, in.UpdatedAt), id, func(s *training.Session) error {
		return applySessionUpdate(s, in)
	})
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, sessionFrom(out))
	return nil
}

// applySessionUpdate applies the present fields of a PATCH and re-derives
// local_date if the start or the zone moved.
func applySessionUpdate(s *training.Session, in sessionUpdateIn) error {
	if in.StartedAt.Set {
		if in.StartedAt.Null {
			return training.FieldErrors{"/started_at": "cannot be null"}.Err()
		}
		s.StartedAt = in.StartedAt.Value
	}
	if in.Timezone.Set {
		s.Timezone = in.Timezone.Value
	}
	if in.StartedAt.Set || in.Timezone.Set {
		loc, err := time.LoadLocation(s.Timezone)
		if err != nil || s.Timezone == "Local" {
			return training.FieldErrors{"/timezone": "not a known IANA time zone"}.Err()
		}
		s.LocalDate = training.LocalDate(s.StartedAt, loc)
	}
	if in.EndedAt.Set {
		s.EndedAt = in.EndedAt.Ptr()
	}
	if in.Title.Set {
		s.Title = in.Title.Value
	}
	if in.Notes.Set {
		s.Notes = in.Notes.Value
	}
	if in.PerceivedFatigue.Set {
		s.PerceivedFatigue = in.PerceivedFatigue.Ptr()
	}
	if in.BodyweightKg.Set {
		s.BodyweightKg = in.BodyweightKg.Ptr()
	}
	if in.IsRestDay.Set {
		s.IsRestDay = in.IsRestDay.Value
	}
	if in.Status.Set {
		if !training.CanTransition(s.Status, in.Status.Value) {
			return training.FieldErrors{"/status": "a " + s.Status + " session cannot become " + in.Status.Value + " by an edit"}.Err()
		}
		s.Status = in.Status.Value
	}
	return training.ValidateSession(*s)
}

func (h *handlers) deleteSession(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	if err := h.Store.DeleteSession(r.Context(), writer(r, nil), id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *handlers) listSessions(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	f := store.SessionFilter{Limit: 20}
	for name, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
		if v := q.Get(name); v != "" {
			d, err := time.Parse(dateLayout, v)
			if err != nil {
				return errBadRequest{name + " is not a date (YYYY-MM-DD)"}
			}
			*dst = &d
		}
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return errBadRequest{"limit must be between 1 and 100"}
		}
		f.Limit = n
	}
	if v := q.Get("cursor"); v != "" {
		at, id, err := decodeCursor(v)
		if err != nil {
			return errBadRequest{"cursor is not one this server issued"}
		}
		f.CursorStartedAt, f.CursorID = &at, &id
	}

	want := f.Limit
	f.Limit++ // one extra row says whether there is another page
	rows, err := h.Store.ListSessions(r.Context(), principalFrom(r.Context()).UserID, f)
	if err != nil {
		return err
	}
	out := sessionPageOut{Items: []sessionSummaryOut{}}
	if len(rows) > want {
		last := rows[want-1]
		c := encodeCursor(last.StartedAt, last.ID)
		out.NextCursor = &c
		rows = rows[:want]
	}
	for _, s := range rows {
		out.Items = append(out.Items, sessionSummaryOut{
			ID: s.ID, StartedAt: utc(s.StartedAt), EndedAt: utcPtr(s.EndedAt), Timezone: s.Timezone,
			LocalDate: s.LocalDate.Format(dateLayout), Title: s.Title, Status: s.Status,
			IsRestDay: s.IsRestDay, BlockCount: s.BlockCount, SetCount: s.SetCount,
		})
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func encodeCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func decodeCursor(c string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("cursor encoding: %w", err)
	}
	ts, idStr, ok := strings.Cut(string(raw), "|")
	if !ok {
		return time.Time{}, uuid.Nil, errBadRequest{"cursor"}
	}
	at, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("cursor time: %w", err)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("cursor id: %w", err)
	}
	return at, id, nil
}

func (h *handlers) putBlock(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	blockID, err := pathID(r, "blockId")
	if err != nil {
		return err
	}
	var in blockIn
	if err := h.body(w, r, "BlockWrite", &in); err != nil {
		return err
	}
	b := training.Block{
		ID: blockID, SessionID: sessionID, OrderIndex: in.OrderIndex, Kind: orDefault(in.Kind, "straight"),
		RoundsPlanned: in.RoundsPlanned, RoundsDone: in.RoundsDone, IntervalS: in.IntervalS, Notes: in.Notes,
	}
	if err := training.ValidateBlock(b); err != nil {
		return err
	}
	out, created, err := h.Store.PutBlock(r.Context(), writer(r, in.UpdatedAt), sessionID, b)
	if err != nil {
		return err
	}
	WriteJSON(w, r, createdOrOK(created), blockFrom(out))
	return nil
}

func (h *handlers) deleteBlock(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	blockID, err := pathID(r, "blockId")
	if err != nil {
		return err
	}
	if err := h.Store.DeleteBlock(r.Context(), writer(r, nil), sessionID, blockID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// putSet is the one way a set is written, whether it has one element or
// several.
func (h *handlers) putSet(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	setID, err := pathID(r, "setId")
	if err != nil {
		return err
	}
	var in setIn
	if err := h.body(w, r, "SetEntryWrite", &in); err != nil {
		return err
	}
	set := in.toDomain(setID, sessionID)
	if err := training.ValidateSet(&set); err != nil {
		return err
	}
	out, created, err := h.Store.PutSet(r.Context(), writer(r, in.UpdatedAt), sessionID, set)
	if err != nil {
		return err
	}
	WriteJSON(w, r, createdOrOK(created), setFrom(out))
	return nil
}

func (h *handlers) deleteSet(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	setID, err := pathID(r, "setId")
	if err != nil {
		return err
	}
	if err := h.Store.DeleteSet(r.Context(), writer(r, nil), sessionID, setID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *handlers) reorder(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	var in reorderIn
	if err := h.body(w, r, "ReorderRequest", &in); err != nil {
		return err
	}
	sets := make([]store.SetOrder, len(in.Sets))
	for i, s := range in.Sets {
		sets[i] = store.SetOrder{BlockID: s.BlockID, SetIDs: s.SetIDs}
	}
	out, err := h.Store.Reorder(r.Context(), writer(r, nil), sessionID, in.Blocks, sets)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, sessionFrom(out))
	return nil
}

func (h *handlers) lastSet(w http.ResponseWriter, r *http.Request) error {
	exerciseID, err := pathID(r, "exerciseId")
	if err != nil {
		return err
	}
	ls, err := h.Store.LastSetWithExercise(r.Context(), principalFrom(r.Context()).UserID, exerciseID)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, lastSetOut{
		SessionID: ls.SessionID, LocalDate: ls.LocalDate.Format(dateLayout), Set: setFrom(ls.Set),
	})
	return nil
}

func createdOrOK(created bool) int {
	if created {
		return http.StatusCreated
	}
	return http.StatusOK
}
