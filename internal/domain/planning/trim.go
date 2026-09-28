package planning

import (
	"math"
	"slices"
)

const eps = 1e-9

// itemRef addresses one item of the plan.
type itemRef struct{ s, b, i int }

func (g *gen) item(r itemRef) *Item {
	return &g.plan.Sessions[r.s].Blocks[r.b].Items[r.i]
}

// itemLoad is the load of all sets of an item per account.
func (g *gen) itemLoad(it Item) map[string]float64 {
	ex := g.k.exercises[it.Exercise]
	out := map[string]float64{}
	if ex == nil || it.Sets <= 0 {
		return out
	}
	for a, u := range g.k.setLoad(ex, it.Kind, it.LoadKg, g.s.Profile.BodyweightKg) {
		out[a] = float64(u * float64(it.Sets))
	}
	return out
}

func (g *gen) weekLoads() map[string]float64 {
	out := map[string]float64{}
	g.eachItem(func(_ itemRef, it *Item) {
		for a, u := range g.itemLoad(*it) {
			out[a] += u
		}
	})
	return out
}

func (g *gen) sessionLoads(si int) map[string]float64 {
	out := map[string]float64{}
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			for a, u := range g.itemLoad(it) {
				out[accountStructure(a)] += u
			}
		}
	}
	return out
}

func (g *gen) eachItem(fn func(itemRef, *Item)) {
	for si := range g.plan.Sessions {
		for bi := range g.plan.Sessions[si].Blocks {
			for ii := range g.plan.Sessions[si].Blocks[bi].Items {
				fn(itemRef{si, bi, ii}, &g.plan.Sessions[si].Blocks[bi].Items[ii])
			}
		}
	}
}

// weekFactor is f(a): the minimum of the applicable factors (LOAD-02).
func (g *gen) weekFactor(a string) float64 {
	k := g.k
	f := 1.0
	for _, r := range k.regionsOf(a) {
		if rs, ok := g.s.Regions[r]; ok && rs.PriorInjury {
			f = math.Min(f, k.T.PriorInjury)
		}
	}
	if !g.s.Profile.HealthConsent {
		f = math.Min(f, k.T.PriorInjury)
	}
	months := g.s.Profile.TrainingMonths
	if months >= k.T.RiskWindowLo && months < k.T.RiskWindowHi {
		f = math.Min(f, k.T.RiskWindow)
	}
	return f
}

// weekCap is the weekly cap of an account and the rule that set it. A later
// rule only names the cap when it is the tighter one, so the headroom carry
// (PAR-S-35) follows the rule that actually binds.
func (g *gen) weekCap(a string, target float64) (float64, string) {
	k := g.k
	R, _ := k.reference(g.hist, a, g.week)
	var cp float64
	rule := RuleWeekCap
	tighten := func(v float64, r string) {
		if v < cp {
			cp, rule = v, r
		}
	}
	switch {
	case g.s.Entry[a] && g.weekIdx < 3:
		// The steps are shares of the target, but never more than their
		// ratio above the week logged before, so a target that grows with a
		// new rung does not enlarge the step (PAR-S-43).
		steps := []float64{k.T.EntryStep1, k.T.EntryStep2, k.T.EntryStep3}
		idx := max(0, min(g.weekIdx-g.breachWeeks(a), 2))
		cp, rule = float64(steps[idx]*target), RuleEntryRamp
		if prev := g.hist.weekly[g.week.AddDate(0, 0, -7)][a]; idx > 0 && prev > 0 {
			cp = math.Min(cp, float64(prev*steps[idx]/steps[idx-1]))
		}
	case R == 0:
		cp, rule = float64(k.T.NewTypeFraction*target), RuleNewLoad
	default:
		cp = float64(R*(1+float64(k.capRate(a)*g.weekFactor(a)))) + g.s.Headroom[a]
		// Right after the entry ramp the mean still holds the ramp weeks;
		// until it is made of full weeks the cap does not fall below the
		// week logged before (PAR-S-43).
		if g.s.Entry[a] && float64(g.weekIdx) < 3+k.T.RefWeeks {
			cp = math.Max(cp, g.hist.weekly[g.week.AddDate(0, 0, -7)][a])
		}
	}
	// Break ramp (§6.11). With a logged level before the pause, the ramp
	// fractions of that level replace the cap of straight-arm and wrist
	// accounts: the athlete returns to a load they tolerated (05 §6.2).
	// Without one there is no tolerated level, so LOAD-04 and LOAD-02 stay
	// and the ramp only adds a bound (§7.2). Bent-arm accounts follow the
	// stream-B factors as an upper bound.
	if b := g.s.Break; b != nil {
		weeks := math.Max(0, math.Floor(daysBetween(b.Since, g.week)/7))
		ref := b.Reference[a]
		if isStraightAccount(a) && b.StraightDays >= k.T.LayoffDays {
			steps := []float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}
			frac := steps[min(b.Step, 3)]
			if ref > 0 {
				cp, rule = float64(frac*ref), RuleBreak
			} else {
				tighten(float64(frac*target), RuleBreak)
			}
		} else if f := g.breakFactor(b.Days); f < 1 {
			base := target
			if ref > 0 {
				base = ref
			}
			tighten(math.Min(1, float64(f*math.Pow(k.T.BreakGrowth, weeks)))*base, RuleBreak)
		}
	}
	for _, r := range k.regionsOf(a) {
		rs, ok := g.s.Regions[r]
		if !ok {
			continue
		}
		if rs.HoldAtRef {
			ref := R
			if v := rs.Reference[a]; v > 0 {
				ref = v
			}
			if ref > 0 {
				tighten(ref, RulePain)
			}
		}
		st := rttStage(rs.State)
		if st < 1 || st > 4 || !slices.Contains(k.rampAccounts(r), a) {
			continue
		}
		frac := k.rampFraction(rs)
		if ref := rs.Reference[a]; ref > 0 {
			// A logged pre-complaint level: the ramp replaces the cap (§7.2).
			cp, rule = float64(frac*ref), RuleRamp
		} else {
			tighten(float64(frac*target), RuleRamp)
		}
	}
	return cp, rule
}

// breakFactor is the week-1 volume of a return after a pause (spec §6.11).
func (g *gen) breakFactor(days float64) float64 {
	k := g.k
	switch {
	case days >= k.T.BreakHalf:
		return k.T.BreakFactorHalf
	case days >= k.T.BreakQuarter:
		return k.T.BreakFactorQuarter
	case days >= k.T.BreakMonth:
		return k.T.BreakFactorMonth
	case days >= k.T.BreakMid:
		return k.T.BreakFactorMid
	}
	return 1
}

// breachWeeks counts pain-rule breaches on an account's regions since the
// onboarding; each one holds the entry ramp for a week (LOAD-04b).
func (g *gen) breachWeeks(a string) int {
	n := 0
	for _, r := range g.k.regionsOf(a) {
		for _, t := range g.s.Regions[r].Breaches {
			if !t.Before(g.s.Profile.OnboardedAt) {
				n++
			}
		}
	}
	return n
}

// sessionCap is the spike cap of a structure in session si (LOAD-03). It
// always admits one whole set more than the 30-day maximum, the smallest
// set of the structure in the session (PAR-S-48). The entry and break ramps
// use the same cap (ENT-R-3): their steps grow the week, not the session.
func (g *gen) sessionCap(si int, structure string, target float64) float64 {
	k := g.k
	unit := g.smallestSet(si, structure)
	m := g.hist.sessionMax[structure]
	if m == 0 {
		return math.Max(float64(k.T.NewTypeFraction*target), unit)
	}
	return math.Max(float64(m*(1+k.T.SpikeCap)), m+unit)
}

// smallestSet is the load of the smallest working set on a structure in
// session si.
func (g *gen) smallestSet(si int, structure string) float64 {
	best := 0.0
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			if it.Sets <= 0 || it.Kind != KindWorking {
				continue
			}
			u := 0.0
			for a, v := range g.itemLoad(it) {
				if accountStructure(a) == structure {
					u += v / float64(it.Sets)
				}
			}
			if u > eps && (best == 0 || u < best) {
				best = u
			}
		}
	}
	return best
}

// budget is the straight-arm set budget of a session (LOAD-06).
func (g *gen) budget(si int) (float64, int) {
	k := g.k
	og, sets := 0.0, 0
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			ex := k.exercises[it.Exercise]
			if ex == nil || ex.StraightArm != ArmStraight || it.Kind != KindWorking || it.Sets <= 0 {
				continue
			}
			sets += it.Sets
			og = math.Max(og, ex.OG)
		}
	}
	switch {
	case og >= k.T.AdvancedOG:
		return k.T.BudgetAdvanced, sets
	case og >= k.T.IntermediateOG:
		return k.T.BudgetIntermediate, sets
	default:
		return k.T.BudgetBeginner, sets
	}
}

// violation is one broken cap.
type violation struct {
	kind    string // week, session, budget
	account string
	session int
	rule    string
	cap     float64
}

// trimLoads enforces the caps and the budget by deterministic cuts
// (LOAD-02–LOAD-06, LOAD-10). Every step removes a set or lowers a rung, so
// the loop ends.
func (g *gen) trimLoads() {
	targetWeek := g.weekLoads()
	targetSess := make([]map[string]float64, len(g.plan.Sessions))
	for si := range g.plan.Sessions {
		targetSess[si] = g.sessionLoads(si)
	}
	caps := map[string]float64{}
	rules := map[string]string{}
	for _, a := range sortedKeys(targetWeek) {
		caps[a], rules[a] = g.weekCap(a, targetWeek[a])
	}
	for iter := 0; iter < 10000; iter++ {
		v, ok := g.findViolation(caps, rules, targetSess)
		if !ok {
			break
		}
		if !g.cut(v) {
			break
		}
	}
	// Carry the unused fraction of a set (PAR-S-35).
	week := g.weekLoads()
	g.caps, g.capRules, g.targets = caps, rules, targetWeek
	g.plan.Headroom = map[string]float64{}
	for _, a := range sortedKeys(caps) {
		if rules[a] != RuleWeekCap {
			continue
		}
		unit := 0.0
		g.eachItem(func(_ itemRef, it *Item) {
			if it.Sets > 0 {
				unit = math.Max(unit, g.itemLoad(*it)[a]/float64(it.Sets))
			}
		})
		if left := caps[a] - week[a]; left > eps && unit > 0 {
			g.plan.Headroom[a] = math.Min(left, unit)
		}
	}
}

func (g *gen) findViolation(caps map[string]float64, rules map[string]string, targetSess []map[string]float64) (violation, bool) {
	week := g.weekLoads()
	for _, a := range sortedKeys(week) {
		if week[a] > caps[a]+eps {
			return violation{kind: "week", account: a, rule: rules[a], cap: caps[a]}, true
		}
	}
	for si := range g.plan.Sessions {
		sl := g.sessionLoads(si)
		for _, st := range sortedKeys(sl) {
			c := g.sessionCap(si, st, targetSess[si][st])
			if sl[st] > c+eps {
				return violation{kind: "session", account: st, session: si, rule: RuleSessionCap, cap: c}, true
			}
		}
		if b, n := g.budget(si); float64(n) > b {
			return violation{kind: "budget", session: si, rule: RuleBudget, cap: b}, true
		}
	}
	return violation{}, false
}

// contributes reports whether an item adds to the violated quantity.
func (g *gen) contributes(v violation, r itemRef, it *Item) bool {
	if it.Sets <= 0 {
		return false
	}
	switch v.kind {
	case "week":
		return g.itemLoad(*it)[v.account] > 0
	case "session":
		if r.s != v.session {
			return false
		}
		for a := range g.itemLoad(*it) {
			if accountStructure(a) == v.account {
				return true
			}
		}
		return false
	default:
		ex := g.k.exercises[it.Exercise]
		return r.s == v.session && ex.StraightArm == ArmStraight && it.Kind == KindWorking
	}
}

// cutStep ranks an item by the LOAD-10 order; the floor is the smallest
// set count that step may reach. Working items keep at least one set until
// step 7, so a halved first week (LOAD-04) plans fewer sets of every
// exercise rather than fewer exercises.
func (g *gen) cutStep(si int, it *Item, block string) (int, int) {
	switch {
	case it.Kind == KindWarmup:
		return 1, 0
	case it.Offer:
		// Offers belong to the max block (ADAPT-05) and go last in it.
		return 5, 0
	case it.Calibration && it.Exercise != g.ladderRung(it.Skill) && hasRule(it.Reasons, RuleBreak) && g.hasWork(si, it.Skill):
		// A break calibration on top of the work goes first (§6.11).
		return 1, 0
	case it.Role == RoleSupport && it.Stimulus != StimPrehab:
		return 1, 1
	case it.Stimulus == StimBalance || it.Stimulus == StimTechnique || it.Stimulus == StimPrehab:
		return 2, 0
	case block == BlockVolume:
		return 3, 0
	case block == BlockStrength || block == BlockEnd:
		return 4, 1
	case block == BlockMax:
		return 5, 1
	}
	return 6, 0
}

// cut performs one LOAD-10 step for a violation.
func (g *gen) cut(v violation) bool {
	k := g.k
	type cand struct {
		ref   itemRef
		step  int
		floor int
	}
	var cands []cand
	g.eachItem(func(r itemRef, it *Item) {
		if !g.contributes(v, r, it) {
			return
		}
		step, floor := g.cutStep(r.s, it, g.plan.Sessions[r.s].Blocks[r.b].Role)
		cands = append(cands, cand{r, step, floor})
	})
	slices.SortStableFunc(cands, func(a, b cand) int {
		if a.step != b.step {
			return a.step - b.step
		}
		ia, ib := g.item(a.ref), g.item(b.ref)
		// Within a step, the item with the most sets first, so cuts spread
		// over the items instead of emptying one after the other.
		if ia.Sets != ib.Sets {
			return ib.Sets - ia.Sets
		}
		if ia.Priority != ib.Priority {
			return ib.Priority - ia.Priority // lowest priority (largest number) first
		}
		if a.ref.s != b.ref.s {
			return b.ref.s - a.ref.s
		}
		return b.ref.i - a.ref.i
	})
	reason := k.reason(RuleTrim, "cap", v.rule)
	for _, c := range cands {
		it := g.item(c.ref)
		if it.Sets > c.floor {
			it.Sets--
			g.note(it, reason, v.rule)
			return true
		}
	}
	// Step 6: one rung lower for held max items. A calibration set at
	// another rung measures that rung and is not lowered.
	for _, c := range cands {
		it := g.item(c.ref)
		if g.plan.Sessions[c.ref.s].Blocks[c.ref.b].Role != BlockMax || it.Stimulus != StimSkill || it.Exercise != g.ladderRung(it.Skill) {
			continue
		}
		ex := k.exercises[it.Exercise]
		if lower := g.lowerRung(ex); lower != nil && g.lighter(lower, ex, v) {
			if ok, _, _ := g.feasible(lower, lower.OG); ok {
				it.Exercise = lower.Slug
				g.note(it, k.reason(RuleNewRung, "exercise", lower.Name), v.rule)
				return true
			}
		}
	}
	// Step 7: drop the exposure, lowest priority first.
	slices.SortStableFunc(cands, func(a, b cand) int {
		ia, ib := g.item(a.ref), g.item(b.ref)
		if ia.Priority != ib.Priority {
			return ib.Priority - ia.Priority
		}
		return a.step - b.step
	})
	for _, c := range cands {
		it := g.item(c.ref)
		if it.Sets > 0 {
			it.Sets = 0
			g.note(it, reason, v.rule)
			return true
		}
	}
	return false
}

func hasRule(rs []Reason, id string) bool {
	return slices.ContainsFunc(rs, func(r Reason) bool { return r.RuleID == id })
}

// hasWork reports whether session si has working sets of a skill at its
// working rung.
func (g *gen) hasWork(si int, skill string) bool {
	rung := g.ladderRung(skill)
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			if it.Skill == skill && it.Exercise == rung && it.Kind == KindWorking && it.Sets > 0 {
				return true
			}
		}
	}
	return false
}

// lighter reports whether a set of lower loads the violated quantity at
// least the spike margin (PAR-D-31) less than a set of ex; a rung that is
// barely lighter is no remedy, and the exposure is cut instead.
func (g *gen) lighter(lower, ex *Exercise, v violation) bool {
	if v.kind == "budget" {
		return lower.StraightArm != ArmStraight
	}
	bw := g.s.Profile.BodyweightKg
	sum := func(e *Exercise) float64 {
		t := 0.0
		for a, u := range g.k.setLoad(e, KindWorking, 0, bw) {
			if a == v.account || v.kind == "session" && accountStructure(a) == v.account {
				t += u
			}
		}
		return t
	}
	return sum(lower) <= sum(ex)*(1-g.k.T.SpikeCap)+eps
}

func (g *gen) note(it *Item, r Reason, capRule string) {
	for _, x := range it.Reasons {
		if x.RuleID == r.RuleID && x.Args["cap"] == r.Args["cap"] {
			return
		}
	}
	if capRule != "" && capRule != r.RuleID {
		it.Reasons = append(it.Reasons, g.k.reason(capRule))
	}
	it.Reasons = append(it.Reasons, r)
}

// minutes estimates the duration of a session (spec §5.8).
func (g *gen) minutes(ps PlannedSession) float64 {
	k := g.k
	total := 0.0
	for _, b := range ps.Blocks {
		if b.Role == BlockWarmup {
			total += b.Minutes
			continue
		}
		var secs float64
		var sets, maxRest float64
		for _, it := range b.Items {
			if it.Sets <= 0 {
				continue
			}
			work := float64(it.HoldS)
			if it.Reps > 0 {
				work = float64(float64(it.Reps) * k.T.SecondsPerRep)
				if it.HoldS > 0 {
					work = float64(float64(it.Reps) * float64(it.HoldS))
				}
			}
			n := float64(it.Sets)
			if b.Paired {
				secs += float64(n * work)
				sets += n
				maxRest = math.Max(maxRest, math.Max(k.T.RestPaired, float64(it.RestS)/2))
				continue
			}
			secs += float64(n*(work+float64(it.RestS))) - float64(it.RestS) + k.T.TransitionS
		}
		if b.Paired && sets > 0 {
			secs += float64((sets-1)*maxRest) + k.T.TransitionS
		}
		total += secs / 60
	}
	return total
}

// trimTime fits every session into the available minutes (PAR-B-68, §5.8).
func (g *gen) trimTime() {
	k := g.k
	limit := float64(g.s.Profile.SessionMinutes)
	for si := range g.plan.Sessions {
		ps := &g.plan.Sessions[si]
		steps := []func() bool{
			func() bool { return g.pairMax(ps) },
			func() bool {
				return g.cutWhere(ps, func(b string, it *Item) bool {
					return it.Role == RoleSupport && it.Kind == KindWorking && it.Stimulus != StimPrehab
				}, 0)
			},
			func() bool { return g.cutWhere(ps, func(b string, it *Item) bool { return b == BlockVolume }, 2) },
			func() bool { return g.shortenRests(ps) },
			func() bool { return g.shrinkBalance(ps) },
			func() bool {
				return g.cutWhere(ps, func(b string, it *Item) bool { return b == BlockVolume && it.Priority > 1 }, 0)
			},
			func() bool { return g.cutWhere(ps, func(b string, it *Item) bool { return b == BlockEnd }, 0) },
			func() bool { return g.cutWhere(ps, func(b string, it *Item) bool { return b == BlockStrength }, 2) },
			func() bool {
				return g.cutWhere(ps, func(b string, it *Item) bool { return b == BlockMax && it.Priority > 1 }, 0)
			},
			func() bool {
				return g.cutWhere(ps, func(b string, it *Item) bool {
					return b == BlockStrength && (it.Priority > 1 || it.Role == RoleSupport)
				}, 0)
			},
		}
		for _, step := range steps {
			for g.minutes(*ps) > limit+eps {
				if !step() {
					break
				}
			}
		}
		ps.EstMinutes = math.Round(g.minutes(*ps))
		if ps.EstMinutes > limit {
			ps.Reasons = append(ps.Reasons, k.reason(RuleTrim, "cap", "time"))
		}
		// Drop emptied items and blocks.
		var blocks []Block
		for _, b := range ps.Blocks {
			b.Items = slices.DeleteFunc(b.Items, func(it Item) bool { return it.Sets <= 0 })
			if len(b.Items) > 0 || b.Role == BlockWarmup {
				blocks = append(blocks, b)
			}
		}
		ps.Blocks = blocks
	}
}

// cutWhere removes one set from the lowest-priority matching item, not
// below floor.
func (g *gen) cutWhere(ps *PlannedSession, match func(string, *Item) bool, floor int) bool {
	bestB, bestI := -1, -1
	for bi := range ps.Blocks {
		for ii := range ps.Blocks[bi].Items {
			it := &ps.Blocks[bi].Items[ii]
			if it.Sets <= floor || !match(ps.Blocks[bi].Role, it) {
				continue
			}
			if bestB < 0 || it.Priority > ps.Blocks[bestB].Items[bestI].Priority ||
				it.Priority == ps.Blocks[bestB].Items[bestI].Priority && (bi > bestB || bi == bestB && ii > bestI) {
				bestB, bestI = bi, ii
			}
		}
	}
	if bestB < 0 {
		return false
	}
	it := &ps.Blocks[bestB].Items[bestI]
	it.Sets--
	g.note(it, g.k.reason(RuleTrim, "cap", "time"), "")
	return true
}

// pairMax pairs two opposite-direction straight-arm max items when time is
// short (LOAD-08, §7.7).
func (g *gen) pairMax(ps *PlannedSession) bool {
	for bi := range ps.Blocks {
		b := &ps.Blocks[bi]
		if b.Role != BlockMax || b.Paired {
			continue
		}
		dirs := map[string]bool{}
		for _, it := range b.Items {
			ex := g.k.exercises[it.Exercise]
			if ex.StraightArm == ArmStraight && it.Stimulus == StimSkill {
				dirs[ex.Direction] = true
			}
		}
		if dirs[DirPush] && dirs[DirPull] {
			b.Paired = true
			b.Reasons = append(b.Reasons, g.k.reason(RulePairing))
			return true
		}
	}
	return false
}

func (g *gen) shortenRests(ps *PlannedSession) bool {
	k := g.k
	changed := false
	for bi := range ps.Blocks {
		for ii := range ps.Blocks[bi].Items {
			it := &ps.Blocks[bi].Items[ii]
			var floor float64
			switch {
			case ps.Blocks[bi].Role == BlockVolume:
				floor = k.T.RestVolumeFloor
			case it.Stimulus == StimStrength && g.exp == ExpNovice:
				floor = k.T.RestStrengthNovice
			case it.Stimulus == StimStrength:
				floor = math.Min(k.T.RestStrengthTrained, float64(it.RestS))
			case it.Stimulus == StimAccessory || it.Stimulus == StimConditioning:
				floor = k.T.RestAccessory
			default:
				continue
			}
			if float64(it.RestS) > floor {
				it.RestS = g.rest(floor)
				changed = true
			}
		}
	}
	return changed
}

func (g *gen) shrinkBalance(ps *PlannedSession) bool {
	k := g.k
	for bi := range ps.Blocks {
		if ps.Blocks[bi].Role != BlockBalance {
			continue
		}
		for ii := range ps.Blocks[bi].Items {
			it := &ps.Blocks[bi].Items[ii]
			if it.Stimulus != StimBalance {
				continue
			}
			minSets := int(math.Floor(k.T.BalanceFloorMin * 60 / (float64(it.HoldS) + float64(it.RestS))))
			if it.Sets > minSets {
				it.Sets = minSets
				return true
			}
		}
	}
	return false
}

// round3 rounds for display; plans stay byte-stable because the inputs are.
func round3(x float64) float64 { return math.Round(x*1000) / 1000 }
