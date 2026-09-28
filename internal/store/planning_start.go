package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/planning"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// StartSession writes a planned session into the training log (spec
// §10.2): a draft session pointing at the planned session, its blocks in
// order, and one planned set entry with one element per planned set. The
// planned session row is locked, so two starts of one session cannot both
// write; the planner lock already serialises the user's calls.
func (t *plannerTx) StartSession(ctx context.Context, userID uuid.UUID, in planning.SessionStart) (uuid.UUID, bool, error) {
	if err := t.check(userID); err != nil {
		return uuid.Nil, false, err
	}
	q := t.q
	ps, err := q.LockActivePlannedSession(ctx, dbgen.LockActivePlannedSessionParams{ID: in.PlannedSessionID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, fmt.Errorf("planned session %s: %w", in.PlannedSessionID, planning.ErrNotFound)
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("locking planned session: %w", err)
	}
	if id := ps.WorkoutSessionID; id != nil {
		prev, err := q.GetSessionAny(ctx, dbgen.GetSessionAnyParams{ID: *id, UserID: userID})
		switch {
		case err == nil && prev.DeletedAt == nil:
			return *id, false, nil
		case err != nil && !errors.Is(err, pgx.ErrNoRows):
			return uuid.Nil, false, fmt.Errorf("reading the started session: %w", err)
		}
		// The athlete deleted the started session; the planned one starts anew.
	}

	exercises, err := exerciseIDs(ctx, q, in.Draft)
	if err != nil {
		return uuid.Nil, false, err
	}
	w := Writer{UserID: userID, DeviceID: in.DeviceID, At: in.At}
	row, err := q.InsertPlannedWorkoutSession(ctx, dbgen.InsertPlannedWorkoutSessionParams{
		ID: in.SessionID, UserID: userID, StartedAt: in.StartedAt, Timezone: in.Timezone,
		LocalDate: dateOf(in.LocalDate), PlannedSessionID: &in.PlannedSessionID,
		ClientID: w.DeviceID, UpdatedAt: w.At,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, fmt.Errorf("session %s: %w", in.SessionID, planning.ErrSessionIDTaken)
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("inserting session: %w", err)
	}
	for bi, b := range in.Draft.Blocks {
		if err := writeDraftBlock(ctx, q, w, row.ID, bi, b, exercises); err != nil {
			return uuid.Nil, false, fmt.Errorf("block %d: %w", bi, err)
		}
	}
	if err := q.MarkPlannedSessionStarted(ctx, dbgen.MarkPlannedSessionStartedParams{
		ID: in.PlannedSessionID, UserID: userID, WorkoutSessionID: &row.ID,
	}); err != nil {
		return uuid.Nil, false, fmt.Errorf("marking planned session started: %w", err)
	}
	return row.ID, true, nil
}

// DraftSets returns the live sets of a started draft in log order. A
// session that is no live draft (completed, abandoned, deleted) does not
// follow the plan: ok is false.
func (t *plannerTx) DraftSets(ctx context.Context, userID, sessionID uuid.UUID) ([]domain.DraftState, bool, error) {
	if err := t.check(userID); err != nil {
		return nil, false, err
	}
	row, err := lockSession(ctx, t.q, userID, sessionID)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrDeleted) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if row.Status != training.StatusDraft {
		return nil, false, nil
	}
	rows, err := t.q.ListDraftSets(ctx, dbgen.ListDraftSetsParams{SessionID: sessionID, UserID: userID})
	if err != nil {
		return nil, false, fmt.Errorf("reading the draft's sets: %w", err)
	}
	out := make([]domain.DraftState, len(rows))
	for i, r := range rows {
		out[i] = domain.DraftState{ID: r.ID.String(), BlockID: r.BlockID.String(), ItemID: uuidString(r.PlannedItemID), Open: r.Open}
	}
	return out, true, nil
}

// AdjustSession writes an adjustment into a draft (ADR 0017). Removed and
// replaced sets become tombstones, as a deletion by the athlete would; a
// replacement takes the old set's place. A block left without sets goes
// too. The rows carry no device: the server wrote them.
func (t *plannerTx) AdjustSession(ctx context.Context, userID, sessionID uuid.UUID, a domain.Adjustment, at time.Time) error {
	if err := t.check(userID); err != nil {
		return err
	}
	q := t.q
	if _, err := lockSession(ctx, q, userID, sessionID); err != nil {
		return fmt.Errorf("locking the draft: %w", err)
	}
	var added []domain.DraftSet
	for _, r := range a.Replace {
		added = append(added, r.Set)
	}
	for _, ap := range a.Append {
		added = append(added, ap.Set)
	}
	for _, b := range a.Blocks {
		added = append(added, b.Sets...)
	}
	exercises, err := exerciseIDs(ctx, q, domain.SessionDraft{Blocks: []domain.DraftBlock{{Sets: added}}})
	if err != nil {
		return err
	}
	w := Writer{UserID: userID, At: at}
	emptied := map[uuid.UUID]bool{}

	for _, r := range a.Relink {
		id, item, err := parseTwo(r.SetID, r.ItemID)
		if err != nil {
			return err
		}
		if err := q.RelinkPlannedSet(ctx, dbgen.RelinkPlannedSetParams{ID: id, UserID: userID, SessionID: sessionID, PlannedItemID: &item}); err != nil {
			return fmt.Errorf("relinking set %s: %w", id, err)
		}
	}
	for _, sid := range a.Remove {
		prev, ok, err := openSet(ctx, q, userID, sessionID, sid)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := removeSet(ctx, q, w, sessionID, prev.ID); err != nil {
			return err
		}
		emptied[prev.BlockID] = true
	}
	for _, r := range a.Replace {
		prev, ok, err := openSet(ctx, q, userID, sessionID, r.SetID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := removeSet(ctx, q, w, sessionID, prev.ID); err != nil {
			return err
		}
		set := r.Set
		set.Round = intPtr16(prev.RoundIndex)
		if err := writeDraftSet(ctx, q, w, sessionID, prev.BlockID, int(prev.OrderIndex), set, exercises[set.Exercise]); err != nil {
			return fmt.Errorf("replacing set %s: %w", prev.ID, err)
		}
	}
	for _, ap := range a.Append {
		block, err := uuid.Parse(ap.BlockID)
		if err != nil {
			return fmt.Errorf("block id %q: %w", ap.BlockID, err)
		}
		order, err := q.NextSetOrder(ctx, dbgen.NextSetOrderParams{BlockID: block, UserID: userID})
		if err != nil {
			return fmt.Errorf("numbering sets: %w", err)
		}
		if err := writeDraftSet(ctx, q, w, sessionID, block, int(order), ap.Set, exercises[ap.Set.Exercise]); err != nil {
			return fmt.Errorf("appending a set: %w", err)
		}
	}
	for _, b := range a.Blocks {
		order, err := q.NextBlockOrder(ctx, dbgen.NextBlockOrderParams{SessionID: sessionID, UserID: userID})
		if err != nil {
			return fmt.Errorf("numbering blocks: %w", err)
		}
		if err := writeDraftBlock(ctx, q, w, sessionID, int(order), b, exercises); err != nil {
			return fmt.Errorf("adding a block: %w", err)
		}
	}
	for block := range emptied {
		n, err := q.CountLiveSetsOfBlock(ctx, dbgen.CountLiveSetsOfBlockParams{BlockID: block, UserID: userID})
		if err != nil {
			return fmt.Errorf("counting sets: %w", err)
		}
		if n > 0 {
			continue
		}
		if err := q.SoftDeleteBlocksOfSession(ctx, dbgen.SoftDeleteBlocksOfSessionParams{
			UpdatedAt: w.At, SessionID: sessionID, UserID: userID, OnlyID: &block,
		}); err != nil {
			return fmt.Errorf("removing an empty block: %w", err)
		}
	}
	return nil
}

// openSet reads a set the adjustment changes. A set that is no longer an
// open planned set of the session is left alone: ok is false. The session
// lock makes that impossible within one adjustment; the check keeps a
// performed set safe regardless.
func openSet(ctx context.Context, q *dbgen.Queries, userID, sessionID uuid.UUID, id string) (dbgen.SetEntry, bool, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return dbgen.SetEntry{}, false, fmt.Errorf("set id %q: %w", id, err)
	}
	e, err := q.GetSetEntry(ctx, dbgen.GetSetEntryParams{ID: sid, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return e, false, nil
	}
	if err != nil {
		return e, false, fmt.Errorf("reading set %s: %w", sid, err)
	}
	ok := e.SessionID == sessionID && e.DeletedAt == nil && e.IsPlanned && e.CompletedAt == nil
	return e, ok, nil
}

// removeSet tombstones a set and its elements, as DeleteSet does.
func removeSet(ctx context.Context, q *dbgen.Queries, w Writer, sessionID, setID uuid.UUID) error {
	if err := q.SoftDeleteSetElements(ctx, dbgen.SoftDeleteSetElementsParams{
		UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID, SetEntryID: &setID, Keep: []uuid.UUID{},
	}); err != nil {
		return fmt.Errorf("removing the elements of set %s: %w", setID, err)
	}
	if err := q.SoftDeleteSetEntries(ctx, dbgen.SoftDeleteSetEntriesParams{
		UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID, OnlyID: &setID,
	}); err != nil {
		return fmt.Errorf("removing set %s: %w", setID, err)
	}
	return nil
}

func parseTwo(a, b string) (uuid.UUID, uuid.UUID, error) {
	x, err := uuid.Parse(a)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("id %q: %w", a, err)
	}
	y, err := uuid.Parse(b)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("id %q: %w", b, err)
	}
	return x, y, nil
}

// exerciseIDs resolves the draft's exercise slugs to the log's catalogue.
// The planner's exercises are content rows (U-50); one that is missing
// means the content and the knowledge base disagree, and the planner is
// unavailable until they are seeded together.
func exerciseIDs(ctx context.Context, q *dbgen.Queries, d domain.SessionDraft) (map[string]uuid.UUID, error) {
	var slugs []string
	seen := map[string]bool{}
	for _, b := range d.Blocks {
		for _, s := range b.Sets {
			if !seen[s.Exercise] {
				seen[s.Exercise] = true
				slugs = append(slugs, s.Exercise)
			}
		}
	}
	rows, err := q.ExerciseIDsBySlug(ctx, slugs)
	if err != nil {
		return nil, fmt.Errorf("reading exercises: %w", err)
	}
	ids := make(map[string]uuid.UUID, len(rows))
	for _, r := range rows {
		ids[r.Slug] = r.ID
	}
	for _, slug := range slugs {
		if _, ok := ids[slug]; !ok {
			return nil, fmt.Errorf("exercise %q is not in the catalogue: %w", slug, planning.ErrUnavailable)
		}
	}
	return ids, nil
}

func writeDraftBlock(ctx context.Context, q *dbgen.Queries, w Writer, sessionID uuid.UUID, order int, b domain.DraftBlock, exercises map[string]uuid.UUID) error {
	blockID, err := uuid.Parse(b.ID)
	if err != nil {
		return fmt.Errorf("block id %q: %w", b.ID, err)
	}
	var rounds *int
	if b.Kind == domain.DraftSuperset {
		n := 0
		for _, s := range b.Sets {
			n = max(n, deref(s.Round)+1)
		}
		rounds = &n
	}
	if _, err := q.UpsertBlock(ctx, dbgen.UpsertBlockParams{
		ID: blockID, UserID: w.UserID, SessionID: sessionID,
		OrderIndex: int32(order), //nolint:gosec // a session has a handful of blocks
		Kind:       b.Kind, RoundsPlanned: int16Ptr(rounds), ClientID: w.DeviceID, UpdatedAt: w.At,
	}); err != nil {
		return fmt.Errorf("writing block: %w", err)
	}
	for i, s := range b.Sets {
		if err := writeDraftSet(ctx, q, w, sessionID, blockID, i, s, exercises[s.Exercise]); err != nil {
			return fmt.Errorf("set %d: %w", i, err)
		}
	}
	return nil
}

func writeDraftSet(ctx context.Context, q *dbgen.Queries, w Writer, sessionID, blockID uuid.UUID, order int, s domain.DraftSet, exercise uuid.UUID) error {
	setID, err := uuid.Parse(s.ID)
	if err != nil {
		return fmt.Errorf("set id %q: %w", s.ID, err)
	}
	elementID, err := uuid.Parse(s.ElementID)
	if err != nil {
		return fmt.Errorf("element id %q: %w", s.ElementID, err)
	}
	// Plans generated before items had IDs start with no link to the item.
	item, err := parseOptionalUUID(s.ItemID)
	if err != nil {
		return fmt.Errorf("plan item id: %w", err)
	}
	rest := s.RestS
	if err := q.InsertPlannedSetEntry(ctx, dbgen.InsertPlannedSetEntryParams{
		ID: setID, UserID: w.UserID, SessionID: sessionID, BlockID: blockID,
		OrderIndex: int32(order), //nolint:gosec // a block has a few dozen sets at most
		RoundIndex: int16Ptr(s.Round), Kind: s.Kind, RestAfterPlannedS: int32Ptr(&rest), Rir: int16Ptr(s.RIR),
		PlannedItemID: item, ClientID: w.DeviceID, UpdatedAt: w.At,
	}); err != nil {
		return fmt.Errorf("writing set: %w", err)
	}
	e := training.Element{ID: elementID, ExerciseID: exercise, Measure: training.Measure(s.Measure), Reps: s.Reps,
		LoadKg: s.LoadKg, IsEccentricOnly: s.Eccentric}
	if s.HoldS != nil {
		hold := float64(*s.HoldS)
		e.HoldSeconds = &hold
	}
	e.AssistanceClass = training.ClassOf(e)
	hold, err := numericPtr(e.HoldSeconds)
	if err != nil {
		return err
	}
	load, err := numeric(e.LoadKg)
	if err != nil {
		return err
	}
	if _, err := q.UpsertSetElement(ctx, dbgen.UpsertSetElementParams{
		ID: e.ID, UserID: w.UserID, SessionID: sessionID, SetEntryID: setID, ExerciseID: e.ExerciseID,
		Measure: string(e.Measure), Reps: int32Ptr(e.Reps), HoldSeconds: hold, LoadKg: load,
		IsEccentricOnly: e.IsEccentricOnly, AssistanceClass: string(e.AssistanceClass),
		ClientID: w.DeviceID, UpdatedAt: w.At,
	}); err != nil {
		return fmt.Errorf("writing element: %w", err)
	}
	return nil
}
