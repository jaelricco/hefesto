// Package memory implements the planning ports in memory, for tests and
// local runs without Postgres. Values are copied through JSON, as a
// database would, so callers never share maps with the store.
package memory

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
	service "github.com/jaelricco/hefesto/internal/planning"
)

// Store implements the Transactor and, through it, SnapshotStore,
// PlanStore, DecisionLog, SessionLog and ConsentLog. InTx serialises the calls of one
// user like the Postgres adapter does; it does not roll back.
type Store struct {
	mu        sync.Mutex
	snapshots map[uuid.UUID][]byte
	plans     map[string][]byte
	decisions map[uuid.UUID][]service.Decision
	consents  map[uuid.UUID][]service.ConsentRecord
	started   map[uuid.UUID]service.SessionStart
	completed map[uuid.UUID]planning.LoggedSession
	deload    map[uuid.UUID]bool
	users     map[uuid.UUID]*sync.Mutex
	// planned holds every planned session ever saved, replaced ones too,
	// by user.
	planned map[uuid.UUID]map[uuid.UUID][]byte
	// offline holds the log sessions started offline, by user, with their
	// planned sessions; linked the log sessions a plan ever pointed at.
	offline map[uuid.UUID]map[uuid.UUID]uuid.UUID
	linked  map[uuid.UUID]bool
}

// New returns an empty store.
func New() *Store {
	return &Store{
		snapshots: map[uuid.UUID][]byte{},
		plans:     map[string][]byte{},
		decisions: map[uuid.UUID][]service.Decision{},
		consents:  map[uuid.UUID][]service.ConsentRecord{},
		started:   map[uuid.UUID]service.SessionStart{},
		completed: map[uuid.UUID]planning.LoggedSession{},
		deload:    map[uuid.UUID]bool{},
		users:     map[uuid.UUID]*sync.Mutex{},
		planned:   map[uuid.UUID]map[uuid.UUID][]byte{},
		offline:   map[uuid.UUID]map[uuid.UUID]uuid.UUID{},
		linked:    map[uuid.UUID]bool{},
	}
}

// InTx runs fn while holding the user's lock.
func (s *Store) InTx(_ context.Context, userID uuid.UUID, fn func(service.Stores) error) error {
	s.mu.Lock()
	l, ok := s.users[userID]
	if !ok {
		l = &sync.Mutex{}
		s.users[userID] = l
	}
	s.mu.Unlock()
	l.Lock()
	defer l.Unlock()
	return fn(service.Stores{Snapshots: s, Plans: s, Decisions: s, Sessions: s, Consents: s})
}

// Snapshot returns the stored snapshot of a user.
func (s *Store) Snapshot(_ context.Context, userID uuid.UUID) (planning.Snapshot, bool, error) {
	s.mu.Lock()
	raw, ok := s.snapshots[userID]
	s.mu.Unlock()
	if !ok {
		return planning.Snapshot{}, false, nil
	}
	var out planning.Snapshot
	if err := json.Unmarshal(raw, &out); err != nil {
		return planning.Snapshot{}, false, fmt.Errorf("decoding snapshot: %w", err)
	}
	return out, true, nil
}

// SaveSnapshot replaces the snapshot of a user.
func (s *Store) SaveSnapshot(_ context.Context, userID uuid.UUID, snap planning.Snapshot) error {
	raw, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("encoding snapshot: %w", err)
	}
	s.mu.Lock()
	s.snapshots[userID] = raw
	s.mu.Unlock()
	return nil
}

func planKey(userID uuid.UUID, week time.Time) string {
	return userID.String() + "/" + week.UTC().Format(time.DateOnly)
}

// ActivePlan returns the stored plan of a week.
func (s *Store) ActivePlan(_ context.Context, userID uuid.UUID, week time.Time) (planning.Plan, bool, error) {
	s.mu.Lock()
	raw, ok := s.plans[planKey(userID, week)]
	s.mu.Unlock()
	if !ok {
		return planning.Plan{}, false, nil
	}
	var out planning.Plan
	if err := json.Unmarshal(raw, &out); err != nil {
		return planning.Plan{}, false, fmt.Errorf("decoding plan: %w", err)
	}
	return out, true, nil
}

// SavePlan replaces the plan of its week. Its sessions stay readable by
// PlannedSessionAny after a new plan replaces it.
func (s *Store) SavePlan(_ context.Context, userID uuid.UUID, p planning.Plan) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encoding plan: %w", err)
	}
	sessions := map[uuid.UUID][]byte{}
	var linked []uuid.UUID
	for _, ps := range p.Sessions {
		id, err := uuid.Parse(ps.ID)
		if err != nil {
			return fmt.Errorf("planned session id %q: %w", ps.ID, err)
		}
		if log, err := uuid.Parse(ps.WorkoutSessionID); err == nil {
			linked = append(linked, log)
		}
		ps.Status, ps.WorkoutSessionID, ps.CheckIn = "", "", false
		if sessions[id], err = json.Marshal(ps); err != nil {
			return fmt.Errorf("encoding planned session: %w", err)
		}
	}
	s.mu.Lock()
	s.plans[planKey(userID, p.WeekStart)] = raw
	if s.planned[userID] == nil {
		s.planned[userID] = map[uuid.UUID][]byte{}
	}
	maps.Copy(s.planned[userID], sessions)
	for _, id := range linked {
		s.linked[id] = true
	}
	s.mu.Unlock()
	return nil
}

// PlannedSessionAny returns a planned session of any plan saved, as
// planned.
func (s *Store) PlannedSessionAny(_ context.Context, userID, sessionID uuid.UUID) (planning.PlannedSession, bool, error) {
	s.mu.Lock()
	raw, ok := s.planned[userID][sessionID]
	s.mu.Unlock()
	if !ok {
		return planning.PlannedSession{}, false, nil
	}
	var out planning.PlannedSession
	if err := json.Unmarshal(raw, &out); err != nil {
		return planning.PlannedSession{}, false, fmt.Errorf("decoding planned session: %w", err)
	}
	return out, true, nil
}

// PlannedSession finds a session of one of the user's active plans.
func (s *Store) PlannedSession(ctx context.Context, userID, sessionID uuid.UUID) (planning.Plan, int, bool, error) {
	s.mu.Lock()
	var keys []string
	for key := range s.plans {
		if strings.HasPrefix(key, userID.String()+"/") {
			keys = append(keys, key)
		}
	}
	s.mu.Unlock()
	for _, key := range keys {
		week, err := time.Parse(time.DateOnly, strings.TrimPrefix(key, userID.String()+"/"))
		if err != nil {
			return planning.Plan{}, 0, false, fmt.Errorf("plan key %q: %w", key, err)
		}
		p, ok, err := s.ActivePlan(ctx, userID, week)
		if err != nil {
			return planning.Plan{}, 0, false, err
		}
		if !ok {
			continue
		}
		for i, ps := range p.Sessions {
			if ps.ID == sessionID.String() {
				return p, i, true, nil
			}
		}
	}
	return planning.Plan{}, 0, false, nil
}

// StartSession records the start of a planned session and marks it
// started in its plan. The draft is kept as given; Started returns it.
func (s *Store) StartSession(ctx context.Context, userID uuid.UUID, in service.SessionStart) (uuid.UUID, bool, error) {
	p, i, ok, err := s.PlannedSession(ctx, userID, in.PlannedSessionID)
	if err != nil {
		return uuid.Nil, false, err
	}
	if !ok {
		return uuid.Nil, false, service.ErrNotFound
	}
	if id := p.Sessions[i].WorkoutSessionID; id != "" {
		out, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, false, fmt.Errorf("log session %q: %w", id, err)
		}
		return out, false, nil
	}
	s.mu.Lock()
	_, taken := s.started[in.SessionID]
	s.mu.Unlock()
	if taken {
		return uuid.Nil, false, service.ErrSessionIDTaken
	}
	p.Sessions[i].Status, p.Sessions[i].WorkoutSessionID = service.SessionStarted, in.SessionID.String()
	p.Sessions[i].CheckIn = in.Lighter
	if err := s.SavePlan(ctx, userID, p); err != nil {
		return uuid.Nil, false, err
	}
	s.mu.Lock()
	s.started[in.SessionID] = in
	s.mu.Unlock()
	return in.SessionID, true, nil
}

// DraftSets returns the sets of a started draft. The memory store keeps no
// log: every set is open until an adjustment removes it.
func (s *Store) DraftSets(_ context.Context, _ uuid.UUID, sessionID uuid.UUID) ([]planning.DraftState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, ok := s.started[sessionID]
	if !ok {
		return nil, false, nil
	}
	var out []planning.DraftState
	for _, b := range in.Draft.Blocks {
		for _, set := range b.Sets {
			out = append(out, planning.DraftState{ID: set.ID, BlockID: b.ID, ItemID: set.ItemID, Open: true})
		}
	}
	return out, true, nil
}

// AdjustSession applies an adjustment to a started draft.
func (s *Store) AdjustSession(_ context.Context, _ uuid.UUID, sessionID uuid.UUID, a planning.Adjustment, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, ok := s.started[sessionID]
	if !ok {
		return service.ErrNotFound
	}
	relink := map[string]string{}
	for _, r := range a.Relink {
		relink[r.SetID] = r.ItemID
	}
	replace := map[string]planning.DraftSet{}
	for _, r := range a.Replace {
		replace[r.SetID] = r.Set
	}
	var blocks []planning.DraftBlock
	for _, b := range in.Draft.Blocks {
		var sets []planning.DraftSet
		for _, set := range b.Sets {
			switch {
			case slices.Contains(a.Remove, set.ID):
				continue
			case replace[set.ID].ID != "":
				r := replace[set.ID]
				r.Round = set.Round
				set = r
			case relink[set.ID] != "":
				set.ItemID = relink[set.ID]
			}
			sets = append(sets, set)
		}
		for _, add := range a.Append {
			if add.BlockID == b.ID {
				sets = append(sets, add.Set)
			}
		}
		if len(sets) > 0 {
			b.Sets = sets
			blocks = append(blocks, b)
		}
	}
	blocks = append(blocks, a.Blocks...)
	in.Draft.Blocks = blocks
	s.started[sessionID] = in
	return nil
}

// Complete records a completed log session, as the log would after
// POST /v1/sessions/{id}/complete. The memory store keeps no log; tests
// hand it the session the planner reads.
func (s *Store) Complete(sess planning.LoggedSession) error {
	id, err := uuid.Parse(sess.ID)
	if err != nil {
		return fmt.Errorf("session id %q: %w", sess.ID, err)
	}
	s.mu.Lock()
	s.completed[id] = sess
	delete(s.started, id)
	s.mu.Unlock()
	return nil
}

// CompletedSession returns a session recorded with Complete.
func (s *Store) CompletedSession(_ context.Context, _ uuid.UUID, sessionID uuid.UUID) (planning.LoggedSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.completed[sessionID]
	return sess, ok, nil
}

// MarkCompleted marks the planned sessions of a log session completed in
// the stored plans.
func (s *Store) MarkCompleted(ctx context.Context, userID, sessionID uuid.UUID, deload bool) error {
	s.mu.Lock()
	var weeks []string
	for key := range s.plans {
		if strings.HasPrefix(key, userID.String()+"/") {
			weeks = append(weeks, strings.TrimPrefix(key, userID.String()+"/"))
		}
	}
	if deload {
		s.deload[sessionID] = true
	}
	s.mu.Unlock()
	for _, w := range weeks {
		week, err := time.Parse(time.DateOnly, w)
		if err != nil {
			return fmt.Errorf("plan week %q: %w", w, err)
		}
		p, ok, err := s.ActivePlan(ctx, userID, week)
		if err != nil || !ok {
			return err
		}
		changed := false
		for i := range p.Sessions {
			if p.Sessions[i].WorkoutSessionID == sessionID.String() {
				p.Sessions[i].Status, changed = service.SessionCompleted, true
			}
		}
		if changed {
			if err := s.SavePlan(ctx, userID, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeloadDay reports whether a completed session marked its day a deload day.
func (s *Store) DeloadDay(sessionID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deload[sessionID]
}

// PendingCompletions lists recorded completions from the day of the
// onboarding on that no decision applied, oldest first.
func (s *Store) PendingCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error) {
	snap, ok, err := s.Snapshot(ctx, userID)
	if err != nil || !ok {
		return nil, err
	}
	since := snap.Profile.OnboardedAt
	s.mu.Lock()
	defer s.mu.Unlock()
	type entry struct {
		id   uuid.UUID
		date time.Time
	}
	var pending []entry
	for id, sess := range s.completed {
		applied := slices.ContainsFunc(s.decisions[userID], func(d service.Decision) bool {
			return d.Trigger == service.TriggerSession && d.SourceID == sess.ID
		})
		if !applied && !sess.Date.Before(civilDay(since)) {
			pending = append(pending, entry{id, sess.Date})
		}
	}
	slices.SortFunc(pending, func(a, b entry) int { return cmp.Or(a.date.Compare(b.date), bytes.Compare(a.id[:], b.id[:])) })
	var out []uuid.UUID
	for _, e := range pending[:min(limit, len(pending))] {
		out = append(out, e.id)
	}
	return out, nil
}

func civilDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// StartOffline records a session a client started offline and synced, as
// the log would after the sync: its draft is kept as given, and no plan
// points at it until the planner links it (spec §10.5).
func (s *Store) StartOffline(userID uuid.UUID, in service.SessionStart) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started[in.SessionID] = in
	if s.offline[userID] == nil {
		s.offline[userID] = map[uuid.UUID]uuid.UUID{}
	}
	s.offline[userID][in.SessionID] = in.PlannedSessionID
}

// PendingOfflineStarts lists the recorded offline starts no plan pointed
// at yet, of planned sessions from since on, oldest first.
func (s *Store) PendingOfflineStarts(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]service.OfflineStart, error) {
	s.mu.Lock()
	var out []service.OfflineStart
	for log, planned := range s.offline[userID] {
		_, draft := s.started[log]
		_, done := s.completed[log]
		if !s.linked[log] && (draft || done) {
			out = append(out, service.OfflineStart{SessionID: log, PlannedSessionID: planned})
		}
	}
	s.mu.Unlock()
	slices.SortFunc(out, func(a, b service.OfflineStart) int { return bytes.Compare(a.SessionID[:], b.SessionID[:]) })
	var recent []service.OfflineStart
	for _, o := range out {
		ps, ok, err := s.PlannedSessionAny(ctx, userID, o.PlannedSessionID)
		if err != nil {
			return nil, err
		}
		if ok && !ps.Date.Before(civilDay(since)) && len(recent) < limit {
			recent = append(recent, o)
		}
	}
	return recent, nil
}

// LinkStart points the planned session of an active plan at a log session
// started offline: completed when the log session is, else started.
func (s *Store) LinkStart(ctx context.Context, userID, plannedSessionID, sessionID uuid.UUID) error {
	p, i, ok, err := s.PlannedSession(ctx, userID, plannedSessionID)
	if err != nil || !ok {
		return err
	}
	s.mu.Lock()
	_, done := s.completed[sessionID]
	s.mu.Unlock()
	p.Sessions[i].Status, p.Sessions[i].WorkoutSessionID, p.Sessions[i].CheckIn = service.SessionStarted, sessionID.String(), false
	if done {
		p.Sessions[i].Status = service.SessionCompleted
	}
	return s.SavePlan(ctx, userID, p)
}

// Started returns the start that created a log session.
func (s *Store) Started(sessionID uuid.UUID) (service.SessionStart, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, ok := s.started[sessionID]
	return in, ok
}

// Recorded returns the recorded event of a trigger and source.
func (s *Store) Recorded(_ context.Context, userID uuid.UUID, trigger, sourceID string) (service.Decision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.decisions[userID] {
		if d.Trigger == trigger && d.SourceID == sourceID {
			return clone(d)
		}
	}
	return service.Decision{}, false, nil
}

// Record stores an event and its changes.
func (s *Store) Record(_ context.Context, userID uuid.UUID, d service.Decision) error {
	d, _, err := clone(d)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.decisions[userID] = append(s.decisions[userID], d)
	s.mu.Unlock()
	return nil
}

// ListDecisions returns recorded events newest first, after the cursor.
func (s *Store) ListDecisions(_ context.Context, userID uuid.UUID, after *service.Cursor, limit int) ([]service.Decision, error) {
	s.mu.Lock()
	all := slices.Clone(s.decisions[userID])
	s.mu.Unlock()
	newest := func(a, b service.Decision) int {
		return cmp.Or(b.At.Compare(a.At), bytes.Compare(b.ID[:], a.ID[:]))
	}
	slices.SortFunc(all, newest)
	out := []service.Decision{}
	for _, d := range all {
		if after != nil && newest(service.Decision{At: after.At, ID: after.ID}, d) >= 0 {
			continue
		}
		if len(out) == limit {
			break
		}
		c, _, err := clone(d)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// RecordConsent appends a decision on the consent.
func (s *Store) RecordConsent(_ context.Context, userID uuid.UUID, r service.ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, prev := range s.consents[userID] {
		if prev.Source == r.Source && prev.ChangeID == r.ChangeID {
			return fmt.Errorf("consent %s %q recorded twice", r.Source, r.ChangeID)
		}
	}
	if r.Text != nil {
		t := *r.Text
		r.Text = &t
	}
	s.consents[userID] = append(s.consents[userID], r)
	return nil
}

// ConsentRecords returns up to limit decisions on the consent, newest
// first.
func (s *Store) ConsentRecords(_ context.Context, userID uuid.UUID, limit int) ([]service.ConsentRecord, error) {
	s.mu.Lock()
	all := slices.Clone(s.consents[userID])
	s.mu.Unlock()
	slices.SortFunc(all, func(a, b service.ConsentRecord) int {
		return cmp.Or(b.At.Compare(a.At), bytes.Compare(b.ID[:], a.ID[:]))
	})
	out := all[:min(max(limit, 0), len(all))]
	for i := range out {
		if out[i].Text != nil {
			t := *out[i].Text
			out[i].Text = &t
		}
	}
	return out, nil
}

// Decisions returns the recorded events of a user in the order recorded.
func (s *Store) Decisions(userID uuid.UUID) []service.Decision {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.decisions[userID])
}

// clone copies a decision through JSON, as a database would.
func clone(d service.Decision) (service.Decision, bool, error) {
	raw, err := json.Marshal(d.Changes)
	if err != nil {
		return service.Decision{}, false, fmt.Errorf("encoding changes: %w", err)
	}
	var cs []planning.Change
	if err := json.Unmarshal(raw, &cs); err != nil {
		return service.Decision{}, false, fmt.Errorf("decoding changes: %w", err)
	}
	if len(cs) == 0 {
		cs = nil
	}
	d.Changes = cs
	return d, true, nil
}
