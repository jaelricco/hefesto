package planning_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/planning"
	"github.com/jaelricco/hefesto/internal/planning/memory"
)

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now() time.Time { return c.t }

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func newService(t *testing.T) (*planning.Service, *memory.Store, *fixedClock) {
	t.Helper()
	kb := planning.LoadContentKnowledge("../../content", false, quiet)
	if _, err := kb.Current(context.Background()); err != nil {
		t.Fatal(err)
	}
	store := memory.New()
	clock := &fixedClock{t: time.Date(2026, time.September, 28, 7, 0, 0, 0, time.UTC)}
	return &planning.Service{Knowledge: kb, Snapshots: store, Plans: store, Decisions: store, Clock: clock, Log: quiet}, store, clock
}

func answers() domain.Answers {
	return domain.Answers{
		BirthYear: 1995, HealthConsent: true, DisclaimerAck: true, BodyweightKg: 75, DataConfidence: "estimated",
		Screening:       []bool{false, false, false, false, false, false},
		Goals:           []domain.Goal{{Skill: "front-lever", TargetLevel: "full", Priority: 1}},
		SessionsPerWeek: 3, SessionMinutes: 60,
		Equipment:     []string{"gym"},
		TrainingLevel: domain.LevelTrained, TrainingAge: "1_to_4_years", LastRegular: "current_or_lt_3_weeks",
		Classes: map[string]string{"pull_up_class": "12_15", "push_up_class": "21_30"},
		Stages:  map[string]domain.StageAnswer{"front-lever": {Level: "tuck", Class: "10_19"}},
	}
}

func TestServiceOnboardPlanAndComplete(t *testing.T) {
	ctx := context.Background()
	svc, store, clock := newService(t)
	user := uuid.New()

	if _, err := svc.Plan(ctx, user); !errors.Is(err, planning.ErrNotOnboarded) {
		t.Fatalf("plan before onboarding: %v", err)
	}
	res, p, err := svc.Onboard(ctx, user, answers())
	if err != nil || res.Status != domain.OnboardingComplete || p == nil || len(p.Sessions) == 0 {
		t.Fatalf("onboarding: %v %+v %v", err, res, p)
	}
	again, err := svc.Plan(ctx, user)
	if err != nil || again.InputHash != p.InputHash {
		t.Fatalf("stored plan not returned: %v", err)
	}

	// Perform the first planned session.
	ps := p.Sessions[0]
	sess := domain.LoggedSession{ID: "s-1", Date: ps.Date}
	for _, b := range ps.Blocks {
		for _, it := range b.Items {
			if it.Kind != domain.KindWorking || it.Offer || it.Stimulus == domain.StimPrehab {
				continue
			}
			v := float64(it.Reps)
			if it.HoldS > 0 {
				v = float64(it.HoldS)
			}
			r := 2.0
			sess.Sets = append(sess.Sets, domain.LoggedSet{ID: "x", Exercise: it.Exercise, Kind: it.Kind, Assist: it.Assist, Value: v, Reserve: &r})
		}
	}
	clock.t = ps.Date.Add(20 * time.Hour)
	if _, err := svc.CompleteSession(ctx, user, sess); err != nil {
		t.Fatal(err)
	}
	// A retried completion changes nothing (decision log).
	if cs, err := svc.CompleteSession(ctx, user, sess); err != nil || cs != nil {
		t.Fatalf("retry: %v %v", cs, err)
	}
	if n := len(store.Decisions(user)); n != 1 {
		t.Errorf("%d decisions recorded, want 1", n)
	}
	snap, _, _ := store.Snapshot(ctx, user)
	if len(snap.History) != 1 {
		t.Errorf("history %d, want 1", len(snap.History))
	}

	// Next Monday: the week starts once.
	clock.t = time.Date(2026, time.October, 5, 6, 0, 0, 0, time.UTC)
	if _, err := svc.StartWeek(ctx, user); err != nil {
		t.Fatal(err)
	}
	if cs, err := svc.StartWeek(ctx, user); err != nil || cs != nil {
		t.Fatalf("second week start: %v %v", cs, err)
	}
	next, err := svc.Plan(ctx, user)
	if err != nil || !next.WeekStart.Equal(planning.WeekStart(clock.t)) {
		t.Fatalf("plan of the new week: %v %v", next.WeekStart, err)
	}
}

func TestServiceSymptomsStopAndClearance(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newService(t)
	user := uuid.New()
	if _, _, err := svc.Onboard(ctx, user, answers()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReportSymptoms(ctx, user, "sym-1"); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Plan(ctx, user)
	if err != nil || !p.Stopped {
		t.Fatalf("plan after symptoms: stopped=%v %v", p.Stopped, err)
	}
	if _, err := svc.ConfirmClearance(ctx, user, "clr-1", ""); err != nil {
		t.Fatal(err)
	}
	if p, _ := svc.Plan(ctx, user); p.Stopped {
		t.Error("still stopped after the global clearance")
	}
}

func TestServiceOnboardingQuestionsStoreNothing(t *testing.T) {
	ctx := context.Background()
	svc, store, _ := newService(t)
	user := uuid.New()
	a := answers()
	a.TrainingLevel = domain.LevelSedentary
	a.Stages = map[string]domain.StageAnswer{"front-lever": {Level: "full", Class: "4_9"}}
	res, p, err := svc.Onboard(ctx, user, a)
	if err != nil || res.Status != domain.OnboardingNeedsAnswers || p != nil {
		t.Fatalf("%v %+v %v", err, res, p)
	}
	if _, ok, _ := store.Snapshot(ctx, user); ok {
		t.Error("snapshot stored while questions are open")
	}
}

func TestServiceUnavailableKnowledge(t *testing.T) {
	ctx := context.Background()
	kb := planning.LoadContentKnowledge(t.TempDir(), false, quiet)
	store := memory.New()
	svc := &planning.Service{Knowledge: kb, Snapshots: store, Plans: store, Decisions: store, Clock: planning.SystemClock{}}
	if _, _, err := svc.Onboard(ctx, uuid.New(), answers()); !errors.Is(err, planning.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
	// In production, draft content is not accepted either (KB-13, ENT-10).
	prod := planning.LoadContentKnowledge("../../content", true, quiet)
	if _, err := prod.Current(ctx); !errors.Is(err, planning.ErrUnavailable) {
		t.Errorf("draft knowledge base accepted in production: %v", err)
	}
}
