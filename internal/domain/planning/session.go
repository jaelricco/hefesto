package planning

import (
	"math"
	"slices"
	"strings"
)

// template returns the session template for the available minutes.
func (k *Knowledge) template(minutes int) Template {
	best := k.templates[0]
	for _, t := range k.templates {
		if t.Minutes <= minutes {
			best = t
		}
	}
	return best
}

// buildSessions turns the slots into sessions with blocks and dosed items
// (SESS-01–SESS-09, DOSE-*).
func (g *gen) buildSessions() {
	k := g.k
	tpl := k.template(g.s.Profile.SessionMinutes)
	prehabLeft := int(k.T.PrehabSessions)
	fulls := 0
	for _, sl := range g.slots {
		if sl.full {
			fulls++
		}
	}
	prehabLeft = min(prehabLeft, fulls)
	exposure := map[string]int{}
	for i, sl := range g.slots {
		ps := PlannedSession{Index: i, Date: sl.date, Kind: SessionFull}
		if !sl.full {
			ps.Kind = SessionLight
		}
		ps.Reasons = append(ps.Reasons, k.reason(RuleTemplate, "minutes", tpl.Minutes))
		warm := Block{Role: BlockWarmup, Minutes: tpl.WarmupMin}
		warm.Reasons = append(warm.Reasons, k.reason(RuleWarmup, "minutes", tpl.WarmupMin))
		var balance, maxB, volB, strB, endB []Item
		for _, a := range sl.ladders {
			idx := exposure[a.skill.Slug]
			exposure[a.skill.Slug]++
			reasons := append([]Reason(nil), a.reasons...)
			if a.region.RuleID != "" {
				reasons = append(reasons, a.region)
			}
			switch {
			case a.stim == StimBalance:
				mins := k.T.BalanceMin
				if float64(g.s.Profile.SessionMinutes) < k.T.BalanceShortUnder {
					mins = k.T.BalanceShortMin
				}
				balance = append(balance, g.balanceItem(a, mins, reasons))
			case !sl.full:
				continue
			case a.role == RoleGoal && (a.stim == StimSkill || a.stim == StimSkillReps || a.stim == StimConditioning || a.stim == StimEccentric || a.stim == StimStrength):
				maxB = append(maxB, g.nextRungCalibration(a, idx)...)
				maxB = append(maxB, g.primary(a, idx, reasons)...)
				if tpl.VolumeBlocks && a.stim == StimSkill {
					if v, ok := g.volumeItem(a); ok {
						volB = append(volB, v)
					}
				}
			default:
				strB = append(strB, g.probeItems(a)...)
				strB = append(strB, g.nextRungCalibration(a, idx)...)
				strB = append(strB, g.secondary(a, idx, reasons))
			}
		}
		if !sl.full {
			for _, a := range g.ladders {
				if a.role == RoleGoal && a.stim == StimSkill && a.rung.StraightArm == ArmStraight && g.techniqueFits(a, sl) {
					balance = append(balance, g.techniqueItem(a))
				}
			}
			ps.Reasons = append(ps.Reasons, k.reason(RuleLightDay))
		}
		if sl.full && prehabLeft > 0 {
			warm.Items = append(warm.Items, g.prehabItems()...)
			endB = append(endB, g.loadedPrehab()...)
			prehabLeft--
		}
		if !sl.full {
			warm.Items = append(warm.Items, g.prehabItems()...)
		}
		warm.Items = append(warm.Items, g.rampSets(maxB)...)
		ps.Blocks = append(ps.Blocks, warm)
		add := func(role string, items []Item, r Reason) {
			if len(items) > 0 {
				ps.Blocks = append(ps.Blocks, Block{Role: role, Items: items, Reasons: []Reason{r}})
			}
		}
		add(BlockBalance, balance, k.reason(RuleBalance))
		add(BlockMax, maxB, k.reason(RuleMaxBlock))
		add(BlockVolume, volB, k.reason(RuleVolume))
		add(BlockStrength, strB, k.reason(RuleStrength))
		add(BlockEnd, endB, k.reason(RuleEndBlock))
		ps.Reasons = append(ps.Reasons, k.reason(RuleOrder))
		g.plan.Sessions = append(g.plan.Sessions, ps)
	}
}

func (g *gen) baseItem(a *active, ex *Exercise, stim string, reasons []Reason) Item {
	return Item{Exercise: ex.Slug, Skill: a.skill.Slug, Stimulus: stim, Kind: KindWorking, Assist: a.assist,
		Class: a.class, Priority: a.priority, Role: a.role, Monitor: a.monitor, Calibration: a.calib,
		Reasons: slices.Clone(reasons)}
}

func (g *gen) rest(v float64) int {
	r := g.k.T.RestRound
	return int(math.Round(v/r) * r)
}

// primary doses the first block of a goal ladder.
func (g *gen) primary(a *active, idx int, reasons []Reason) []Item {
	out := g.probeItems(a)
	switch a.stim {
	case StimSkill:
		out = append(out, g.skillHold(a, reasons))
	case StimSkillReps:
		out = append(out, g.skillReps(a, reasons))
	case StimConditioning:
		out = append(out, g.conditioning(a, reasons))
	case StimEccentric:
		out = append(out, g.eccentric(a, reasons))
	default:
		out = append(out, g.strength(a, idx, reasons))
	}
	return out
}

// probeItems is the offer of ADAPT-05 or ADAPT-10: two short attempts at
// the next rung (or the first concentric repetition), only on acceptance.
func (g *gen) probeItems(a *active) []Item {
	k := g.k
	if a.probe == nil {
		return nil
	}
	rule := RuleProbe
	if a.stim == StimEccentric {
		rule = RuleEccToConc
	}
	p := g.baseItem(a, a.probe, StimSkill, []Reason{k.reason(rule, "exercise", a.probe.Name), k.reason(RuleProbeGate)})
	p.Assist, p.Calibration, p.Offer = "", false, true
	p.Sets, p.RestS, p.Reserve = int(k.T.ProbeAttempts), g.rest(k.T.RestMax), 1
	if a.probe.Measure == MeasureHold {
		p.HoldS = int(k.T.ProbeHold)
	} else {
		// The first concentric attempt (ADAPT-10) is a calibration set: as
		// many clean repetitions as the reserve allows, at least one.
		p.Stimulus, p.Reps, p.Calibration, p.Sets = StimSkillReps, 1, true, 1
		p.Reserve = int(k.T.RIRStrength)
	}
	p.StopRules = stopRules()
	return []Item{p}
}

// nextRungCalibration is the weekly calibration set at the next rung while a
// novice ladder waits at 3 × 8 for that rung's value to reach PAR-S-33
// (ADAPT-07).
func (g *gen) nextRungCalibration(a *active, idx int) []Item {
	k := g.k
	ls := g.s.Ladders[a.skill.Slug]
	if idx != 0 || g.exp != ExpNovice || a.stim != StimStrength || ls.Rung != a.rung.Slug || ls.RepTarget < k.T.NoviceRepsHi {
		return nil
	}
	next := g.nextRung(a.rung)
	if next == nil {
		return nil
	}
	if ok, _, _ := g.feasible(next, next.OG); !ok {
		return nil
	}
	if e, ok := g.capacity(next, AssistNone); ok && k.Dose(e) >= k.T.NextRungMinDose {
		return nil
	}
	it := g.baseItem(a, next, StimStrength, []Reason{k.reason(RuleDoubleProg, "reps", k.T.NoviceRestart)})
	it.Sets, it.Reps, it.Reserve, it.Calibration = 1, int(k.T.NoviceRestart), int(k.T.RIRStrength), true
	it.RestS = g.rest(k.T.RestStrengthNovice)
	return []Item{it}
}

// secondary doses feeder and support ladders in the strength block.
func (g *gen) secondary(a *active, idx int, reasons []Reason) Item {
	switch a.stim {
	case StimEccentric:
		return g.eccentric(a, reasons)
	case StimConditioning:
		return g.conditioning(a, reasons)
	case StimSkill:
		return g.conditioning(a, reasons)
	case StimSkillReps:
		return g.skillReps(a, reasons)
	}
	if a.role == RoleSupport {
		return g.accessory(a, reasons)
	}
	return g.strength(a, idx, reasons)
}

func stopRules() []string {
	return []string{RuleStopForm, RuleStopFails, RuleStopDrop, RuleStopPain}
}

// holdFor is the set hold from a dose value: min(fraction · d, d − 2),
// at least the minimum set hold (08 §4, DOSE-01).
func (g *gen) holdFor(d float64) float64 {
	k := g.k
	h := math.Min(float64(k.T.SetHoldFraction*d), d-k.T.MinSetHold)
	return math.Max(k.T.MinSetHold, math.Floor(h))
}

func (g *gen) skillHold(a *active, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimSkill, reasons)
	d := k.Dose(a.cap)
	if !a.hasCap || d < 2*k.T.MinSetHold {
		// Calibration start without a usable value (§4.3, PAR-S-39).
		it.HoldS, it.Sets, it.Reserve, it.Calibration = int(k.T.VolumeHoldMin), int(k.T.MinSets), 2, true
	} else {
		h := g.holdFor(d)
		if capped, ok := g.holdGrowthCap(a.rung, h); ok {
			h = capped
			it.Reasons = append(it.Reasons, k.reason(RuleHoldGrowth))
		}
		it.HoldS = int(h)
		it.Sets = int(clamp(math.Round(k.T.TargetTotalHold/h), k.T.MinSets, k.T.MaxSets))
		it.Reserve = int(math.Floor(d - h))
	}
	it.RestS = g.rest(k.T.RestMax)
	if a.rung.OG >= k.T.HeaviestOG {
		it.RestS = g.rest(k.T.RestHeaviest)
	}
	it.StopRules = stopRules()
	it.Reasons = append(it.Reasons, k.reason(RuleDoseMaxHold, "hold_s", it.HoldS, "max_s", math.Floor(d), "sets", it.Sets))
	return it
}

// holdGrowthCap applies ADAPT-04: within a rung the set hold grows by at
// most PAR-B-33 per week over the longest working hold logged on it in the
// last seven days.
func (g *gen) holdGrowthCap(ex *Exercise, h float64) (float64, bool) {
	prev := 0.0
	for _, sess := range g.s.History {
		if gap := daysBetween(sess.Date, g.week); gap <= 0 || gap > 7 {
			continue
		}
		for _, set := range sess.Sets {
			if set.Exercise == ex.Slug && set.Kind == KindWorking && (set.Assist == "" || set.Assist == AssistNone) {
				prev = math.Max(prev, set.Value)
			}
		}
	}
	if prev > 0 && h > prev+g.k.T.HoldStepPerWeek {
		return prev + g.k.T.HoldStepPerWeek, true
	}
	return h, false
}

func (g *gen) skillReps(a *active, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimSkillReps, reasons)
	d := k.Dose(a.cap)
	it.Reps = int(math.Max(1, math.Floor(d)-1))
	it.Sets = int(k.T.DefaultMaxSets)
	it.Reserve = 1
	it.RestS = g.rest(k.T.RestMax)
	it.StopRules = stopRules()
	it.Reasons = append(it.Reasons, k.reason(RuleDoseMaxReps, "reps", it.Reps))
	return it
}

func (g *gen) volumeItem(a *active) (Item, bool) {
	k := g.k
	vr := g.lowerRung(a.rung)
	assist := AssistNone
	if vr != nil {
		if ok, _, _ := g.feasible(vr, vr.OG); !ok {
			vr = nil
		}
	}
	if vr == nil {
		if !g.hasBands || !a.rung.Assistable {
			return Item{}, false
		}
		vr, assist = a.rung, AssistBand
	}
	// A new rung keeps its volume one rung lower (LOAD-09); that is the
	// default here anyway.
	e, ok := g.capacity(vr, assist)
	if !ok {
		e = k.derivedPrior(a.cap, vr, g.now)
	}
	d := k.Dose(e)
	raw := math.Min(float64(k.T.SetHoldFraction*d), d-k.T.MinSetHold)
	if raw < k.T.VolumeHoldMin {
		return Item{}, false
	}
	h := clamp(math.Floor(raw), k.T.VolumeHoldMin, k.T.VolumeHoldMax)
	it := g.baseItem(a, vr, StimSkill, nil)
	it.Assist = assist
	it.Calibration = false
	it.HoldS = int(h)
	it.Sets = int(clamp(math.Round(k.T.TargetTotalHold/h), k.T.VolumeSetsMin, k.T.VolumeSetsMax))
	it.Reserve = int(math.Floor(d - h))
	it.RestS = g.rest(k.T.RestVolume)
	it.StopRules = stopRules()
	it.Reasons = append(it.Reasons, k.reason(RuleDoseVolume, "hold_s", it.HoldS, "sets", it.Sets))
	return it, true
}

func (g *gen) conditioning(a *active, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimConditioning, reasons)
	d := k.Dose(a.cap)
	h := clamp(math.Floor(math.Min(float64(k.T.SetHoldFraction*d), d-k.T.MinSetHold)), k.T.CondHoldMin, k.T.CondHoldMax)
	reserve := math.Floor(d - h)
	if !a.hasCap || d < k.T.CondHoldMin+k.T.MinSetHold {
		h = math.Max(k.T.MinSetHold, math.Floor(math.Min(float64(k.T.SetHoldFraction*d), d-k.T.MinSetHold)))
		reserve = math.Floor(d - h)
		if !a.hasCap {
			// No estimate: a calibration set at the lower bound of the
			// conditioning window, stopped with the usual reserve (§4.3).
			h, reserve = k.T.CondHoldMin, k.T.MinSetHold
		}
		it.Calibration = true
	}
	it.HoldS = int(h)
	it.Sets = int(k.T.CondSetsHi)
	if k.T.CondSetsHi*h > k.T.CondMaxTotal {
		it.Sets = int(k.T.CondSetsLo)
	}
	it.Reserve = int(math.Max(0, reserve))
	it.RestS = g.rest(k.T.RestAccessory)
	it.Reasons = append(it.Reasons, k.reason(RuleDoseCond, "hold_s", it.HoldS, "sets", it.Sets))
	return it
}

// strength doses bent-arm work: double progression for novices, weekly
// undulation otherwise (DOSE-05, DOSE-06, DOSE-11).
func (g *gen) strength(a *active, idx int, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimStrength, reasons)
	d := k.Dose(a.cap)
	ls := g.s.Ladders[a.skill.Slug]
	it.Reserve = int(k.T.RIRStrength)
	if g.exp == ExpNovice {
		reps := ls.RepTarget
		if reps == 0 || ls.Rung != a.rung.Slug {
			reps = clamp(math.Floor(d)-k.T.RIRStrength, k.T.NoviceRepsLo, k.T.NoviceRepsHi)
			if math.Floor(d)-k.T.RIRStrength < k.T.NoviceRepsLo {
				// Below the range (see SEL-09 fallback): what the reserve
				// allows, at least one; double progression climbs from here.
				reps = math.Max(1, math.Floor(d)-k.T.RIRStrength)
			}
		}
		it.Reps, it.Sets = int(reps), int(k.T.NoviceSets)
		it.RestS = g.rest(k.T.RestStrengthNovice)
		it.Reasons = append(it.Reasons, k.reason(RuleDoseNovice, "reps", it.Reps))
	} else {
		lo, hi, rest := g.undulation(a, idx)
		reps := clamp(math.Floor(d)-k.T.RIRStrength, 1, hi)
		it.Reps, it.Sets = int(reps), int(k.T.StrengthSets)
		it.RestS = g.rest(rest)
		it.Reasons = append(it.Reasons, k.reason(RuleDoseTrained, "reps", it.Reps, "range_lo", lo, "range_hi", hi))
	}
	if a.rung.Loadable && g.nextRung(a.rung) == nil && math.Floor(d)-k.T.RIRStrength >= k.T.RepToLoad && g.hasWeights() {
		load := ls.LoadKg
		if load == 0 {
			load = g.plate(float64(k.T.LoadIncrementPct / 100 * g.s.Profile.BodyweightKg))
		}
		if max := g.s.Profile.MaxAddedLoadKg; max > 0 {
			load = math.Min(load, max)
		}
		it.LoadKg = load
		it.Reps = int(k.T.HeavyHi)
		it.Calibration = true
		it.Reasons = append(it.Reasons, k.reason(RuleDoseLoad, "load_kg", load))
	}
	if !a.hasCap {
		it.Reps, it.Calibration = int(k.T.NoviceRestart), true
	}
	return it
}

// undulation returns the rep range and rest of the idx-th exposure of the
// week: heavy, then light, then medium (PAR-S-13).
func (g *gen) undulation(a *active, idx int) (float64, float64, float64) {
	k := g.k
	n := len(a.days)
	order := []int{0, 2, 1} // heavy, light, medium
	if n == 2 {
		order = []int{0, 1}
	}
	kind := order[idx%len(order)]
	if n <= 1 {
		kind = 0
	}
	switch kind {
	case 1:
		return k.T.MediumLo, k.T.MediumHi, k.T.RestLight
	case 2:
		return k.T.LightLo, k.T.LightHi, k.T.RestLight
	default:
		return k.T.HeavyLo, k.T.HeavyHi, k.T.RestStrengthTrained
	}
}

func (g *gen) accessory(a *active, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimAccessory, reasons)
	d := k.Dose(a.cap)
	if a.rung.Measure == MeasureHold {
		return g.conditioning(a, reasons)
	}
	it.Reps = int(clamp(math.Floor(d)-k.T.RIRStrength, 1, k.T.AccessoryReps))
	it.Sets = int(k.T.AccessorySets)
	it.Reserve = int(k.T.RIRStrength)
	it.RestS = g.rest(k.T.RestAccessory)
	if !a.hasCap {
		it.Reps, it.Calibration = int(k.T.NoviceRestart), true
	}
	it.Reasons = append(it.Reasons, k.reason(RuleDoseAccess, "reps", it.Reps))
	return it
}

func (g *gen) eccentric(a *active, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimEccentric, reasons)
	secs := g.s.Ladders[a.skill.Slug].EccS
	if secs == 0 {
		secs = k.T.EccentricStart
	}
	it.Sets, it.Reps, it.HoldS = int(k.T.EccentricSets), int(k.T.EccentricReps), int(secs)
	it.Class = classHard
	it.RestS = g.rest(k.T.RestEccentric)
	it.Calibration = false
	it.Reasons = append(it.Reasons, k.reason(RuleDoseEcc, "seconds", secs))
	return it
}

func (g *gen) balanceItem(a *active, minutes float64, reasons []Reason) Item {
	k := g.k
	it := g.baseItem(a, a.rung, StimBalance, reasons)
	it.HoldS = int(k.T.BalanceSetS)
	it.RestS = g.rest(k.T.BalanceSetS)
	it.Sets = int(math.Floor(minutes * 60 / (k.T.BalanceSetS + float64(it.RestS))))
	it.Calibration = false
	it.Reasons = append(it.Reasons, k.reason(RuleDoseBalance, "minutes", minutes))
	return it
}

// techniqueFits reports whether a technique item on a light day respects
// the 24 h spacing of straight-arm technique (LOAD-05).
func (g *gen) techniqueFits(a *active, sl *slot) bool {
	for _, st := range g.k.structuresAt(a.rung) {
		for _, other := range g.slots {
			if other == sl {
				continue
			}
			if c, ok := other.cls[st]; ok && c > classLight {
				if dayGap(other.day, sl.day) < g.spacing(c, st) || dayGap(sl.day, other.day) < g.k.T.SpacingModerate {
					return false
				}
			}
		}
	}
	return true
}

func (g *gen) techniqueItem(a *active) Item {
	k := g.k
	ex := a.rung
	if lower := g.lowerRung(ex); lower != nil {
		if ok, _, _ := g.feasible(lower, lower.OG); ok {
			ex = lower
		}
	}
	e, ok := g.capacity(ex, AssistNone)
	if !ok {
		e = a.cap
	}
	d := k.Dose(e)
	h := math.Max(k.T.MinSetHold, math.Floor(math.Min(float64(k.T.TechniqueFrac*d), k.T.TechniqueMax)))
	it := g.baseItem(a, ex, StimTechnique, nil)
	it.Class = classModerate
	it.HoldS, it.Sets = int(h), int(k.T.TechniqueTries)
	it.Reserve = int(math.Max(0, math.Floor(d-h)))
	it.RestS = g.rest(math.Max(h, k.T.BalanceSetS))
	it.Calibration = false
	it.Reasons = append(it.Reasons, k.reason(RuleDoseTech, "hold_s", it.HoldS))
	return it
}

// prehabRegions returns the regions for prehab: injured in the last 12
// months first, then the authored order (PAR-D-01); at most two.
// prehabRegions picks at most two regions for prehab: regions injured in the
// last 12 months first, then the order of the body file, which follows
// PAR-D-01 (shoulder, wrist, elbow, lower back). Regions that share a prehab
// programme (both shoulder regions, both wrist regions) count once.
func (g *gen) prehabRegions() []string {
	var ordered []string
	for _, id := range sortedKeys(g.s.Regions) {
		if g.s.Regions[id].PriorInjury {
			ordered = append(ordered, id)
		}
	}
	ordered = append(ordered, g.k.regionIDs...)
	var out []string
	seen := map[string]bool{}
	for _, id := range ordered {
		p, ok := g.k.prehab[id]
		if !ok || slices.Contains(out, id) {
			continue
		}
		key := strings.Join(p.Activation, ",")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, id)
		if len(out) == 2 {
			break
		}
	}
	return out
}

func (g *gen) prehabItems() []Item {
	k := g.k
	var out []Item
	for _, r := range g.prehabRegions() {
		p := k.prehab[r]
		for _, slug := range p.Activation {
			ex := k.exercises[slug]
			if ok, _, _ := g.feasible(ex, ex.OG); !ok {
				continue
			}
			out = append(out, Item{Exercise: slug, Stimulus: StimPrehab, Kind: KindWarmup, Sets: int(k.T.PrehabSets),
				Reps: int(k.T.PrehabReps), Reserve: int(k.T.RIRStrength), RestS: g.rest(k.T.RestPrehab),
				Class: classLight, Priority: 99, Role: RoleSupport,
				Reasons: []Reason{k.reason(RuleDosePrehab, "region", k.regions[r].Name, "label", p.Label)}})
			break
		}
	}
	return out
}

func (g *gen) loadedPrehab() []Item {
	k := g.k
	var out []Item
	for _, r := range g.prehabRegions() {
		for _, slug := range k.prehab[r].Loaded {
			ex := k.exercises[slug]
			if ok, _, _ := g.feasible(ex, ex.OG); !ok {
				continue
			}
			out = append(out, Item{Exercise: slug, Stimulus: StimPrehab, Kind: KindWorking, Sets: int(k.T.PrehabSets),
				Reps: int(k.T.PrehabReps), Reserve: int(k.T.RIRStrength), RestS: g.rest(k.T.RestPrehab),
				Class: classLight, Priority: 99, Role: RoleSupport,
				Reasons: []Reason{k.reason(RuleDosePrehab, "region", k.regions[r].Name, "label", k.prehab[r].Label)}})
			break
		}
	}
	return out
}

// rampSets adds two warm-up sets before the first held max block: two rungs
// below and one rung below at half the set hold (SESS-03).
func (g *gen) rampSets(maxB []Item) []Item {
	k := g.k
	for _, m := range maxB {
		if m.Stimulus != StimSkill || m.Offer {
			continue
		}
		ex := k.exercises[m.Exercise]
		var out []Item
		one := g.lowerRung(ex)
		var two *Exercise
		if one != nil {
			two = g.lowerRung(one)
		}
		for _, r := range []*Exercise{two, one} {
			if r == nil {
				continue
			}
			if ok, _, _ := g.feasible(r, r.OG); !ok {
				continue
			}
			out = append(out, Item{Exercise: r.Slug, Skill: m.Skill, Stimulus: StimSkill, Kind: KindWarmup, Sets: 1,
				HoldS: int(math.Max(k.T.MinSetHold, math.Floor(float64(m.HoldS)/2))), Reserve: m.Reserve, RestS: g.rest(k.T.RestAccessory),
				Class: classLight, Priority: m.Priority, Role: m.Role,
				Reasons: []Reason{k.reason(RuleWarmup, "exercise", r.Name)}})
		}
		return out
	}
	return nil
}

func (g *gen) hasWeights() bool {
	return g.equipment["weight_vest"] || g.equipment["dumbbells_or_plates"] || g.equipment["dip_belt"]
}

// plate rounds a load up to the smallest plate, at least one plate
// (PAR-B-32, PAR-S-18).
func (g *gen) plate(kg float64) float64 {
	p := g.s.Profile.SmallestPlateKg
	if p <= 0 {
		p = g.k.T.SmallestPlate
	}
	return math.Max(p, math.Ceil(kg/p)*p)
}

// applyDeload scales a deload week and regional pain deloads (WEEK-07,
// ADAPT-14).
func (g *gen) applyDeload() {
	k := g.k
	for si := range g.plan.Sessions {
		ps := &g.plan.Sessions[si]
		if g.deload != "" && ps.Kind == SessionFull {
			ps.Kind = SessionDeload
		}
		for bi := range ps.Blocks {
			for ii := range ps.Blocks[bi].Items {
				it := &ps.Blocks[bi].Items[ii]
				if it.Kind != KindWorking || it.Offer {
					continue
				}
				if g.deload != "" {
					it.Sets = max(1, int(math.Floor(float64(float64(it.Sets)*k.T.DeloadSets))))
					it.Reserve += int(k.T.DeloadReserve)
					it.Reasons = append(it.Reasons, k.reason(RuleDeload, "kind", g.deload))
				}
				ex := k.exercises[it.Exercise]
				for _, id := range sortedKeys(g.s.Regions) {
					rs := g.s.Regions[id]
					if ps.Date.Before(rs.PainDeloadTo) && float64(k.maxRating(id, ex)) >= k.T.SpacingRating {
						it.Sets = max(1, int(math.Floor(float64(float64(it.Sets)*k.T.PainDeloadVol))))
						it.Reasons = append(it.Reasons, regional(k.reason(RulePain, "region", k.regions[id].Name), id))
					}
				}
			}
		}
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
