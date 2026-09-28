// Package planning is the application service around the training planner
// (docs/algorithm/spec.md §10.1): it loads the user's snapshot, calls the
// pure core in internal/domain/planning and stores the results. Data access
// goes through the ports below; internal/store implements them for
// Postgres, the memory package for tests and local runs.
package planning

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// Stores are the ports one call of the service reads and writes.
type Stores struct {
	Snapshots SnapshotStore
	Plans     PlanStore
	Decisions DecisionLog
}

// Transactor runs fn with stores bound to one transaction that holds the
// user's planner lock. Events of one user therefore apply one at a time, and
// either all writes of an event land or none (spec §9.4, ADR 0013).
type Transactor interface {
	InTx(ctx context.Context, userID uuid.UUID, fn func(Stores) error) error
}

// SnapshotStore reads and writes the planner state of one user. The core
// returns a whole new snapshot from every event, so the port stores it
// whole; the Postgres adapter splits it into the tables of spec §4.9. Every
// logged session and pain report in a snapshot carries an ID; the history
// is append-only.
type SnapshotStore interface {
	Snapshot(ctx context.Context, userID uuid.UUID) (planning.Snapshot, bool, error)
	SaveSnapshot(ctx context.Context, userID uuid.UUID, s planning.Snapshot) error
}

// PlanStore keeps the generated week plans. A plan and its sessions carry
// the IDs the service gave them.
type PlanStore interface {
	ActivePlan(ctx context.Context, userID uuid.UUID, week time.Time) (planning.Plan, bool, error)
	SavePlan(ctx context.Context, userID uuid.UUID, p planning.Plan) error
	// PlannedSession returns the active plan that holds a planned session,
	// and the session's index in it.
	PlannedSession(ctx context.Context, userID, sessionID uuid.UUID) (planning.Plan, int, bool, error)
}

// Decision is one applied event with the changes it made (spec §6.14).
type Decision struct {
	ID       uuid.UUID
	Trigger  string
	SourceID string
	At       time.Time
	Changes  []planning.Change
}

// Cursor is a position in a list ordered by time, then ID, newest first.
type Cursor struct {
	At time.Time
	ID uuid.UUID
}

// DecisionLog records every change the user sees and makes events
// idempotent: an event recorded before (same trigger and source) is not
// applied again, and its recorded changes are the answer (spec §9.4).
type DecisionLog interface {
	Recorded(ctx context.Context, userID uuid.UUID, trigger, sourceID string) (Decision, bool, error)
	Record(ctx context.Context, userID uuid.UUID, d Decision) error
	// ListDecisions returns up to limit events, newest first, after the
	// cursor when there is one.
	ListDecisions(ctx context.Context, userID uuid.UUID, after *Cursor, limit int) ([]Decision, error)
}

// KnowledgeSource returns the validated knowledge base, or
// ErrUnavailable when it failed validation (spec §2.6).
type KnowledgeSource interface {
	Current(ctx context.Context) (*planning.Knowledge, error)
}

// Clock is the time source; tests fix it.
type Clock interface{ Now() time.Time }

// SystemClock is the wall clock.
type SystemClock struct{}

// Now returns the current time in UTC, to the microsecond: Postgres keeps
// microseconds, and a snapshot must read back exactly as it was written.
func (SystemClock) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// IDSource creates the IDs of rows the planner itself creates: plans and
// their sessions, decisions and the pain baseline from the onboarding.
type IDSource interface{ New() uuid.UUID }

// UUIDv7 is the ID source of the service: time-ordered UUIDs (CLAUDE.md).
type UUIDv7 struct{}

// New returns a new UUIDv7.
func (UUIDv7) New() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Errors of the service, mapped to problems at the HTTP edge (spec §10.4).
var (
	// ErrUnavailable: the knowledge base is missing or invalid (503).
	ErrUnavailable = errors.New("planning unavailable")
	// ErrNotOnboarded: SAFE-01, there is no snapshot yet (409).
	ErrNotOnboarded = errors.New("onboarding required")
	// ErrAlreadyOnboarded: the onboarding runs once; later changes go
	// through the profile and the goals (409).
	ErrAlreadyOnboarded = errors.New("already onboarded")
	// ErrConsentRequired: pain reports are health data and need the
	// consent (409, onboarding.md §3.1).
	ErrConsentRequired = errors.New("health data consent required")
	// ErrNotFound: no such plan or planned session (404).
	ErrNotFound = errors.New("not found")
)
