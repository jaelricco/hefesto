package planning

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"
)

// This file exposes internals to the external test package: the invariants
// of spec §12.1 and single rules for table tests.

// CheckInvariants returns every violation of the plan properties I-1 … I-11
// (spec §12.1) for a plan generated from s at now for week.
func CheckInvariants(k *Knowledge, s Snapshot, now, week time.Time, p Plan) []string {
	var out []string
	fail := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }

	// I-1 determinism: a second run is byte-identical.
	again, err := Generate(k, s, now, week)
	if err != nil {
		fail("I-1: second run failed: %v", err)
	} else {
		a, _ := json.Marshal(p)
		b, _ := json.Marshal(again)
		if string(a) != string(b) {
			fail("I-1: two runs differ")
		}
		if p.InputHash != InputHash(k, s, now, week) {
			fail("I-1: input hash is not stable")
		}
	}

	g := &gen{k: k, s: s, now: now, week: week, plan: &p, equipment: map[string]bool{}}
	for _, e := range s.Profile.Equipment {
		g.equipment[e] = true
	}
	g.hist = k.history(s, week)

	// I-6, I-7: reasons exist, IDs resolve, no forbidden wording.
	checkReason := func(where string, r Reason) {
		if _, ok := k.rules[r.RuleID]; !ok {
			fail("I-6: %s: unknown rule %s", where, r.RuleID)
		}
		for _, id := range r.Params {
			if _, ok := k.params[id]; !ok {
				fail("I-6: %s: unknown parameter %s", where, id)
			}
		}
		for _, id := range r.Sources {
			if _, ok := k.sources[id]; !ok {
				fail("I-6: %s: unknown source %s", where, id)
			}
		}
		if f := k.Forbidden(r.Text); f != "" {
			fail("I-7: %s: %q contains %q", where, r.Text, f)
		}
	}
	for _, r := range p.Reasons {
		checkReason("plan", r)
	}
	for _, r := range p.Hints {
		checkReason("hint", r)
	}
	for _, e := range p.Exclusions {
		checkReason("exclusion "+e.Exercise, e.Reason)
	}

	for si, ps := range p.Sessions {
		for _, b := range ps.Blocks {
			for _, it := range b.Items {
				where := fmt.Sprintf("session %d %s", si+1, it.Exercise)
				ex := k.exercises[it.Exercise]
				if ex == nil {
					fail("I-6: %s: unknown exercise", where)
					continue
				}
				if len(it.Reasons) == 0 {
					fail("I-6: %s: item without a reason", where)
				}
				for _, r := range it.Reasons {
					checkReason(where, r)
				}
				if it.Sets <= 0 {
					continue
				}
				// I-9 equipment.
				for _, req := range ex.Requires {
					if !g.equipment[req] {
						fail("I-9: %s needs %s", where, req)
					}
				}
				// I-2 regions.
				for id, rs := range s.Regions {
					if rs.State == StateLocked && k.maxRating(id, ex) >= 1 {
						fail("I-2: %s loads the locked region %s", where, id)
					}
					if rs.Complaint && k.cell(id, ex) == ActionExclude {
						fail("I-2: %s is X for %s", where, id)
					}
				}
				// I-5 reserve and offers.
				skillSet := it.Stimulus == StimSkill || it.Stimulus == StimSkillReps ||
					(ex.StraightArm == ArmStraight && it.Stimulus == StimConditioning)
				if skillSet && it.Kind == KindWorking && !it.Offer && it.Reserve < 1 {
					fail("I-5: %s has no reserve", where)
				}
				if it.Offer && (!s.Profile.HealthConsent || s.Screening.AnyYes && !s.Screening.Cleared) {
					fail("I-5: %s offers a probe despite screening or consent", where)
				}
			}
		}
		// I-8 time.
		if ps.EstMinutes > float64(s.Profile.SessionMinutes)+eps && !hasTimeTrim(ps) {
			fail("I-8: session %d takes %.1f of %d min without a time trim", si+1, ps.EstMinutes, s.Profile.SessionMinutes)
		}
		// I-3 budget.
		if b, n := g.budget(si); float64(n) > b {
			fail("I-3: session %d has %d straight-arm sets, budget %.0f", si+1, n, b)
		}
	}

	// I-3 week caps and I-11 ramps.
	for _, l := range p.Loads {
		if l.Planned > l.Cap+2e-3 { // loads are rounded to 3 decimals
			fail("I-3: %s planned %.3f over cap %.3f (%s)", l.Account, l.Planned, l.Cap, l.Rule)
		}
		for _, r := range k.regionsOf(l.Account) {
			rs, ok := s.Regions[r]
			if !ok || !rs.Complaint {
				continue
			}
			if R, _ := k.reference(g.hist, l.Account, week); R == 0 && l.Cap > float64(k.T.NewTypeFraction*l.Target)+2e-3 { // loads are rounded to 3 decimals
				fail("I-11: %s of region %s may grow to %.3f, more than a new account (%.3f)", l.Account, r,
					l.Cap, k.T.NewTypeFraction*l.Target)
			}
		}
		// A break ramp without a logged level before the pause only bounds
		// LOAD-04 and LOAD-02 (§7.2), so a new straight-arm account stays at
		// the ramp share of its target.
		if b := s.Break; b != nil && b.StraightDays >= k.T.LayoffDays && b.Reference[l.Account] == 0 && isStraightAccount(l.Account) {
			steps := []float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}
			if R, _ := k.reference(g.hist, l.Account, week); R == 0 && l.Cap > steps[min(b.Step, 3)]*l.Target+2e-3 {
				fail("I-11: %s may grow to %.3f after a pause without a logged level", l.Account, l.Cap)
			}
		}
	}

	// I-4 spacing between planned days and against the log.
	type mark struct {
		day      time.Time
		class    int
		straight bool
		planned  bool
	}
	byStructure := map[string][]mark{}
	for _, ps := range p.Sessions {
		for _, b := range ps.Blocks {
			for _, it := range b.Items {
				if it.Sets <= 0 || it.Kind != KindWorking || it.Class == classLight {
					continue
				}
				for _, st := range k.structuresAt(k.exercises[it.Exercise]) {
					byStructure[st] = append(byStructure[st], mark{ps.Date, it.Class, k.exercises[it.Exercise].StraightArm != ArmNone, true})
				}
			}
		}
	}
	for st, marks := range byStructure {
		for _, h := range g.hist.hardAt[st] {
			marks = append(marks, mark{h.day, h.class, h.straight, false})
		}
		slices.SortFunc(marks, func(a, b mark) int { return a.day.Compare(b.day) })
		for i := 1; i < len(marks); i++ {
			gap := marks[i].day.Sub(marks[i-1].day).Hours()
			if gap == 0 || !marks[i].planned {
				continue // same session or day; two logged sessions are history
			}
			if need := g.spacing(marks[i-1].class, st, marks[i-1].straight); gap < need {
				fail("I-4: %s loaded %s and %s (%.0f h, needs %.0f h)", st,
					marks[i-1].day.Format("Mon"), marks[i].day.Format("Mon"), gap, need)
			}
		}
	}

	// I-10 the week is fully accounted for: sessions and rest days.
	if len(p.Sessions)+len(p.RestDays) != 7 && !p.Stopped {
		fail("I-10: %d sessions and %d rest days", len(p.Sessions), len(p.RestDays))
	}
	return out
}

func hasTimeTrim(ps PlannedSession) bool {
	for _, r := range ps.Reasons {
		if r.RuleID == RuleTrim && r.Args["cap"] == "time" {
			return true
		}
	}
	return false
}

// HoldDose is DOSE-01 for a dose value d: sets and hold time.
func HoldDose(k *Knowledge, d float64) (sets, hold int) {
	g := &gen{k: k}
	h := g.holdFor(d)
	return int(clamp(math.Round(k.T.TargetTotalHold/h), k.T.MinSets, k.T.MaxSets)), int(h)
}

// WeekFactor is f(a) of LOAD-02.
func WeekFactor(k *Knowledge, s Snapshot, account string) float64 {
	g := &gen{k: k, s: s}
	return g.weekFactor(account)
}

// SetLoad is LOAD-01 for one working set without added load.
func SetLoad(k *Knowledge, exercise string, bw float64) map[string]float64 {
	return k.setLoad(k.exercises[exercise], KindWorking, 0, bw)
}

// ProbesAllowedWhy reports which ADAPT-06a condition blocks probes.
func ProbesAllowedWhy(k *Knowledge, s Snapshot, week time.Time, exercise string) string {
	g := &gen{k: k, s: s, week: week}
	g.hist = k.history(s, week)
	ex := k.exercises[exercise]
	if s.Screening.AnyYes && !s.Screening.Cleared || !s.Profile.HealthConsent {
		return "screening/consent"
	}
	if s.Break != nil && ex.StraightArm != ArmNone {
		return "break"
	}
	for acc := range k.setLoad(ex, KindWorking, 0, s.Profile.BodyweightKg) {
		if ref, _ := k.reference(g.hist, acc, week); ref == 0 {
			return "no reference for " + acc
		}
	}
	return ""
}
