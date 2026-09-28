package store

import (
	"context"
	"errors"
	"fmt"

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
		LocalDate: dateOf(in.LocalDate), Title: in.Draft.Title, PlannedSessionID: &in.PlannedSessionID,
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
