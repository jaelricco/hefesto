package planning_test

import (
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// truths are the simulated athletes of the personas (maximum reps or
// seconds of the exercises the onboarding asked about).
var truths = map[string]map[string]float64{
	"1-beginner-park-muscle-up":          {"pull-up": 2, "dip-pb": 0, "push-up": 10, "support-hold-pb": 20, "dead-hang": 40, "row-horizontal": 8},
	"2-advanced-gym-planche-front-lever": {"planche-tuck": 14, "front-lever-advanced-tuck": 6, "pull-up": 14, "push-up": 25, "dip-pb": 15, "hollow-hold": 45},
	"3-elbow-inner-planche":              {"planche-tuck": 14, "pull-up": 14, "push-up": 25, "dip-pb": 15, "hollow-hold": 45},
	"4-returner-six-months":              {"planche-tuck": 8, "pull-up": 9, "push-up": 16, "dip-pb": 10, "hollow-hold": 25},
	"5-full-planche-in-8-weeks":          {"push-up": 6, "hollow-hold": 12},
	"6-contradictory-answers":            {"push-up": 1, "pull-up": 0, "dead-hang": 15},
}

func athleteFor(name string) *athlete {
	t := map[string]float64{}
	for k, v := range truths[name] {
		t[k] = v
	}
	return &athlete{truth: t, growth: 1.03}
}

// painFree reports mild, stable pain after every session for the regions
// with a complaint, so ramps can advance (a missing report holds them,
// PAR-S-34).
func painFree(t *testing.T, k *planning.Knowledge, nrs float64) hookFunc {
	return func(i int, ps planning.PlannedSession, s planning.Snapshot) (planning.Snapshot, []planning.Change) {
		var all []planning.Change
		for id, rs := range s.Regions {
			if !rs.Complaint {
				continue
			}
			for _, r := range []planning.PainReport{
				{Region: id, Timepoint: planning.PainDuring, NRS: nrs, At: ps.Date.Add(18 * time.Hour)},
				{Region: id, Timepoint: planning.PainMorning, NRS: nrs, At: ps.Date.Add(32 * time.Hour)},
			} {
				r.SessionID = sessionID(ps)
				var ch []planning.Change
				var err error
				s, ch, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventPain, At: r.At, Pain: &r})
				if err != nil {
					t.Fatalf("pain: %v", err)
				}
				all = append(all, ch...)
			}
		}
		return s, all
	}
}

func sessionID(ps planning.PlannedSession) string { return ps.Date.Format("20060102") }

// TestPersonaWeeks records twelve simulated weeks per persona as golden
// files; docs/algorithm/personas.md discusses them. Every week's plan must
// satisfy the invariants.
func TestPersonaWeeks(t *testing.T) {
	k := kb(t)
	for _, p := range personas {
		t.Run(p.name, func(t *testing.T) {
			s, _ := start(t, k, p.answers())
			weeks, _ := simulate(t, k, s, athleteFor(p.name), 12, painFree(t, k, 1))
			golden(t, filepath.Join("testdata", p.name+".weeks.txt"), summary(k, weeks))
		})
	}
}

// Spec §12.5: persona 4 returns after six months without logs. The ramp
// steps 0.25 → 0.5 → 0.75 → 1.0 advance weekly, but without a logged level
// before the pause they only bound LOAD-04 and LOAD-02 (§7.2); the ramp
// ends once its last step has held for the reference weeks.
func TestScenarioReturnerRamp(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona4())
	var steps []int
	weeks, end := simulate(t, k, s, athleteFor("4-returner-six-months"), 10, nil)
	for _, w := range weeks {
		for _, l := range w.plan.Loads {
			if l.Account == "biceps_distal/SA" && l.Cap > 0.25*l.Target+1e-3 && l.Rule == "LOAD-04" {
				t.Errorf("week 1 cap %.3f above the ramp share of %.3f", l.Cap, l.Target)
			}
		}
	}
	for _, w := range weeks[:4] {
		if w.after.Break != nil {
			steps = append(steps, w.after.Break.Step)
		}
	}
	if want := []int{1, 2, 3, 3}; !slices.Equal(steps, want) {
		t.Errorf("ramp steps %v, want %v", steps, want)
	}
	if end.Break != nil {
		t.Errorf("break still active after 10 weeks: %+v", end.Break)
	}
	for _, l := range weeks[len(weeks)-1].plan.Loads {
		if l.Rule == "ADAPT-16" && l.Account == "wrist" && l.Cap > l.Target+1e-3 {
			t.Errorf("wrist cap %.3f above its target %.3f", l.Cap, l.Target)
		}
	}
}

// Spec §6.11 with a logged level: persona 2 trains four weeks, then pauses.
// The pause keeps growing while the user is away, a pause of 28 days or more
// starts the straight-arm accounts at 25 % of the level before the pause,
// and a longer pause never allows more load.
func TestScenarioLoggedBreak(t *testing.T) {
	k := kb(t)
	s0, _ := start(t, k, persona2())
	_, s0 = simulate(t, k, s0, athleteFor("2-advanced-gym-planche-front-lever"), 4, nil)
	last := s0.History[len(s0.History)-1].Date
	prev := math.Inf(1)
	for _, days := range []int{10, 20, 30, 37, 44, 51, 65, 90, 130} {
		s := s0
		back := weekOf(last.AddDate(0, 0, days))
		for wk := weekOf(last).AddDate(0, 0, 7); !wk.After(back); wk = wk.AddDate(0, 0, 7) {
			var err error
			if s, _, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventWeek, At: wk}); err != nil {
				t.Fatal(err)
			}
		}
		p, err := planning.Generate(k, s, back.Add(7*time.Hour), back)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range planning.CheckInvariants(k, s, back.Add(7*time.Hour), back, p) {
			t.Errorf("%d days: %s", days, v)
		}
		if s.Break == nil && days >= 15 {
			t.Fatalf("%d days: no break", days)
		}
		var sa float64
		for _, l := range p.Loads {
			if strings.HasSuffix(l.Account, "/SA") || l.Account == "wrist" {
				sa += l.Cap
				if days >= 28+7 && l.Rule == "ADAPT-16" {
					if ref := s.Break.Reference[l.Account]; ref > 0 && math.Abs(l.Cap-0.25*ref) > 2e-3 {
						t.Errorf("%d days: %s cap %.3f, want 25 %% of %.3f", days, l.Account, l.Cap, ref)
					}
				}
			}
		}
		if sa > prev+1e-6 {
			t.Errorf("%d days away allow %.2f straight-arm units, more than a shorter pause (%.2f)", days, sa, prev)
		}
		prev = sa
	}
}

func weekOf(d time.Time) time.Time {
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// Spec §12.5: persona 3 reports 6/10 during session 2 → red-flag questions,
// a pain deload; three breaches within 14 days → referral.
func TestScenarioElbowPain(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona3())
	var changes []planning.Change
	hook := func(i int, ps planning.PlannedSession, s planning.Snapshot) (planning.Snapshot, []planning.Change) {
		nrs := 1.0
		if i >= 1 && i <= 3 {
			nrs = 6
		}
		r := planning.PainReport{Region: "elbow_inner", Timepoint: planning.PainDuring, NRS: nrs,
			At: ps.Date.Add(18 * time.Hour), SessionID: sessionID(ps)}
		s, ch, err := planning.Adapt(k, s, planning.Event{Kind: planning.EventPain, At: r.At, Pain: &r})
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, ch...)
		return s, ch
	}
	_, end := simulate(t, k, s, athleteFor("3-elbow-inner-planche"), 2, hook)
	kinds := map[string]int{}
	for _, c := range changes {
		kinds[c.Kind]++
		for _, r := range c.Reasons {
			if r.Region == "" && c.Region != "" {
				t.Errorf("reason %s about %s without the region mark (EXPL-07)", r.RuleID, c.Region)
			}
		}
	}
	if kinds[planning.ChangeRedFlags] == 0 {
		t.Error("no red-flag questions after 6/10")
	}
	if kinds[planning.ChangeDeload] == 0 {
		t.Error("no pain deload after 6/10")
	}
	if kinds[planning.ChangeReferral] == 0 || end.Regions["elbow_inner"].Referral != "advise" {
		t.Errorf("no referral after three breaches: %v, %+v", kinds, end.Regions["elbow_inner"])
	}
}

// Spec §12.5: red flag RF-10 stops training until the user confirms a
// clearance.
func TestScenarioRedFlagStops(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona3())
	at := now.Add(24 * time.Hour)
	s, _, err := planning.Adapt(k, s, planning.Event{Kind: planning.EventRedFlags, At: at, Region: "elbow_inner",
		Answers: map[string]bool{"RF-10": true}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := planning.Generate(k, s, at, monday)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Stopped || len(p.Sessions) != 0 {
		t.Fatalf("RF-10 did not stop the plan: stopped=%v, %d sessions", p.Stopped, len(p.Sessions))
	}
	if !hasRule(p.Reasons, "SAFE-02") {
		t.Error("no SAFE-02 reason")
	}
	s, _, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventClearance, At: at.Add(48 * time.Hour), Region: "elbow_inner"})
	if err != nil {
		t.Fatal(err)
	}
	p, err = planning.Generate(k, s, at.Add(72*time.Hour), monday.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if p.Stopped {
		t.Error("still stopped after the clearance")
	}
}

// Spec §12.5: a week without training changes nothing but the week; no
// wording about catching up, and the caps look at the weeks with load.
func TestScenarioMissedWeek(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	weeks, s := simulate(t, k, s, athleteFor("2-advanced-gym-planche-front-lever"), 4, nil)
	before := weeks[len(weeks)-1].plan
	gap := monday.AddDate(0, 0, 7*4)
	s, _, err := planning.Adapt(k, s, planning.Event{Kind: planning.EventWeek, At: gap.AddDate(0, 0, 7)})
	if err != nil {
		t.Fatal(err)
	}
	wk := gap.AddDate(0, 0, 7)
	p, err := planning.Generate(k, s, wk.Add(7*time.Hour), wk)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range planning.CheckInvariants(k, s, wk.Add(7*time.Hour), wk, p) {
		t.Error(v)
	}
	if s.Break != nil {
		t.Errorf("one week off started a break ramp: %+v", s.Break)
	}
	if len(p.Sessions) != len(before.Sessions) {
		t.Errorf("%d sessions after the free week, %d before", len(p.Sessions), len(before.Sessions))
	}
	for _, l := range p.Loads {
		if l.Rule == "LOAD-04" {
			t.Errorf("%s treated as new after one free week", l.Account)
		}
	}
}

// Spec §12.5: the muscle-up ladder becomes a goal ladder once its
// prerequisites (5 pull-ups, 8 dips: the unlock thresholds of research 02)
// are met.
func TestScenarioMuscleUpActivates(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona1())
	a := &athlete{truth: map[string]float64{"pull-up": 6, "dip-pb": 7, "push-up": 20, "support-hold-pb": 40, "dead-hang": 60,
		"row-horizontal": 15, "muscle-up-bar-kipping": 1}, growth: 1.06}
	weeks, _ := simulate(t, k, s, a, 16, nil)
	first := -1
	for i, w := range weeks {
		if slices.ContainsFunc(items(w.plan), func(it planning.Item) bool { return it.Skill == "muscle-up" && it.Role == planning.RoleGoal }) {
			first = i
			break
		}
	}
	if first < 0 {
		t.Fatalf("the muscle-up ladder never became active in 16 weeks\n%s", summary(k, weeks))
	}
	if first == 0 {
		t.Error("muscle-up active in week 1 before the prerequisites")
	}
}

// Spec §12.5: a ladder without progress gets a stagnation deload in the
// following week, not extra volume.
func TestScenarioPlateau(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	a := athleteFor("2-advanced-gym-planche-front-lever")
	a.growth = 1.0
	weeks, _ := simulate(t, k, s, a, 12, nil)
	found := false
	for _, w := range weeks {
		if w.plan.Deload == planning.DeloadStagnation {
			found = true
		}
	}
	if !found {
		t.Errorf("no stagnation deload in 12 weeks without progress\n%s", summary(k, weeks))
	}
}
