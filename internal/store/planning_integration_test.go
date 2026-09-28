//go:build integration

package store_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/planning"
	"github.com/jaelricco/hefesto/internal/planning/memory"
	"github.com/jaelricco/hefesto/internal/store"
	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

// These tests pin the planner's persistence (ADR 0013): a snapshot reads
// back exactly as it was written, plans and decisions keep their rules, and
// the service behaves the same on Postgres as in memory.

var monday = time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)

func plannerUser(t *testing.T, db *pgxpool.Pool) uuid.UUID {
	t.Helper()
	u := id()
	exec(t, db, `INSERT INTO users (id, email) VALUES ($1, $2)`, u, u.String()+"@example.test")
	return u
}

// workoutSession inserts the log row a planner session refers to.
func workoutSession(t *testing.T, db *pgxpool.Pool, user, session uuid.UUID, day time.Time) {
	t.Helper()
	exec(t, db, `INSERT INTO workout_sessions (id, user_id, started_at, timezone, local_date, updated_at)
	             VALUES ($1, $2, $3::timestamptz, 'UTC', ($3::timestamptz AT TIME ZONE 'UTC')::date, $3::timestamptz)`,
		session, user, day)
}

func inTx(t *testing.T, p *store.Planner, user uuid.UUID, fn func(planning.Stores) error) {
	t.Helper()
	if err := p.InTx(context.Background(), user, fn); err != nil {
		t.Fatal(err)
	}
}

func loadSnapshot(t *testing.T, p *store.Planner, user uuid.UUID) domain.Snapshot {
	t.Helper()
	var s domain.Snapshot
	inTx(t, p, user, func(st planning.Stores) error {
		var ok bool
		var err error
		s, ok, err = st.Snapshots.Snapshot(context.Background(), user)
		if err == nil && !ok {
			err = errors.New("no snapshot")
		}
		return err
	})
	return s
}

func saveSnapshot(t *testing.T, p *store.Planner, user uuid.UUID, s domain.Snapshot) {
	t.Helper()
	inTx(t, p, user, func(st planning.Stores) error { return st.Snapshots.SaveSnapshot(context.Background(), user, s) })
}

func sameJSON(t *testing.T, what string, want, got any) {
	t.Helper()
	a, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("%s differs\nwant %s\ngot  %s", what, a, b)
	}
}

func f64(v float64) *float64 { return &v }

// at is a timestamp with microseconds, as the service's clock gives them.
func at(day, hour int) time.Time {
	return monday.AddDate(0, 0, day).Add(time.Duration(hour)*time.Hour + 123456*time.Microsecond)
}

// fullSnapshot sets every field of every part of a snapshot, so a column
// the adapter forgets shows up as a difference.
func fullSnapshot(session, pain uuid.UUID) domain.Snapshot {
	d1, d2 := monday.AddDate(0, 0, 70), monday.AddDate(0, 0, 90)
	return domain.Snapshot{
		Profile: domain.Profile{BirthYear: 1990, SessionsPerWeek: 4, SessionMinutes: 60,
			Equipment: []string{"floor", "gym", "pull_up_bar"}, BodyweightKg: 72.35, TrainingLevel: domain.LevelTrained,
			TrainingMonths: 13.466666666666667, LastRegular: "3_to_6_weeks", HealthConsent: true, DisclaimerAck: true,
			OnboardedAt: monday, PreferredDays: []time.Weekday{time.Monday, time.Thursday, time.Sunday},
			Mobility: map[string]string{"shoulder_flexion": "full"}, MaxAddedLoadKg: 10, SmallestPlateKg: 1.25},
		Goals: []domain.Goal{
			{Skill: "planche", TargetLevel: "full", Priority: 1, TargetDate: &d1},
			{Skill: "front-lever", TargetLevel: "advanced-tuck", Priority: 2, TargetDate: &d2},
		},
		Capacities: map[string]domain.Estimate{
			"planche-tuck|none": {Mu: 12.345678901234567, Sigma: 2.5, Origin: domain.OriginLog, At: at(3, 18),
				Seen: at(4, 18), N: 7, Pending: f64(9.75)},
			"pull-up|band": {Mu: 6.1, Sigma: 1.3, Origin: domain.OriginSelf, At: at(0, 7), Seen: at(1, 7), N: 1, Pending: f64(1e-7)},
		},
		Ladders: map[string]domain.LadderState{"planche": {Rung: "planche-tuck", Status: domain.StatusCalibrated,
			Since: monday, Exposures: 5, Claimed: "planche-tuck", CapRung: "planche-advanced-tuck", ProbeOffer: true,
			RepTarget: 8.5, LoadKg: 2.5, EccS: 3, LastUp: at(2, 19), CapTo: "planche-lean", CapUntil: monday.AddDate(0, 0, 14)}},
		Regions: map[string]domain.RegionState{"elbow_inner": {State: domain.StateRTT2, EnteredVia: "pain_report",
			Since: monday, StartFraction: 0.5, Step: 2, StepSince: monday.AddDate(0, 0, 7), StepSessions: 1,
			Reference: map[string]float64{"elbow_medial/SA": 3.25}, PriorInjury: true, Complaint: true,
			ComplaintAt: monday, Restrictions: []string{"hang"}, Breaches: []time.Time{at(1, 20), at(5, 8)},
			PainDeloadTo: monday.AddDate(0, 0, 10), RestUntil: monday.AddDate(0, 0, 3), HoldAtRef: true, Referral: "soft"}},
		Screening: domain.Screening{ExertionSymptoms: true, AnyYes: true, Cleared: true},
		Constraints: []domain.Constraint{
			{Kind: domain.ConstraintLocked, Region: "wrist_back_extension", Created: monday},
			{Kind: domain.ConstraintStopped, Created: monday},
		},
		Phase: domain.Phase{MesoStart: monday, LastDeload: monday.AddDate(0, 0, -7), DeloadWeek: monday.AddDate(0, 0, 35),
			DeloadKind: domain.DeloadPlanned, DeloadNext: domain.DeloadPain, Calibrate: []string{"pull-up|none"}},
		History: []domain.LoggedSession{{ID: session.String(), Date: monday, Deload: true, Fatigue: f64(6),
			Sets: []domain.LoggedSet{{ID: "a", Exercise: "planche-tuck", Kind: domain.KindWorking, Assist: domain.AssistNone,
				LoadKg: 5, Value: 8, Reserve: f64(2), Form: f64(4), Failed: true, Partial: true, Eccentric: true}}}},
		Pain: []domain.PainReport{{ID: pain.String(), Region: "elbow_inner", Timepoint: domain.PainDuring, NRS: 3.5,
			At: at(0, 18), LastedOver: true, Persisted: true, SuddenSharp: true, SessionID: session.String()}},
		Unlocked: map[string]bool{"planche/tuck": true},
		Headroom: map[string]float64{"elbow_medial/SA": 0.37},
		Entry:    map[string]bool{"wrist": true},
		Break: &domain.BreakState{Days: 42, StraightDays: 35, Since: monday, Step: 2, StepSince: monday.AddDate(0, 0, 7),
			StepSessions: 1, Logged: true, Reference: map[string]float64{"wrist": 4.2}, Base: map[string]float64{"wrist": 6.28}},
	}
}

// noZeroField fails for every field of v that has its zero value, so the
// fixture keeps covering fields the snapshot gains.
func noZeroField(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			t.Errorf("%s is nil", path)
			return
		}
		noZeroField(t, v.Elem(), path)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			if v.Interface().(time.Time).IsZero() {
				t.Errorf("%s is the zero time", path)
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				noZeroField(t, v.Field(i), path+"."+v.Type().Field(i).Name)
			}
		}
	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			t.Errorf("%s is empty", path)
			return
		}
		if v.Kind() == reflect.Slice {
			noZeroField(t, v.Index(0), path+"[0]")
		} else {
			noZeroField(t, v.MapIndex(v.MapKeys()[0]), path+"[k]")
		}
	default:
		if v.IsZero() {
			t.Errorf("%s is zero", path)
		}
	}
}

func TestPlannerSnapshotRoundTrip(t *testing.T) {
	db := pgtest.New(t)
	p := store.NewPlanner(db)
	user := plannerUser(t, db)
	session, session2 := id(), id()
	workoutSession(t, db, user, session, monday)
	workoutSession(t, db, user, session2, monday.AddDate(0, 0, 3))

	full := fullSnapshot(session, id())
	noZeroField(t, reflect.ValueOf(full), "Snapshot")
	saveSnapshot(t, p, user, full)
	got := loadSnapshot(t, p, user)
	if !reflect.DeepEqual(full, got) {
		sameJSON(t, "snapshot", full, got)
		t.Fatalf("snapshot differs beyond its JSON:\nwant %+v\ngot  %+v", full, got)
	}

	// Saving the same snapshot again writes no new rows.
	saveSnapshot(t, p, user, got)
	if n := count(t, db, `SELECT count(*) FROM user_pain_reports WHERE user_id = $1`, user); n != 1 {
		t.Fatalf("%d pain rows after a second save", n)
	}

	// Change every part: the history and the pain reports only grow, keyed
	// parts gain, change and lose entries, goals swap priorities.
	next := loadSnapshot(t, p, user)
	next.Profile.PreferredDays = nil
	next.Profile.Mobility = nil
	next.Goals = []domain.Goal{
		{Skill: "front-lever", TargetLevel: "advanced-tuck", Priority: 1},
		{Skill: "muscle-up", TargetLevel: "bar-kipping", Priority: 2},
	}
	delete(next.Capacities, "pull-up|band")
	next.Capacities["dip|none"] = domain.Estimate{Mu: 8, Sigma: 2, Origin: domain.OriginDerived, N: 0}
	l := next.Ladders["planche"]
	l.Rung, l.CapTo, l.CapUntil = "planche-advanced-tuck", "", time.Time{}
	next.Ladders["planche"] = l
	next.Ladders["dip"] = domain.LadderState{Claimed: "dip-pb", Status: domain.StatusClaimed, Since: monday}
	delete(next.Regions, "elbow_inner")
	next.Regions["wrist_back_extension"] = domain.RegionState{State: domain.StateLocked, Since: monday}
	next.Screening = domain.Screening{}
	next.Constraints = []domain.Constraint{{Kind: domain.ConstraintExcluded, Region: "knee", Created: monday.AddDate(0, 0, 2)}}
	next.Phase.DeloadNext, next.Phase.Calibrate = "", nil
	next.History = append(next.History, domain.LoggedSession{ID: session2.String(), Date: monday.AddDate(0, 0, 3),
		Sets: []domain.LoggedSet{{ID: "b", Exercise: "dip-pb", Kind: domain.KindWorking, Assist: domain.AssistNone, Value: 7}}})
	next.Pain = append(next.Pain, domain.PainReport{ID: id().String(), Region: "knee", Timepoint: domain.PainDaily, NRS: 0, At: at(3, 7)})
	next.Headroom, next.Break = nil, nil
	saveSnapshot(t, p, user, next)
	got = loadSnapshot(t, p, user)
	sameJSON(t, "changed snapshot", next, got)
	if n := count(t, db, `SELECT count(*) FROM user_goals WHERE user_id = $1 AND status = 'dropped'`, user); n != 1 {
		t.Fatalf("%d dropped goals, want planche", n)
	}
	if n := count(t, db, `SELECT count(*) FROM planning_constraints WHERE user_id = $1 AND cleared_at IS NOT NULL`, user); n != 2 {
		t.Fatalf("%d lifted constraints, want 2", n)
	}
}

func TestPlannerRejectsBadIDsAndOtherUsers(t *testing.T) {
	db := pgtest.New(t)
	p := store.NewPlanner(db)
	user, other := plannerUser(t, db), plannerUser(t, db)
	ctx := context.Background()
	s := fullSnapshot(id(), id())
	s.History[0].ID = "20260928"
	err := p.InTx(ctx, user, func(st planning.Stores) error { return st.Snapshots.SaveSnapshot(ctx, user, s) })
	if err == nil || !strings.Contains(err.Error(), "must be the workout session's UUID") {
		t.Fatalf("a session without a UUID was stored: %v", err)
	}
	// The history refers to the log: another user's session is refused.
	theirs := id()
	workoutSession(t, db, other, theirs, monday)
	s = fullSnapshot(theirs, id())
	if err := p.InTx(ctx, user, func(st planning.Stores) error { return st.Snapshots.SaveSnapshot(ctx, user, s) }); err == nil {
		t.Fatal("stored a history entry for another user's session")
	}
	err = p.InTx(ctx, user, func(st planning.Stores) error {
		_, _, err := st.Snapshots.Snapshot(ctx, other)
		return err
	})
	if !errors.Is(err, store.ErrPlannerUser) {
		t.Fatalf("read another user's snapshot in this user's transaction: %v", err)
	}
	var ok bool
	inTx(t, p, user, func(st planning.Stores) error {
		var err error
		_, ok, err = st.Snapshots.Snapshot(ctx, user)
		return err
	})
	if ok {
		t.Fatal("a failed save left a snapshot behind")
	}
}

func TestPlannerPlans(t *testing.T) {
	db := pgtest.New(t)
	p := store.NewPlanner(db)
	user := plannerUser(t, db)
	ctx := context.Background()
	ids := map[[2]int]string{} // (plan, session) → ID, stable per fixture
	idOf := func(hash byte, i int) string {
		key := [2]int{int(hash), i}
		if _, ok := ids[key]; !ok {
			ids[key] = id().String()
		}
		return ids[key]
	}
	plan := func(hash byte, sessions int) domain.Plan {
		pl := domain.Plan{ID: idOf(hash, -1), WeekStart: monday, RulesetVersion: "0.1.0",
			InputHash: "sha256:" + strings.Repeat(string("0123456789abcdef"[hash]), 64),
			Reasons:   []domain.Reason{{RuleID: "WEEK-01", Args: map[string]string{"sessions": "3"}}}, Disclaimer: "x",
			Loads: []domain.AccountLoad{{Account: "wrist", Target: 1.5, Planned: 1, Cap: 1.2, Rule: "LOAD-02"}}}
		for i := 0; i < sessions; i++ {
			pl.Sessions = append(pl.Sessions, domain.PlannedSession{ID: idOf(hash, i), Index: i, Date: monday.AddDate(0, 0, 2*i),
				Kind: domain.SessionFull, EstMinutes: 42, Blocks: []domain.Block{{Role: domain.BlockWarmup, Minutes: 7}}})
		}
		return pl
	}
	inTx(t, p, user, func(st planning.Stores) error {
		if _, ok, err := st.Plans.ActivePlan(ctx, user, monday); ok || err != nil {
			t.Fatalf("plan before any was saved: %v %v", ok, err)
		}
		if err := st.Plans.SavePlan(ctx, user, plan(1, 3)); err != nil {
			return err
		}
		return st.Plans.SavePlan(ctx, user, plan(2, 2))
	})
	var got domain.Plan
	inTx(t, p, user, func(st planning.Stores) error {
		var ok bool
		var err error
		got, ok, err = st.Plans.ActivePlan(ctx, user, monday)
		if err == nil && !ok {
			err = errors.New("no active plan")
		}
		return err
	})
	sameJSON(t, "plan", plan(2, 2), got)
	// Sessions are found in the active plan only, and only by their user.
	other := plannerUser(t, db)
	inTx(t, p, user, func(st planning.Stores) error {
		pl, i, ok, err := st.Plans.PlannedSession(ctx, user, uuid.MustParse(idOf(2, 1)))
		if err != nil || !ok || i != 1 || pl.ID != idOf(2, -1) {
			t.Fatalf("session of the active plan: %v %d %v %v", pl.ID, i, ok, err)
		}
		if _, _, ok, err := st.Plans.PlannedSession(ctx, user, uuid.MustParse(idOf(1, 0))); ok || err != nil {
			t.Fatalf("session of a superseded plan: %v %v", ok, err)
		}
		return nil
	})
	inTx(t, p, other, func(st planning.Stores) error {
		if _, _, ok, err := st.Plans.PlannedSession(ctx, other, uuid.MustParse(idOf(2, 1))); ok || err != nil {
			t.Fatalf("another user's session: %v %v", ok, err)
		}
		return nil
	})
	if n := count(t, db, `SELECT count(*) FROM training_plans WHERE user_id = $1 AND status = 'superseded'`, user); n != 1 {
		t.Fatalf("%d superseded plans, want 1", n)
	}
	if n := count(t, db, `SELECT count(*) FROM planned_sessions s JOIN training_plans p ON p.id = s.plan_id
	                      WHERE p.user_id = $1 AND p.status = 'active'`, user); n != 2 {
		t.Fatalf("%d planned sessions in the active plan, want 2", n)
	}
	violates(t, func() error {
		_, err := db.Exec(ctx, `UPDATE training_plans SET status = 'active' WHERE user_id = $1`, user)
		return err
	}(), "training_plans_active_uk")
	// A planned session cannot belong to another user's plan.
	var planID uuid.UUID
	if err := db.QueryRow(ctx, `SELECT id FROM training_plans WHERE user_id = $1 AND status = 'active'`, user).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(ctx, `INSERT INTO planned_sessions (id, user_id, plan_id, order_index, scheduled_date, kind, est_minutes)
	                        VALUES ($1, $2, $3, 5, $4, 'full', 30)`, id(), other, planID, monday)
	violates(t, err, "planned_sessions_plan_id_user_id_fkey")
}

func TestPlannerDecisions(t *testing.T) {
	db := pgtest.New(t)
	p := store.NewPlanner(db)
	user := plannerUser(t, db)
	ctx := context.Background()
	changes := []domain.Change{{Kind: domain.ChangeRegion, Region: "knee", To: "rtt_1", Reasons: []domain.Reason{{RuleID: "INJ-03"}}}}
	decisions := []planning.Decision{
		{ID: id(), Trigger: planning.TriggerSession, SourceID: "s1", At: at(0, 9)},
		{ID: id(), Trigger: planning.TriggerPain, SourceID: "p1", At: at(1, 9), Changes: changes},
		{ID: id(), Trigger: planning.TriggerSymptoms, SourceID: "x1", At: at(1, 9)},
	}
	inTx(t, p, user, func(st planning.Stores) error {
		if _, seen, err := st.Decisions.Recorded(ctx, user, planning.TriggerSession, "s1"); seen || err != nil {
			t.Fatalf("seen before recording: %v %v", seen, err)
		}
		for _, d := range decisions {
			if err := st.Decisions.Record(ctx, user, d); err != nil {
				return err
			}
		}
		return nil
	})
	inTx(t, p, user, func(st planning.Stores) error {
		d, seen, err := st.Decisions.Recorded(ctx, user, planning.TriggerPain, "p1")
		if !seen || err != nil {
			t.Fatalf("recorded event not found: %v %v", seen, err)
		}
		sameJSON(t, "recorded decision", decisions[1], d)
		if _, seen, err := st.Decisions.Recorded(ctx, user, planning.TriggerSession, "s1"); !seen || err != nil {
			t.Fatalf("an event without changes is not seen: %v %v", seen, err)
		}
		// Newest first; events at the same instant in ID order, descending.
		want := []planning.Decision{decisions[1], decisions[2], decisions[0]}
		if bytes.Compare(decisions[1].ID[:], decisions[2].ID[:]) < 0 {
			want[0], want[1] = decisions[2], decisions[1]
		}
		page, err := st.Decisions.ListDecisions(ctx, user, nil, 2)
		if err != nil {
			return err
		}
		sameJSON(t, "first page", want[:2], page)
		page, err = st.Decisions.ListDecisions(ctx, user, &planning.Cursor{At: page[1].At, ID: page[1].ID}, 2)
		if err != nil {
			return err
		}
		sameJSON(t, "second page", want[2:], page)
		return nil
	})
	err := p.InTx(ctx, user, func(st planning.Stores) error {
		return st.Decisions.Record(ctx, user, planning.Decision{ID: id(), Trigger: planning.TriggerSession, SourceID: "s1", At: at(2, 9)})
	})
	if err == nil {
		t.Fatal("recorded the same event twice")
	}
	_, err = db.Exec(ctx, `UPDATE plan_decisions SET changes = '[]' WHERE user_id = $1`, user)
	violates(t, err, "plan_decisions_append_only")
	_, err = db.Exec(ctx, `DELETE FROM plan_decisions WHERE user_id = $1`, user)
	violates(t, err, "plan_decisions_append_only")
	exec(t, db, `DELETE FROM users WHERE id = $1`, user) // the cascade may delete them
	if n := count(t, db, `SELECT count(*) FROM plan_decisions`); n != 0 {
		t.Fatalf("%d decisions survived their user", n)
	}
}

func TestPlannerGoalAndConstraintChecks(t *testing.T) {
	db := pgtest.New(t)
	user := plannerUser(t, db)
	ctx := context.Background()
	exec(t, db, `INSERT INTO user_goals (id, user_id, skill, target_level, priority) VALUES ($1, $2, 'planche', 'full', 1)`, id(), user)
	_, err := db.Exec(ctx, `INSERT INTO user_goals (id, user_id, skill, target_level, priority) VALUES ($1, $2, 'dip', 'full', 1)`, id(), user)
	violates(t, err, "user_goals_priority_uk")
	_, err = db.Exec(ctx, `INSERT INTO user_goals (id, user_id, skill, target_level) VALUES ($1, $2, 'dip', 'full')`, id(), user)
	violates(t, err, "user_goals_active_priority_ck")
	_, err = db.Exec(ctx, `INSERT INTO planning_constraints (id, user_id, kind, created_at, position) VALUES ($1, $2, 'region_locked', now(), 0)`, id(), user)
	violates(t, err, "planning_constraints_region_kind_ck")
	_, err = db.Exec(ctx, `INSERT INTO user_pain_reports (id, user_id, region, timepoint, nrs, reported_at, position)
	                       VALUES ($1, $2, 'knee', 'during', 11, now(), 0)`, id(), user)
	violates(t, err, "user_pain_reports_nrs_ck")
}

// Events of one user apply one at a time; other users are not held up.
func TestPlannerSerialisesAUser(t *testing.T) {
	db := pgtest.New(t)
	p := store.NewPlanner(db)
	user, other := plannerUser(t, db), plannerUser(t, db)
	ctx := context.Background()
	entered, release := make(chan struct{}), make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- p.InTx(ctx, user, func(planning.Stores) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	if err := p.InTx(ctx, other, func(planning.Stores) error { return nil }); err != nil {
		t.Fatalf("another user waited: %v", err)
	}
	second := make(chan error, 1)
	go func() { second <- p.InTx(ctx, user, func(planning.Stores) error { return nil }) }()
	select {
	case err := <-second:
		t.Fatalf("a second transaction of the user ran while the first held the lock: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	for _, ch := range []chan error{first, second} {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("transaction did not finish")
		}
	}
}

type clock struct{ t time.Time }

func (c *clock) Now() time.Time { return c.t }

// seqIDs gives both services the same IDs.
type seqIDs struct{ n int }

func (s *seqIDs) New() uuid.UUID {
	s.n++
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprint(s.n)))
}

func plannerAnswers(kind string) domain.Answers {
	a := domain.Answers{BirthYear: 1995, HealthConsent: true, DisclaimerAck: true, BodyweightKg: 75,
		DataConfidence: "estimated", Screening: []bool{false, false, false, false, false, false},
		Goals:           []domain.Goal{{Skill: "planche", TargetLevel: "full", Priority: 1}},
		SessionsPerWeek: 3, SessionMinutes: 60, Equipment: []string{"gym", "rings", "resistance_bands"},
		TrainingLevel: domain.LevelTrained, TrainingAge: "1_to_4_years", LastRegular: "current_or_lt_3_weeks",
		Classes: map[string]string{"push_up_class": "21_30", "pull_up_class": "12_15", "dip_class": "13_20"},
		Stages:  map[string]domain.StageAnswer{"planche": {Level: "tuck", Class: "10_19"}}}
	switch kind {
	case "complaint":
		a.BodyMap = map[string]domain.BodyMapEntry{"elbow_inner": {Current: true}}
		a.Complaints = map[string]domain.Complaint{"elbow_inner": {PainDaily: 1.5, PainTraining: 3.5, Onset: "gradual",
			DurationWeeks: 2, Suspected: "no", Assessment: "no"}}
		a.RedFlags = map[string]map[string]bool{"elbow_inner": flags()}
	case "returner":
		a.LastRegular = "17_to_26_weeks"
		a.PreBreak = map[string]string{"planche": "advanced-tuck"}
		a.Stages = map[string]domain.StageAnswer{"planche": {Level: "unknown"}}
	case "redflags":
		// Daily pain above the green limit: the region starts at stage 0.
		a.BodyMap = map[string]domain.BodyMapEntry{"elbow_inner": {Current: true}, "knee": {Past12: true}}
		a.Complaints = map[string]domain.Complaint{"elbow_inner": {PainDaily: 3, PainTraining: 3, Onset: "gradual",
			DurationWeeks: 2, Suspected: "no", Assessment: "no"}}
		a.RedFlags = map[string]map[string]bool{"elbow_inner": flags()}
	case "stopped":
		// A red flag that stops training, already in the onboarding.
		a.BodyMap = map[string]domain.BodyMapEntry{"wrist_back_extension": {Current: true}}
		a.Complaints = map[string]domain.Complaint{"wrist_back_extension": {PainDaily: 1, PainTraining: 2,
			Onset: "sudden", DurationWeeks: 1, Suspected: "no", Assessment: "no"}}
		a.RedFlags = map[string]map[string]bool{"wrist_back_extension": flags("RF-07")}
	}
	return a
}

// flags answers the red-flag questions of an adult's arm region: the named
// ones yes, the others no.
func flags(yes ...string) map[string]bool {
	out := map[string]bool{}
	for _, id := range []string{"RF-01", "RF-02", "RF-03", "RF-04", "RF-05", "RF-06", "RF-07", "RF-10"} {
		out[id] = slices.Contains(yes, id)
	}
	return out
}

// perform logs a planned session as planned.
func perform(ps domain.PlannedSession, sessionID uuid.UUID) domain.LoggedSession {
	sess := domain.LoggedSession{ID: sessionID.String(), Date: ps.Date, Deload: ps.Kind == domain.SessionDeload}
	for _, b := range ps.Blocks {
		for _, it := range b.Items {
			if it.Stimulus == domain.StimPrehab || it.Offer {
				continue
			}
			for i := 0; i < it.Sets; i++ {
				v := float64(it.Reps)
				if it.HoldS > 0 && it.Reps == 0 {
					v = float64(it.HoldS)
				}
				assist := it.Assist
				if assist == "" {
					assist = domain.AssistNone
				}
				sess.Sets = append(sess.Sets, domain.LoggedSet{ID: fmt.Sprintf("%s-%d", it.Exercise, i), Exercise: it.Exercise,
					Kind: it.Kind, Assist: assist, LoadKg: it.LoadKg, Value: v, Reserve: f64(float64(it.Reserve)), Form: f64(4)})
			}
		}
	}
	return sess
}

// The service writes the same snapshots, plans and changes through Postgres
// as through the memory store: the adapter loses nothing the core reads.
func TestPlannerServiceMatchesMemory(t *testing.T) {
	kb := planning.LoadContentKnowledge(filepath.Join(pgtest.RepoRoot(), "content"), false,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, kind := range []string{"advanced", "complaint", "returner", "redflags", "stopped"} {
		t.Run(kind, func(t *testing.T) {
			db := pgtest.New(t)
			pg, mem := store.NewPlanner(db), memory.New()
			user := plannerUser(t, db)
			clk := &clock{t: monday.Add(7 * time.Hour)}
			svcs := []*planning.Service{
				{Knowledge: kb, Store: mem, Clock: clk, IDs: &seqIDs{}},
				{Knowledge: kb, Store: pg, Clock: clk, IDs: &seqIDs{}},
			}
			ctx := context.Background()
			both := func(what string, call func(*planning.Service) (any, error)) {
				t.Helper()
				var out [2]any
				for i, svc := range svcs {
					v, err := call(svc)
					if err != nil {
						t.Fatalf("%s (%d): %v", what, i, err)
					}
					out[i] = v
				}
				sameJSON(t, what, out[0], out[1])
				snaps := [2]domain.Snapshot{}
				for i, st := range []planning.Transactor{mem, pg} {
					if err := st.InTx(ctx, user, func(s planning.Stores) error {
						var err error
						snaps[i], _, err = s.Snapshots.Snapshot(ctx, user)
						return err
					}); err != nil {
						t.Fatal(err)
					}
				}
				sameJSON(t, what+": snapshot", snaps[0], snaps[1])
			}

			both("onboarding", func(svc *planning.Service) (any, error) {
				res, p, err := svc.Onboard(ctx, user, plannerAnswers(kind))
				return []any{res, p}, err
			})
			switch kind {
			case "redflags":
				// Stage 0 ends on green daily pain and negative red flags;
				// a stop names its region and goes with its clearance.
				rid := id().String()
				both("daily pain", func(svc *planning.Service) (any, error) {
					return svc.ReportPain(ctx, user, rid, domain.PainReport{Region: "elbow_inner",
						Timepoint: domain.PainDaily, NRS: 1, At: clk.t})
				})
				both("red flags negative", func(svc *planning.Service) (any, error) {
					return svc.AnswerRedFlags(ctx, user, "rf-1", "elbow_inner", flags())
				})
				both("red flag stop", func(svc *planning.Service) (any, error) {
					return svc.AnswerRedFlags(ctx, user, "rf-2", "knee", flags("RF-10"))
				})
				both("clearance", func(svc *planning.Service) (any, error) {
					return svc.ConfirmClearance(ctx, user, "cl-1", "knee")
				})
			case "stopped":
				both("screening clearance", func(svc *planning.Service) (any, error) {
					return svc.ConfirmClearance(ctx, user, "cl-1", "")
				})
			}
			for week := 0; week < 3; week++ {
				var plan domain.Plan
				both(fmt.Sprintf("week %d plan", week+1), func(svc *planning.Service) (any, error) {
					var err error
					plan, err = svc.Plan(ctx, user)
					return plan, err
				})
				for _, ps := range plan.Sessions {
					sid := id()
					workoutSession(t, db, user, sid, ps.Date)
					clk.t = ps.Date.Add(20 * time.Hour)
					sess := perform(ps, sid)
					both("session "+ps.Date.Format(time.DateOnly), func(svc *planning.Service) (any, error) {
						return svc.CompleteSession(ctx, user, sess)
					})
					if kind == "complaint" {
						rid := id().String()
						both("pain "+ps.Date.Format(time.DateOnly), func(svc *planning.Service) (any, error) {
							return svc.ReportPain(ctx, user, rid, domain.PainReport{Region: "elbow_inner",
								Timepoint: domain.PainDuring, NRS: 1, At: clk.t, SessionID: sid.String()})
						})
					}
				}
				clk.t = monday.AddDate(0, 0, 7*(week+1)).Add(7 * time.Hour)
				both(fmt.Sprintf("week %d start", week+2), func(svc *planning.Service) (any, error) {
					return svc.StartWeek(ctx, user)
				})
			}
			both("profile", func(svc *planning.Service) (any, error) {
				return svc.UpdateProfile(ctx, user, domain.ProfileUpdate{SessionsPerWeek: 4, SessionMinutes: 45,
					Equipment: []string{"gym", "parallettes"}, BodyweightKg: 74, PreferredDays: []time.Weekday{time.Monday, time.Thursday},
					Mobility: map[string]string{"wrist": "partly"}, MaxAddedLoadKg: 20, SmallestPlateKg: 1.25})
			})
			date := clk.t.AddDate(0, 5, 0)
			both("goals", func(svc *planning.Service) (any, error) {
				goals, realism, err := svc.SetGoals(ctx, user, []domain.Goal{{Skill: "handstand", TargetLevel: "free-30s", Priority: 2},
					{Skill: "planche", TargetLevel: "full", Priority: 1, TargetDate: &date}})
				return []any{goals, realism}, err
			})
			var plan domain.Plan
			both("regenerate", func(svc *planning.Service) (any, error) {
				var err error
				plan, err = svc.Regenerate(ctx, user)
				return plan, err
			})
			if len(plan.Sessions) > 0 {
				both("planned session", func(svc *planning.Service) (any, error) {
					p, ps, err := svc.PlannedSession(ctx, user, uuid.MustParse(plan.Sessions[0].ID))
					return []any{p.ID, ps}, err
				})
			}
			both("symptoms", func(svc *planning.Service) (any, error) {
				return svc.ReportSymptoms(ctx, user, "symptoms-1")
			})
			both("symptoms again", func(svc *planning.Service) (any, error) {
				return svc.ReportSymptoms(ctx, user, "symptoms-1")
			})
			both("decisions", func(svc *planning.Service) (any, error) {
				first, err := svc.Decisions(ctx, user, nil, 3)
				if err != nil || len(first) < 3 {
					return first, err
				}
				rest, err := svc.Decisions(ctx, user, &planning.Cursor{At: first[2].At, ID: first[2].ID}, 100)
				return []any{first, rest}, err
			})
			if kind != "stopped" && kind != "advanced" && kind != "returner" {
				both("pain reports", func(svc *planning.Service) (any, error) {
					return svc.PainReports(ctx, user, nil, 100)
				})
			}
		})
	}
}
