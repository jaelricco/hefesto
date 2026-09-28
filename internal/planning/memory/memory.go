// Package memory implements the planning ports in memory, for tests and
// local runs without Postgres. Values are copied through JSON, as a
// database would, so callers never share maps with the store.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
	service "github.com/jaelricco/hefesto/internal/planning"
)

// Decision is one recorded event with its changes.
type Decision struct {
	Trigger  string
	SourceID string
	Changes  []planning.Change
}

// Store implements the Transactor and, through it, SnapshotStore,
// PlanStore and DecisionLog. InTx serialises the calls of one user like the
// Postgres adapter does; it does not roll back.
type Store struct {
	mu        sync.Mutex
	snapshots map[uuid.UUID][]byte
	plans     map[string][]byte
	decisions map[uuid.UUID][]Decision
	users     map[uuid.UUID]*sync.Mutex
}

// New returns an empty store.
func New() *Store {
	return &Store{
		snapshots: map[uuid.UUID][]byte{},
		plans:     map[string][]byte{},
		decisions: map[uuid.UUID][]Decision{},
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

// Seen reports whether an event was already recorded.
func (s *Store) Seen(_ context.Context, userID uuid.UUID, trigger, sourceID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.decisions[userID] {
		if d.Trigger == trigger && d.SourceID == sourceID {
			return true, nil
		}
	}
	return false, nil
}

// Record stores an event and its changes.
func (s *Store) Record(_ context.Context, userID uuid.UUID, trigger, sourceID string, cs []planning.Change) error {
	s.mu.Lock()
	s.decisions[userID] = append(s.decisions[userID], Decision{Trigger: trigger, SourceID: sourceID, Changes: cs})
	s.mu.Unlock()
	return nil
}

// Decisions returns the recorded events of a user, oldest first.
func (s *Store) Decisions(userID uuid.UUID) []Decision {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Decision(nil), s.decisions[userID]...)
}
