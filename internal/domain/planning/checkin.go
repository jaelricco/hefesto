package planning

import (
	"math"
	"slices"
)

// CheckIn is what the athlete may tell when starting a session (spec
// §6.12, ENT-7): hours slept in the last 24 hours, and fatigue on the
// 1–10 scale of perceived_fatigue (10 is the most tired). Both are
// optional. The planner never stores them; only whether they made the
// session lighter.
type CheckIn struct {
	SleepH  *float64
	Fatigue *int
}

// Check-in ranges.
const (
	maxSleepH  = 24
	minFatigue = 1
	maxFatigue = 10
)

// ValidateCheckIn checks the ranges of a check-in; its pointers name the
// check-in in the start request.
func ValidateCheckIn(c CheckIn) error {
	f := fields{}
	if c.SleepH != nil && (math.IsNaN(*c.SleepH) || *c.SleepH < 0 || *c.SleepH > maxSleepH) {
		f.add("/check_in/sleep_hours", "must be 0–24")
	}
	if c.Fatigue != nil && (*c.Fatigue < minFatigue || *c.Fatigue > maxFatigue) {
		f.add("/check_in/fatigue", "must be 1–10")
	}
	return f.err()
}

// Tired reports whether a check-in asks for a lighter session: sleep of at
// most PAR-E-41 hours, or fatigue of at least PAR-S-28 (ADAPT-17).
func (k *Knowledge) Tired(c CheckIn) bool {
	return c.SleepH != nil && *c.SleepH <= k.T.SleepShort ||
		c.Fatigue != nil && float64(*c.Fatigue) >= k.T.CheckinFatigue
}

// Technique makes a planned session lighter after a tired check-in
// (ADAPT-17; PAR-E-43, PAR-E-19): in its max blocks, offers go and every
// hold, skill or conditioning, becomes submaximal technique,
// min(0.5 · d, 10 s) for three tries (DOSE-09, PAR-S-26), where
// d = hold + reserve is the dose value the plan used. A hold that would
// leave no reserve goes. Rep work (strength, rep skills, eccentrics) and
// every other block stay: upper-body strength was unaffected by short
// sleep (PAR-E-42). Item IDs stay, so the plan and a draft started from it
// agree. changed is false when the session has nothing to make lighter.
// Applying it twice changes nothing more.
func (k *Knowledge) Technique(ps PlannedSession) (out PlannedSession, changed bool) {
	out = ps
	out.Blocks = nil
	for _, b := range ps.Blocks {
		if b.Role != BlockMax {
			out.Blocks = append(out.Blocks, b)
			continue
		}
		nb := b
		nb.Items = nil
		for _, it := range b.Items {
			switch {
			case it.Offer:
				changed = true
				continue
			case (it.Stimulus == StimSkill || it.Stimulus == StimConditioning) && it.HoldS > 0:
				changed = true
				tech, ok := k.techniqueHold(it)
				if !ok {
					continue
				}
				it = tech
			}
			nb.Items = append(nb.Items, it)
		}
		if len(nb.Items) > 0 {
			out.Blocks = append(out.Blocks, nb)
		}
	}
	if !changed {
		return ps, false
	}
	out.Reasons = slices.Clone(ps.Reasons)
	if !slices.ContainsFunc(out.Reasons, func(r Reason) bool { return r.RuleID == RuleCheckin }) {
		out.Reasons = append(out.Reasons, k.reason(RuleCheckin))
	}
	return out, true
}

// techniqueHold turns a skill hold into technique holds from the same dose
// value, as the plan doses technique on light days (DOSE-09). ok is false
// when no hold leaves a reserve.
func (k *Knowledge) techniqueHold(it Item) (Item, bool) {
	d := float64(it.HoldS + it.Reserve)
	h := math.Max(k.T.MinSetHold, math.Floor(math.Min(float64(k.T.TechniqueFrac*d), k.T.TechniqueMax)))
	reserve := math.Floor(d - h)
	if reserve < 1 {
		return Item{}, false
	}
	out := it
	out.Stimulus, out.Class, out.Calibration = StimTechnique, classModerate, false
	out.HoldS, out.Sets, out.Reserve = int(h), int(k.T.TechniqueTries), int(reserve)
	out.RestS = int(math.Round(math.Max(h, k.T.BalanceSetS)/k.T.RestRound) * k.T.RestRound)
	out.Reasons = append(slices.Clone(it.Reasons), k.reason(RuleDoseTech, "hold_s", out.HoldS))
	return out, true
}
