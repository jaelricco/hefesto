package planning_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
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
	return &planning.Service{Knowledge: kb, Store: store, Clock: clock, Log: quiet}, store, clock
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
	first, err := svc.CompleteSession(ctx, user, sess)
	if err != nil || first.Replayed {
		t.Fatal(first, err)
	}
	// A retried completion changes nothing and answers with the changes
	// recorded the first time (decision log).
	if again, err := svc.CompleteSession(ctx, user, sess); err != nil || !again.Replayed ||
		!reflect.DeepEqual(again.Changes, first.Changes) {
		t.Fatalf("retry: %+v %v, want the first %+v", again, err, first)
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
	if out, err := svc.StartWeek(ctx, user); err != nil || !out.Replayed {
		t.Fatalf("second week start: %+v %v", out, err)
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
	svc := &planning.Service{Knowledge: kb, Store: store, Clock: planning.SystemClock{}}
	if _, _, err := svc.Onboard(ctx, uuid.New(), answers()); !errors.Is(err, planning.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
	// In production, draft content is not accepted either (KB-13, ENT-10).
	prod := planning.LoadContentKnowledge("../../content", true, quiet)
	if _, err := prod.Current(ctx); !errors.Is(err, planning.ErrUnavailable) {
		t.Errorf("draft knowledge base accepted in production: %v", err)
	}
}

func TestServiceOnboardsOnce(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newService(t)
	user := uuid.New()
	if _, _, err := svc.Onboard(ctx, user, answers()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Onboard(ctx, user, answers()); !errors.Is(err, planning.ErrAlreadyOnboarded) {
		t.Fatalf("second onboarding: %v", err)
	}
	// Open questions do not hide that the user is onboarded.
	a := answers()
	a.TrainingLevel = domain.LevelSedentary
	a.Stages = map[string]domain.StageAnswer{"front-lever": {Level: "full", Class: "4_9"}}
	if _, _, err := svc.Onboard(ctx, user, a); !errors.Is(err, planning.ErrAlreadyOnboarded) {
		t.Fatalf("second onboarding with questions: %v", err)
	}
}

func TestServiceWeeksSessionsAndRegeneration(t *testing.T) {
	ctx := context.Background()
	svc, _, clock := newService(t)
	user := uuid.New()
	_, first, err := svc.Onboard(ctx, user, answers())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.Sessions[0].ID == "" {
		t.Fatalf("plan without IDs: %q %q", first.ID, first.Sessions[0].ID)
	}
	if p, err := svc.WeekPlan(ctx, user, clock.t.AddDate(0, 0, 2)); err != nil || p.ID != first.ID {
		t.Fatalf("this week's plan: %v %v", p.ID, err)
	}
	for _, week := range []time.Time{clock.t.AddDate(0, 0, -7), clock.t.AddDate(0, 0, 7)} {
		if _, err := svc.WeekPlan(ctx, user, week); !errors.Is(err, planning.ErrNotFound) {
			t.Errorf("week of %s: %v, want not found", week.Format(time.DateOnly), err)
		}
	}
	sid := uuid.MustParse(first.Sessions[1].ID)
	plan, ps, err := svc.PlannedSession(ctx, user, sid)
	if err != nil || plan.ID != first.ID || ps.Index != first.Sessions[1].Index {
		t.Fatalf("planned session: %v %v %v", plan.ID, ps.Index, err)
	}

	// Regenerating keeps the plan's content and replaces its handles.
	again, err := svc.Regenerate(ctx, user)
	if err != nil || again.InputHash != first.InputHash || again.ID == first.ID {
		t.Fatalf("regenerated %s (%s), first %s (%s): %v", again.ID, again.InputHash, first.ID, first.InputHash, err)
	}
	if _, _, err := svc.PlannedSession(ctx, user, sid); !errors.Is(err, planning.ErrNotFound) {
		t.Errorf("a session of the replaced plan: %v", err)
	}
}

func TestServiceProfileAndGoals(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newService(t)
	user := uuid.New()
	if _, err := svc.UpdateProfile(ctx, user, domain.ProfileUpdate{}); !errors.Is(err, planning.ErrNotOnboarded) {
		t.Fatalf("before onboarding: %v", err)
	}
	if _, _, err := svc.Onboard(ctx, user, answers()); err != nil {
		t.Fatal(err)
	}
	prof, err := svc.UpdateProfile(ctx, user, domain.ProfileUpdate{SessionsPerWeek: 2, SessionMinutes: 45,
		Equipment: []string{"gym"}, BodyweightKg: 80})
	if err != nil || prof.SessionsPerWeek != 2 || prof.BodyweightKg != 80 {
		t.Fatalf("profile: %+v %v", prof, err)
	}
	if p, err := svc.Plan(ctx, user); err != nil || len(p.Sessions) > 2 {
		t.Fatalf("the plan does not follow the profile: %d sessions, %v", len(p.Sessions), err)
	}
	var ve *domain.ValidationError
	if _, err := svc.UpdateProfile(ctx, user, domain.ProfileUpdate{SessionsPerWeek: 9, SessionMinutes: 45, BodyweightKg: 80}); !errors.As(err, &ve) {
		t.Fatalf("invalid profile: %v", err)
	}

	goals, _, err := svc.SetGoals(ctx, user, []domain.Goal{
		{Skill: "handstand", TargetLevel: "free-30s", Priority: 2},
		{Skill: "front-lever", TargetLevel: "full", Priority: 1},
	})
	if err != nil || len(goals) != 2 || goals[0].Skill != "front-lever" {
		t.Fatalf("goals: %+v %v", goals, err)
	}
	if _, _, err := svc.SetGoals(ctx, user, nil); !errors.As(err, &ve) {
		t.Fatalf("no goals: %v", err)
	}
	_, snap, err := svc.View(ctx, user)
	if err != nil || len(snap.Goals) != 2 || snap.Profile.SessionsPerWeek != 2 {
		t.Fatalf("view: %+v %v", snap.Goals, err)
	}
}

func TestServicePainNeedsConsentAndValidInput(t *testing.T) {
	ctx := context.Background()
	svc, _, clock := newService(t)
	without, with := uuid.New(), uuid.New()
	a := answers()
	a.HealthConsent = false
	if _, _, err := svc.Onboard(ctx, without, a); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Onboard(ctx, with, answers()); err != nil {
		t.Fatal(err)
	}
	report := domain.PainReport{Region: "elbow_inner", Timepoint: domain.PainDuring, NRS: 2, At: clock.t}
	if _, err := svc.ReportPain(ctx, without, uuid.NewString(), report); !errors.Is(err, planning.ErrConsentRequired) {
		t.Fatalf("pain without consent: %v", err)
	}
	var ve *domain.ValidationError
	bad := report
	bad.Region, bad.NRS = "toe", 11
	if _, err := svc.ReportPain(ctx, with, uuid.NewString(), bad); !errors.As(err, &ve) || len(ve.Fields) != 2 {
		t.Fatalf("invalid pain report: %v", err)
	}
	ids := []string{uuid.NewString(), uuid.NewString()}
	for i, id := range ids {
		r := report
		r.At = clock.t.Add(time.Duration(i) * time.Hour)
		if _, err := svc.ReportPain(ctx, with, id, r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.PainReports(ctx, with, nil, 1)
	if err != nil || len(got) != 1 || got[0].ID != ids[1] {
		t.Fatalf("newest pain report: %+v %v", got, err)
	}
	at := got[0].At
	rest, err := svc.PainReports(ctx, with, &planning.Cursor{At: at, ID: uuid.MustParse(ids[1])}, 10)
	if err != nil || len(rest) != 1 || rest[0].ID != ids[0] {
		t.Fatalf("next page: %+v %v", rest, err)
	}

	// Red-flag answers name every question asked for the region.
	if _, err := svc.AnswerRedFlags(ctx, with, "rf-1", "elbow_inner", map[string]bool{}); !errors.As(err, &ve) {
		t.Fatalf("unanswered red flags: %v", err)
	}
}

func TestServiceDecisionsNewestFirst(t *testing.T) {
	ctx := context.Background()
	svc, _, clock := newService(t)
	user := uuid.New()
	if _, _, err := svc.Onboard(ctx, user, answers()); err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"sym-1", "sym-2", "sym-3"} {
		clock.t = clock.t.Add(time.Duration(i) * time.Minute)
		if _, err := svc.ReportSymptoms(ctx, user, id); err != nil {
			t.Fatal(err)
		}
	}
	page, err := svc.Decisions(ctx, user, nil, 2)
	if err != nil || len(page) != 2 || page[0].SourceID != "sym-3" || page[1].SourceID != "sym-2" {
		t.Fatalf("first page: %+v %v", page, err)
	}
	last := page[1]
	page, err = svc.Decisions(ctx, user, &planning.Cursor{At: last.At, ID: last.ID}, 2)
	if err != nil || len(page) != 1 || page[0].SourceID != "sym-1" || len(page[0].Changes) == 0 {
		t.Fatalf("second page: %+v %v", page, err)
	}
}

func TestServiceStartPlannedSession(t *testing.T) {
	ctx := context.Background()
	svc, store, clock := newService(t)
	user := uuid.New()
	_, p, err := svc.Onboard(ctx, user, answers())
	if err != nil {
		t.Fatal(err)
	}
	ps := p.Sessions[0]
	if ps.Status != planning.SessionPlanned {
		t.Fatalf("a new session is %q", ps.Status)
	}
	for _, b := range ps.Blocks {
		for _, it := range b.Items {
			if _, err := uuid.Parse(it.ID); err != nil {
				t.Fatalf("item %s without ID: %q", it.Exercise, it.ID)
			}
		}
	}
	planned := uuid.MustParse(ps.ID)
	in := planning.SessionStart{SessionID: uuid.New(), StartedAt: clock.t, Timezone: "Europe/Zurich", LocalDate: clock.t, At: clock.t}

	if _, _, err := svc.StartPlannedSession(ctx, user, uuid.New(), in); !errors.Is(err, planning.ErrNotFound) {
		t.Errorf("an unknown planned session: %v", err)
	}
	id, created, err := svc.StartPlannedSession(ctx, user, planned, in)
	if err != nil || !created || id != in.SessionID {
		t.Fatalf("start: %v %v %v", id, created, err)
	}
	got, ok := store.Started(id)
	if !ok || got.PlannedSessionID != planned || len(got.Draft.Blocks) == 0 {
		t.Fatalf("stored start: %+v", got)
	}
	// A second start, even with another ID, answers with the first session.
	other := in
	other.SessionID = uuid.New()
	if again, created, err := svc.StartPlannedSession(ctx, user, planned, other); err != nil || created || again != id {
		t.Errorf("second start: %v %v %v", again, created, err)
	}
	if _, now, err := svc.PlannedSession(ctx, user, planned); err != nil || now.Status != planning.SessionStarted ||
		now.WorkoutSessionID != id.String() {
		t.Errorf("planned session after the start: %q %q %v", now.Status, now.WorkoutSessionID, err)
	}
	// Another planned session cannot take the log session's ID.
	next := uuid.MustParse(p.Sessions[1].ID)
	if _, _, err := svc.StartPlannedSession(ctx, user, next, in); !errors.Is(err, planning.ErrSessionIDTaken) {
		t.Errorf("a taken ID: %v", err)
	}

	// A new plan keeps the start on the same day.
	again, err := svc.Regenerate(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if s := again.Sessions[0]; s.ID == ps.ID || s.Status != planning.SessionStarted || s.WorkoutSessionID != id.String() {
		t.Errorf("regenerated session: %s %q %q", s.ID, s.Status, s.WorkoutSessionID)
	}
	for _, s := range again.Sessions[1:] {
		if s.Status != planning.SessionPlanned || s.WorkoutSessionID != "" {
			t.Errorf("session on %s: %q %q", s.Date.Format(time.DateOnly), s.Status, s.WorkoutSessionID)
		}
	}

	// A stop starts nothing (SAFE-02).
	if _, err := svc.ReportSymptoms(ctx, user, "sym-1"); err != nil {
		t.Fatal(err)
	}
	stopped, err := svc.Plan(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.StartPlannedSession(ctx, user, next, planning.SessionStart{SessionID: uuid.New()})
	var se *planning.StoppedError
	if !errors.Is(err, planning.ErrTrainingStopped) || !errors.As(err, &se) || se.Rule != domain.RuleStopped {
		t.Errorf("start while stopped: %v (plan has %d sessions)", err, len(stopped.Sessions))
	}
}

// followsPlan checks that a started draft holds exactly the sets the planned
// session of its day asks for: nothing is performed in the memory store, so
// every set is still open.
func followsPlan(t *testing.T, kb *domain.Knowledge, store *memory.Store, logID uuid.UUID, p domain.Plan) []string {
	t.Helper()
	sig := func(s domain.DraftSet) string {
		v := func(p *int) int {
			if p == nil {
				return -1
			}
			return *p
		}
		return fmt.Sprintf("%s %s %s %d %d %.2f %d %d", s.ItemID, s.Exercise, s.Kind, v(s.Reps), v(s.HoldS), s.LoadKg, s.RestS, v(s.RIR))
	}
	var want, got, ids []string
	for _, ps := range p.Sessions {
		if ps.WorkoutSessionID == logID.String() {
			for _, b := range domain.Materialize(kb, ps, func() string { return "" }).Blocks {
				for _, s := range b.Sets {
					want = append(want, sig(s))
				}
			}
		}
	}
	in, _ := store.Started(logID)
	for _, b := range in.Draft.Blocks {
		for _, s := range b.Sets {
			got = append(got, sig(s))
			ids = append(ids, s.ID)
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(want, got) {
		t.Errorf("the draft does not follow the plan:\n want %v\n got  %v", want, got)
	}
	return ids
}

// A started draft follows every new plan of the week (ADR 0017).
func TestServiceStartedDraftFollowsThePlan(t *testing.T) {
	ctx := context.Background()
	svc, store, clock := newService(t)
	kb, err := svc.Knowledge.Current(ctx)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.New()
	_, p, err := svc.Onboard(ctx, user, answers())
	if err != nil {
		t.Fatal(err)
	}
	logID := uuid.New()
	if _, _, err := svc.StartPlannedSession(ctx, user, uuid.MustParse(p.Sessions[0].ID),
		planning.SessionStart{SessionID: logID, StartedAt: clock.t, Timezone: "UTC", LocalDate: clock.t, At: clock.t}); err != nil {
		t.Fatal(err)
	}
	started, err := svc.Plan(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	before := followsPlan(t, kb, store, logID, started)

	// The same plan again: the sets stay, only their items are new.
	again, err := svc.Regenerate(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	if ids := followsPlan(t, kb, store, logID, again); !slices.Equal(ids, before) {
		t.Errorf("an unchanged plan rewrote the draft: %v → %v", before, ids)
	}

	// Goal and profile changes move the plan; the draft follows each of them.
	changed := false
	for _, change := range []func() error{
		func() error {
			_, _, err := svc.SetGoals(ctx, user, []domain.Goal{{Skill: "planche", TargetLevel: "full", Priority: 1}})
			return err
		},
		func() error {
			_, err := svc.UpdateProfile(ctx, user, domain.ProfileUpdate{SessionsPerWeek: 2, SessionMinutes: 30,
				Equipment: []string{"pull_up_bar"}, BodyweightKg: 75})
			return err
		},
		func() error {
			_, _, err := svc.SetGoals(ctx, user, []domain.Goal{{Skill: "front-lever", TargetLevel: "full", Priority: 1}})
			return err
		},
	} {
		if err := change(); err != nil {
			t.Fatal(err)
		}
		now, err := svc.Plan(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		ids := followsPlan(t, kb, store, logID, now)
		changed = changed || !slices.Equal(ids, before)
		before = ids
	}
	if !changed {
		t.Error("no change adjusted the draft; the test exercises nothing")
	}

	// A stop empties the draft and says so; a retry answers the same.
	out, err := svc.ReportSymptoms(ctx, user, "sym-1")
	if err != nil {
		t.Fatal(err)
	}
	var adjusted bool
	for _, c := range out.Changes {
		adjusted = adjusted || c.Kind == domain.ChangeAdjusted && c.Session == logID.String()
	}
	if in, _ := store.Started(logID); !adjusted || len(in.Draft.Blocks) != 0 {
		t.Errorf("after a stop: adjusted=%v, %d blocks left", adjusted, len(in.Draft.Blocks))
	}
	if replay, err := svc.ReportSymptoms(ctx, user, "sym-1"); err != nil || !replay.Replayed || !reflect.DeepEqual(replay.Changes, out.Changes) {
		t.Errorf("replay: %+v %v", replay, err)
	}
}
