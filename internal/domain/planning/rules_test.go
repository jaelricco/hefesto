package planning_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// DOSE-01 point rule (spec §12.2): h = min(0.70 d, d − 2) ≥ 2 s, sets =
// clamp(round(60 / h), 2, 5).
func TestDose01(t *testing.T) {
	k := kb(t)
	for _, c := range []struct {
		d          float64
		sets, hold int
	}{
		{4, 5, 2}, {6, 5, 4}, {10, 5, 7}, {20, 4, 14}, {30, 3, 21},
	} {
		sets, hold := planning.HoldDose(k, c.d)
		if sets != c.sets || hold != c.hold {
			t.Errorf("d=%g: %d × %d s, want %d × %d s", c.d, sets, hold, c.sets, c.hold)
		}
	}
}

// PAR-F-30 and PAR-F-31: confidence class by σ/μ and the dose offset.
func TestConfidenceAndDose(t *testing.T) {
	k := kb(t)
	for _, c := range []struct {
		mu, sigma float64
		conf      string
		dose      float64
	}{
		{10, 1, planning.ConfHigh, 10},
		{10, 1.5, planning.ConfMid, 9.25},
		{10, 2.9, planning.ConfMid, 8.55},
		{10, 3, planning.ConfLow, 7},
		{2, 3, planning.ConfLow, 0},
		{0, 1, planning.ConfLow, 0},
	} {
		e := planning.Estimate{Mu: c.mu, Sigma: c.sigma}
		if got := k.Confidence(e); got != c.conf {
			t.Errorf("μ=%g σ=%g: confidence %s, want %s", c.mu, c.sigma, got, c.conf)
		}
		if got := k.Dose(e); got < c.dose-1e-9 || got > c.dose+1e-9 {
			t.Errorf("μ=%g σ=%g: dose %g, want %g", c.mu, c.sigma, got, c.dose)
		}
	}
}

// LOAD-02: f(a) is the minimum of the applying factors, never their product
// (spec §12.2: prior injury and no consent together give 0.5, not 0.25).
func TestWeekFactorIsTheMinimum(t *testing.T) {
	k := kb(t)
	base := planning.Snapshot{Profile: planning.Profile{HealthConsent: true, TrainingMonths: 60},
		Regions: map[string]planning.RegionState{}}
	for _, c := range []struct {
		name   string
		modify func(*planning.Snapshot)
		want   float64
	}{
		{"none", func(*planning.Snapshot) {}, 1},
		{"risk window", func(s *planning.Snapshot) { s.Profile.TrainingMonths = 12 }, 0.75},
		{"prior injury", func(s *planning.Snapshot) {
			s.Regions["elbow_inner"] = planning.RegionState{State: planning.StateNormal, PriorInjury: true}
		}, 0.5},
		{"prior injury, no consent, risk window", func(s *planning.Snapshot) {
			s.Profile.HealthConsent, s.Profile.TrainingMonths = false, 12
			s.Regions["elbow_inner"] = planning.RegionState{State: planning.StateNormal, PriorInjury: true}
		}, 0.5},
	} {
		s := base
		s.Regions = map[string]planning.RegionState{}
		c.modify(&s)
		if got := planning.WeekFactor(k, s, "elbow_medial/SA"); got != c.want {
			t.Errorf("%s: f = %g, want %g", c.name, got, c.want)
		}
	}
}

// LOAD-01: straight-arm work loads the /SA accounts, bent-arm work the /BA
// accounts; the wrist has one account; assistance counts in full.
func TestSetLoadAccounts(t *testing.T) {
	k := kb(t)
	tuck := planning.SetLoad(k, "planche-tuck", 75)
	if tuck["biceps_distal/SA"] == 0 || tuck["biceps_distal/BA"] != 0 || tuck["wrist"] == 0 {
		t.Errorf("tuck planche load %v", tuck)
	}
	full := planning.SetLoad(k, "planche", 75)
	if full["biceps_distal/SA"] <= tuck["biceps_distal/SA"] {
		t.Errorf("full planche %v not heavier than tuck %v", full, tuck)
	}
	pull := planning.SetLoad(k, "pull-up", 75)
	if pull["biceps_distal/BA"] == 0 || pull["biceps_distal/SA"] != 0 {
		t.Errorf("pull-up load %v", pull)
	}
	for a, u := range full {
		if u > 1+1e-9 {
			t.Errorf("%s: %g units, a set is at most 1", a, u)
		}
	}
}

// INJ-01/INJ-02: urgencies and actions of the red flags (research 05 §9).
func TestRedFlags(t *testing.T) {
	k := kb(t)
	for _, c := range []struct {
		region  string
		answers map[string]bool
		minor   bool
		urgency string
		stop    bool
		lock    bool
		rtt0    bool
	}{
		{"elbow_inner", map[string]bool{"RF-10": true}, false, planning.UrgencyNow, true, true, false},
		{"elbow_inner", map[string]bool{"RF-01": true}, false, planning.UrgencySoon, false, true, false},
		{"elbow_inner", map[string]bool{"RF-04": true}, false, planning.UrgencyAdvise, false, false, true},
		{"elbow_inner", map[string]bool{"RF-05": true, "RF-05-displaced": true}, false, planning.UrgencyNow, true, true, false},
		{"lower_back", map[string]bool{"RF-08": true}, false, planning.UrgencyNow, true, true, false},
		// Region-specific flags are not asked elsewhere, so they have no effect.
		{"elbow_inner", map[string]bool{"RF-08": true}, false, "", false, false, false},
		{"wrist_back_extension", map[string]bool{"RF-12": true}, false, "", false, false, false},
		{"wrist_back_extension", map[string]bool{"RF-12": true}, true, planning.UrgencyAdvise, false, false, true},
		{"elbow_inner", nil, false, "", false, false, false},
	} {
		got := k.EvaluateRedFlags(c.region, c.answers, c.minor)
		if got.Urgency != c.urgency || got.Stop != c.stop || got.Lock != c.lock || got.RTT0 != c.rtt0 {
			t.Errorf("%s %v minor=%v: %+v", c.region, c.answers, c.minor, got)
		}
		for _, r := range got.Reasons {
			if r.Region != c.region {
				t.Errorf("red-flag reason without region mark (EXPL-07): %+v", r)
			}
		}
	}
	// Every region asks RF-01 … RF-07 and RF-10 (KB-10).
	for _, region := range []string{"shoulder_front", "knee", "other", "chest"} {
		ids := map[string]bool{}
		for _, q := range k.RedFlags(region, false) {
			ids[q.ID] = true
		}
		for _, id := range []string{"RF-01", "RF-02", "RF-03", "RF-04", "RF-05", "RF-06", "RF-07", "RF-10"} {
			if !ids[id] {
				t.Errorf("%s does not ask %s", region, id)
			}
		}
	}
}

// GOAL-05: the realism check sums the PAR-A-45 bands; the date never
// changes the plan.
func TestRealism(t *testing.T) {
	k := kb(t)
	date := monday.AddDate(0, 0, 8*7)
	s := planning.Snapshot{Capacities: map[string]planning.Estimate{}, Unlocked: map[string]bool{}}
	r, ok := k.RealismFor(s, planning.Goal{Skill: "planche", TargetLevel: "tuck", Priority: 1, TargetDate: &date}, monday)
	if !ok {
		t.Fatal("no realism result")
	}
	// Tuck is OG 5 from 0: four steps ≤ 4 (2–8 weeks) and one of 5–8 (4–13).
	if r.LowerWeeks != 12 || r.ShownFrom != 28.5 || r.ShownTo != 45 || !r.Unrealistic {
		t.Errorf("realism %+v", r)
	}
	if !strings.Contains(r.Reason.Text, "Erfahrungswerte") || !strings.Contains(r.Reason.Text, "keine Prognose") {
		t.Errorf("realism text without the experience-value label: %q", r.Reason.Text)
	}
}

// KB-01 … KB-13: the validation finds each kind of defect in otherwise
// valid files.
func TestKnowledgeBaseValidation(t *testing.T) {
	files, issues, err := content.ReadTraining("../../../content")
	if err != nil || content.HasErrors(issues, false) {
		t.Fatalf("reading the knowledge base: %v %v", err, issues)
	}
	if _, iss := planning.Build(files); planning.HasErrors(iss, false) {
		t.Fatalf("the repository knowledge base has errors: %v", iss)
	}
	for _, c := range []struct {
		check  string
		mutate func(f *planning.Files)
	}{
		{"KB-01", func(f *planning.Files) { f.Manifest.RulesetVersion = "" }},
		{"KB-02", func(f *planning.Files) { f.Parameters[0].Sources = append(f.Parameters[0].Sources, "Z-999") }},
		{"KB-02", func(f *planning.Files) { f.Skills[0].Rungs = append(f.Skills[0].Rungs, "no-such-exercise") }},
		{"KB-03", func(f *planning.Files) { f.Parameters = drop(f.Parameters, "PAR-D-09") }},
		{"KB-03", func(f *planning.Files) {
			f.Rules = dropRule(f.Rules, "LOAD-02")
		}},
		{"KB-04", func(f *planning.Files) {
			// pull-up/strict-5 ⇄ hang-foundation/arch-hang
			for i := range f.Skills {
				if f.Skills[i].Slug == "hang-foundation" {
					f.Skills[i].Levels[0].Prerequisites = []string{"pull-up/strict-5"}
				}
			}
		}},
		{"KB-05", func(f *planning.Files) {
			for i := range f.Skills {
				if f.Skills[i].Slug == "planche" {
					r := f.Skills[i].Rungs
					r[1], r[2] = r[2], r[1]
				}
			}
		}},
		{"KB-06", func(f *planning.Files) { delete(f.Body.Matrix[0].Cells, "front_lever") }},
		{"KB-07", func(f *planning.Files) { f.Exercises[0].LoadFamily = "" }},
		{"KB-08", func(f *planning.Files) { f.Skills[1].Levels[0].Exercise = "planche" }},
		{"KB-09", func(f *planning.Files) { f.Sessions.Templates = f.Sessions.Templates[1:] }},
		{"KB-10", func(f *planning.Files) { f.Body.RedFlags[0].Regions = []string{"elbow_inner"} }},
		{"KB-12", func(f *planning.Files) { f.Rules[0].Text += " Das heilt deine Sehne." }},
	} {
		f := fresh(t)
		c.mutate(&f)
		_, iss := planning.Build(f)
		found := false
		for _, i := range iss {
			if i.Check == c.check && !i.Warn {
				found = true
			}
		}
		if !found {
			t.Errorf("%s not reported: %v", c.check, iss)
		}
	}
}

func drop(ps []planning.Param, id string) []planning.Param {
	var out []planning.Param
	for _, p := range ps {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return out
}

func dropRule(rs []planning.Rule, id string) []planning.Rule {
	var out []planning.Rule
	for _, r := range rs {
		if r.ID != id {
			out = append(out, r)
		}
	}
	return out
}

// fresh re-reads the files, which is the simplest deep copy.
func fresh(t *testing.T) planning.Files {
	t.Helper()
	f, _, err := content.ReadTraining("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func BenchmarkGenerate(b *testing.B) {
	k := kb(b)
	s, _, err := planning.Start(k, persona2(), now)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err := planning.Generate(k, s, now, monday); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	files, _, err := content.ReadTraining("../../../content")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		planning.Build(files)
	}
}

// benchHistory is persona 2 after twelve simulated weeks.
func benchHistory(b *testing.B) (*planning.Knowledge, planning.Snapshot, planning.LoggedSession) {
	k := kb(b)
	s, _ := start(b, k, persona2())
	a := athleteFor("2-advanced-gym-planche-front-lever")
	weeks, s := simulate(b, k, s, a, 12, nil)
	last := weeks[len(weeks)-1].plan
	sess := a.perform(k, last.Sessions[0], "bench")
	sess.Date = monday.AddDate(0, 0, 7*12)
	return k, s, sess
}

func BenchmarkGenerateWithHistory(b *testing.B) {
	k, s, _ := benchHistory(b)
	wk := monday.AddDate(0, 0, 7*12)
	b.ResetTimer()
	for range b.N {
		if _, err := planning.Generate(k, s, wk.Add(7*time.Hour), wk); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdapt(b *testing.B) {
	k, s, sess := benchHistory(b)
	b.ResetTimer()
	for range b.N {
		if _, _, err := planning.Adapt(k, s, planning.Event{Kind: planning.EventSession, At: sess.Date.Add(20 * time.Hour), Session: &sess}); err != nil {
			b.Fatal(err)
		}
	}
}

// ADAPT-04: the set hold of a rung grows by at most 2 s over last week's
// longest working hold, whatever the estimate says.
func TestHoldGrowthCap(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	s.Ladders = map[string]planning.LadderState{}
	s.Capacities["front-lever-tuck|none"] = planning.Estimate{Mu: 20, Sigma: 2, Origin: planning.OriginLog, At: monday.AddDate(0, 0, -3), N: 5}
	var sets []planning.LoggedSet
	for i := range 6 {
		v := 7.0
		if i > 0 {
			v = 6
		}
		sets = append(sets, planning.LoggedSet{ID: "a", Exercise: "front-lever-tuck", Kind: planning.KindWorking, Assist: planning.AssistNone, Value: v})
	}
	for _, d := range []int{-10, -3} {
		s.History = append(s.History, planning.LoggedSession{ID: "h" + monday.AddDate(0, 0, d).Format("0102"), Date: monday.AddDate(0, 0, d), Sets: sets})
	}
	p, err := planning.Generate(k, s, now, monday)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range items(p) {
		if it.Exercise == "front-lever-tuck" && !it.Offer {
			found = true
			if it.HoldS > 9 || !hasRule(it.Reasons, "ADAPT-04") {
				t.Errorf("tuck front lever %d s (reasons %v), want ≤ 9 s by ADAPT-04", it.HoldS, it.Reasons)
			}
		}
	}
	if !found {
		t.Fatal("no tuck front lever planned")
	}
}

// Spec §8.3 with PAR-D-18, PAR-D-26, PAR-D-28 and PAR-S-47: what one
// session's pain reports do to a region in the ramp.
func TestRampPainRules(t *testing.T) {
	k := kb(t)
	day := monday.AddDate(0, 0, 14)
	for _, c := range []struct {
		name      string
		reports   []planning.PainReport
		state     string
		step      int
		sessions  int  // step sessions afterwards (one before)
		restUntil int  // days after the session, 0 = no rest
		deload    bool // PAR-D-18 pain deload
	}{
		{name: "no soreness counts and completes the step", state: planning.StateRTT3, step: 2, reports: []planning.PainReport{
			{Timepoint: planning.PainBefore, NRS: 1}, {Timepoint: planning.PainDuring, NRS: 1}, {Timepoint: planning.PainMorning, NRS: 1}}},
		{name: "soreness above the baseline does not count", state: planning.StateRTT2, step: 2, sessions: 1, reports: []planning.PainReport{
			{Timepoint: planning.PainBefore, NRS: 0}, {Timepoint: planning.PainDuring, NRS: 2}, {Timepoint: planning.PainMorning, NRS: 0}}},
		{name: "next-day soreness repeats the step and rests a day", state: planning.StateRTT2, step: 2, restUntil: 2, reports: []planning.PainReport{
			{Timepoint: planning.PainDuring, NRS: 0}, {Timepoint: planning.PainMorning, NRS: 1}}},
		{name: "pain lasting over an hour", state: planning.StateRTT2, step: 2, restUntil: 2, reports: []planning.PainReport{
			{Timepoint: planning.PainAfter, NRS: 2, LastedOver: true}}},
		{name: "warm-up pain over 15 min goes a step back and rests two days", state: planning.StateRTT2, step: 1, restUntil: 3, reports: []planning.PainReport{
			{Timepoint: planning.PainWarmup, NRS: 3, Persisted: true}}},
		{name: "a broken threshold deloads and restarts the step", state: planning.StateRTT2, step: 2, deload: true, reports: []planning.PainReport{
			{Timepoint: planning.PainDuring, NRS: 6}}},
		{name: "8/10 lasting over an hour does both", state: planning.StateRTT2, step: 2, restUntil: 2, deload: true, reports: []planning.PainReport{
			{Timepoint: planning.PainAfter, NRS: 8, LastedOver: true}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _ := start(t, k, persona3())
			rs := s.Regions["elbow_inner"]
			rs.State, rs.Step, rs.StartFraction, rs.StepSessions, rs.StepSince = planning.StateRTT2, 2, 0.25, 1, day.AddDate(0, 0, -14)
			s.Regions["elbow_inner"] = rs
			sess := planning.LoggedSession{ID: "s", Date: day, Sets: []planning.LoggedSet{
				{ID: "1", Exercise: "planche-lean", Kind: planning.KindWorking, Assist: planning.AssistNone, Value: 5, Reserve: f64(3)}}}
			var err error
			s, _, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventSession, At: day.Add(19 * time.Hour), Session: &sess})
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range c.reports {
				r.Region, r.SessionID = "elbow_inner", "s"
				r.At = day.Add(18 * time.Hour)
				if r.Timepoint == planning.PainBefore {
					r.At = day.Add(17 * time.Hour)
				}
				if r.Timepoint == planning.PainMorning {
					r.At = day.Add(31 * time.Hour)
				}
				if s, _, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventPain, At: r.At, Pain: &r}); err != nil {
					t.Fatal(err)
				}
			}
			got := s.Regions["elbow_inner"]
			if got.State != c.state || got.Step != c.step || got.StepSessions != c.sessions {
				t.Errorf("state %s step %d sessions %d, want %s step %d sessions %d", got.State, got.Step, got.StepSessions, c.state, c.step, c.sessions)
			}
			wantRest := time.Time{}
			if c.restUntil > 0 {
				wantRest = day.AddDate(0, 0, c.restUntil)
			}
			if !got.RestUntil.Equal(wantRest) {
				t.Errorf("rest until %v, want %v", got.RestUntil, wantRest)
			}
			if deload := got.PainDeloadTo.After(day); deload != c.deload {
				t.Errorf("pain deload %v, want %v", deload, c.deload)
			}
		})
	}
}

// PAR-D-28: a resting region gets no load until its rest ends.
func TestRestingRegionIsNotPlanned(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona3())
	rs := s.Regions["elbow_inner"]
	rs.RestUntil = monday.AddDate(0, 0, 3)
	s.Regions["elbow_inner"] = rs
	p, err := planning.Generate(k, s, now, monday)
	if err != nil {
		t.Fatal(err)
	}
	planned := false
	for _, ps := range p.Sessions {
		for _, b := range ps.Blocks {
			for _, it := range b.Items {
				if it.Exercise != "planche-lean" && it.Exercise != "planche-tuck" {
					continue
				}
				planned = true
				if ps.Date.Before(rs.RestUntil) {
					t.Errorf("%s planned on %s during the rest", it.Exercise, ps.Date.Format(time.DateOnly))
				}
			}
		}
	}
	if !planned {
		t.Error("the planche is not planned after the rest either")
	}
}
