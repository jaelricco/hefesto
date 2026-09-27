package planning

import (
	"math"
	"slices"
	"time"
)

// Experience classes (spec §4.2, PAR-S-20).
const (
	ExpNovice       = "novice"
	ExpIntermediate = "intermediate"
	ExpAdvanced     = "advanced"
)

func experience(p Profile) string {
	switch {
	case p.TrainingMonths < 6 || p.TrainingLevel == LevelSedentary:
		return ExpNovice
	case p.TrainingMonths >= 48 && p.TrainingLevel == LevelHighlyTrained:
		return ExpAdvanced
	default:
		return ExpIntermediate
	}
}

// slot is one training day of the week.
type slot struct {
	day     time.Weekday
	date    time.Time
	full    bool
	ladders []*active
	cls     map[string]int // structure → highest class planned that day
}

// weekdayOrder sorts Monday first.
func weekdayOrder(d time.Weekday) int { return (int(d) + 6) % 7 }

// planDays picks the training days and which are full (WEEK-01, WEEK-02).
func (g *gen) planDays() {
	k, p := g.k, g.s.Profile
	days := slices.Clone(p.PreferredDays)
	if len(days) != p.SessionsPerWeek {
		days = slices.Clone(k.days[p.SessionsPerWeek])
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleDays, "sessions", p.SessionsPerWeek))
	}
	slices.SortFunc(days, func(a, b time.Weekday) int { return weekdayOrder(a) - weekdayOrder(b) })
	maxFull := map[string]float64{ExpNovice: k.T.FullNovice, ExpIntermediate: k.T.FullIntermediate, ExpAdvanced: k.T.FullAdvanced}[g.exp]
	nFull := min(len(days), int(maxFull))
	fullSet := bestSpread(days, nFull)
	for i, d := range days {
		off := weekdayOrder(d)
		g.slots = append(g.slots, &slot{day: d, date: g.week.AddDate(0, 0, off), full: fullSet[i], cls: map[string]int{}})
	}
	g.fullCount = nFull
	if nFull < len(days) {
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleSessionKind, "full", nFull, "light", len(days)-nFull))
	}
	if nFull >= int(k.T.SplitAt) {
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleSplit, "full", nFull))
	}
}

// bestSpread chooses n of the days maximising the smallest circular gap;
// ties go to the earliest combination.
func bestSpread(days []time.Weekday, n int) []bool {
	best := make([]bool, len(days))
	if n >= len(days) {
		for i := range best {
			best[i] = true
		}
		return best
	}
	bestGap := -1
	for mask := 0; mask < 1<<len(days); mask++ {
		if popcount(mask) != n {
			continue
		}
		var chosen []int
		for i := range days {
			if mask&(1<<i) != 0 {
				chosen = append(chosen, weekdayOrder(days[i]))
			}
		}
		gap := 7
		for i := range chosen {
			next := chosen[(i+1)%len(chosen)]
			d := (next - chosen[i] + 7) % 7
			if d == 0 {
				d = 7
			}
			gap = min(gap, d)
		}
		if gap > bestGap {
			bestGap = gap
			for i := range days {
				best[i] = mask&(1<<i) != 0
			}
		}
	}
	return best
}

func popcount(x int) int {
	n := 0
	for ; x != 0; x &= x - 1 {
		n++
	}
	return n
}

// spacing returns the hours a stimulus of class c needs before the next
// hard or moderate stimulus of the same structure (LOAD-05).
func (g *gen) spacing(c int, structure string) float64 {
	k := g.k
	switch c {
	case classHard:
		if g.inRamp(structure) {
			return k.T.SpacingRamp
		}
		return k.T.SpacingHard
	case classModerate:
		if g.inRamp(structure) {
			return k.T.SpacingRamp
		}
		return k.T.SpacingModerate
	}
	return 0
}

// inRamp reports whether a structure belongs to a region in the ramp.
func (g *gen) inRamp(structure string) bool {
	for id, rs := range g.s.Regions {
		if st := rttStage(rs.State); st >= 1 && st <= 4 && g.k.regionHas(id, structure) {
			return true
		}
	}
	return false
}

// fits checks spacing, direction and history for placing a on slot i.
func (g *gen) fits(a *active, i int) bool {
	sl := g.slots[i]
	if a.class > classLight && !sl.full {
		return false
	}
	if a.stim != StimBalance && !sl.full {
		return false
	}
	for _, other := range sl.ladders {
		if other == a {
			return false
		}
	}
	if a.rung.StraightArm == ArmStraight && a.stim == StimSkill {
		same := 0
		for _, o := range sl.ladders {
			if o.stim == StimSkill && o.rung.StraightArm == ArmStraight && o.rung.Direction == a.rung.Direction {
				same++
			}
		}
		if same >= 2 {
			return false
		}
	}
	if a.class == classLight {
		return true
	}
	for _, st := range g.k.structuresAt(a.rung) {
		for j, other := range g.slots {
			if j == i {
				continue
			}
			oc, ok := other.cls[st]
			if !ok || oc == classLight {
				continue
			}
			if dayGap(sl.day, other.day) < g.spacing(a.class, st) || dayGap(other.day, sl.day) < g.spacing(oc, st) {
				return false
			}
		}
		for _, m := range g.hist.hardAt[st] {
			gap := sl.date.Sub(m.day).Hours()
			if gap >= 0 && gap < g.spacing(m.class, st) {
				return false
			}
		}
	}
	return true
}

// place records a on slot i.
func (g *gen) place(a *active, i int) {
	sl := g.slots[i]
	sl.ladders = append(sl.ladders, a)
	a.days = append(a.days, i)
	for _, st := range g.k.structuresAt(a.rung) {
		if a.class > sl.cls[st] || sl.cls[st] == 0 {
			sl.cls[st] = max(sl.cls[st], a.class)
		}
	}
}

// allocate spreads the ladders over the days (WEEK-05, WEEK-06).
func (g *gen) allocate() {
	k := g.k
	for _, a := range g.ladders {
		if a.rung == nil {
			continue
		}
		a.freq = g.frequency(a)
		want := min(a.freq, len(g.slots))
		for n := 0; n < want; n++ {
			best, bestScore := -1, math.Inf(-1)
			for i := range g.slots {
				if !g.fits(a, i) {
					continue
				}
				score := g.score(a, i)
				if score > bestScore {
					best, bestScore = i, score
				}
			}
			if best < 0 {
				break
			}
			g.place(a, best)
		}
		if len(a.days) < want {
			a.reasons = append(a.reasons, k.reason(RuleFrequency, "skill", a.skill.Name, "planned", len(a.days), "target", want))
		}
		if len(a.days) == 0 {
			g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleTooFewDays, "skill", a.skill.Name))
		}
	}
	for _, sl := range g.slots {
		slices.SortStableFunc(sl.ladders, func(x, y *active) int {
			if x.priority != y.priority {
				return x.priority - y.priority
			}
			return roleRank(x.role) - roleRank(y.role)
		})
	}
}

// score prefers a day with an opposite-direction straight-arm skill
// (pairing, PAR-B-81), then the largest gap to the ladder's own days, then
// the earliest day.
func (g *gen) score(a *active, i int) float64 {
	sl := g.slots[i]
	s := 0.0
	if a.stim == StimSkill && a.rung.StraightArm == ArmStraight {
		for _, o := range sl.ladders {
			if o.stim == StimSkill && o.rung.StraightArm == ArmStraight && o.rung.Direction != a.rung.Direction {
				s += 1000
			}
		}
	}
	gap := 7.0
	for _, d := range a.days {
		fw := dayGap(g.slots[d].day, sl.day) / 24
		bw := dayGap(sl.day, g.slots[d].day) / 24
		gap = math.Min(gap, math.Min(fw, bw))
	}
	s += float64(gap * 10)
	// Prefer less crowded days, so light ladders fill days the hard ones
	// leave empty (spec §5.4, a fourth day without straight-arm work).
	s -= float64(len(sl.ladders) * 5)
	s -= float64(weekdayOrder(sl.day)) * 0.01
	return s
}
