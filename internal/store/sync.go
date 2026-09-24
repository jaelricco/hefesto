package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// ---------------------------------------------------------------------- pull

// SyncSession is a session row as sync carries it: no tree, tombstones kept.
type SyncSession struct {
	training.Session
	ServerSeq int64
	DeletedAt *time.Time
}

// SyncBlock is a block row without its sets.
type SyncBlock struct {
	training.Block
	ServerSeq int64
	DeletedAt *time.Time
}

// SyncSet is a set with its live elements. ServerSeq is the latest change to
// the set or any of its elements.
type SyncSet struct {
	training.SetEntry
	ServerSeq int64
	DeletedAt *time.Time
}

// Bodyweight is an entry of the bodyweight log.
type Bodyweight struct {
	ID           uuid.UUID
	MeasuredAt   time.Time
	LocalDate    time.Time
	BodyweightKg float64
	Note         string
	UpdatedAt    time.Time
	ServerSeq    int64
	DeletedAt    *time.Time
}

// SyncPage is one page of the change feed.
type SyncPage struct {
	Cursor     int64
	HasMore    bool
	Sessions   []SyncSession
	Blocks     []SyncBlock
	Sets       []SyncSet
	Bodyweight []Bodyweight
}

// Pull returns the athlete's changes after cursor, at most limit units, from
// one consistent snapshot. Rows whose parents changed after the page are
// brought forward with their parents, so every page applies on its own.
// deviceID, when known, records cursor as acknowledged by that device.
func (s *Store) Pull(ctx context.Context, userID uuid.UUID, deviceID *uuid.UUID, cursor int64, limit int) (SyncPage, error) {
	if deviceID != nil {
		if err := s.q.AckSyncCursor(ctx, dbgen.AckSyncCursorParams{Cursor: cursor, DeviceID: *deviceID, UserID: userID}); err != nil {
			return SyncPage{}, fmt.Errorf("acknowledging cursor: %w", err)
		}
	}
	page := SyncPage{
		Cursor: cursor, Sessions: []SyncSession{}, Blocks: []SyncBlock{}, Sets: []SyncSet{}, Bodyweight: []Bodyweight{},
	}
	opts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, s.pool, opts, func(tx pgx.Tx) error {
		q := dbgen.New(tx)
		changes, err := q.SyncChanges(ctx, dbgen.SyncChangesParams{
			UserID: userID, Cursor: cursor,
			PageLimit: int32(limit + 1), //nolint:gosec // bounded by the handler
		})
		if err != nil {
			return fmt.Errorf("reading changes: %w", err)
		}
		if len(changes) > limit {
			page.HasMore = true
			changes = changes[:limit]
		}
		if len(changes) == 0 {
			return nil
		}
		page.Cursor = changes[len(changes)-1].Seq

		ids := map[string][]uuid.UUID{}
		for _, c := range changes {
			ids[c.Entity] = append(ids[c.Entity], c.ID)
		}
		return page.load(ctx, q, userID, ids)
	})
	if err != nil {
		return SyncPage{}, fmt.Errorf("pulling changes: %w", err)
	}
	return page, nil
}

// load fills the page with the rows named by ids, then brings forward the
// parents the client cannot have yet.
func (p *SyncPage) load(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, ids map[string][]uuid.UUID) error {
	sets, err := loadSyncSets(ctx, q, userID, ids["set"])
	if err != nil {
		return err
	}
	p.Sets = sets

	haveBlock := map[uuid.UUID]bool{}
	for _, id := range ids["block"] {
		haveBlock[id] = true
	}
	var moreBlocks []uuid.UUID
	for _, st := range sets {
		if !haveBlock[st.BlockID] {
			haveBlock[st.BlockID] = true
			moreBlocks = append(moreBlocks, st.BlockID)
		}
	}
	blocks, err := p.loadBlocks(ctx, q, userID, ids["block"], moreBlocks)
	if err != nil {
		return err
	}
	p.Blocks = blocks

	haveSession := map[uuid.UUID]bool{}
	for _, id := range ids["session"] {
		haveSession[id] = true
	}
	var moreSessions []uuid.UUID
	for _, b := range blocks {
		if !haveSession[b.SessionID] {
			haveSession[b.SessionID] = true
			moreSessions = append(moreSessions, b.SessionID)
		}
	}
	for _, st := range sets {
		if !haveSession[st.SessionID] {
			haveSession[st.SessionID] = true
			moreSessions = append(moreSessions, st.SessionID)
		}
	}
	sessions, err := p.loadSessions(ctx, q, userID, ids["session"], moreSessions)
	if err != nil {
		return err
	}
	p.Sessions = sessions

	if len(ids["bodyweight"]) > 0 {
		rows, err := q.SyncBodyweight(ctx, dbgen.SyncBodyweightParams{UserID: userID, Ids: ids["bodyweight"]})
		if err != nil {
			return fmt.Errorf("reading bodyweight: %w", err)
		}
		for _, r := range rows {
			p.Bodyweight = append(p.Bodyweight, bodyweightFromRow(r))
		}
		sort.Slice(p.Bodyweight, func(i, j int) bool { return p.Bodyweight[i].ServerSeq < p.Bodyweight[j].ServerSeq })
	}
	return nil
}

// loadBlocks reads the page's blocks, plus those parents in extra that
// changed after the page (earlier ones the client already has).
func (p *SyncPage) loadBlocks(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, inPage, extra []uuid.UUID) ([]SyncBlock, error) {
	all := append(append([]uuid.UUID{}, inPage...), extra...)
	out := []SyncBlock{}
	if len(all) == 0 {
		return out, nil
	}
	rows, err := q.SyncBlocks(ctx, dbgen.SyncBlocksParams{UserID: userID, Ids: all})
	if err != nil {
		return nil, fmt.Errorf("reading blocks: %w", err)
	}
	isExtra := idSet(extra)
	for _, b := range rows {
		if isExtra[b.ID] && b.ServerSeq <= p.Cursor {
			continue
		}
		out = append(out, SyncBlock{
			Block: training.Block{
				ID: b.ID, SessionID: b.SessionID, OrderIndex: int(b.OrderIndex), Kind: b.Kind,
				RoundsPlanned: intPtr16(b.RoundsPlanned), RoundsDone: intPtr16(b.RoundsDone),
				IntervalS: intPtr32(b.IntervalS), Notes: b.Notes, UpdatedAt: b.UpdatedAt,
			},
			ServerSeq: b.ServerSeq, DeletedAt: b.DeletedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerSeq < out[j].ServerSeq })
	return out, nil
}

func (p *SyncPage) loadSessions(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, inPage, extra []uuid.UUID) ([]SyncSession, error) {
	all := append(append([]uuid.UUID{}, inPage...), extra...)
	out := []SyncSession{}
	if len(all) == 0 {
		return out, nil
	}
	rows, err := q.SyncSessions(ctx, dbgen.SyncSessionsParams{UserID: userID, Ids: all})
	if err != nil {
		return nil, fmt.Errorf("reading sessions: %w", err)
	}
	isExtra := idSet(extra)
	for _, r := range rows {
		if isExtra[r.ID] && r.ServerSeq <= p.Cursor {
			continue
		}
		out = append(out, SyncSession{Session: sessionFromRow(r), ServerSeq: r.ServerSeq, DeletedAt: r.DeletedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerSeq < out[j].ServerSeq })
	return out, nil
}

// loadSyncSets reads whole sets: the entry, its live elements with their
// assistance and media, and the latest change to any of them.
func loadSyncSets(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, ids []uuid.UUID) ([]SyncSet, error) {
	out := []SyncSet{}
	if len(ids) == 0 {
		return out, nil
	}
	entries, err := q.SyncSetEntries(ctx, dbgen.SyncSetEntriesParams{UserID: userID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("reading sets: %w", err)
	}
	seqs, err := q.SyncSetSeqs(ctx, dbgen.SyncSetSeqsParams{UserID: userID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("reading set versions: %w", err)
	}
	seq := make(map[uuid.UUID]int64, len(seqs))
	for _, s := range seqs {
		seq[s.ID] = s.Seq
	}
	elements, err := q.SyncElementsOfSets(ctx, dbgen.SyncElementsOfSetsParams{UserID: userID, SetIds: ids})
	if err != nil {
		return nil, fmt.Errorf("reading elements: %w", err)
	}
	assists, err := q.SyncAssistanceOfSets(ctx, dbgen.SyncAssistanceOfSetsParams{UserID: userID, SetIds: ids})
	if err != nil {
		return nil, fmt.Errorf("reading assistance: %w", err)
	}
	links, err := q.SyncElementMediaOfSets(ctx, dbgen.SyncElementMediaOfSetsParams{UserID: userID, SetIds: ids})
	if err != nil {
		return nil, fmt.Errorf("reading media links: %w", err)
	}

	byElement := make(map[uuid.UUID]*training.Assistance, len(assists))
	for _, a := range assists {
		byElement[a.SetElementID] = assistanceFromRow(a)
	}
	media := map[uuid.UUID][]uuid.UUID{}
	for _, l := range links {
		media[l.SetElementID] = append(media[l.SetElementID], l.MediaID)
	}
	bySet := map[uuid.UUID][]training.Element{}
	for _, e := range elements {
		el := elementFromRow(e, byElement[e.ID])
		el.MediaIDs = nonNilIDs(media[e.ID])
		bySet[e.SetEntryID] = append(bySet[e.SetEntryID], el)
	}
	for _, e := range entries {
		st := setFromRow(e)
		st.Elements = nonNilElements(bySet[e.ID])
		out = append(out, SyncSet{SetEntry: st, ServerSeq: seq[e.ID], DeletedAt: e.DeletedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ServerSeq < out[j].ServerSeq })
	return out, nil
}

func idSet(ids []uuid.UUID) map[uuid.UUID]bool {
	m := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// ---------------------------------------------------------------- bodyweight

// PutBodyweight creates or replaces a bodyweight entry. LocalDate must be set.
func (s *Store) PutBodyweight(ctx context.Context, w Writer, b Bodyweight) (Bodyweight, error) {
	kg, err := numeric(b.BodyweightKg)
	if err != nil {
		return Bodyweight{}, err
	}
	row, err := s.q.UpsertBodyweight(ctx, dbgen.UpsertBodyweightParams{
		ID: b.ID, UserID: w.UserID, MeasuredAt: b.MeasuredAt, LocalDate: dateOf(b.LocalDate),
		BodyweightKg: kg, Note: b.Note, ClientID: w.DeviceID, UpdatedAt: w.At,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		if prev, err := s.q.GetBodyweight(ctx, dbgen.GetBodyweightParams{ID: b.ID, UserID: w.UserID}); err == nil && prev.DeletedAt != nil {
			return Bodyweight{}, ErrDeleted
		}
		return Bodyweight{}, ErrNotFound // another user's id
	}
	if err != nil {
		return Bodyweight{}, translate(fmt.Errorf("writing bodyweight: %w", err))
	}
	return bodyweightFromRow(row), nil
}

// DeleteBodyweight tombstones a bodyweight entry.
func (s *Store) DeleteBodyweight(ctx context.Context, w Writer, id uuid.UUID) error {
	n, err := s.q.SoftDeleteBodyweight(ctx, dbgen.SoftDeleteBodyweightParams{
		ClientID: w.DeviceID, UpdatedAt: w.At, ID: id, UserID: w.UserID,
	})
	if err != nil {
		return fmt.Errorf("deleting bodyweight: %w", err)
	}
	if n == 0 {
		if prev, err := s.q.GetBodyweight(ctx, dbgen.GetBodyweightParams{ID: id, UserID: w.UserID}); err == nil && prev.DeletedAt != nil {
			return ErrDeleted
		}
		return ErrNotFound
	}
	return nil
}

func bodyweightFromRow(r dbgen.UserBodyweightLog) Bodyweight {
	return Bodyweight{
		ID: r.ID, MeasuredAt: r.MeasuredAt, LocalDate: r.LocalDate.Time, BodyweightKg: numericToFloat(r.BodyweightKg),
		Note: r.Note, UpdatedAt: r.UpdatedAt, ServerSeq: r.ServerSeq, DeletedAt: r.DeletedAt,
	}
}

// --------------------------------------------------------------- idempotency

// Idempotency errors.
var (
	ErrIdempotencyInProgress = errors.New("a request with this idempotency key is in progress")
	ErrIdempotencyKeyReuse   = errors.New("this idempotency key was used with a different request")
)

// StoredResponse is the first response to an idempotent request.
type StoredResponse struct {
	Status int
	Body   []byte
}

// ClaimIdempotencyKey claims key for a request with this fingerprint. It
// returns the stored response when the same request already completed, and
// nothing when the caller now holds the key and must Finish or Release it.
func (s *Store) ClaimIdempotencyKey(ctx context.Context, userID uuid.UUID, key string, fingerprint []byte) (*StoredResponse, error) {
	for range 2 { // a second try covers a key that expired between the two statements
		_, err := s.q.ClaimIdempotencyKey(ctx, dbgen.ClaimIdempotencyKeyParams{UserID: userID, Key: key, RequestFingerprint: fingerprint})
		if err == nil {
			return nil, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("claiming idempotency key: %w", err)
		}
		k, err := s.q.GetIdempotencyKey(ctx, dbgen.GetIdempotencyKeyParams{UserID: userID, Key: key})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading idempotency key: %w", err)
		}
		switch {
		case string(k.RequestFingerprint) != string(fingerprint):
			return nil, ErrIdempotencyKeyReuse
		case k.State == "completed" && k.ResponseStatus != nil:
			return &StoredResponse{Status: int(*k.ResponseStatus), Body: k.ResponseBody}, nil
		default:
			return nil, ErrIdempotencyInProgress
		}
	}
	return nil, ErrIdempotencyInProgress
}

// FinishIdempotencyKey stores the response for replays.
func (s *Store) FinishIdempotencyKey(ctx context.Context, userID uuid.UUID, key string, r StoredResponse) error {
	status := int32(r.Status) //nolint:gosec // an HTTP status
	if err := s.q.CompleteIdempotencyKey(ctx, dbgen.CompleteIdempotencyKeyParams{
		ResponseStatus: &status, ResponseBody: r.Body, UserID: userID, Key: key,
	}); err != nil {
		return fmt.Errorf("storing idempotent response: %w", err)
	}
	return nil
}

// ReleaseIdempotencyKey gives up a claimed key after a failure, so a retry
// with the same key runs again.
func (s *Store) ReleaseIdempotencyKey(ctx context.Context, userID uuid.UUID, key string) error {
	if err := s.q.ReleaseIdempotencyKey(ctx, dbgen.ReleaseIdempotencyKeyParams{UserID: userID, Key: key}); err != nil {
		return fmt.Errorf("releasing idempotency key: %w", err)
	}
	return nil
}

// PurgeExpiredIdempotencyKeys deletes keys past their 24 hours.
func (s *Store) PurgeExpiredIdempotencyKeys(ctx context.Context) (int64, error) {
	n, err := s.q.PurgeExpiredIdempotencyKeys(ctx)
	if err != nil {
		return 0, fmt.Errorf("purging idempotency keys: %w", err)
	}
	return n, nil
}
