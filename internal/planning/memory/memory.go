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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
	service "github.com/jaelricco/hefesto/internal/planning"
)

// Store implements the Transactor and, through it, SnapshotStore,
// PlanStore and DecisionLog. InTx serialises the calls of one user like the
// Postgres adapter does; it does not roll back.
type Store struct {
	mu        sync.Mutex
	snapshots map[uuid.UUID][]byte
	plans     map[string][]byte
	decisions map[uuid.UUID][]service.Decision
	users     map[uuid.UUID]*sync.Mutex
}

// New returns an empty store.
func New() *Store {
	return &Store{
		snapshots: map[uuid.UUID][]byte{},
		plans:     map[string][]byte{},
		decisions: map[uuid.UUID][]service.Decision{},
		users:     map[uuid.UUID]*sync.Mutex{},
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
	return fn(service.Stores{Snapshots: s, Plans: s, Decisions: s})
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

// SavePlan replaces the plan of its week.
func (s *Store) SavePlan(_ context.Context, userID uuid.UUID, p planning.Plan) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encoding plan: %w", err)
	}
	s.mu.Lock()
	s.plans[planKey(userID, p.WeekStart)] = raw
	s.mu.Unlock()
	return nil
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
