package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// CompletedSession reads a completed log session as the planner's history
// records it (ADR 0018). Only performed sets count, in the order performed.
func (t *plannerTx) CompletedSession(ctx context.Context, userID, sessionID uuid.UUID) (domain.LoggedSession, bool, error) {
	if err := t.check(userID); err != nil {
		return domain.LoggedSession{}, false, err
	}
	row, err := t.q.GetCompletedSessionForPlanner(ctx, dbgen.GetCompletedSessionForPlannerParams{ID: sessionID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LoggedSession{}, false, nil
	}
	if err != nil {
		return domain.LoggedSession{}, false, fmt.Errorf("reading the completed session: %w", err)
	}
	if row.IsRestDay {
		return domain.LoggedSession{}, false, nil
	}
	els, err := t.q.ListPerformedElements(ctx, dbgen.ListPerformedElementsParams{SessionID: sessionID, UserID: userID})
	if err != nil {
		return domain.LoggedSession{}, false, fmt.Errorf("reading the performed sets: %w", err)
	}
	sess := domain.LoggedSession{ID: sessionID.String(), Date: row.LocalDate.Time.UTC(), Deload: row.Deload}
	if f := row.PerceivedFatigue; f != nil {
		v := float64(*f)
		sess.Fatigue = &v
	}
	for _, e := range els {
		if set, ok := loggedSet(e); ok {
			sess.Sets = append(sess.Sets, set)
		}
	}
	return sess, true, nil
}

// loggedSet turns a performed element into a set of the planner's history.
// Only what the planner can compare counts (ADR 0018): reps or seconds,
// without assistance or with a band. Other assistance (a partner, a
// machine, ...) is no measure of either capacity, and distances have none.
// The reserve is the logged RIR of a rep set; the log keeps none for holds.
func loggedSet(e dbgen.ListPerformedElementsRow) (domain.LoggedSet, bool) {
	var assist string
	switch e.Assistance {
	case "none":
		assist = domain.AssistNone
	case "band":
		assist = domain.AssistBand
	default:
		return domain.LoggedSet{}, false
	}
	set := domain.LoggedSet{ID: e.ID.String(), Exercise: e.Exercise, Kind: e.Kind, Assist: assist,
		LoadKg: numericToFloat(e.LoadKg), Failed: e.Failed, Partial: e.IsPartialRom, Eccentric: e.IsEccentricOnly}
	switch e.Measure {
	case domain.MeasureReps:
		if e.Reps == nil {
			return domain.LoggedSet{}, false
		}
		set.Value = float64(*e.Reps)
		if e.Rir != nil {
			r := float64(*e.Rir)
			set.Reserve = &r
		}
	case domain.MeasureHold:
		hold := numericPtrToFloat(e.HoldSeconds)
		if hold == nil {
			return domain.LoggedSet{}, false
		}
		set.Value = *hold
	default:
		return domain.LoggedSet{}, false
	}
	if f := e.FormQuality; f != nil {
		v := float64(*f)
		set.Form = &v
	}
	return set, true
}

// MarkCompleted marks the planned sessions of a log session completed, in
// every plan that carries it, and a deload session's day a deload day.
func (t *plannerTx) MarkCompleted(ctx context.Context, userID, sessionID uuid.UUID, deload bool) error {
	if err := t.check(userID); err != nil {
		return err
	}
	if err := t.q.MarkPlannedSessionsCompleted(ctx, dbgen.MarkPlannedSessionsCompletedParams{
		WorkoutSessionID: &sessionID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("marking planned sessions completed: %w", err)
	}
	if deload {
		if err := t.q.MarkDeloadDay(ctx, dbgen.MarkDeloadDayParams{SessionID: sessionID, UserID: userID}); err != nil {
			return fmt.Errorf("marking the deload day: %w", err)
		}
	}
	return nil
}

// PendingCompletions lists completed sessions without a planner decision.
func (t *plannerTx) PendingCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error) {
	if err := t.check(userID); err != nil {
		return nil, err
	}
	ids, err := t.q.ListPendingCompletions(ctx, dbgen.ListPendingCompletionsParams{
		UserID: userID, MaxRows: int32(min(limit, maxPage)), //nolint:gosec // bounded by maxPage
	})
	if err != nil {
		return nil, fmt.Errorf("listing pending completions: %w", err)
	}
	return ids, nil
}
