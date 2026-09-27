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

// SnapshotStore reads and writes the planner state of one user. The core
// returns a whole new snapshot from every event, so the port stores it
// whole; a Postgres adapter splits it into the tables of spec §4.9.
type SnapshotStore interface {
	Snapshot(ctx context.Context, userID uuid.UUID) (planning.Snapshot, bool, error)
	SaveSnapshot(ctx context.Context, userID uuid.UUID, s planning.Snapshot) error
}

// PlanStore keeps the generated week plans.
type PlanStore interface {
	ActivePlan(ctx context.Context, userID uuid.UUID, week time.Time) (planning.Plan, bool, error)
	SavePlan(ctx context.Context, userID uuid.UUID, p planning.Plan) error
}

// DecisionLog records every change the user sees and makes events
// idempotent: an event already seen (same trigger and source) is skipped
// (spec §9.4).
type DecisionLog interface {
	Seen(ctx context.Context, userID uuid.UUID, trigger, sourceID string) (bool, error)
	Record(ctx context.Context, userID uuid.UUID, trigger, sourceID string, cs []planning.Change) error
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

// Now returns the current time in UTC.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// Errors of the service. The HTTP layer maps ErrUnavailable to 503
// planning-unavailable and ErrNotOnboarded to 409 (spec §10.4).
var (
	ErrUnavailable  = errors.New("planning unavailable")
	ErrNotOnboarded = errors.New("onboarding required")
)
