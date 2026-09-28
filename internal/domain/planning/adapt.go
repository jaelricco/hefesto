package planning

import (
	"errors"
	"math"
	"slices"
	"time"
)

// Event kinds for Adapt (spec §6.1).
const (
	EventSession   = "session_completed"
	EventPain      = "pain_report"
	EventRedFlags  = "red_flags"
	EventClearance = "clearance"
	EventSymptoms  = "symptoms"
	EventWeek      = "week_start"
)

// Event is one trigger for adaptation.
type Event struct {
	Kind     string             `json:"kind"`
	At       time.Time          `json:"at"`
	Session  *LoggedSession     `json:"session,omitempty"`
	Pain     *PainReport        `json:"pain,omitempty"`
	Region   string             `json:"region,omitempty"`
	Answers  map[string]bool    `json:"answers,omitempty"`
	Headroom map[string]float64 `json:"headroom,omitempty"` // from the ending week's plan
}

// Change is one adaptation the user sees (spec §6.14).
type Change struct {
	Kind    string   `json:"kind"`
	Skill   string   `json:"skill,omitempty"`
	Region  string   `json:"region,omitempty"`
	From    string   `json:"from,omitempty"`
	To      string   `json:"to,omitempty"`
	Reasons []Reason `json:"reasons"`
}

// Change kinds.
const (
	ChangeCapacity    = "capacity"
	ChangeCalibration = "calibration_due"
	ChangeProbe       = "probe_offered"
	ChangeRungDown    = "rung_down"
	ChangeTarget      = "target_changed"
	ChangeDeload      = "deload_scheduled"
	ChangeRegion      = "region_state"
	ChangeReferral    = "referral_suggested"
	ChangeRamp        = "ramp_step"
	ChangeRedFlags    = "ask_red_flags"
	ChangeStopped     = "training_stopped"
	ChangeBreak       = "ramp_started"
)

// ErrUnknownEvent is returned for an event kind Adapt does not know.
var ErrUnknownEvent = errors.New("unknown event kind")

// Adapt applies one event and returns the new snapshot and the changes
// (spec §6). The input snapshot is not modified.
func Adapt(k *Knowledge, s Snapshot, ev Event) (Snapshot, []Change, error) {
	a := &adapter{k: k, s: s.clone(), at: ev.At}
	switch ev.Kind {
	case EventSession:
		if ev.Session == nil {
			return s, nil, ErrUnknownEvent
		}
		a.session(*ev.Session)
	case EventPain:
		if ev.Pain == nil {
			return s, nil, ErrUnknownEvent
		}
		a.pain(*ev.Pain)
	case EventRedFlags:
		a.redFlags(ev.Region, ev.Answers)
	case EventClearance:
		a.clearance(ev.Region)
	case EventSymptoms:
		a.s.Constraints = append(a.s.Constraints, Constraint{Kind: ConstraintStopped, Created: civil(ev.At, time.UTC)})
		a.change(Change{Kind: ChangeStopped, Reasons: []Reason{k.reason(RuleStopped)}})
	case EventWeek:
		a.week(civil(ev.At, time.UTC), ev.Headroom)
	default:
		return s, nil, ErrUnknownEvent
	}
	return a.s.canonical(), a.changes, nil
}

type adapter struct {
	k       *Knowledge
	s       Snapshot
	at      time.Time
	changes []Change
}

func (a *adapter) change(c Change) { a.changes = append(a.changes, c) }

// session applies a completed session: capacities, ladders, plateaus,
// fatigue and ramps (ADAPT-01–ADAPT-14).
func (a *adapter) session(sess LoggedSession) {
	k, s := a.k, &a.s
	if slices.ContainsFunc(s.History, func(h LoggedSession) bool { return h.ID == sess.ID }) {
		return // idempotent
	}
	s.History = append(s.History, sess)
	slices.SortStableFunc(s.History, func(x, y LoggedSession) int { return x.Date.Compare(y.Date) })

	first := map[string]bool{}
	for _, set := range sess.Sets {
		ex, ok := k.exercises[set.Exercise]
		if !ok || set.Kind == KindWarmup {
			continue
		}
		assist := set.Assist
		if assist == "" {
			assist = AssistNone
		}
		key := CapKey(ex.Slug, assist)
		isFirst := !first[key]
		first[key] = true
		est, has := s.Capacities[key]
		if !has {
			// The first observation of an exercise updates the prior the
			// plan used: derived from a harder rung or the unassisted value
			// (PAR-S-39).
			est, has = k.capacityAt(*s, ex, assist, sess.Date)
		}
		o, ok := k.classify(set, ex, isFirst, est.Mu)
		if !ok {
			continue
		}
		if !has {
			sd := math.Max(o.R, k.minSD(ex, o.X))
			if o.LowerBound {
				// A lower bound without any prior: the bound itself, as wide
				// as a derived value (PAR-S-31, PAR-F-26).
				sd = math.Max(k.minSD(ex, o.X), float64(k.T.DerivedFrac*o.X))
			}
			s.Capacities[key] = Estimate{Mu: o.X, Sigma: sd, Origin: OriginLog, At: sess.Date, N: 1}
			continue
		}
		est = k.predict(est, ex, sess.Date)
		next, contradicted := k.update(est, ex, o, sess.Date)
		s.Capacities[key] = next
		// A reserve too large to count in full hides the capacity: the
		// next exposure calibrates (reserve 2–3, spec §4.3).
		if o.Capped && !slices.Contains(s.Phase.Calibrate, key) {
			s.Phase.Calibrate = append(s.Phase.Calibrate, key)
		}
		if contradicted {
			if !slices.Contains(s.Phase.Calibrate, key) {
				s.Phase.Calibrate = append(s.Phase.Calibrate, key)
			}
			a.change(Change{Kind: ChangeCalibration, Skill: ex.Skill, To: ex.Slug, Reasons: []Reason{k.reason(RuleContradict, "exercise", ex.Name)}})
		} else if !o.LowerBound {
			s.Phase.Calibrate = slices.DeleteFunc(s.Phase.Calibrate, func(c string) bool { return c == key })
		}
	}
	a.ladders(sess)
	a.plateau()
	a.fatigue()
	a.rampSessions(sess)
}

// ladders updates ladder states from the sets of a session.
func (a *adapter) ladders(sess LoggedSession) {
	k, s := a.k, &a.s
	bySkill := map[string][]LoggedSet{}
	for _, set := range sess.Sets {
		if ex, ok := k.exercises[set.Exercise]; ok && ex.Skill != "" && set.Kind != KindWarmup {
			bySkill[ex.Skill] = append(bySkill[ex.Skill], set)
		}
	}
	for _, skill := range sortedKeys(bySkill) {
		sets := bySkill[skill]
		ls := s.Ladders[skill]
		// The working rung is the hardest non-eccentric exercise logged.
		var top *Exercise
		for _, set := range sets {
			ex := k.exercises[set.Exercise]
			if !ex.Eccentric && (top == nil || ex.Rung > top.Rung) {
				top = ex
			}
		}
		ls.Exposures++
		if top != nil {
			if cur := k.exercises[ls.Rung]; cur == nil || top.Rung > cur.Rung {
				ls.LastUp = sess.Date
			}
			if ls.Rung != top.Slug {
				ls.Since = sess.Date
				ls.RepTarget = 0
				ls.ProbeOffer = false // earned again on the new rung
			}
			ls.Rung = top.Slug
			ls.Status = StatusCalibrated
			a.progression(&ls, top, sets, sess)
		}
		a.eccentric(&ls, sets)
		s.Ladders[skill] = ls
	}
}

// progression applies the double progression, probe offers and
// regressions of one ladder (ADAPT-05, ADAPT-07, ADAPT-12).
func (a *adapter) progression(ls *LadderState, top *Exercise, sets []LoggedSet, sess LoggedSession) {
	k := a.k
	var mine []LoggedSet
	for _, set := range sets {
		if set.Exercise == top.Slug {
			mine = append(mine, set)
		}
	}
	if len(mine) == 0 {
		return
	}
	if top.Measure == MeasureReps && experience(k, a.s.Profile) == ExpNovice {
		target := ls.RepTarget
		if target == 0 {
			target = mine[0].Value
		}
		all := true
		for _, set := range mine {
			if set.Value < target || set.Failed || set.Reserve != nil && *set.Reserve < 1 {
				all = false
			}
		}
		if all {
			next := math.Min(target+1, k.T.RepToLoad)
			if target >= k.T.NoviceRepsHi {
				next = target // the next rung takes over once its dose allows (ADAPT-07)
			}
			// Announce only a new target inside the novice range.
			if next != ls.RepTarget && next <= k.T.NoviceRepsHi {
				a.change(Change{Kind: ChangeTarget, Skill: top.Skill, To: formatNum(next), Reasons: []Reason{k.reason(RuleDoubleProg, "reps", next)}})
			}
			ls.RepTarget = next
		} else if ls.RepTarget == 0 {
			ls.RepTarget = target
		}
	}
	if top.Measure == MeasureHold {
		f := mine[0]
		x := f.Value
		if f.Reserve != nil {
			x += *f.Reserve
		}
		good := f.Form == nil || *f.Form >= k.T.OfferMinForm
		prevOK := a.lastFirstSetOK(top, sess.ID)
		if x >= k.T.StageOffer && good && prevOK && !ls.ProbeOffer {
			ls.ProbeOffer = true
			// The offer is the next rung, or this rung without the band.
			probe := top
			if f.Assist != AssistBand {
				if next := k.nextRungOf(top); next != nil {
					probe = next
				}
			}
			a.offer(top, probe, RuleProbe)
		}
	}
	// ADAPT-12: form below 3 or two failures in this and the previous session.
	bad := func(ss []LoggedSet) bool {
		fails := 0
		for i, set := range ss {
			if i == 0 && set.Form != nil && *set.Form < k.T.FormMin {
				return true
			}
			if set.Failed {
				fails++
			}
		}
		return float64(fails) >= k.T.FailStop
	}
	if bad(mine) && a.previousBad(top, sess.ID, bad) {
		sk := k.skills[top.Skill]
		for r := top.Rung - 1; r >= 0; r-- {
			if lower := k.exercises[sk.Rungs[r]]; !lower.Eccentric {
				ls.CapTo, ls.CapUntil = lower.Slug, sess.Date.AddDate(0, 0, 7)
				a.change(Change{Kind: ChangeRungDown, Skill: top.Skill, From: top.Slug, To: lower.Slug, Reasons: []Reason{k.reason(RuleRungDown, "exercise", lower.Name)}})
				break
			}
		}
	}
}

// lastFirstSetOK reports whether the previous session's first set of an
// exercise reached the stage-offer threshold with form ≥ 4 (ADAPT-05).
func (a *adapter) lastFirstSetOK(ex *Exercise, skip string) bool {
	for i := len(a.s.History) - 1; i >= 0; i-- {
		h := a.s.History[i]
		if h.ID == skip {
			continue
		}
		for _, set := range h.Sets {
			if set.Exercise != ex.Slug || set.Kind == KindWarmup {
				continue
			}
			x := set.Value
			if set.Reserve != nil {
				x += *set.Reserve
			}
			return x >= a.k.T.StageOffer && (set.Form == nil || *set.Form >= 4)
		}
	}
	return false
}

func (a *adapter) previousBad(ex *Exercise, skip string, bad func([]LoggedSet) bool) bool {
	for i := len(a.s.History) - 1; i >= 0; i-- {
		h := a.s.History[i]
		if h.ID == skip {
			continue
		}
		var mine []LoggedSet
		for _, set := range h.Sets {
			if set.Exercise == ex.Slug && set.Kind != KindWarmup {
				mine = append(mine, set)
			}
		}
		if len(mine) > 0 {
			return bad(mine)
		}
	}
	return false
}

// offer announces a probe offer when next week's plan can show it
// (ADAPT-06a).
func (a *adapter) offer(from, probe *Exercise, rule string) {
	k := a.k
	week := weekStart(civil(a.at, time.UTC)).AddDate(0, 0, 7)
	if !k.probeGate(a.s, k.history(a.s, week), probe, week) {
		return
	}
	a.change(Change{Kind: ChangeProbe, Skill: from.Skill, From: from.Slug, To: probe.Slug,
		Reasons: []Reason{k.reason(rule, "exercise", probe.Name)}})
}

// eccentric lengthens eccentric reps and offers the first concentric
// attempt (ADAPT-10).
func (a *adapter) eccentric(ls *LadderState, sets []LoggedSet) {
	k := a.k
	for _, set := range sets {
		ex := k.exercises[set.Exercise]
		if !ex.Eccentric {
			continue
		}
		if ls.EccS == 0 {
			ls.EccS = k.T.EccentricStart
		}
		if ls.EccS < k.T.EccentricTarget {
			ls.EccS += k.T.EccentricStep
		} else if !ls.ProbeOffer {
			ls.ProbeOffer = true
			probe := ex
			if next := k.nextRungOf(ex); next != nil {
				probe = next
			}
			a.offer(ex, probe, RuleEccToConc)
		}
		return
	}
}

// plateau schedules a stagnation deload for the priority-1 goal ladder
// (ADAPT-13).
func (a *adapter) plateau() {
	k, s := a.k, &a.s
	if len(s.Goals) == 0 || s.Phase.DeloadNext != "" {
		return
	}
	g := s.Goals[0]
	ls, ok := s.Ladders[g.Skill]
	ex := k.exercises[ls.Rung]
	if !ok || ex == nil {
		return
	}
	last := s.History[len(s.History)-1].Date
	if daysBetween(ls.Since, last) < 7*k.T.PlateauMinWeeks {
		return
	}
	if !s.Phase.LastDeload.IsZero() && daysBetween(s.Phase.LastDeload, last) < 7*k.T.PlateauDeloadGap {
		return
	}
	// A deload session is no evidence of a plateau, and the gap of PAR-S-17
	// counts from the most recent one, also while its week is running.
	for i := len(s.History) - 1; i >= 0; i-- {
		if s.History[i].Deload {
			if daysBetween(s.History[i].Date, last) < 7*k.T.PlateauDeloadGap {
				return
			}
			break
		}
	}
	if k.skills[g.Skill].LimitingFactor == LimitBalance && float64(ls.Exposures) < k.T.BalanceReview {
		return
	}
	var xs []float64
	for i := len(s.History) - 1; i >= 0 && len(xs) < 3; i-- {
		if s.History[i].Deload {
			continue
		}
		for _, set := range s.History[i].Sets {
			if set.Exercise == ex.Slug && set.Kind != KindWarmup {
				x := set.Value
				if set.Reserve != nil {
					x += *set.Reserve
				}
				xs = append(xs, x)
				break
			}
		}
	}
	if len(xs) == 3 && xs[0] <= xs[2] && xs[1] <= xs[2] {
		s.Phase.DeloadNext = DeloadStagnation
		a.change(Change{Kind: ChangeDeload, Skill: g.Skill, To: DeloadStagnation, Reasons: []Reason{k.reason(RulePlateau, "exercise", ex.Name)}})
	}
}

// fatigue schedules a deload after high perceived fatigue (PAR-B-49 b).
func (a *adapter) fatigue() {
	k, s := a.k, &a.s
	if s.Phase.DeloadNext != "" {
		return
	}
	n, high := 0, 0
	for i := len(s.History) - 1; i >= 0 && float64(n) < k.T.FatigueWindow; i-- {
		n++
		if f := s.History[i].Fatigue; f != nil && *f >= k.T.FatigueHigh {
			high++
		}
	}
	if float64(high) >= k.T.FatigueCount {
		s.Phase.DeloadNext = DeloadFatigue
		a.change(Change{Kind: ChangeDeload, To: DeloadFatigue, Reasons: []Reason{k.reason(RuleDeload, "kind", DeloadFatigue)}})
	}
}

// rampSessions counts the sessions of a break ramp: those that loaded a
// straight-arm or wrist account (PAR-D-26 counts sessions of the step). A
// ramp after a complaint counts on the next morning's report instead (pain).
func (a *adapter) rampSessions(sess LoggedSession) {
	b := a.s.Break
	if b == nil {
		return
	}
	for _, set := range sess.Sets {
		ex, ok := a.k.exercises[set.Exercise]
		if !ok || set.Kind == KindWarmup {
			continue
		}
		for acc := range a.k.setLoad(ex, set.Kind, set.LoadKg, a.s.Profile.BodyweightKg) {
			if isStraightAccount(acc) {
				b.StepSessions++
				return
			}
		}
	}
}

// breakStep advances the break ramp at a week start: at least PAR-D-26
// sessions on the step and seven days since it began, so each step is one
// planned week.
func (a *adapter) breakStep(week time.Time) {
	k, b := a.k, a.s.Break
	if b == nil || b.Step >= 3 {
		return
	}
	if float64(b.StepSessions) >= k.T.RampSessions && daysBetween(b.StepSince, week) >= k.T.RampMinDays {
		b.Step++
		b.StepSessions, b.StepSince = 0, week
		a.change(Change{Kind: ChangeRamp, To: formatNum([]float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}[b.Step]), Reasons: []Reason{k.reason(RuleBreak)}})
	}
}

// touches reports whether a session loaded a region at 2 or more.
func (a *adapter) touches(sess LoggedSession, region string) bool {
	for _, set := range sess.Sets {
		if ex, ok := a.k.exercises[set.Exercise]; ok && a.k.maxRating(region, ex) >= 2 {
			return true
		}
	}
	return false
}

// advance moves a region one volume step or ramp stage (INJ-03).
func (a *adapter) advance(rs *RegionState, id string, day time.Time) {
	k := a.k
	if float64(rs.StepSessions) < k.T.RampSessions || daysBetween(rs.StepSince, day) < k.T.RampMinDays {
		return
	}
	st := rttStage(rs.State)
	from := rs.State
	switch {
	case st == 1:
		rs.State = StateRTT2
		rs.Step = min(rs.Step+1, 2)
	case st == 2 && rs.Step < 2:
		rs.Step++
	case st < 5:
		rs.State = rttState(st + 1)
	}
	rs.StepSessions, rs.StepSince = 0, day
	a.change(Change{Kind: ChangeRamp, Region: id, From: from, To: rs.State,
		Reasons: []Reason{regional(k.reason(RuleRamp, "region", k.regions[id].Name), id)}})
}

// pain applies a pain report (spec §8.6, INJ-07, INJ-08).
func (a *adapter) pain(r PainReport) {
	k, s := a.k, &a.s
	if !s.Profile.HealthConsent {
		return
	}
	if _, ok := k.regions[r.Region]; !ok {
		return
	}
	s.Pain = append(s.Pain, r)
	rs, ok := s.Regions[r.Region]
	if !ok {
		rs = RegionState{State: StateNormal, Since: civil(r.At, time.UTC), StepSince: civil(r.At, time.UTC)}
	}
	name := k.regions[r.Region].Name
	ask := r.SuddenSharp || r.NRS > k.T.PainAccept || (r.Timepoint == PainDaily && r.NRS > k.T.PainGreen) || !rs.Complaint
	// Stage 0 without a referral ends with daily pain within the green limit
	// and negative red-flag answers (spec §8.3), so a green daily report asks
	// the questions again.
	exit0 := rs.State == StateRTT0 && rs.Referral != "advise" && r.Timepoint == PainDaily && r.NRS <= k.T.PainGreen
	if ask && (r.NRS > 0 || r.SuddenSharp) || exit0 {
		a.change(Change{Kind: ChangeRedFlags, Region: r.Region, Reasons: []Reason{regional(k.reason(RuleRedFlagAsk, "region", name), r.Region)}})
	}
	day := civil(r.At, time.UTC)
	if !rs.Complaint {
		if r.NRS > k.T.PainGreen || float64(a.painDays(r.Region, day)) >= k.T.PainEntryCount {
			rs.Complaint, rs.ComplaintAt, rs.EnteredVia = true, day, "pain_report"
			rs.StartFraction = k.T.RTTStart
			rs.Step = k.startStep(rs.StartFraction)
			rs.State, rs.Since, rs.StepSince = StateRTT1, day, day
			if r.Timepoint == PainDaily && r.NRS > k.T.PainGreen {
				rs.State = StateRTT0
			}
			rs.Reference = a.references(r.Region)
			a.change(Change{Kind: ChangeRegion, Region: r.Region, From: StateNormal, To: rs.State, Reasons: []Reason{regional(k.reason(RuleRegionState, "region", name, "state", rs.State), r.Region)}})
		}
		s.Regions[r.Region] = rs
		return
	}
	// A session-bound report of a region with a complaint.
	if r.SessionID != "" && (r.Timepoint == PainMorning || r.Timepoint == PainAfter || r.Timepoint == PainDuring || r.Timepoint == PainWarmup) {
		base := baselineBefore(s.Pain, r.Region, r.SessionID, r.At)
		v := k.sessionPain(s.Pain, r.Region, r.SessionID, base)
		ruleBroken := v.status == painBreach || k.weeklyTrendRising(s.Pain, r.Region, r.At)
		st := rttStage(rs.State)
		ramp := st >= 1 && st <= 5
		switch {
		case ruleBroken || ramp && (v.lasted || v.persisted || v.nextDay):
			restFrom := day
			if r.Timepoint != PainMorning {
				restFrom = day.AddDate(0, 0, 1)
			}
			a.breach(&rs, r.Region, day, restFrom, ruleBroken, v)
		case r.Timepoint == PainMorning && v.status == painGreen && !v.sore && st >= 1 && st <= 4:
			// The morning report completes a session without soreness
			// (PAR-D-26, PAR-S-47).
			for _, h := range s.History {
				if h.ID == r.SessionID && a.touches(h, r.Region) {
					rs.StepSessions++
					a.advance(&rs, r.Region, day)
					break
				}
			}
		}
	}
	if rs.HoldAtRef && a.greenWeek(r.Region, day) {
		rs.HoldAtRef = false
	}
	a.referrals(&rs, r.Region, day)
	s.Regions[r.Region] = rs
}

// breach handles a violated pain rule: the pain deload when a threshold,
// the morning rule or the weekly trend is broken (PAR-D-18), and in the ramp
// the soreness rules (PAR-D-28): repeat the step and rest the region a day,
// or after persisting warm-up pain go one step back and rest two days. Every
// breach restarts the step (PAR-D-26).
func (a *adapter) breach(rs *RegionState, id string, day, restFrom time.Time, deload bool, v painVerdict) {
	k, s := a.k, &a.s
	name := k.regions[id].Name
	if st := rttStage(rs.State); st >= 1 && st <= 5 {
		from := rs.State
		rest := 0.0
		switch {
		case v.persisted:
			a.retreat(rs)
			rest = k.T.RestWarmup
		case v.lasted || v.nextDay:
			rest = k.T.RestNextDay
		}
		if until := restFrom.AddDate(0, 0, int(rest)); rest > 0 && until.After(rs.RestUntil) {
			rs.RestUntil = until
		}
		rs.StepSessions, rs.StepSince = 0, day
		if rest > 0 {
			a.change(Change{Kind: ChangeRamp, Region: id, From: from, To: rs.State,
				Reasons: []Reason{regional(k.reason(RulePain, "region", name), id)}})
		}
	}
	if n := len(rs.Breaches); n > 0 && rs.Breaches[n-1].Equal(day) {
		return
	}
	rs.Breaches = append(rs.Breaches, day)
	if deload {
		rs.PainDeloadTo = day.AddDate(0, 0, int(k.T.PainDeloadDays))
		rs.HoldAtRef = true
		if rs.Reference == nil {
			rs.Reference = a.references(id)
		}
		for skill, ls := range s.Ladders {
			cur := k.exercises[ls.Rung]
			if cur == nil || k.maxRating(id, cur) < 2 {
				continue
			}
			sk := k.skills[skill]
			for r := cur.Rung - 1; r >= 0; r-- {
				if lower := k.exercises[sk.Rungs[r]]; !lower.Eccentric {
					ls.CapTo, ls.CapUntil = lower.Slug, rs.PainDeloadTo
					s.Ladders[skill] = ls
					break
				}
			}
		}
		a.change(Change{Kind: ChangeDeload, Region: id, To: DeloadPain, Reasons: []Reason{regional(k.reason(RulePain, "region", name), id)}})
	}
	recent := 0
	for _, t := range rs.Breaches {
		if daysBetween(t, day) <= k.T.BreachWindow {
			recent++
		}
	}
	// INJ-08: reaching the breach count is its own referral message, also
	// when an earlier timer already advised one.
	if float64(recent) == k.T.BreachCount {
		rs.Referral = "advise"
		a.change(Change{Kind: ChangeReferral, Region: id, Reasons: []Reason{regional(k.reason(RuleReferral, "region", name), id)}})
	}
}

// retreat moves a region one volume step or stage back (PAR-D-28), the
// inverse of advance; in stage 1 the start share drops one step.
func (a *adapter) retreat(rs *RegionState) {
	start := a.k.startStep(rs.StartFraction)
	switch st := rttStage(rs.State); {
	case st >= 4:
		rs.State = rttState(st - 1)
	case st == 3:
		rs.State, rs.Step = StateRTT2, 2
	case st == 2 && rs.Step > start+1:
		rs.Step--
	case st == 2:
		rs.State, rs.Step = StateRTT1, start
	case st == 1:
		rs.Step = max(rs.Step-1, 0)
	}
}

// referrals applies the complaint timers (PAR-D-32, PAR-D-19, RF-11).
func (a *adapter) referrals(rs *RegionState, id string, day time.Time) {
	k := a.k
	if !rs.Complaint || rs.ComplaintAt.IsZero() {
		return
	}
	improving := a.greenWeek(id, day)
	age := daysBetween(rs.ComplaintAt, day)
	name := k.regions[id].Name
	switch {
	case age >= k.T.ReferralDays && !improving && rs.State != StateRTT0:
		rs.State, rs.Referral = StateRTT0, "advise"
		a.change(Change{Kind: ChangeReferral, Region: id, To: StateRTT0, Reasons: []Reason{regional(k.reason(RuleReferral, "region", name), id)}})
	case age >= k.T.ReferralSoft && !improving && rs.Referral == "":
		rs.Referral = "soft"
		a.change(Change{Kind: ChangeReferral, Region: id, Reasons: []Reason{regional(k.reason(RuleReferral, "region", name), id)}})
	}
}

// greenWeek reports whether every report of a region in the last 7 days is
// within the green limit (PAR-D-14).
func (a *adapter) greenWeek(id string, day time.Time) bool {
	n := 0
	for _, r := range a.s.Pain {
		if r.Region != id || daysBetween(civil(r.At, time.UTC), day) > 7 {
			continue
		}
		n++
		if r.NRS > a.k.T.PainGreen || r.LastedOver || r.Persisted {
			return false
		}
	}
	return n > 0
}

// painDays counts days with a report above 0 in the entry window (PAR-S-42).
func (a *adapter) painDays(id string, day time.Time) int {
	days := map[time.Time]bool{}
	for _, r := range a.s.Pain {
		d := civil(r.At, time.UTC)
		if r.Region == id && r.NRS > 0 && daysBetween(d, day) < a.k.T.PainEntryDays {
			days[d] = true
		}
	}
	return len(days)
}

// references returns the logged pre-complaint R of a region's ramp accounts.
func (a *adapter) references(id string) map[string]float64 {
	k := a.k
	week := weekStart(civil(a.at, time.UTC))
	h := k.history(a.s, week)
	out := map[string]float64{}
	for _, acc := range k.rampAccounts(id) {
		if r, _ := k.reference(h, acc, week); r > 0 {
			out[acc] = r
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// redFlags applies red-flag answers for a region (INJ-02).
func (a *adapter) redFlags(id string, answers map[string]bool) {
	k, s := a.k, &a.s
	if _, ok := k.regions[id]; !ok {
		return
	}
	day := civil(a.at, time.UTC)
	out := k.EvaluateRedFlags(id, answers, isMinor(k, s.Profile.BirthYear, a.at))
	rs := s.Regions[id]
	if rs.State == "" {
		rs = RegionState{State: StateNormal, Since: day, StepSince: day}
	}
	from := rs.State
	noStructures := len(k.regions[id].Structures) == 0
	if len(out.Flags) == 0 {
		// All negative: stage 0 without a referral ends once daily pain is
		// within the green limit (spec §8.3).
		if rs.State == StateRTT0 && rs.Referral != "advise" && a.dailyGreen(id) {
			a.enterRamp(&rs, id, k.T.RTTStart, day, "red_flags_negative")
			if s.Profile.HealthConsent {
				s.Regions[id] = rs
			}
		}
		return
	}
	switch {
	case out.Stop || noStructures && out.Urgency != "" && out.Urgency != UrgencyAdvise:
		rs.State = StateLocked
		s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintStopped, Region: id, Created: day},
			Constraint{Kind: ConstraintLocked, Region: id, Created: day})
		a.change(Change{Kind: ChangeStopped, Region: id, Reasons: out.Reasons})
	case out.Lock:
		rs.State = StateLocked
		s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintLocked, Region: id, Created: day})
	case out.RTT0:
		rs.State, rs.Referral = StateRTT0, "advise"
	default:
		return
	}
	rs.Complaint, rs.Since = true, day
	if rs.ComplaintAt.IsZero() {
		rs.ComplaintAt = day
	}
	if s.Profile.HealthConsent {
		s.Regions[id] = rs
	}
	a.change(Change{Kind: ChangeRegion, Region: id, From: from, To: rs.State, Reasons: out.Reasons})
}

// clearance lifts a stop or a lock after the user confirms a professional
// clearance (spec §8.3). An empty region clears the global stop and every
// locked region.
func (a *adapter) clearance(id string) {
	k, s := a.k, &a.s
	day := civil(a.at, time.UTC)
	// A clearance moves a locked region, or one in stage 0, to stage 1 at
	// the start share after a referral (PAR-D-33).
	unlock := func(r string) {
		rs, ok := s.Regions[r]
		if !ok || rs.State != StateLocked && rs.State != StateRTT0 {
			return
		}
		a.enterRamp(&rs, r, k.T.RTTStartReferral, day, "clearance")
		s.Regions[r] = rs
	}
	if id == "" {
		s.Constraints = slices.DeleteFunc(s.Constraints, func(c Constraint) bool { return c.Kind == ConstraintStopped || c.Kind == ConstraintLocked })
		s.Screening.Cleared = true
		for _, r := range sortedKeys(s.Regions) {
			unlock(r)
		}
		return
	}
	// A stop raised by this region's red flags goes with its clearance; a
	// stop without a region (exertion symptoms) needs the global one.
	s.Constraints = slices.DeleteFunc(s.Constraints, func(c Constraint) bool {
		return (c.Kind == ConstraintLocked || c.Kind == ConstraintStopped) && c.Region == id
	})
	unlock(id)
}

// enterRamp moves a region to stage 1 at a start share.
func (a *adapter) enterRamp(rs *RegionState, id string, start float64, day time.Time, via string) {
	k := a.k
	from := rs.State
	rs.State, rs.StartFraction, rs.EnteredVia = StateRTT1, start, via
	rs.Step, rs.StepSessions, rs.StepSince, rs.Since = k.startStep(start), 0, day, day
	a.change(Change{Kind: ChangeRegion, Region: id, From: from, To: StateRTT1,
		Reasons: []Reason{regional(k.reason(RuleRegionState, "region", k.regions[id].Name, "state", StateRTT1), id)}})
}

// dailyGreen reports whether the latest daily report of a region is within
// the green limit (PAR-D-14).
func (a *adapter) dailyGreen(id string) bool {
	var last *PainReport
	for i := range a.s.Pain {
		r := &a.s.Pain[i]
		if r.Region == id && r.Timepoint == PainDaily && (last == nil || !r.At.Before(last.At)) {
			last = r
		}
	}
	return last != nil && last.NRS <= a.k.T.PainGreen
}

// week runs the week-start bookkeeping: mesocycle, deload weeks, pauses,
// stage-5 exits, calibration intervals and headroom (spec §6).
func (a *adapter) week(week time.Time, headroom map[string]float64) {
	k, s := a.k, &a.s
	week = weekStart(week)
	prev := week.AddDate(0, 0, -7)
	ph := &s.Phase
	plannedPrev := !ph.MesoStart.IsZero() && math.Floor(daysBetween(ph.MesoStart, prev)/7)+1 == k.T.MesoWeeks
	if ph.DeloadWeek.Equal(prev) || plannedPrev {
		ph.MesoStart, ph.LastDeload = week, prev
	}
	if ph.DeloadNext != "" {
		ph.DeloadWeek, ph.DeloadKind, ph.DeloadNext = week, ph.DeloadNext, ""
		a.change(Change{Kind: ChangeDeload, To: ph.DeloadKind, Reasons: []Reason{k.reason(RuleDeloadWeek, "kind", ph.DeloadKind)}})
	}
	if headroom != nil {
		s.Headroom = cloneMap(headroom)
	}
	// Training age grows with regular training (PAR-S-20).
	n := 0
	for _, h := range s.History {
		if daysBetween(h.Date, week) <= 28 && h.Date.Before(week) {
			n++
		}
	}
	if n >= 4 {
		s.Profile.TrainingMonths += 7.0 / 30.0
	}
	a.breaks(week)
	// Stage 5 back to normal after two weeks without a breach (PAR-S-32).
	for _, id := range sortedKeys(s.Regions) {
		rs := s.Regions[id]
		if rs.State != StateRTT5 {
			continue
		}
		clean := daysBetween(rs.StepSince, week) >= 7*k.T.RTT5Weeks
		for _, t := range rs.Breaches {
			if daysBetween(t, week) < 7*k.T.RTT5Weeks {
				clean = false
			}
		}
		if clean {
			rs.State, rs.Complaint = StateNormal, false
			s.Regions[id] = rs
			a.change(Change{Kind: ChangeRegion, Region: id, From: StateRTT5, To: StateNormal,
				Reasons: []Reason{regional(k.reason(RuleRegionState, "region", k.regions[id].Name, "state", StateNormal), id)}})
		}
	}
	// A calibration set every 4–6 weeks (PAR-B-29).
	for _, key := range sortedKeys(s.Capacities) {
		e := s.Capacities[key]
		if !e.At.IsZero() && daysBetween(e.At, week) >= 7*k.T.CalibrationWeeks && !slices.Contains(ph.Calibrate, key) {
			ph.Calibrate = append(ph.Calibrate, key)
		}
	}
}

// breaks detects a pause from the log at a week start, keeps it current
// while the user stays away, advances its ramp and ends it (ADAPT-16,
// spec §6.11). The pause of the straight-arm and wrist accounts counts from
// their own last load, so a pause of only those accounts ramps them too.
func (a *adapter) breaks(week time.Time) {
	k, s := a.k, &a.s
	if len(s.History) == 0 {
		return
	}
	h := k.history(*s, week)
	lastAny := s.History[len(s.History)-1].Date
	var lastStraight time.Time
	for acc, t := range h.lastLoad {
		if isStraightAccount(acc) && t.After(lastStraight) {
			lastStraight = t
		}
	}
	days := daysBetween(lastAny, week)
	straight := 0.0
	if !lastStraight.IsZero() && daysBetween(lastStraight, week) >= k.T.LayoffDays {
		straight = daysBetween(lastStraight, week)
	}
	b := s.Break
	switch {
	case b == nil && (days >= k.T.BreakMid || straight > 0):
		ref := map[string]float64{}
		for acc, t := range h.lastLoad {
			if r, _ := k.reference(h, acc, weekStart(t).AddDate(0, 0, 7)); r > 0 {
				ref[acc] = r
			}
		}
		s.Break = &BreakState{Days: days, StraightDays: straight, Since: week, StepSince: week, Logged: true, Reference: ref}
		// The rung before the pause caps the ladder during the ramp.
		for _, skill := range sortedKeys(s.Ladders) {
			ls := s.Ladders[skill]
			if ex := k.exercises[ls.Rung]; ex != nil && ex.StraightArm == ArmStraight {
				ls.CapRung = ls.Rung
				s.Ladders[skill] = ls
			}
		}
		a.change(Change{Kind: ChangeBreak, To: formatNum(math.Max(days, straight)), Reasons: []Reason{k.reason(RuleBreak, "days", math.Max(days, straight))}})
	case b != nil && b.Logged && lastAny.Before(b.Since):
		// Still away: the pause grows and the ramp starts in the week the
		// user comes back.
		b.Days = days
		if straight > 0 {
			b.StraightDays = straight
		}
		b.Since, b.Step, b.StepSince, b.StepSessions = week, 0, week, 0
	case b != nil:
		a.breakStep(week)
		weeks := math.Floor(daysBetween(b.Since, week) / 7)
		grown := k.breakFactorFor(b.Days) * math.Pow(k.T.BreakGrowth, weeks)
		// The ramp ends when its last step (1.0) has held for the weeks of
		// the reference mean, so LOAD-02 takes over from a mean of full
		// weeks rather than of ramp weeks (PAR-S-14).
		rampDone := b.StraightDays < k.T.LayoffDays ||
			b.Step >= 3 && daysBetween(b.StepSince, week) >= 7*k.T.RefWeeks
		if rampDone && grown >= 1 {
			s.Break = nil
		}
	}
}

// breakFactorFor mirrors gen.breakFactor for the adapter.
func (k *Knowledge) breakFactorFor(days float64) float64 {
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
