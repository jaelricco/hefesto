package planning

import (
	"math"
	"time"
)

// Realism is the realism check of a dated goal (spec §3.6, GOAL-05).
type Realism struct {
	Skill         string  `json:"skill"`
	TargetLevel   string  `json:"target_level"`
	LowerWeeks    float64 `json:"lower_weeks"`
	ShownFrom     float64 `json:"shown_from_weeks"`
	ShownTo       float64 `json:"shown_to_weeks"`
	WeeksToDate   float64 `json:"weeks_to_date"`
	Unrealistic   bool    `json:"unrealistic"`
	Milestone     string  `json:"milestone,omitempty"` // next level of the goal skill
	MilestoneFrom float64 `json:"milestone_from_weeks,omitempty"`
	MilestoneTo   float64 `json:"milestone_to_weeks,omitempty"`
	Reason        Reason  `json:"reason"`
}

// band returns the PAR-A-45 week band for a step to OG level o.
func (k *Knowledge) band(o float64) (float64, float64) {
	switch {
	case o <= 4:
		return k.T.RealLo4, k.T.RealHi4
	case o <= 8:
		return k.T.RealLo8, k.T.RealHi8
	case o <= 12:
		return k.T.RealLo12, k.T.RealHi12
	default:
		return k.T.RealLoTop, k.T.RealHiTop
	}
}

// span sums the bands from ordinal a (exclusive) to t (inclusive): the lower
// bound and the shown upper half.
func (k *Knowledge) span(a, t float64) (lower, from, to float64) {
	for o := math.Floor(a) + 1; o <= math.Ceil(t); o++ {
		lo, hi := k.band(o)
		lower += lo
		from += (lo + hi) / 2
		to += hi
	}
	return lower, from, to
}

// RealismFor checks a goal against its date. The goal is never rejected
// (O-8); the plan follows the working rung, not the date.
func (k *Knowledge) RealismFor(s Snapshot, g Goal, now time.Time) (Realism, bool) {
	sk := k.skills[g.Skill]
	target := k.level(g.Skill + "/" + g.TargetLevel)
	if sk == nil || target == nil || g.TargetDate == nil {
		return Realism{}, false
	}
	cur := 0.0
	next := -1
	for i, l := range sk.Levels {
		if k.levelMet(s, g.Skill+"/"+l.Slug) {
			cur = l.OG
			continue
		}
		next = i
		break
	}
	lower, from, to := k.span(cur, target.OG)
	weeks := g.TargetDate.Sub(now).Hours() / (24 * 7)
	r := Realism{Skill: g.Skill, TargetLevel: g.TargetLevel, LowerWeeks: lower, ShownFrom: from, ShownTo: to,
		WeeksToDate: math.Floor(weeks)}
	r.Unrealistic = weeks < lower
	// The milestone is the first unmet level one OG step or more above the
	// current one; a level on the same ordinal (e.g. the planche lean) has no
	// band of its own and is the working rung anyway.
	for next >= 0 && next < len(sk.Levels) && sk.Levels[next].OG <= cur {
		next++
	}
	if r.Unrealistic && next >= 0 && next < len(sk.Levels) {
		ms := sk.Levels[next]
		_, mf, mt := k.span(cur, ms.OG)
		r.Milestone, r.MilestoneFrom, r.MilestoneTo = ms.Slug, mf, mt
	}
	r.Reason = k.reason(RuleRealism, "from", from, "to", to, "lower", lower)
	return r, true
}
