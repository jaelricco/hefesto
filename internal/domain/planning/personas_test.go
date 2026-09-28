package planning_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// The six personas of spec §12.4 as onboarding answers.

func base() planning.Answers {
	return planning.Answers{
		BirthYear: 1995, HealthConsent: true, DisclaimerAck: true,
		BodyweightKg: 75, DataConfidence: "estimated",
		Screening: []bool{false, false, false, false, false, false},
		Classes:   map[string]string{},
	}
}

// Persona 1: beginner, outdoor park only, twice a week, goal muscle-up.
func persona1() planning.Answers {
	a := base()
	a.Goals = []planning.Goal{{Skill: "muscle-up", TargetLevel: "bar-kipping", Priority: 1}}
	a.SessionsPerWeek, a.SessionMinutes = 2, 45
	a.Equipment = []string{"outdoor_park"}
	a.TrainingLevel, a.TrainingAge, a.LastRegular = planning.LevelRecreational, "lt_6_months", "current_or_lt_3_weeks"
	a.Classes = map[string]string{"push_up_class": "8_12", "pull_up_class": "1_3", "dip_class": "0", "support_hold_class": "10_30"}
	a.Stages = map[string]planning.StageAnswer{"muscle-up": {Level: "none"}}
	return a
}

// Persona 2: advanced, gym, four times a week, planche and front lever.
func persona2() planning.Answers {
	a := base()
	a.Goals = []planning.Goal{
		{Skill: "planche", TargetLevel: "full", Priority: 1},
		{Skill: "front-lever", TargetLevel: "full", Priority: 2},
	}
	a.SessionsPerWeek, a.SessionMinutes = 4, 60
	a.Equipment = []string{"gym", "rings", "resistance_bands", "parallettes"}
	a.TrainingLevel, a.TrainingAge, a.LastRegular = planning.LevelTrained, "1_to_4_years", "current_or_lt_3_weeks"
	a.Classes = map[string]string{"push_up_class": "21_30", "pull_up_class": "12_15", "dip_class": "13_20", "hollow_hold_class": "30_60"}
	a.Stages = map[string]planning.StageAnswer{
		"planche":     {Level: "tuck", Class: "10_19"},
		"front-lever": {Level: "advanced-tuck", Class: "4_9"},
	}
	return a
}

// Persona 3: like persona 2 with medial elbow complaints, goal planche.
func persona3() planning.Answers {
	a := persona2()
	a.Goals = []planning.Goal{{Skill: "planche", TargetLevel: "full", Priority: 1}}
	a.Stages = map[string]planning.StageAnswer{"planche": {Level: "tuck", Class: "10_19"}}
	a.BodyMap = map[string]planning.BodyMapEntry{"elbow_inner": {Current: true}}
	a.Complaints = map[string]planning.Complaint{"elbow_inner": {
		PainDaily: 1.5, PainTraining: 3.5, Onset: "gradual", DurationWeeks: 6, Suspected: "no", Assessment: "no",
	}}
	a.RedFlags = map[string]map[string]bool{"elbow_inner": noFlags("elbow_inner")}
	return a
}

// noFlags answers every red-flag question of an adult's region with no
// (onboarding.md §3.7: every question asked is answered).
func noFlags(region string) map[string]bool {
	out := map[string]bool{}
	for _, id := range []string{"RF-01", "RF-02", "RF-03", "RF-04", "RF-05", "RF-06", "RF-07", "RF-10"} {
		out[id] = false
	}
	if region == "lower_back" {
		out["RF-08"], out["RF-09"] = false, false
	}
	return out
}

// Persona 4: returner after six months without training.
func persona4() planning.Answers {
	a := base()
	a.Goals = []planning.Goal{{Skill: "planche", TargetLevel: "full", Priority: 1}}
	a.SessionsPerWeek, a.SessionMinutes = 3, 60
	a.Equipment = []string{"gym", "rings"}
	a.TrainingLevel, a.TrainingAge, a.LastRegular = planning.LevelTrained, "1_to_4_years", "17_to_26_weeks"
	a.PreBreak = map[string]string{"planche": "advanced-tuck"}
	a.Classes = map[string]string{"push_up_class": "13_20", "pull_up_class": "8_11", "dip_class": "8_12", "hollow_hold_class": "15_30"}
	a.Stages = map[string]planning.StageAnswer{"planche": {Level: "unknown"}}
	return a
}

// Persona 5: beginner who wants the full planche in eight weeks.
func persona5() planning.Answers {
	a := base()
	target := monday.AddDate(0, 0, 8*7)
	a.Goals = []planning.Goal{{Skill: "planche", TargetLevel: "full", Priority: 1, TargetDate: &target}}
	a.SessionsPerWeek, a.SessionMinutes = 3, 45
	a.Equipment = []string{"floor", "wall"}
	a.TrainingLevel, a.TrainingAge, a.LastRegular = planning.LevelRecreational, "lt_6_months", "never"
	a.Classes = map[string]string{"push_up_class": "4_7", "hollow_hold_class": "lt_15"}
	a.Stages = map[string]planning.StageAnswer{"planche": {Level: "none"}}
	return a
}

// Persona 6: contradictory answers — sedentary, no push-ups, but straddle
// planche and full front lever.
func persona6() planning.Answers {
	a := base()
	a.Goals = []planning.Goal{
		{Skill: "planche", TargetLevel: "full", Priority: 1},
		{Skill: "front-lever", TargetLevel: "full", Priority: 2},
	}
	a.SessionsPerWeek, a.SessionMinutes = 3, 60
	a.Equipment = []string{"outdoor_park"}
	a.TrainingLevel, a.TrainingAge, a.LastRegular = planning.LevelSedentary, "lt_6_months", "current_or_lt_3_weeks"
	a.Classes = map[string]string{"push_up_class": "0", "pull_up_class": "0", "dead_hang_class": "10_30"}
	a.Stages = map[string]planning.StageAnswer{
		"planche":     {Level: "straddle", Class: "4_9"},
		"front-lever": {Level: "full", Class: "4_9"},
	}
	return a
}

type persona struct {
	name    string
	answers func() planning.Answers
}

var personas = []persona{
	{"1-beginner-park-muscle-up", persona1},
	{"2-advanced-gym-planche-front-lever", persona2},
	{"3-elbow-inner-planche", persona3},
	{"4-returner-six-months", persona4},
	{"5-full-planche-in-8-weeks", persona5},
	{"6-contradictory-answers", persona6},
}

// start runs the onboarding; persona 6 answers its clarification questions
// with «no», as the spec scenario expects.
func start(t testing.TB, k *planning.Knowledge, a planning.Answers) (planning.Snapshot, planning.OnboardingResult) {
	t.Helper()
	s, res, err := planning.Start(k, a, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == planning.OnboardingNeedsAnswers {
		a.Clarifications = map[string]bool{}
		for _, q := range res.Questions {
			a.Clarifications[q.ID] = false
		}
		s2, res2, err := planning.Start(k, a, now)
		if err != nil {
			t.Fatal(err)
		}
		res2.Questions = res.Questions
		return s2, res2
	}
	return s, res
}

func TestPersonas(t *testing.T) {
	k := kb(t)
	for _, p := range personas {
		t.Run(p.name, func(t *testing.T) {
			s, res := start(t, k, p.answers())
			plan, err := planning.Generate(k, s, now, monday)
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range planning.CheckInvariants(k, s, now, monday, plan) {
				t.Error(v)
			}
			got := renderOnboarding(res) + render(k, plan)
			golden(t, filepath.Join("testdata", p.name+".txt"), got)
		})
	}
}

func renderOnboarding(res planning.OnboardingResult) string {
	var b strings.Builder
	b.WriteString("onboarding: " + res.Status + "\n")
	for _, q := range res.Questions {
		b.WriteString("  asked " + q.ID + ": " + q.Text + "\n")
	}
	for _, r := range res.Reasons {
		b.WriteString("  [" + r.RuleID + "] " + r.Text + "\n")
	}
	for _, r := range res.Hints {
		b.WriteString("  hint [" + r.RuleID + "] " + r.Text + "\n")
	}
	if res.Disclaimer == "" {
		b.WriteString("  MISSING DISCLAIMER\n")
	}
	return b.String()
}

// golden compares got with the file, or rewrites it with -update. A change
// to a golden file is a reviewed decision (spec §12.3).
func golden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // test fixture path
	if err != nil {
		t.Fatalf("%v (run go test -run TestPersonas -update)", err)
	}
	if string(want) != got {
		t.Errorf("%s differs; run go test ./internal/domain/planning -run TestPersonas -update and review the diff\n--- got\n%s", path, got)
	}
}

// items returns every planned working item with at least one set.
func items(p planning.Plan) []planning.Item {
	var out []planning.Item
	for _, s := range p.Sessions {
		for _, b := range s.Blocks {
			for _, it := range b.Items {
				if it.Sets > 0 && it.Kind == planning.KindWorking {
					out = append(out, it)
				}
			}
		}
	}
	return out
}

func hasExercise(p planning.Plan, slug string) bool {
	return slices.ContainsFunc(items(p), func(it planning.Item) bool { return it.Exercise == slug })
}

func hasRule(rs []planning.Reason, id string) bool {
	return slices.ContainsFunc(rs, func(r planning.Reason) bool { return r.RuleID == id })
}

// TestPersonaExpectations checks the expected properties of spec §12.4.
func TestPersonaExpectations(t *testing.T) {
	k := kb(t)
	plan := func(t *testing.T, a planning.Answers) (planning.Snapshot, planning.OnboardingResult, planning.Plan) {
		t.Helper()
		s, res := start(t, k, a)
		p, err := planning.Generate(k, s, now, monday)
		if err != nil {
			t.Fatal(err)
		}
		return s, res, p
	}
	straight := func(it planning.Item) bool {
		ex, _ := k.Exercise(it.Exercise)
		return ex.StraightArm == planning.ArmStraight
	}

	t.Run("1 beginner park muscle-up", func(t *testing.T) {
		_, _, p := plan(t, persona1())
		if len(p.Sessions) != 2 || p.Sessions[0].Date.Weekday() != monday.Weekday() || p.Sessions[1].Date.Weekday() != monday.AddDate(0, 0, 3).Weekday() {
			t.Errorf("want two sessions Monday and Thursday, got %d", len(p.Sessions))
		}
		for _, slug := range []string{"pull-up-negative", "dip-negative", "dead-hang", "support-hold-pb", "row-vertical"} {
			if !hasExercise(p, slug) {
				t.Errorf("feeder %s missing", slug)
			}
		}
		for _, it := range items(p) {
			if it.Assist != "" && it.Assist != planning.AssistNone {
				t.Errorf("%s assisted without bands", it.Exercise)
			}
			if strings.HasPrefix(it.Exercise, "muscle-up") {
				t.Errorf("muscle-up block planned before the 5 + 5 foundations: %s", it.Exercise)
			}
		}
		if !hasRule(p.Hints, "GOAL-04") {
			t.Error("no readiness hint for the muscle-up")
		}
	})

	t.Run("2 advanced gym planche front lever", func(t *testing.T) {
		_, _, p := plan(t, persona2())
		days := map[string]int{}
		free := 0
		for _, s := range p.Sessions {
			n := 0
			for _, b := range s.Blocks {
				for _, it := range b.Items {
					if it.Sets > 0 && it.Kind == planning.KindWorking && straight(it) {
						n++
						days[it.Skill]++
						if it.Stimulus == planning.StimSkill && it.RestS < 300 {
							t.Errorf("%s rest %d s, want ≥ 300", it.Exercise, it.RestS)
						}
					}
				}
			}
			if n == 0 {
				free++
			}
		}
		if days["planche"] < 2 || days["front-lever"] < 2 {
			t.Errorf("planche on %d, front lever on %d days", days["planche"], days["front-lever"])
		}
		if free == 0 {
			t.Error("no day without straight-arm work")
		}
		if !slices.ContainsFunc(items(p), func(it planning.Item) bool { return it.Calibration }) {
			t.Error("no calibration sets in the first week")
		}
	})

	t.Run("3 medial elbow complaint", func(t *testing.T) {
		s, _, p := plan(t, persona3())
		if st := s.Regions["elbow_inner"].State; st != planning.StateRTT1 {
			t.Errorf("elbow_inner is %s, want rtt_1", st)
		}
		exposures := 0
		for _, it := range items(p) {
			if it.Offer {
				t.Errorf("probe %s offered with a complaint", it.Exercise)
			}
			if it.Skill == "planche" {
				exposures++
				if !hasRule(it.Reasons, "INJ-05") || !it.Monitor {
					t.Errorf("planche item %s not modified and monitored", it.Exercise)
				}
			}
		}
		if exposures > 2 {
			t.Errorf("planche %d times; 72 h spacing allows 2 in this week", exposures)
		}
		if !slices.ContainsFunc(p.Monitor, func(m planning.RegionMonitor) bool { return m.Region == "elbow_inner" }) {
			t.Error("no pain monitoring for elbow_inner")
		}
	})

	t.Run("4 returner after six months", func(t *testing.T) {
		s, _, p := plan(t, persona4())
		if s.Break == nil || s.Break.Days < 119 {
			t.Fatalf("break %+v, want ≥ 119 days", s.Break)
		}
		for _, it := range items(p) {
			if it.Skill == "planche" && it.Exercise != "planche-lean" {
				t.Errorf("planche rung %s, want two rungs under the advanced tuck (lean)", it.Exercise)
			}
			if it.Offer {
				t.Errorf("probe %s offered during the break ramp", it.Exercise)
			}
		}
		for _, l := range p.Loads {
			if l.Planned > 0.25*l.Target+2e-3 && strings.HasSuffix(l.Account, "/SA") {
				t.Errorf("%s at %.2f of target, want ≤ 25 %% in the first ramp week", l.Account, l.Planned/l.Target)
			}
		}
	})

	t.Run("5 full planche in eight weeks", func(t *testing.T) {
		_, res, p := plan(t, persona5())
		if len(res.Realism) != 1 {
			t.Fatalf("realism %v", res.Realism)
		}
		r := res.Realism[0]
		if !r.Unrealistic || r.LowerWeeks != 48 || r.ShownFrom != 105 || r.ShownTo != 162 || r.Milestone == "" {
			t.Errorf("realism %+v", r)
		}
		for _, it := range items(p) {
			if it.Skill == "planche" {
				t.Errorf("planche block %s for a beginner without the push-up foundation", it.Exercise)
			}
		}
		if !slices.ContainsFunc(items(p), func(it planning.Item) bool { return it.Skill == "push-up" }) {
			t.Error("no push-up foundation")
		}
	})

	t.Run("6 contradictory answers", func(t *testing.T) {
		s, res, p := plan(t, persona6())
		if len(res.Questions) == 0 || len(res.Questions) > 2 {
			t.Errorf("%d questions, want 1–2", len(res.Questions))
		}
		for _, skill := range []string{"planche", "front-lever"} {
			if c := s.Ladders[skill].Claimed; c != "" {
				t.Errorf("%s keeps the implausible claim %s", skill, c)
			}
		}
		for _, it := range items(p) {
			if straight(it) {
				t.Errorf("straight-arm %s planned without foundations", it.Exercise)
			}
		}
	})
}
