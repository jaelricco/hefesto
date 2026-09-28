package planning

import (
	"math"
	"slices"
	"strings"
	"time"
)

// Roles of an active ladder (spec §3.4).
const (
	RoleGoal    = "goal"
	RoleFeeder  = "feeder"
	RoleSupport = "support"
)

// active is one ladder the week trains.
type active struct {
	skill    *Skill
	role     string
	priority int
	goal     *Goal
	reasons  []Reason

	rung   *Exercise // working rung
	assist string
	stim   string
	class  int
	freq   int
	cap    Estimate // dose source for the working rung
	hasCap bool
	calib  bool
	// maintain: a support whose recommended level is reached keeps a
	// maintenance dose (PAR-B-63).
	maintain bool
	probe    *Exercise
	monitor  bool
	modify   bool
	region   Reason
	days     []int
}

// Stimulus types (spec §0.3).
const (
	StimSkill        = "skill"
	StimSkillReps    = "skill_reps"
	StimStrength     = "strength"
	StimConditioning = "conditioning"
	StimEccentric    = "eccentric"
	StimBalance      = "balance"
	StimTechnique    = "technique"
	StimPrehab       = "prehab"
	StimAccessory    = "accessory"
)

// levelMet reports whether a level counts as met for planning: unlocked, or
// the dose value of its exercise reaches its threshold (spec §3.4 step 2).
func (k *Knowledge) levelMet(s Snapshot, ref string) bool {
	if s.Unlocked[ref] {
		return true
	}
	l := k.level(ref)
	if l == nil {
		return false
	}
	e, ok := s.Capacities[CapKey(l.Exercise, AssistNone)]
	return ok && k.Dose(e) >= l.Threshold
}

// firstUnmet returns the index of the first unmet level of a skill, or
// len(levels) when all are met.
func (k *Knowledge) firstUnmet(s Snapshot, sk *Skill) int {
	for i, l := range sk.Levels {
		if !k.levelMet(s, sk.Slug+"/"+l.Slug) {
			return i
		}
	}
	return len(sk.Levels)
}

// activeLadders derives the ladders of the week from the goals (GOAL-01,
// GOAL-02, GOAL-06).
func (g *gen) activeLadders() {
	k, s := g.k, g.s
	byskill := map[string]*active{}
	add := func(skill, role string, prio int, goal *Goal, r Reason) *active {
		if a, ok := byskill[skill]; ok {
			if prio < a.priority || (prio == a.priority && roleRank(role) < roleRank(a.role)) {
				a.priority, a.role = prio, role
				if goal != nil {
					a.goal = goal
				}
			}
			a.maintain = false
			return a
		}
		a := &active{skill: k.skills[skill], role: role, priority: prio, goal: goal, reasons: []Reason{r}}
		byskill[skill] = a
		return a
	}
	var feed func(ref string, prio int, depth int)
	feed = func(ref string, prio int, depth int) {
		skill, _, _ := strings.Cut(ref, "/")
		sk := k.skills[skill]
		if sk == nil || depth > 8 {
			return
		}
		idx := k.firstUnmet(s, sk)
		if idx >= len(sk.Levels) {
			return
		}
		var unmet []string
		for _, p := range sk.Levels[idx].Prerequisites {
			if !k.levelMet(s, p) {
				unmet = append(unmet, p)
			}
		}
		if len(unmet) == 0 || sk.Foundation {
			add(skill, RoleFeeder, prio, nil, k.reason(RuleGoalPath, "skill", sk.Name, "role", RoleFeeder))
		}
		for _, p := range unmet {
			feed(p, prio, depth+1)
		}
	}
	for i := range s.Goals {
		goal := &s.Goals[i]
		sk := k.skills[goal.Skill]
		if sk == nil {
			continue
		}
		idx := k.firstUnmet(s, sk)
		if idx >= len(sk.Levels) {
			idx = len(sk.Levels) - 1
		}
		lvl := sk.Levels[idx]
		var unmet []string
		for _, p := range lvl.Prerequisites {
			if !k.levelMet(s, p) {
				unmet = append(unmet, p)
			}
		}
		if len(unmet) == 0 {
			add(goal.Skill, RoleGoal, goal.Priority, goal, k.reason(RuleGoalOrder, "skill", sk.Name, "priority", goal.Priority))
		} else {
			for _, p := range unmet {
				feed(p, goal.Priority, 0)
			}
			if lvl.Hint != "" {
				g.plan.Hints = append(g.plan.Hints, k.reason(RuleReadiness, "skill", sk.Name, "hint", lvl.Hint))
			}
		}
		for _, e := range lvl.Recommended {
			if e.Weight < k.T.RecommendedMin || k.levelMet(s, e.To) {
				continue
			}
			skill, _, _ := strings.Cut(e.To, "/")
			add(skill, RoleSupport, goal.Priority, nil, k.reason(RuleGoalPath, "skill", k.skills[skill].Name, "role", RoleSupport))
		}
		// Foundations recommended for this or an earlier level that are
		// reached keep a maintenance dose (PAR-B-63).
		for i := 0; i <= idx; i++ {
			for _, e := range sk.Levels[i].Recommended {
				skill, _, _ := strings.Cut(e.To, "/")
				if e.Weight < k.T.RecommendedMin || !k.levelMet(s, e.To) {
					continue
				}
				if _, ok := byskill[skill]; !ok {
					add(skill, RoleSupport, goal.Priority, nil, k.reason(RuleMaintenance, "skill", k.skills[skill].Name)).maintain = true
				}
			}
		}
	}
	// GOAL-06: if the week only pushes or only pulls, add the other side.
	dirs := map[string]bool{}
	for _, a := range byskill {
		if a.role != RoleSupport {
			dirs[k.skillDirection(a.skill)] = true
		}
	}
	prio := len(s.Goals) + 1
	switch {
	case dirs[DirPush] && !dirs[DirPull]:
		add(k.antagonist(s, DirPull), RoleSupport, prio, nil, k.reason(RuleAntagonist, "direction", DirPull))
	case dirs[DirPull] && !dirs[DirPush]:
		add(k.antagonist(s, DirPush), RoleSupport, prio, nil, k.reason(RuleAntagonist, "direction", DirPush))
	}
	for _, id := range sortedKeys(byskill) {
		if byskill[id].skill != nil {
			g.ladders = append(g.ladders, byskill[id])
		}
	}
	slices.SortStableFunc(g.ladders, func(x, y *active) int {
		if x.priority != y.priority {
			return x.priority - y.priority
		}
		if roleRank(x.role) != roleRank(y.role) {
			return roleRank(x.role) - roleRank(y.role)
		}
		return strings.Compare(x.skill.Slug, y.skill.Slug)
	})
}

func roleRank(r string) int {
	switch r {
	case RoleGoal:
		return 0
	case RoleFeeder:
		return 1
	default:
		return 2
	}
}

// skillDirection is the direction of a skill's rungs (push, pull or none).
func (k *Knowledge) skillDirection(sk *Skill) string {
	for _, r := range sk.Rungs {
		if d := k.exercises[r].Direction; d != DirNone && d != "" {
			return d
		}
	}
	return DirNone
}

// antagonist picks the foundation ladder of a direction: the pull-up or
// push-up skill, else any foundation skill of that direction.
func (k *Knowledge) antagonist(s Snapshot, dir string) string {
	pref := map[string]string{DirPull: "pull-up", DirPush: "push-up"}[dir]
	if _, ok := k.skills[pref]; ok {
		return pref
	}
	for _, id := range k.skillIDs {
		sk := k.skills[id]
		if sk.Foundation && k.skillDirection(sk) == dir {
			return id
		}
	}
	return pref
}

// capacity returns the estimate for an exercise now: observed, or derived
// from the nearest harder rung of its skill (PAR-S-39).
func (g *gen) capacity(ex *Exercise, assist string) (Estimate, bool) {
	return g.k.capacityAt(g.s, ex, assist, g.now)
}

// capacityAt is capacity for any snapshot and time; Adapt uses it as the
// prior of an exercise's first observation.
func (k *Knowledge) capacityAt(s Snapshot, ex *Exercise, assist string, now time.Time) (Estimate, bool) {
	if e, ok := s.Capacities[CapKey(ex.Slug, assist)]; ok {
		return k.predict(e, ex, now), true
	}
	if assist == AssistBand {
		if e, ok := k.capacityAt(s, ex, AssistNone, now); ok {
			return k.derivedPrior(e, ex, now), true
		}
		return Estimate{}, false
	}
	sk := k.skills[ex.Skill]
	if sk == nil {
		return Estimate{}, false
	}
	for r := ex.Rung + 1; r < len(sk.Rungs); r++ {
		harder := k.exercises[sk.Rungs[r]]
		if harder.Measure != ex.Measure || harder.Eccentric {
			continue
		}
		if e, ok := s.Capacities[CapKey(harder.Slug, AssistNone)]; ok {
			return k.derivedPrior(k.predict(e, harder, now), ex, now), true
		}
	}
	return Estimate{}, false
}

// feasible reports whether an exercise can be planned: equipment, regions,
// supinated gate (SEL-02–SEL-04). The reason explains an exclusion.
func (g *gen) feasible(ex *Exercise, workingOG float64) (bool, Reason, regionVerdict) {
	k := g.k
	for _, req := range ex.Requires {
		if !g.equipment[req] {
			return false, k.reason(RuleEquipment, "exercise", ex.Name, "equipment", req), regionVerdict{}
		}
	}
	if ex.Supinated && workingOG < k.T.SupinatedMinOG {
		return false, k.reason(RuleSupinated, "exercise", ex.Name), regionVerdict{}
	}
	v := k.judge(g.s, ex)
	if v.exclude {
		return false, v.reason, v
	}
	return true, Reason{}, v
}

// selectRung picks the working rung of a ladder (SEL-07–SEL-09).
func (g *gen) selectRung(a *active) {
	k := g.k
	sk := a.skill
	ls := g.s.Ladders[sk.Slug]
	top := len(sk.Rungs) - 1
	// §6.11: the rung before the pause caps the ladder during a break; in
	// the ramp steps before 1.0 one rung below it, because the ramp stages
	// before the target rung are regressions (05 §6.2).
	if ls.CapRung != "" && g.s.Break != nil {
		if e := k.exercises[ls.CapRung]; e != nil {
			c := e.Rung
			if g.breakRamp() && e.StraightArm == ArmStraight {
				c = max(c-1, 0)
			}
			top = min(top, c)
		}
	}
	// PAR-B-57: at most one step up per ladder and week.
	if cur := k.exercises[ls.Rung]; cur != nil && !ls.LastUp.IsZero() && g.now.Sub(ls.LastUp) < 7*24*time.Hour {
		top = min(top, cur.Rung)
	}
	if ls.CapTo != "" && g.now.Before(ls.CapUntil) {
		if e := k.exercises[ls.CapTo]; e != nil {
			top = min(top, e.Rung)
		}
	}
	// SEL-08: in the first two exposures stay one rung under the claim.
	claimCap := top
	if ls.Claimed != "" && ls.Exposures < 2 {
		if e := k.exercises[ls.Claimed]; e != nil {
			claimCap = max(e.Rung-1, 0)
			a.calib = true
		}
	}
	// §6.11: after the longest breaks the first week starts two rungs under
	// the pre-break level; afterwards calibration and SEL-07 decide, capped
	// by that level.
	if b := g.s.Break; b != nil && b.Days >= k.T.BreakHalf && ls.CapRung != "" && !g.week.After(b.Since) {
		if e := k.exercises[ls.CapRung]; e != nil {
			claimCap = min(claimCap, max(e.Rung-2, 0))
		}
	}
	top = min(top, claimCap)
	lo := k.T.MinSetHold * 2 // 08 §4: set ≥ 2 s plus reserve ≥ 2 s
	// WEEK-07: a deload week holds the rung of the week before.
	if g.deload != "" {
		if ex := k.exercises[ls.Rung]; ex != nil && ex.Skill == sk.Slug && ex.Rung <= top {
			if ok, _, v := g.feasible(ex, ex.OG); ok {
				e, has := g.capacity(ex, AssistNone)
				if ex.Measure == MeasureHold && (!has || k.Dose(e) < lo) && g.hasBands && ex.Assistable {
					if eb, okb := g.capacity(ex, AssistBand); okb {
						e, has = eb, true
					}
					a.assist = AssistBand
				}
				g.setRung(a, ex, e, v)
				a.hasCap = has
				a.reasons = append(a.reasons, k.reason(RuleDeloadWeek, "kind", g.deload))
				return
			}
		}
	}
	repLo := k.T.HeavyLo
	if g.exp == ExpNovice {
		repLo = k.T.NoviceRepsLo
	}
	var fallback *Exercise
	var ecc *Exercise
	var feasibleRungs []*Exercise // non-eccentric, hardest first
	verdicts := map[string]regionVerdict{}
	known := -1 // lowest rung with an estimate of its own
	for r := top; r >= 0; r-- {
		ex := k.exercises[sk.Rungs[r]]
		if _, own := g.s.Capacities[CapKey(ex.Slug, AssistNone)]; own && !ex.Eccentric {
			known = r
		}
		ok, why, verdict := g.feasible(ex, ex.OG)
		if !ok {
			g.exclude(ex, why)
			continue
		}
		if ex.Eccentric {
			if ecc == nil {
				ecc = ex
			}
			continue
		}
		fallback = ex
		verdicts[ex.Slug] = verdict
		feasibleRungs = append(feasibleRungs, ex)
		e, has := g.capacity(ex, AssistNone)
		if !has {
			continue
		}
		d := k.Dose(e)
		if ex.Measure == MeasureHold && d >= lo || ex.Measure == MeasureReps && math.Floor(d)-k.T.RIRStrength >= repLo {
			// ADAPT-06: at x > 30 s the stage window is left; the next rung
			// takes over, with a band while it has no value of its own.
			if next := g.nextRung(ex); next != nil && ex.Measure == MeasureHold && ex.HoldClass == HoldSkill &&
				d > k.T.StageSwitch && next.Rung <= top {
				if ok, _, v := g.feasible(next, next.OG); ok {
					ne, has := g.capacity(next, AssistNone)
					if !has || k.Dose(ne) < lo {
						if g.hasBands && next.Assistable {
							if eb, okb := g.capacity(next, AssistBand); okb {
								ne = eb
							}
							a.assist = AssistBand
							g.setRung(a, next, ne, v)
						} else {
							g.setRung(a, next, ne, v)
						}
						a.hasCap, a.calib = has, true
						a.reasons = append(a.reasons, k.reason(RuleRungUp))
						return
					}
				}
			}
			g.setRung(a, ex, e, verdict)
			a.reasons = append(a.reasons, k.reason(RuleHoldRung, "exercise", ex.Name, "dose", d))
			if a.calib {
				a.reasons = append(a.reasons, k.reason(RuleEntryRung, "exercise", ex.Name))
			}
			return
		}
	}
	// Nothing clears the threshold. For reps, easier rungs only carry the
	// harder rung's estimate as a lower bound (PAR-S-39), so the root would
	// always win; the rung just below the lowest one the user reported is the
	// calibration start instead (SEL-08 spirit), unless that estimate is
	// under one repetition, where the eccentric variant takes over (SEL-09).
	if fallback != nil && fallback.Measure == MeasureReps && known >= 0 {
		e, ok := g.s.Capacities[CapKey(sk.Rungs[known], AssistNone)]
		concentric := ok && k.Dose(e)-k.T.RIRStrength >= 1
		if concentric || ecc == nil {
			for _, ex := range feasibleRungs {
				if ex.Rung < known {
					fallback = ex
					ecc = nil
					break
				}
			}
		}
		// SEL-09 names only d < 1 (eccentric) and the rep range. In between,
		// a rung where one repetition with the target reserve is possible is
		// trained concentrically below the range (DOSE-05 start), rather than
		// eccentrically until the full range is reached.
		if concentric && ecc != nil {
			if kr := k.exercises[sk.Rungs[known]]; slices.Contains(feasibleRungs, kr) {
				fallback, ecc = kr, nil
			}
		}
	}
	// Otherwise: eccentric (reps), band, or the lowest feasible rung with a
	// calibration start.
	if ecc != nil {
		e, _ := g.capacity(ecc, AssistNone)
		_, _, v := g.feasible(ecc, ecc.OG)
		g.setRung(a, ecc, e, v)
		a.stim, a.class = StimEccentric, classHard
		a.reasons = append(a.reasons, k.reason(RuleRepRung, "exercise", ecc.Name))
		return
	}
	if fallback != nil {
		e, has := g.capacity(fallback, AssistNone)
		if g.hasBands && fallback.Assistable {
			// With the band the unassisted value does not apply: its own
			// value, or a calibration start (§4.3).
			eb, okb := g.capacity(fallback, AssistBand)
			e, has = eb, okb
			a.assist = AssistBand
			g.setRung(a, fallback, e, verdicts[fallback.Slug])
		} else {
			g.setRung(a, fallback, e, verdicts[fallback.Slug])
		}
		a.hasCap = has
		a.calib = true
		a.reasons = append(a.reasons, k.reason(RuleHoldRung, "exercise", fallback.Name, "dose", k.Dose(e)))
	}
}

// setRung records the working rung and its stimulus.
func (g *gen) setRung(a *active, ex *Exercise, e Estimate, v regionVerdict) {
	k := g.k
	a.rung, a.cap, a.hasCap = ex, e, true
	a.monitor, a.modify, a.region = v.monitor, v.modify, v.reason
	if k.Confidence(e) == ConfLow || slices.Contains(g.s.Phase.Calibrate, CapKey(ex.Slug, AssistNone)) {
		a.calib = true
	}
	d := k.Dose(e)
	switch {
	case ex.Eccentric:
		a.stim, a.class = StimEccentric, classHard
	case ex.Measure == MeasureHold && ex.HoldClass == HoldBalance:
		a.stim, a.class = StimBalance, classLight
	case ex.Measure == MeasureHold && ex.HoldClass == HoldSkill && d <= k.T.StageSwitch:
		a.stim, a.class = StimSkill, classLight
		if ex.StraightArm == ArmStraight {
			a.class = classHard
		}
	case ex.Measure == MeasureHold:
		a.stim, a.class = StimConditioning, classLight
		if ex.StraightArm == ArmStraight {
			a.class = classModerate
		}
	case a.skill.LimitingFactor == LimitMixed:
		a.stim, a.class = StimSkillReps, classHard
	default:
		a.stim, a.class = StimStrength, classModerate
	}
	if a.modify && a.stim == StimSkill {
		// M: one rung lower is the modification (INJ-05).
		if lower := g.lowerRung(ex); lower != nil {
			le, _ := g.capacity(lower, AssistNone)
			a.rung, a.cap = lower, le
			a.reasons = append(a.reasons, a.region)
		}
	}
	if a.modify && ex.Measure == MeasureReps && !ex.Eccentric && g.hasBands && ex.Assistable && a.assist != AssistBand {
		// M for repetitions: a band takes load off the structure (SEL-10,
		// INJ-05).
		if eb, ok := g.capacity(ex, AssistBand); ok {
			a.cap = eb
		}
		a.assist = AssistBand
		a.reasons = append(a.reasons, a.region)
	}
	// Probe offer (ADAPT-05, ADAPT-10) when the ladder earned it and
	// ADAPT-06a allows: the next rung, or for a band-assisted rung the same
	// rung without the band.
	ls := g.s.Ladders[a.skill.Slug]
	if ls.ProbeOffer && g.deload == "" && g.probesAllowed(ex) {
		next := g.nextRung(ex)
		if a.assist == AssistBand {
			next = ex
		}
		if next != nil {
			if ok, _, _ := g.feasible(next, next.OG); ok {
				a.probe = next
			}
		}
	}
}

// lowerRung returns the next easier non-eccentric rung.
func (g *gen) lowerRung(ex *Exercise) *Exercise {
	sk := g.k.skills[ex.Skill]
	for r := ex.Rung - 1; r >= 0; r-- {
		c := g.k.exercises[sk.Rungs[r]]
		if !c.Eccentric && c.Measure == ex.Measure {
			return c
		}
	}
	return nil
}

func (g *gen) nextRung(ex *Exercise) *Exercise { return g.k.nextRungOf(ex) }

// nextRungOf returns the next harder non-eccentric rung of the same measure.
func (k *Knowledge) nextRungOf(ex *Exercise) *Exercise {
	sk := k.skills[ex.Skill]
	for r := ex.Rung + 1; r < len(sk.Rungs); r++ {
		c := k.exercises[sk.Rungs[r]]
		if !c.Eccentric && c.Measure == ex.Measure {
			return c
		}
	}
	return nil
}

// probesAllowed applies ADAPT-06a for this plan.
func (g *gen) probesAllowed(ex *Exercise) bool {
	return g.k.probeGate(g.s, g.hist, ex, g.week)
}

// probeGate is ADAPT-06a: region normal, no ramp on the accounts (after a
// pause: no straight-arm or wrist load), not in the first week of a new
// load, no screening or consent limits. Plan and adaptation share it, so an
// announced offer is one the plan can show.
func (k *Knowledge) probeGate(s Snapshot, h loadHistory, ex *Exercise, week time.Time) bool {
	if s.Screening.AnyYes && !s.Screening.Cleared || !s.Profile.HealthConsent {
		return false
	}
	loads := k.setLoad(ex, KindWorking, 0, s.Profile.BodyweightKg)
	if b := s.Break; b != nil {
		if ex.StraightArm != ArmNone {
			return false
		}
		for acc := range loads {
			if isStraightAccount(acc) && b.StraightDays >= k.T.LayoffDays {
				return false
			}
		}
	}
	for acc := range loads {
		for _, r := range k.regionsOf(acc) {
			if rs, ok := s.Regions[r]; ok && rs.State != StateNormal {
				return false
			}
		}
		if ref, _ := k.reference(h, acc, week); ref == 0 {
			return false
		}
	}
	return true
}

// frequency is the target number of exposures of a ladder (WEEK-04).
func (g *gen) frequency(a *active) int {
	k := g.k
	strength := k.T.FreqTrained
	if g.exp == ExpNovice {
		strength = k.T.FreqNovice
	}
	f := strength
	switch {
	case a.maintain:
		f = k.T.MaintSessions
	case a.stim == StimBalance:
		f = k.T.FreqBalance
	case a.role == RoleSupport:
		f = math.Min(strength, float64(g.fullCount))
	}
	return int(math.Min(f, k.T.FreqMax))
}

func (g *gen) exclude(ex *Exercise, r Reason) {
	for _, e := range g.plan.Exclusions {
		if e.Exercise == ex.Slug {
			return
		}
	}
	g.plan.Exclusions = append(g.plan.Exclusions, Exclusion{Exercise: ex.Slug, Reason: r})
}

// dayGap returns hours between two weekday indices going forward.
func dayGap(from, to time.Weekday) float64 {
	return float64((int(to)-int(from)+7)%7) * 24
}
