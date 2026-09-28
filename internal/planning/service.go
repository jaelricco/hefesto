package planning

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// Service runs the planner for one user at a time. It holds no state of
// its own; every call reads the snapshot, applies the pure core and writes
// the result, inside one transaction of Store.
type Service struct {
	Knowledge KnowledgeSource
	Store     Transactor
	Clock     Clock
	IDs       IDSource // UUIDv7 when nil
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
	// The pain baseline from the onboarding is the first row of the pain
	// history; the planner gives it its ID.
	for i := range snap.Pain {
		if snap.Pain[i].ID == "" {
			snap.Pain[i].ID = s.newID().String()
		}
	}
	var p *planning.Plan
	err = s.Store.InTx(ctx, userID, func(st Stores) error {
		if _, ok, err := st.Snapshots.Snapshot(ctx, userID); err != nil {
			return fmt.Errorf("onboarding: loading snapshot: %w", err)
		} else if ok {
			return ErrAlreadyOnboarded
		}
		if res.Status == planning.OnboardingNeedsAnswers {
			return nil
		}
		if err := st.Snapshots.SaveSnapshot(ctx, userID, snap); err != nil {
			return fmt.Errorf("onboarding: saving snapshot: %w", err)
		}
		plan, err := s.regenerate(ctx, st, kb, userID, snap)
		p = &plan
		return err
	})
	if err != nil {
		return planning.OnboardingResult{}, nil, err
	}
	return res, p, nil
}

// Plan returns the plan of the current week, generating it on first use.
// Plans change only on events (spec §11.3).
func (s *Service) Plan(ctx context.Context, userID uuid.UUID) (planning.Plan, error) {
	var p planning.Plan
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		kb, snap, err := s.load(ctx, st, userID)
		if err != nil {
			return err
		}
		week := WeekStart(s.Clock.Now())
		stored, ok, err := st.Plans.ActivePlan(ctx, userID, week)
		if err != nil {
			return fmt.Errorf("loading plan: %w", err)
		}
		if ok && stored.RulesetVersion == kb.Version {
			p = stored
			return nil
		}
		p, err = s.regenerate(ctx, st, kb, userID, snap)
		return err
	})
	return p, err
}

// WeekPlan returns the plan of the week that starts on week's Monday: the
// current week's as Plan does, an earlier week's as stored. There is no
// plan of a later week yet (ErrNotFound).
func (s *Service) WeekPlan(ctx context.Context, userID uuid.UUID, week time.Time) (planning.Plan, error) {
	week = WeekStart(week)
	if week.Equal(WeekStart(s.Clock.Now())) {
		return s.Plan(ctx, userID)
	}
	var p planning.Plan
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		if _, _, err := s.load(ctx, st, userID); err != nil {
			return err
		}
		stored, ok, err := st.Plans.ActivePlan(ctx, userID, week)
		if err != nil {
			return fmt.Errorf("loading plan: %w", err)
		}
		if !ok {
			return fmt.Errorf("plan of the week of %s: %w", week.Format(time.DateOnly), ErrNotFound)
		}
		p = stored
		return nil
	})
	return p, err
}

// Regenerate generates the current week's plan anew. The same snapshot on
// the same day gives the same sessions (spec §1.3).
func (s *Service) Regenerate(ctx context.Context, userID uuid.UUID) (planning.Plan, error) {
	var p planning.Plan
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		kb, snap, err := s.load(ctx, st, userID)
		if err != nil {
			return err
		}
		p, err = s.regenerate(ctx, st, kb, userID, snap)
		return err
	})
	return p, err
}

// PlannedSession returns a session of an active plan with the plan it
// belongs to.
func (s *Service) PlannedSession(ctx context.Context, userID, sessionID uuid.UUID) (planning.Plan, planning.PlannedSession, error) {
	var (
		p planning.Plan
		i int
	)
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		var (
			ok  bool
			err error
		)
		p, i, ok, err = st.Plans.PlannedSession(ctx, userID, sessionID)
		if err != nil {
			return fmt.Errorf("loading planned session: %w", err)
		}
		if !ok || i < 0 || i >= len(p.Sessions) {
			return fmt.Errorf("planned session %s: %w", sessionID, ErrNotFound)
		}
		return nil
	})
	if err != nil {
		return planning.Plan{}, planning.PlannedSession{}, err
	}
	return p, p.Sessions[i], nil
}

// Decisions returns up to limit applied events, newest first.
func (s *Service) Decisions(ctx context.Context, userID uuid.UUID, after *Cursor, limit int) ([]Decision, error) {
	var out []Decision
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		var err error
		out, err = st.Decisions.ListDecisions(ctx, userID, after, limit)
		if err != nil {
			return fmt.Errorf("listing decisions: %w", err)
		}
		return nil
	})
	return out, err
}

// View returns the knowledge base and the user's snapshot for reading.
func (s *Service) View(ctx context.Context, userID uuid.UUID) (*planning.Knowledge, planning.Snapshot, error) {
	var (
		kb   *planning.Knowledge
		snap planning.Snapshot
	)
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		var err error
		kb, snap, err = s.load(ctx, st, userID)
		return err
	})
	return kb, snap, err
}

// PainReports returns up to limit stored pain reports, newest first.
func (s *Service) PainReports(ctx context.Context, userID uuid.UUID, after *Cursor, limit int) ([]planning.PainReport, error) {
	_, snap, err := s.View(ctx, userID)
	if err != nil {
		return nil, err
	}
	type entry struct {
		r  planning.PainReport
		id uuid.UUID
	}
	all := make([]entry, 0, len(snap.Pain))
	for _, r := range snap.Pain {
		id, err := uuid.Parse(r.ID)
		if err != nil {
			return nil, fmt.Errorf("pain report %q: %w", r.ID, err)
		}
		all = append(all, entry{r, id})
	}
	slices.SortFunc(all, func(a, b entry) int {
		return cmp.Or(b.r.At.Compare(a.r.At), bytes.Compare(b.id[:], a.id[:]))
	})
	out := []planning.PainReport{}
	for _, e := range all {
		if after != nil && !before(e.r.At, e.id, *after) {
			continue
		}
		if len(out) == limit {
			break
		}
		out = append(out, e.r)
	}
	return out, nil
}

// before reports whether (at, id) comes after the cursor in a list ordered
// newest first.
func before(at time.Time, id uuid.UUID, c Cursor) bool {
	if !at.Equal(c.At) {
		return at.Before(c.At)
	}
	return bytes.Compare(id[:], c.ID[:]) < 0
}

// UpdateProfile applies a profile change and generates the plan anew
// (WEEK-08).
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, u planning.ProfileUpdate) (planning.Profile, error) {
	var out planning.Profile
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		kb, snap, err := s.load(ctx, st, userID)
		if err != nil {
			return err
		}
		next, err := planning.UpdateProfile(snap, u)
		if err != nil {
			return fmt.Errorf("profile: %w", err)
		}
		if err := s.save(ctx, st, kb, userID, next); err != nil {
			return err
		}
		out = next.Profile
		return nil
	})
	return out, err
}

// SetGoals replaces the goals and generates the plan anew (WEEK-08). It
// returns the goals in priority order.
func (s *Service) SetGoals(ctx context.Context, userID uuid.UUID, goals []planning.Goal) ([]planning.Goal, []planning.Realism, error) {
	var (
		out     []planning.Goal
		realism []planning.Realism
	)
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		kb, snap, err := s.load(ctx, st, userID)
		if err != nil {
			return err
		}
		next, err := planning.SetGoals(kb, snap, goals, s.Clock.Now())
		if err != nil {
			return fmt.Errorf("goals: %w", err)
		}
		if err := s.save(ctx, st, kb, userID, next); err != nil {
			return err
		}
		out, realism = next.Goals, kb.Realism(next, s.Clock.Now())
		return nil
	})
	return out, realism, err
}

// save stores a snapshot the user changed and generates the plan anew.
func (s *Service) save(ctx context.Context, st Stores, kb *planning.Knowledge, userID uuid.UUID, next planning.Snapshot) error {
	if err := st.Snapshots.SaveSnapshot(ctx, userID, next); err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}
	_, err := s.regenerate(ctx, st, kb, userID, next)
	return err
}

// Outcome is what an event changed. An event the service applied before is
// not applied again: Replayed is set and Changes are the ones recorded
// then, so a retried request gets the same answer.
type Outcome struct {
	Changes  []planning.Change
	Replayed bool
}

// CompleteSession adapts to a completed session. It is idempotent per
// session ID (the complete endpoint may be retried, sync may replay).
func (s *Service) CompleteSession(ctx context.Context, userID uuid.UUID, sess planning.LoggedSession) (Outcome, error) {
	return s.apply(ctx, userID, TriggerSession, sess.ID, planning.Event{Kind: planning.EventSession, At: s.Clock.Now(), Session: &sess}, nil)
}

// ReportPain adapts to one pain report (spec §8.6); reportID becomes the
// report's ID in the pain history. Pain reports are health data and need
// the user's consent (ErrConsentRequired).
func (s *Service) ReportPain(ctx context.Context, userID uuid.UUID, reportID string, r planning.PainReport) (Outcome, error) {
	r.ID = reportID
	return s.apply(ctx, userID, TriggerPain, reportID, planning.Event{Kind: planning.EventPain, At: s.Clock.Now(), Pain: &r},
		func(kb *planning.Knowledge, snap planning.Snapshot) error {
			if !snap.Profile.HealthConsent {
				return ErrConsentRequired
			}
			return kb.ValidatePain(r, s.Clock.Now())
		})
}

// AnswerRedFlags applies the answers to the red-flag questions of a region
// (spec §8.2). Every question asked for the region must be answered.
func (s *Service) AnswerRedFlags(ctx context.Context, userID uuid.UUID, answersID, region string, answers map[string]bool) (Outcome, error) {
	return s.apply(ctx, userID, TriggerRedFlags, answersID, planning.Event{Kind: planning.EventRedFlags, At: s.Clock.Now(),
		Region: region, Answers: answers},
		func(kb *planning.Knowledge, snap planning.Snapshot) error {
			return kb.ValidateRedFlags(region, answers, kb.IsMinor(snap.Profile.BirthYear, s.Clock.Now()))
		})
}

// ConfirmClearance records that the user confirmed a professional
// clearance for a region, or for everything when region is empty.
func (s *Service) ConfirmClearance(ctx context.Context, userID uuid.UUID, clearanceID, region string) (Outcome, error) {
	return s.apply(ctx, userID, TriggerClearance, clearanceID, planning.Event{Kind: planning.EventClearance, At: s.Clock.Now(), Region: region}, nil)
}

// ReportSymptoms stops training after exertion symptoms (RF-10, SAFE-02).
func (s *Service) ReportSymptoms(ctx context.Context, userID uuid.UUID, reportID string) (Outcome, error) {
	return s.apply(ctx, userID, TriggerSymptoms, reportID, planning.Event{Kind: planning.EventSymptoms, At: s.Clock.Now()}, nil)
}

// StartWeek runs the week-start bookkeeping once per week and user, with
// the headroom of the week that ended (PAR-S-35).
func (s *Service) StartWeek(ctx context.Context, userID uuid.UUID) (Outcome, error) {
	week := WeekStart(s.Clock.Now())
	var out Outcome
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		prev, ok, err := st.Plans.ActivePlan(ctx, userID, week.AddDate(0, 0, -7))
		if err != nil {
			return fmt.Errorf("loading last week's plan: %w", err)
		}
		var headroom map[string]float64
		if ok {
			headroom = prev.Headroom
		}
		out, err = s.applyIn(ctx, st, userID, TriggerWeek, week.Format(time.DateOnly),
			planning.Event{Kind: planning.EventWeek, At: week, Headroom: headroom}, nil)
		return err
	})
	return out, err
}

// check validates an event against the knowledge base and the snapshot
// before it is applied.
type check func(*planning.Knowledge, planning.Snapshot) error

// apply runs one event in its own transaction.
func (s *Service) apply(ctx context.Context, userID uuid.UUID, trigger, sourceID string, ev planning.Event, chk check) (Outcome, error) {
	var out Outcome
	err := s.Store.InTx(ctx, userID, func(st Stores) error {
		var err error
		out, err = s.applyIn(ctx, st, userID, trigger, sourceID, ev, chk)
		return err
	})
	return out, err
}

// applyIn runs one event: answer from the decision log when it was applied
// before, else check, adapt, store the snapshot and the changes, then
// regenerate the week's plan.
func (s *Service) applyIn(ctx context.Context, st Stores, userID uuid.UUID, trigger, sourceID string, ev planning.Event, chk check) (Outcome, error) {
	kb, snap, err := s.load(ctx, st, userID)
	if err != nil {
		return Outcome{}, err
	}
	prev, seen, err := st.Decisions.Recorded(ctx, userID, trigger, sourceID)
	if err != nil {
		return Outcome{}, fmt.Errorf("%s: checking decision log: %w", trigger, err)
	}
	if seen {
		return Outcome{Changes: prev.Changes, Replayed: true}, nil
	}
	if chk != nil {
		if err := chk(kb, snap); err != nil {
			return Outcome{}, fmt.Errorf("%s: %w", trigger, err)
		}
	}
	next, changes, err := planning.Adapt(kb, snap, ev)
	if err != nil {
		return Outcome{}, fmt.Errorf("%s: %w", trigger, err)
	}
	if err := st.Snapshots.SaveSnapshot(ctx, userID, next); err != nil {
		return Outcome{}, fmt.Errorf("%s: saving snapshot: %w", trigger, err)
	}
	d := Decision{ID: s.newID(), Trigger: trigger, SourceID: sourceID, At: s.Clock.Now(), Changes: changes}
	if err := st.Decisions.Record(ctx, userID, d); err != nil {
		return Outcome{}, fmt.Errorf("%s: recording decisions: %w", trigger, err)
	}
	if _, err := s.regenerate(ctx, st, kb, userID, next); err != nil {
		return Outcome{}, err
	}
	return Outcome{Changes: changes}, nil
}

func (s *Service) load(ctx context.Context, st Stores, userID uuid.UUID) (*planning.Knowledge, planning.Snapshot, error) {
	kb, err := s.Knowledge.Current(ctx)
	if err != nil {
		return nil, planning.Snapshot{}, fmt.Errorf("planning: %w", err)
	}
	snap, ok, err := st.Snapshots.Snapshot(ctx, userID)
	if err != nil {
		return nil, planning.Snapshot{}, fmt.Errorf("loading snapshot: %w", err)
	}
	if !ok {
		return nil, planning.Snapshot{}, ErrNotOnboarded
	}
	return kb, snap, nil
}

func (s *Service) regenerate(ctx context.Context, st Stores, kb *planning.Knowledge, userID uuid.UUID, snap planning.Snapshot) (planning.Plan, error) {
	now := s.Clock.Now()
	p, err := planning.Generate(kb, snap, now, WeekStart(now))
	if err != nil {
		return planning.Plan{}, fmt.Errorf("generating plan: %w", err)
	}
	p.ID = s.newID().String()
	for i := range p.Sessions {
		p.Sessions[i].ID = s.newID().String()
	}
	if err := st.Plans.SavePlan(ctx, userID, p); err != nil {
		return planning.Plan{}, fmt.Errorf("saving plan: %w", err)
	}
	if s.Log != nil {
		s.Log.InfoContext(ctx, "plan generated", "user_id", userID, "week", p.WeekStart.Format(time.DateOnly),
			"sessions", len(p.Sessions), "ruleset_version", p.RulesetVersion)
	}
	return p, nil
}

func (s *Service) newID() uuid.UUID {
	if s.IDs == nil {
		return UUIDv7{}.New()
	}
	return s.IDs.New()
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
