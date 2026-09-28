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
	Sessions  SessionLog
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
// the IDs the service gave them. A read fills each session's Status and
// WorkoutSessionID; a save stores them as given.
type PlanStore interface {
	ActivePlan(ctx context.Context, userID uuid.UUID, week time.Time) (planning.Plan, bool, error)
	SavePlan(ctx context.Context, userID uuid.UUID, p planning.Plan) error
	// PlannedSession returns the active plan that holds a planned session,
	// and the session's index in it.
	PlannedSession(ctx context.Context, userID, sessionID uuid.UUID) (planning.Plan, int, bool, error)
}

// SessionLog writes planned sessions into the training log (spec §10.2).
type SessionLog interface {
	// StartSession writes the draft as a draft session of the log and marks
	// the planned session started, in one transaction. A planned session
	// whose log session still exists is not started again: the store
	// returns that session and created false. ErrNotFound when the planned
	// session is in no active plan.
	StartSession(ctx context.Context, userID uuid.UUID, in SessionStart) (id uuid.UUID, created bool, err error)
	// DraftSets returns the live sets of a log session in log order, and
	// false when the session is no live draft: a completed, abandoned or
	// deleted session does not follow the plan.
	DraftSets(ctx context.Context, userID, sessionID uuid.UUID) ([]planning.DraftState, bool, error)
	// AdjustSession writes an adjustment into a draft (ADR 0017). at is the
	// updated_at of the rows it writes.
	AdjustSession(ctx context.Context, userID, sessionID uuid.UUID, a planning.Adjustment, at time.Time) error
	// CompletedSession returns a completed log session as the planner's
	// history records it (ADR 0018): its performed sets in the order
	// performed. ok is false when the session is not completed, is a rest
	// day or does not exist.
	CompletedSession(ctx context.Context, userID, sessionID uuid.UUID) (planning.LoggedSession, bool, error)
	// MarkCompleted marks the planned sessions a log session was started
	// from as completed, and its day a deload day when deload is set.
	MarkCompleted(ctx context.Context, userID, sessionID uuid.UUID, deload bool) error
	// PendingCompletions returns up to limit sessions completed since the
	// day of the onboarding that the planner has not applied yet, oldest
	// first.
	PendingCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error)
}

// SessionStart is a planned session started in the log.
type SessionStart struct {
	PlannedSessionID uuid.UUID
	// SessionID is the log session's ID, given by the client.
	SessionID uuid.UUID
	// DeviceID becomes the client_id of the log rows.
	DeviceID  *uuid.UUID
	StartedAt time.Time
	Timezone  string
	LocalDate time.Time // the athlete's calendar day of StartedAt
	// At is the client's clock for the change, the rows' updated_at.
	At time.Time
	// CheckIn is the athlete's optional check-in at the start (spec
	// §6.12); it is not stored.
	CheckIn *planning.CheckIn
	// Lighter records that the check-in made the session lighter
	// (ADAPT-17); the service sets it.
	Lighter bool
	Draft   planning.SessionDraft
}

// Statuses of a planned session.
const (
	SessionPlanned   = "planned"
	SessionStarted   = "started"
	SessionCompleted = "completed"
)

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
	// ErrTrainingStopped: SAFE-02 or SAFE-07 stop all training, and no
	// planned session starts (409, spec §10.4). The plan says why.
	ErrTrainingStopped = errors.New("training stopped")
	// ErrSessionIDTaken: the log session's ID belongs to another session
	// (409).
	ErrSessionIDTaken = errors.New("session id already in use")
)

// StoppedError is ErrTrainingStopped with the rule that stops training.
type StoppedError struct{ Rule string }

func (e *StoppedError) Error() string { return "training stopped by " + e.Rule }

// Unwrap makes a StoppedError match ErrTrainingStopped.
func (e *StoppedError) Unwrap() error { return ErrTrainingStopped }
