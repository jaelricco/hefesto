package planning_test

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// athlete is a simulated user who performs planned sessions. truth is the
// unassisted maximum per exercise (reps or seconds); easier rungs without a
// value manage twice the value of the next harder rung that has one.
// Trained exercises improve by growth per week.
type athlete struct {
	truth         map[string]float64
	growth        float64
	declineOffers bool
}

func (a *athlete) max(k *planning.Knowledge, slug, assist string) float64 {
	v, ok := a.truth[slug]
	if !ok {
		v = a.derive(k, slug)
	}
	if assist == planning.AssistBand {
		v = v*1.5 + 3
	}
	return v
}

// derive estimates an exercise nobody specified from the nearest rung of its
// ladder that has a value: each easier rung adds half plus a little, each
// harder rung takes a little more than half away. It is a test model, not
// physiology.
func (a *athlete) derive(k *planning.Knowledge, slug string) float64 {
	ex, _ := k.Exercise(slug)
	sk, ok := k.Skill(ex.Skill)
	if !ok {
		return 20
	}
	for d := 1; d < len(sk.Rungs); d++ {
		if r := ex.Rung + d; r < len(sk.Rungs) {
			if hv, ok := a.truth[sk.Rungs[r]]; ok {
				for i := 0; i < d; i++ {
					hv = hv*1.5 + 4
				}
				return hv
			}
		}
		if r := ex.Rung - d; r >= 0 {
			if lv, ok := a.truth[sk.Rungs[r]]; ok {
				for i := 0; i < d; i++ {
					lv = math.Max(0, (lv-4)/1.5)
				}
				return lv
			}
		}
	}
	return 20
}

func f64(v float64) *float64 { return &v }

// perform logs one planned session: every set as planned, capped by the
// athlete's maximum; calibration sets stop two short of it. Offers are
// accepted unless declineOffers is set.
func (a *athlete) perform(k *planning.Knowledge, ps planning.PlannedSession, id string) planning.LoggedSession {
	sess := planning.LoggedSession{ID: id, Date: ps.Date, Deload: ps.Kind == planning.SessionDeload}
	n := 0
	for _, b := range ps.Blocks {
		for _, it := range b.Items {
			if it.Offer && a.declineOffers || it.Stimulus == planning.StimPrehab {
				continue
			}
			ex, _ := k.Exercise(it.Exercise)
			for i := 0; i < it.Sets; i++ {
				n++
				set := planning.LoggedSet{ID: fmt.Sprintf("%s-%d", id, n), Exercise: it.Exercise, Kind: it.Kind,
					Assist: it.Assist, LoadKg: it.LoadKg, Form: f64(4)}
				if set.Assist == "" {
					set.Assist = planning.AssistNone
				}
				if it.Stimulus == planning.StimEccentric {
					set.Value, set.Eccentric = float64(it.Reps), true
					sess.Sets = append(sess.Sets, set)
					continue
				}
				m := a.max(k, it.Exercise, set.Assist)
				planned := float64(it.Reps)
				if ex.Measure == planning.MeasureHold {
					planned = float64(it.HoldS)
				}
				v := math.Min(planned, math.Floor(m))
				if it.Calibration && i == 0 && it.Kind == planning.KindWorking {
					v = math.Max(1, math.Floor(m)-2)
				}
				set.Value, set.Reserve = v, f64(math.Max(0, math.Floor(m)-v))
				sess.Sets = append(sess.Sets, set)
			}
		}
	}
	return sess
}

// grow raises the maximum of every exercise the week trained.
func (a *athlete) grow(k *planning.Knowledge, p planning.Plan) {
	seen := map[string]bool{}
	for _, it := range items(p) {
		if seen[it.Exercise] {
			continue
		}
		seen[it.Exercise] = true
		a.truth[it.Exercise] = a.max(k, it.Exercise, planning.AssistNone) * a.growth
	}
}

// week is one simulated week: its plan and the snapshot after it.
type week struct {
	plan    planning.Plan
	changes []planning.Change
}

// simulate runs n weeks from s: plan, perform every session, adapt, then the
// week_start event with the plan's headroom. hook may add events after a
// session (index across the whole run) and returns the snapshot to continue
// with.
func simulate(t testing.TB, k *planning.Knowledge, s planning.Snapshot, a *athlete, n int,
	hook func(i int, ps planning.PlannedSession, s planning.Snapshot) planning.Snapshot) ([]week, planning.Snapshot) {
	t.Helper()
	var out []week
	start := monday
	idx := 0
	for w := 0; w < n; w++ {
		wk := start.AddDate(0, 0, 7*w)
		at := wk.Add(7 * time.Hour)
		p, err := planning.Generate(k, s, at, wk)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range planning.CheckInvariants(k, s, at, wk, p) {
			t.Errorf("week %d: %s", w+1, v)
		}
		var changes []planning.Change
		for _, ps := range p.Sessions {
			sess := a.perform(k, ps, sessionID(ps))
			var ch []planning.Change
			s, ch, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventSession, At: ps.Date.Add(20 * time.Hour), Session: &sess})
			if err != nil {
				t.Fatal(err)
			}
			changes = append(changes, ch...)
			if hook != nil {
				s = hook(idx, ps, s)
			}
			idx++
		}
		a.grow(k, p)
		next := wk.AddDate(0, 0, 7)
		var ch []planning.Change
		s, ch, err = planning.Adapt(k, s, planning.Event{Kind: planning.EventWeek, At: next, Headroom: p.Headroom})
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, ch...)
		out = append(out, week{plan: p, changes: changes})
	}
	return out, s
}

// summary is one line per week: sessions, straight-arm sets, working sets,
// rungs per skill and the ramp or deload state.
func summary(k *planning.Knowledge, weeks []week) string {
	var b strings.Builder
	for i, w := range weeks {
		sets, sa := 0, 0
		rungs := map[string]string{}
		for _, it := range items(w.plan) {
			sets += it.Sets
			ex, _ := k.Exercise(it.Exercise)
			if ex.StraightArm == planning.ArmStraight {
				sa += it.Sets
			}
			if it.Skill != "" && it.Stimulus != planning.StimPrehab {
				name := it.Exercise
				if it.Assist == planning.AssistBand {
					name += "+band"
				}
				if !strings.Contains(rungs[it.Skill], name) {
					if rungs[it.Skill] != "" {
						rungs[it.Skill] += ","
					}
					rungs[it.Skill] += name
				}
			}
		}
		var parts []string
		for sk, r := range rungs {
			parts = append(parts, sk+"="+r)
		}
		sort.Strings(parts)
		var kinds []string
		for _, c := range w.changes {
			if c.Kind != planning.ChangeCapacity {
				kinds = append(kinds, c.Kind)
			}
		}
		fmt.Fprintf(&b, "week %d: %d sessions, %d sets (%d straight-arm)", i+1, len(w.plan.Sessions), sets, sa)
		if w.plan.Deload != "" {
			fmt.Fprintf(&b, ", deload %s", w.plan.Deload)
		}
		if w.plan.Stopped {
			b.WriteString(", stopped")
		}
		fmt.Fprintf(&b, "\n  %s\n", strings.Join(parts, " "))
		if len(kinds) > 0 {
			fmt.Fprintf(&b, "  changes: %s\n", strings.Join(dedupe(kinds), " "))
		}
	}
	return b.String()
}

func dedupe(in []string) []string {
	var out []string
	seen := map[string]int{}
	for _, s := range in {
		if seen[s] == 0 {
			out = append(out, s)
		}
		seen[s]++
	}
	for i, s := range out {
		if seen[s] > 1 {
			out[i] = fmt.Sprintf("%s×%d", s, seen[s])
		}
	}
	return out
}
