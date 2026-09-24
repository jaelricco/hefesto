package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// Store is the persistence layer's entry point.
type Store struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

// New wraps a pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: dbgen.New(pool)}
}

// Writer identifies who is writing: the user every row belongs to, and the
// device that becomes the rows' client_id (nil when the token has none).
type Writer struct {
	UserID   uuid.UUID
	DeviceID *uuid.UUID
	// At is the client's clock for this change, recorded as updated_at.
	At time.Time
}

// tx runs fn in a transaction with sibling-order constraints deferred, so a
// write may move several siblings through each other's positions.
func (s *Store) tx(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
			return fmt.Errorf("deferring constraints: %w", err)
		}
		return fn(dbgen.New(tx))
	})
	return translate(err)
}

// ----------------------------------------------------------------- sessions

// CreateSession inserts a new draft session.
func (s *Store) CreateSession(ctx context.Context, w Writer, in training.Session) (training.Session, error) {
	var out training.Session
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if in.TemplateID != nil {
			ok, err := q.TemplateBelongsToUser(ctx, dbgen.TemplateBelongsToUserParams{ID: *in.TemplateID, UserID: w.UserID})
			if err != nil {
				return fmt.Errorf("checking template: %w", err)
			}
			if !ok {
				return training.FieldErrors{"/template_id": "no such template"}.Err()
			}
		}
		bw, err := numericPtr(in.BodyweightKg)
		if err != nil {
			return err
		}
		row, err := q.InsertSession(ctx, dbgen.InsertSessionParams{
			ID: in.ID, UserID: w.UserID, StartedAt: in.StartedAt, Timezone: in.Timezone,
			LocalDate: dateOf(in.LocalDate), Title: in.Title, Notes: in.Notes, BodyweightKg: bw,
			IsRestDay: in.IsRestDay, TemplateID: in.TemplateID, ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAlreadyExists
		}
		if err != nil {
			return fmt.Errorf("inserting session: %w", err)
		}
		out = sessionFromRow(row)
		return nil
	})
	return out, err
}

// GetSession returns a live session with its whole tree.
func (s *Store) GetSession(ctx context.Context, userID, id uuid.UUID) (training.Session, error) {
	row, err := s.q.GetSession(ctx, dbgen.GetSessionParams{ID: id, UserID: userID})
	if err != nil {
		return training.Session{}, translate(err)
	}
	return loadTree(ctx, s.q, userID, sessionFromRow(row))
}

// UpdateSession applies edit to the current state of a session under a row
// lock and saves the result. edit validates; its error aborts the update.
func (s *Store) UpdateSession(ctx context.Context, w Writer, id uuid.UUID, edit func(*training.Session) error) (training.Session, error) {
	var out training.Session
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		row, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: id, UserID: w.UserID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		sess := sessionFromRow(row)
		if err := edit(&sess); err != nil {
			return err
		}
		bw, err := numericPtr(sess.BodyweightKg)
		if err != nil {
			return err
		}
		row, err = q.UpdateSession(ctx, dbgen.UpdateSessionParams{
			ID: id, UserID: w.UserID, StartedAt: sess.StartedAt, EndedAt: sess.EndedAt,
			Timezone: sess.Timezone, LocalDate: dateOf(sess.LocalDate), Title: sess.Title,
			Notes: sess.Notes, PerceivedFatigue: int16Ptr(sess.PerceivedFatigue), BodyweightKg: bw,
			IsRestDay: sess.IsRestDay, Status: sess.Status, ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if err != nil {
			return fmt.Errorf("updating session: %w", err)
		}
		out, err = loadTree(ctx, q, w.UserID, sessionFromRow(row))
		return err
	})
	return out, err
}

// DeleteSession tombstones a session and everything in it.
func (s *Store) DeleteSession(ctx context.Context, w Writer, id uuid.UUID) error {
	return s.tx(ctx, func(q *dbgen.Queries) error {
		n, err := q.SoftDeleteSession(ctx, dbgen.SoftDeleteSessionParams{ID: id, UserID: w.UserID, ClientID: w.DeviceID, UpdatedAt: w.At})
		if err != nil {
			return fmt.Errorf("deleting session: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		return deleteChildren(ctx, q, w, id, nil)
	})
}

// deleteChildren tombstones the elements, sets and blocks of a session, or of
// one block of it.
func deleteChildren(ctx context.Context, q *dbgen.Queries, w Writer, sessionID uuid.UUID, blockID *uuid.UUID) error {
	if err := q.SoftDeleteSetElements(ctx, dbgen.SoftDeleteSetElementsParams{
		ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID,
		BlockID: blockID, Keep: []uuid.UUID{},
	}); err != nil {
		return fmt.Errorf("deleting elements: %w", err)
	}
	if err := q.SoftDeleteSetEntries(ctx, dbgen.SoftDeleteSetEntriesParams{
		ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID, BlockID: blockID,
	}); err != nil {
		return fmt.Errorf("deleting sets: %w", err)
	}
	if err := q.SoftDeleteBlocksOfSession(ctx, dbgen.SoftDeleteBlocksOfSessionParams{
		ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID, OnlyID: blockID,
	}); err != nil {
		return fmt.Errorf("deleting blocks: %w", err)
	}
	return nil
}

// SessionSummary is a row of the session list.
type SessionSummary struct {
	ID         uuid.UUID
	StartedAt  time.Time
	EndedAt    *time.Time
	Timezone   string
	LocalDate  time.Time
	Title      string
	Status     string
	IsRestDay  bool
	BlockCount int
	SetCount   int
}

// SessionFilter narrows the list; the cursor is the last row already seen.
type SessionFilter struct {
	From, To        *time.Time
	CursorStartedAt *time.Time
	CursorID        *uuid.UUID
	Limit           int
}

// ListSessions returns one page, newest first.
func (s *Store) ListSessions(ctx context.Context, userID uuid.UUID, f SessionFilter) ([]SessionSummary, error) {
	rows, err := s.q.ListSessions(ctx, dbgen.ListSessionsParams{
		UserID: userID, FromDate: datePtr(f.From), ToDate: datePtr(f.To),
		CursorStartedAt: f.CursorStartedAt, CursorID: f.CursorID,
		PageLimit: int32(f.Limit), //nolint:gosec // bounded by the handler
	})
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	out := make([]SessionSummary, len(rows))
	for i, r := range rows {
		out[i] = SessionSummary{
			ID: r.ID, StartedAt: r.StartedAt, EndedAt: r.EndedAt, Timezone: r.Timezone,
			LocalDate: r.LocalDate.Time, Title: r.Title, Status: r.Status, IsRestDay: r.IsRestDay,
			BlockCount: int(r.BlockCount), SetCount: int(r.SetCount),
		}
	}
	return out, nil
}

// ------------------------------------------------------------------- blocks

// PutBlock creates or replaces a block and reports whether it was created.
func (s *Store) PutBlock(ctx context.Context, w Writer, sessionID uuid.UUID, b training.Block) (training.Block, bool, error) {
	var (
		out     training.Block
		created bool
	)
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if _, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID}); err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		row, err := q.UpsertBlock(ctx, dbgen.UpsertBlockParams{
			ID: b.ID, UserID: w.UserID, SessionID: sessionID,
			OrderIndex: int32(b.OrderIndex), //nolint:gosec // bounded by the API schema
			Kind:       b.Kind, RoundsPlanned: int16Ptr(b.RoundsPlanned), RoundsDone: int16Ptr(b.RoundsDone),
			IntervalS: int32Ptr(b.IntervalS), Notes: b.Notes, ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound // the id is a tombstone, or belongs to another session
		}
		if err != nil {
			return fmt.Errorf("writing block: %w", err)
		}
		created = row.Inserted
		out, err = blockWithSets(ctx, q, w.UserID, sessionID, b.ID)
		return err
	})
	return out, created, err
}

// DeleteBlock tombstones a block and its sets.
func (s *Store) DeleteBlock(ctx context.Context, w Writer, sessionID, blockID uuid.UUID) error {
	return s.tx(ctx, func(q *dbgen.Queries) error {
		if _, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID}); err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		b, err := q.GetBlock(ctx, dbgen.GetBlockParams{ID: blockID, UserID: w.UserID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		if b.SessionID != sessionID || b.DeletedAt != nil {
			return ErrNotFound
		}
		return deleteChildren(ctx, q, w, sessionID, &blockID)
	})
}

// --------------------------------------------------------------------- sets

// PutSet creates or replaces a set entry and its complete, ordered list of
// elements. The set must already have passed training.ValidateSet.
func (s *Store) PutSet(ctx context.Context, w Writer, sessionID uuid.UUID, set training.SetEntry) (training.SetEntry, bool, error) {
	var (
		out     training.SetEntry
		created bool
	)
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if _, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID}); err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		if err := checkSetReferences(ctx, q, w.UserID, sessionID, set); err != nil {
			return err
		}

		rpe, err := numericPtr(set.RPE)
		if err != nil {
			return err
		}
		row, err := q.UpsertSetEntry(ctx, dbgen.UpsertSetEntryParams{
			ID: set.ID, UserID: w.UserID, SessionID: sessionID, BlockID: set.BlockID,
			OrderIndex: int32(set.OrderIndex), //nolint:gosec // bounded by the API schema
			RoundIndex: int16Ptr(set.RoundIndex), Kind: set.Kind, IsPlanned: set.IsPlanned,
			RestAfterPlannedS: int32Ptr(set.RestAfterPlannedS), RestAfterActualS: int32Ptr(set.RestAfterActualS),
			Rpe: rpe, Rir: int16Ptr(set.RIR), CompletedAt: set.CompletedAt, Notes: set.Notes,
			ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound // the id is a tombstone, or belongs to another session
		}
		if err != nil {
			return fmt.Errorf("writing set: %w", err)
		}
		created = row.Inserted

		keep := make([]uuid.UUID, len(set.Elements))
		for i, e := range set.Elements {
			keep[i] = e.ID
		}
		// Elements dropped from the list go first, so their positions are free.
		if err := q.SoftDeleteSetElements(ctx, dbgen.SoftDeleteSetElementsParams{
			ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID,
			SetEntryID: &set.ID, Keep: keep,
		}); err != nil {
			return fmt.Errorf("removing dropped elements: %w", err)
		}
		for i, e := range set.Elements {
			if err := writeElement(ctx, q, w, sessionID, set.ID, i, e); err != nil {
				return err
			}
		}

		out, err = setByID(ctx, q, w.UserID, sessionID, set.ID)
		return err
	})
	return out, created, err
}

func writeElement(ctx context.Context, q *dbgen.Queries, w Writer, sessionID, setID uuid.UUID, i int, e training.Element) error {
	p := fmt.Sprintf("/elements/%d", i)
	hold, err := numericPtr(e.HoldSeconds)
	if err != nil {
		return err
	}
	dist, err := numericPtr(e.DistanceM)
	if err != nil {
		return err
	}
	load, err := numeric(e.LoadKg)
	if err != nil {
		return err
	}
	_, err = q.UpsertSetElement(ctx, dbgen.UpsertSetElementParams{
		ID: e.ID, UserID: w.UserID, SessionID: sessionID, SetEntryID: setID,
		OrderIndex: int32(e.OrderIndex), //nolint:gosec // bounded by MaxElementsPerSet
		ExerciseID: e.ExerciseID, Measure: string(e.Measure), Reps: int32Ptr(e.Reps),
		HoldSeconds: hold, DistanceM: dist, Tempo: e.Tempo, LoadKg: load,
		IsEccentricOnly: e.IsEccentricOnly, IsPartialRom: e.IsPartialROM, RomNote: e.ROMNote,
		FormQuality: int16Ptr(e.FormQuality), Failed: e.Failed, AssistanceClass: string(e.AssistanceClass),
		ClientID: w.DeviceID, UpdatedAt: w.At,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return training.FieldErrors{p + "/id": "this element belongs to another set, or was deleted"}.Err()
	}
	if err != nil {
		return fmt.Errorf("writing element: %w", err)
	}

	var keep *uuid.UUID
	if a := e.Assistance; a != nil {
		est, err := numericPtr(a.EstimatedAssistKg)
		if err != nil {
			return err
		}
		_, err = q.UpsertAssistance(ctx, dbgen.UpsertAssistanceParams{
			ID: a.ID, UserID: w.UserID, SetElementID: e.ID, Type: a.Type, BandID: a.BandID,
			BandCount: int16(a.BandCount), //nolint:gosec // bounded by the API schema
			Anchor:    a.Anchor, EstimatedAssistKg: est, Note: a.Note, ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return training.FieldErrors{p + "/assistance/id": "this assistance belongs to another element, or was deleted"}.Err()
		}
		if err != nil {
			return fmt.Errorf("writing assistance: %w", err)
		}
		keep = &a.ID
	}
	if err := q.SoftDeleteOtherAssistance(ctx, dbgen.SoftDeleteOtherAssistanceParams{
		ClientID: w.DeviceID, UpdatedAt: w.At, SetElementID: e.ID, UserID: w.UserID, KeepID: keep,
	}); err != nil {
		return fmt.Errorf("removing replaced assistance: %w", err)
	}
	return nil
}

// checkSetReferences verifies what the schema cannot: the block is a live
// block of this session, every exercise exists (and a retired one is only
// kept, never newly chosen), and every band is one the user may use.
func checkSetReferences(ctx context.Context, q *dbgen.Queries, userID, sessionID uuid.UUID, set training.SetEntry) error {
	errs := training.FieldErrors{}

	b, err := q.GetBlock(ctx, dbgen.GetBlockParams{ID: set.BlockID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		errs["/block_id"] = "no such block in this session"
	case err != nil:
		return fmt.Errorf("reading block: %w", err)
	case b.SessionID != sessionID || b.DeletedAt != nil:
		errs["/block_id"] = "no such block in this session"
	}

	existing, err := q.ListElementsOfSet(ctx, dbgen.ListElementsOfSetParams{SetEntryID: set.ID, UserID: userID})
	if err != nil {
		return fmt.Errorf("reading elements: %w", err)
	}
	kept := map[uuid.UUID]uuid.UUID{} // element id -> exercise id as stored
	for _, e := range existing {
		kept[e.ID] = e.ExerciseID
	}

	exIDs := make([]uuid.UUID, 0, len(set.Elements))
	var bandIDs []uuid.UUID
	for _, e := range set.Elements {
		exIDs = append(exIDs, e.ExerciseID)
		if e.Assistance != nil && e.Assistance.BandID != nil {
			bandIDs = append(bandIDs, *e.Assistance.BandID)
		}
	}
	statuses, err := q.ExerciseStatuses(ctx, exIDs)
	if err != nil {
		return fmt.Errorf("reading exercises: %w", err)
	}
	status := make(map[uuid.UUID]string, len(statuses))
	for _, s := range statuses {
		status[s.ID] = s.Status
	}
	visible := map[uuid.UUID]bool{}
	if len(bandIDs) > 0 {
		ids, err := q.VisibleBandIDs(ctx, dbgen.VisibleBandIDsParams{Ids: bandIDs, UserID: &userID})
		if err != nil {
			return fmt.Errorf("reading bands: %w", err)
		}
		for _, id := range ids {
			visible[id] = true
		}
	}

	for i, e := range set.Elements {
		p := fmt.Sprintf("/elements/%d", i)
		switch st, ok := status[e.ExerciseID]; {
		case !ok:
			errs[p+"/exercise_id"] = "no such exercise"
		case st == "retired" && kept[e.ID] != e.ExerciseID:
			errs[p+"/exercise_id"] = "this exercise is retired and cannot be newly logged"
		}
		if a := e.Assistance; a != nil && a.BandID != nil && !visible[*a.BandID] {
			errs[p+"/assistance/band_id"] = "no such band"
		}
	}
	return errs.Err()
}

// DeleteSet tombstones a set and its elements.
func (s *Store) DeleteSet(ctx context.Context, w Writer, sessionID, setID uuid.UUID) error {
	return s.tx(ctx, func(q *dbgen.Queries) error {
		if _, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID}); err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		e, err := q.GetSetEntry(ctx, dbgen.GetSetEntryParams{ID: setID, UserID: w.UserID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		if e.SessionID != sessionID || e.DeletedAt != nil {
			return ErrNotFound
		}
		if err := q.SoftDeleteSetElements(ctx, dbgen.SoftDeleteSetElementsParams{
			ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID,
			SetEntryID: &setID, Keep: []uuid.UUID{},
		}); err != nil {
			return fmt.Errorf("deleting elements: %w", err)
		}
		if err := q.SoftDeleteSetEntries(ctx, dbgen.SoftDeleteSetEntriesParams{
			ClientID: w.DeviceID, UpdatedAt: w.At, SessionID: sessionID, UserID: w.UserID, OnlyID: &setID,
		}); err != nil {
			return fmt.Errorf("deleting set: %w", err)
		}
		return nil
	})
}

// ------------------------------------------------------------------ reorder

// SetOrder is the new order of one block's sets.
type SetOrder struct {
	BlockID uuid.UUID
	SetIDs  []uuid.UUID
}

// Reorder renumbers blocks and/or the sets of named blocks in one
// transaction. Each list must name exactly the live children it orders.
func (s *Store) Reorder(ctx context.Context, w Writer, sessionID uuid.UUID, blocks []uuid.UUID, sets []SetOrder) (training.Session, error) {
	var out training.Session
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		row, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		liveBlocks, err := q.ListBlocks(ctx, dbgen.ListBlocksParams{SessionID: sessionID, UserID: w.UserID})
		if err != nil {
			return fmt.Errorf("reading blocks: %w", err)
		}
		liveSets, err := q.ListSetEntries(ctx, dbgen.ListSetEntriesParams{SessionID: sessionID, UserID: w.UserID})
		if err != nil {
			return fmt.Errorf("reading sets: %w", err)
		}

		if blocks != nil {
			current := make([]uuid.UUID, len(liveBlocks))
			for i, b := range liveBlocks {
				current[i] = b.ID
			}
			if err := training.ValidateOrder("/blocks", blocks, current); err != nil {
				return err //nolint:wrapcheck // domain validation error
			}
			if err := q.SetBlockOrder(ctx, dbgen.SetBlockOrderParams{
				Ids: blocks, UserID: w.UserID, SessionID: sessionID, ClientID: w.DeviceID, UpdatedAt: w.At,
			}); err != nil {
				return fmt.Errorf("reordering blocks: %w", err)
			}
		}

		isLiveBlock := map[uuid.UUID]bool{}
		for _, b := range liveBlocks {
			isLiveBlock[b.ID] = true
		}
		for i, so := range sets {
			p := fmt.Sprintf("/sets/%d", i)
			if !isLiveBlock[so.BlockID] {
				return training.FieldErrors{p + "/block_id": "no such block in this session"}.Err()
			}
			var current []uuid.UUID
			for _, e := range liveSets {
				if e.BlockID == so.BlockID {
					current = append(current, e.ID)
				}
			}
			if err := training.ValidateOrder(p+"/set_ids", so.SetIDs, current); err != nil {
				return err //nolint:wrapcheck // domain validation error
			}
			if err := q.SetSetOrder(ctx, dbgen.SetSetOrderParams{
				Ids: so.SetIDs, UserID: w.UserID, BlockID: so.BlockID, ClientID: w.DeviceID, UpdatedAt: w.At,
			}); err != nil {
				return fmt.Errorf("reordering sets: %w", err)
			}
		}
		out, err = loadTree(ctx, q, w.UserID, sessionFromRow(row))
		return err
	})
	return out, err
}

// ----------------------------------------------------------------- last set

// LastSet is the most recent performed set containing an exercise.
type LastSet struct {
	SessionID uuid.UUID
	LocalDate time.Time
	Set       training.SetEntry
}

// LastSetWithExercise finds the most recent performed set that includes the
// exercise, returned whole (a combo is repeated as a combo).
func (s *Store) LastSetWithExercise(ctx context.Context, userID, exerciseID uuid.UUID) (LastSet, error) {
	row, err := s.q.LastSetWithExercise(ctx, dbgen.LastSetWithExerciseParams{UserID: userID, ExerciseID: exerciseID})
	if err != nil {
		return LastSet{}, translate(err)
	}
	set, err := setByID(ctx, s.q, userID, row.SessionID, row.SetEntryID)
	if err != nil {
		return LastSet{}, err
	}
	return LastSet{SessionID: row.SessionID, LocalDate: row.LocalDate.Time, Set: set}, nil
}

// --------------------------------------------------------------------- read

// loadTree fills in a session's live blocks, sets, elements and assistance.
func loadTree(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, sess training.Session) (training.Session, error) {
	blocks, err := q.ListBlocks(ctx, dbgen.ListBlocksParams{SessionID: sess.ID, UserID: userID})
	if err != nil {
		return sess, fmt.Errorf("reading blocks: %w", err)
	}
	entries, err := q.ListSetEntries(ctx, dbgen.ListSetEntriesParams{SessionID: sess.ID, UserID: userID})
	if err != nil {
		return sess, fmt.Errorf("reading sets: %w", err)
	}
	elements, err := q.ListSetElements(ctx, dbgen.ListSetElementsParams{SessionID: sess.ID, UserID: userID})
	if err != nil {
		return sess, fmt.Errorf("reading elements: %w", err)
	}
	assists, err := q.ListAssistance(ctx, dbgen.ListAssistanceParams{SessionID: sess.ID, UserID: userID})
	if err != nil {
		return sess, fmt.Errorf("reading assistance: %w", err)
	}

	byElement := make(map[uuid.UUID]*training.Assistance, len(assists))
	for _, a := range assists {
		byElement[a.SetElementID] = &training.Assistance{
			ID: a.ID, Type: a.Type, BandID: a.BandID, BandCount: int(a.BandCount), Anchor: a.Anchor,
			EstimatedAssistKg: numericPtrToFloat(a.EstimatedAssistKg), Note: a.Note,
		}
	}
	bySet := map[uuid.UUID][]training.Element{}
	for _, e := range elements {
		bySet[e.SetEntryID] = append(bySet[e.SetEntryID], training.Element{
			ID: e.ID, OrderIndex: int(e.OrderIndex), ExerciseID: e.ExerciseID, Measure: training.Measure(e.Measure),
			Reps: intPtr32(e.Reps), HoldSeconds: numericPtrToFloat(e.HoldSeconds), DistanceM: numericPtrToFloat(e.DistanceM),
			Tempo: e.Tempo, LoadKg: numericToFloat(e.LoadKg), IsEccentricOnly: e.IsEccentricOnly,
			IsPartialROM: e.IsPartialRom, ROMNote: e.RomNote, FormQuality: intPtr16(e.FormQuality), Failed: e.Failed,
			AssistanceClass: training.AssistanceClass(e.AssistanceClass), Assistance: byElement[e.ID],
		})
	}
	byBlock := map[uuid.UUID][]training.SetEntry{}
	for _, e := range entries {
		byBlock[e.BlockID] = append(byBlock[e.BlockID], training.SetEntry{
			ID: e.ID, SessionID: e.SessionID, BlockID: e.BlockID, OrderIndex: int(e.OrderIndex),
			RoundIndex: intPtr16(e.RoundIndex), Kind: e.Kind, IsPlanned: e.IsPlanned,
			RestAfterPlannedS: intPtr32(e.RestAfterPlannedS), RestAfterActualS: intPtr32(e.RestAfterActualS),
			RPE: numericPtrToFloat(e.Rpe), RIR: intPtr16(e.Rir), CompletedAt: e.CompletedAt, Notes: e.Notes,
			UpdatedAt: e.UpdatedAt, Elements: nonNilElements(bySet[e.ID]),
		})
	}
	sess.Blocks = make([]training.Block, len(blocks))
	for i, b := range blocks {
		sess.Blocks[i] = training.Block{
			ID: b.ID, SessionID: b.SessionID, OrderIndex: int(b.OrderIndex), Kind: b.Kind,
			RoundsPlanned: intPtr16(b.RoundsPlanned), RoundsDone: intPtr16(b.RoundsDone),
			IntervalS: intPtr32(b.IntervalS), Notes: b.Notes, UpdatedAt: b.UpdatedAt,
			Sets: nonNilSets(byBlock[b.ID]),
		}
	}
	return sess, nil
}

func blockWithSets(ctx context.Context, q *dbgen.Queries, userID, sessionID, blockID uuid.UUID) (training.Block, error) {
	tree, err := loadTree(ctx, q, userID, training.Session{ID: sessionID})
	if err != nil {
		return training.Block{}, err
	}
	for _, b := range tree.Blocks {
		if b.ID == blockID {
			return b, nil
		}
	}
	return training.Block{}, ErrNotFound
}

func setByID(ctx context.Context, q *dbgen.Queries, userID, sessionID, setID uuid.UUID) (training.SetEntry, error) {
	tree, err := loadTree(ctx, q, userID, training.Session{ID: sessionID})
	if err != nil {
		return training.SetEntry{}, err
	}
	for _, b := range tree.Blocks {
		for _, s := range b.Sets {
			if s.ID == setID {
				return s, nil
			}
		}
	}
	return training.SetEntry{}, ErrNotFound
}

func sessionFromRow(r dbgen.WorkoutSession) training.Session {
	return training.Session{
		ID: r.ID, StartedAt: r.StartedAt, EndedAt: r.EndedAt, Timezone: r.Timezone,
		LocalDate: r.LocalDate.Time, Title: r.Title, Notes: r.Notes,
		PerceivedFatigue: intPtr16(r.PerceivedFatigue), BodyweightKg: numericPtrToFloat(r.BodyweightKg),
		Status: r.Status, IsRestDay: r.IsRestDay, TemplateID: r.TemplateID, CompletedAt: r.CompletedAt,
		UpdatedAt: r.UpdatedAt, ServerUpdatedAt: r.ServerUpdatedAt, Blocks: []training.Block{},
	}
}

func nonNilElements(e []training.Element) []training.Element {
	if e == nil {
		return []training.Element{}
	}
	return e
}

func nonNilSets(s []training.SetEntry) []training.SetEntry {
	if s == nil {
		return []training.SetEntry{}
	}
	return s
}
