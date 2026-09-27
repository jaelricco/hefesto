package planning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// Service runs the planner for one user at a time. It holds no state of
// its own; every call reads the snapshot, applies the pure core and writes
// the result.
type Service struct {
	Knowledge KnowledgeSource
	Snapshots SnapshotStore
	Plans     PlanStore
	Decisions DecisionLog
	Clock     Clock
	Log       *slog.Logger
}

// Triggers of the decision log.
const (
	TriggerOnboarding = "onboarding"
	TriggerSession    = "session_completed"
	TriggerPain       = "pain_report"
	TriggerRedFlags   = "red_flags"
	TriggerClearance  = "clearance"
	TriggerSymptoms   = "symptoms"
	TriggerWeek       = "week_start"
)

// Onboard turns the onboarding answers into the start state and the first
// plan. While clarification questions are open nothing is stored; the
// client asks them and sends the answers again (onboarding.md §5.5).
func (s *Service) Onboard(ctx context.Context, userID uuid.UUID, a planning.Answers) (planning.OnboardingResult, *planning.Plan, error) {
	kb, err := s.Knowledge.Current(ctx)
	if err != nil {
		return planning.OnboardingResult{}, nil, fmt.Errorf("onboarding: %w", err)
	}
	snap, res, err := planning.Start(kb, a, s.Clock.Now())
	if err != nil {
		return planning.OnboardingResult{}, nil, fmt.Errorf("onboarding: %w", err)
	}
	if res.Status == planning.OnboardingNeedsAnswers {
		return res, nil, nil
	}
	if err := s.Snapshots.SaveSnapshot(ctx, userID, snap); err != nil {
		return res, nil, fmt.Errorf("onboarding: saving snapshot: %w", err)
	}
	p, err := s.regenerate(ctx, kb, userID, snap)
	if err != nil {
		return res, nil, err
	}
	return res, &p, nil
}

// Plan returns the plan of the current week, generating it on first use.
// Plans change only on events (spec §11.3).
func (s *Service) Plan(ctx context.Context, userID uuid.UUID) (planning.Plan, error) {
	kb, snap, err := s.load(ctx, userID)
	if err != nil {
		return planning.Plan{}, err
	}
	week := WeekStart(s.Clock.Now())
	p, ok, err := s.Plans.ActivePlan(ctx, userID, week)
	if err != nil {
		return planning.Plan{}, fmt.Errorf("loading plan: %w", err)
	}
	if ok && p.RulesetVersion == kb.Version {
		return p, nil
	}
	return s.regenerate(ctx, kb, userID, snap)
}

// CompleteSession adapts to a completed session. It is idempotent per
// session ID (the complete endpoint may be retried, sync may replay).
func (s *Service) CompleteSession(ctx context.Context, userID uuid.UUID, sess planning.LoggedSession) ([]planning.Change, error) {
	return s.apply(ctx, userID, TriggerSession, sess.ID, planning.Event{Kind: planning.EventSession, At: s.Clock.Now(), Session: &sess})
}

// ReportPain adapts to one pain report (spec §8.6).
func (s *Service) ReportPain(ctx context.Context, userID uuid.UUID, reportID string, r planning.PainReport) ([]planning.Change, error) {
	return s.apply(ctx, userID, TriggerPain, reportID, planning.Event{Kind: planning.EventPain, At: s.Clock.Now(), Pain: &r})
}

// AnswerRedFlags applies the answers to the red-flag questions of a region
// (spec §8.2).
func (s *Service) AnswerRedFlags(ctx context.Context, userID uuid.UUID, answersID, region string, answers map[string]bool) ([]planning.Change, error) {
	return s.apply(ctx, userID, TriggerRedFlags, answersID, planning.Event{Kind: planning.EventRedFlags, At: s.Clock.Now(),
		Region: region, Answers: answers})
}

// ConfirmClearance records that the user confirmed a professional
// clearance for a region, or for everything when region is empty.
func (s *Service) ConfirmClearance(ctx context.Context, userID uuid.UUID, clearanceID, region string) ([]planning.Change, error) {
	return s.apply(ctx, userID, TriggerClearance, clearanceID, planning.Event{Kind: planning.EventClearance, At: s.Clock.Now(), Region: region})
}

// ReportSymptoms stops training after exertion symptoms (RF-10, SAFE-02).
func (s *Service) ReportSymptoms(ctx context.Context, userID uuid.UUID, reportID string) ([]planning.Change, error) {
	return s.apply(ctx, userID, TriggerSymptoms, reportID, planning.Event{Kind: planning.EventSymptoms, At: s.Clock.Now()})
}

// StartWeek runs the week-start bookkeeping once per week and user, with
// the headroom of the week that ended (PAR-S-35).
func (s *Service) StartWeek(ctx context.Context, userID uuid.UUID) ([]planning.Change, error) {
	now := s.Clock.Now()
	week := WeekStart(now)
	prev, ok, err := s.Plans.ActivePlan(ctx, userID, week.AddDate(0, 0, -7))
	if err != nil {
		return nil, fmt.Errorf("loading last week's plan: %w", err)
	}
	var headroom map[string]float64
	if ok {
		headroom = prev.Headroom
	}
	return s.apply(ctx, userID, TriggerWeek, week.Format(time.DateOnly), planning.Event{Kind: planning.EventWeek, At: week, Headroom: headroom})
}

// apply runs one event: skip it when already seen, adapt, store the
// snapshot and the changes, then regenerate the week's plan.
func (s *Service) apply(ctx context.Context, userID uuid.UUID, trigger, sourceID string, ev planning.Event) ([]planning.Change, error) {
	kb, snap, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	seen, err := s.Decisions.Seen(ctx, userID, trigger, sourceID)
	if err != nil {
		return nil, fmt.Errorf("%s: checking decision log: %w", trigger, err)
	}
	if seen {
		return nil, nil
	}
	next, changes, err := planning.Adapt(kb, snap, ev)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", trigger, err)
	}
	if err := s.Snapshots.SaveSnapshot(ctx, userID, next); err != nil {
		return nil, fmt.Errorf("%s: saving snapshot: %w", trigger, err)
	}
	if err := s.Decisions.Record(ctx, userID, trigger, sourceID, changes); err != nil {
		return nil, fmt.Errorf("%s: recording decisions: %w", trigger, err)
	}
	if _, err := s.regenerate(ctx, kb, userID, next); err != nil {
		return nil, err
	}
	return changes, nil
}

func (s *Service) load(ctx context.Context, userID uuid.UUID) (*planning.Knowledge, planning.Snapshot, error) {
	kb, err := s.Knowledge.Current(ctx)
	if err != nil {
		return nil, planning.Snapshot{}, fmt.Errorf("planning: %w", err)
	}
	snap, ok, err := s.Snapshots.Snapshot(ctx, userID)
	if err != nil {
		return nil, planning.Snapshot{}, fmt.Errorf("loading snapshot: %w", err)
	}
	if !ok {
		return nil, planning.Snapshot{}, ErrNotOnboarded
	}
	return kb, snap, nil
}

func (s *Service) regenerate(ctx context.Context, kb *planning.Knowledge, userID uuid.UUID, snap planning.Snapshot) (planning.Plan, error) {
	now := s.Clock.Now()
	p, err := planning.Generate(kb, snap, now, WeekStart(now))
	if err != nil {
		return planning.Plan{}, fmt.Errorf("generating plan: %w", err)
	}
	if err := s.Plans.SavePlan(ctx, userID, p); err != nil {
		return planning.Plan{}, fmt.Errorf("saving plan: %w", err)
	}
	if s.Log != nil {
		s.Log.InfoContext(ctx, "plan generated", "user_id", userID, "week", p.WeekStart.Format(time.DateOnly),
			"sessions", len(p.Sessions), "ruleset_version", p.RulesetVersion)
	}
	return p, nil
}

// WeekStart is the Monday 00:00 UTC of t's ISO week.
func WeekStart(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	wd := int(day.Weekday())
	if wd == 0 {
		wd = 7
	}
	return day.AddDate(0, 0, -(wd - 1))
}
