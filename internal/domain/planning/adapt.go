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
	return a.s, a.changes, nil
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
		o, ok := k.classify(set, ex, isFirst, est.Mu)
		if !ok {
			continue
		}
		if !has {
			if o.LowerBound {
				continue
			}
			s.Capacities[key] = Estimate{Mu: o.X, Sigma: math.Max(o.R, k.minSD(ex)), Origin: OriginLog, At: sess.Date, N: 1}
			continue
		}
		est = k.predict(est, ex, sess.Date)
		next, contradicted := k.update(est, ex, o, sess.Date)
		s.Capacities[key] = next
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
	if top.Measure == MeasureReps && experience(a.s.Profile) == ExpNovice {
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
			ls.RepTarget = math.Min(target+1, k.T.RepToLoad)
			if target >= k.T.NoviceRepsHi {
				ls.RepTarget = target // the next rung takes over once its dose allows (ADAPT-07)
			}
			a.change(Change{Kind: ChangeTarget, Skill: top.Skill, To: formatNum(ls.RepTarget), Reasons: []Reason{k.reason(RuleDoubleProg, "reps", ls.RepTarget)}})
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
		good := f.Form == nil || *f.Form >= 4
		prevOK := a.lastFirstSetOK(top, sess.ID)
		if x >= k.T.StageOffer && good && prevOK && !ls.ProbeOffer {
			ls.ProbeOffer = true
			a.change(Change{Kind: ChangeProbe, Skill: top.Skill, From: top.Slug, Reasons: []Reason{k.reason(RuleProbe, "exercise", top.Name)}})
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
			a.change(Change{Kind: ChangeProbe, Skill: ex.Skill, From: ex.Slug, Reasons: []Reason{k.reason(RuleEccToConc, "exercise", ex.Name)}})
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
	if k.skills[g.Skill].LimitingFactor == LimitBalance && float64(ls.Exposures) < k.T.BalanceReview {
		return
	}
	var xs []float64
	for i := len(s.History) - 1; i >= 0 && len(xs) < 3; i-- {
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

// rampSessions counts pain-free sessions at a ramp step and advances it
// (PAR-D-25, PAR-D-26).
func (a *adapter) rampSessions(sess LoggedSession) {
	k, s := a.k, &a.s
	for _, id := range sortedKeys(s.Regions) {
		rs := s.Regions[id]
		st := rttStage(rs.State)
		if st < 1 || st > 4 || !a.touches(sess, id) {
			continue
		}
		base := baselineBefore(s.Pain, id, sess.ID, sess.Date)
		status, _, _ := k.sessionPain(s.Pain, id, sess.ID, base)
		if status == painBreach || status == painNone && !rs.BreakOnly {
			continue
		}
		rs.StepSessions++
		a.advance(&rs, id, sess.Date)
		s.Regions[id] = rs
	}
	if b := s.Break; b != nil {
		b.StepSessions++
		steps := 3
		if float64(b.StepSessions) >= k.T.RampSessions && daysBetween(b.StepSince, sess.Date) >= k.T.RampMinDays && b.Step < steps {
			b.Step++
			b.StepSessions, b.StepSince = 0, sess.Date
			a.change(Change{Kind: ChangeRamp, To: formatNum([]float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}[b.Step]), Reasons: []Reason{k.reason(RuleBreak)}})
		}
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
	if ask && (r.NRS > 0 || r.SuddenSharp) {
		a.change(Change{Kind: ChangeRedFlags, Region: r.Region, Reasons: []Reason{regional(k.reason(RuleRedFlagAsk, "region", name), r.Region)}})
	}
	day := civil(r.At, time.UTC)
	if !rs.Complaint {
		if r.NRS > k.T.PainGreen || a.painDays(r.Region, day) >= 2 {
			rs.Complaint, rs.ComplaintAt, rs.EnteredVia = true, day, "pain_report"
			rs.StartFraction = k.T.RTTStart
			rs.Step = k.startStep(rs.StartFraction)
			rs.State, rs.Since, rs.StepSince = StateRTT1, day, day
			if r.Timepoint == PainDaily && r.NRS > k.T.PainGreen {
				rs.State = StateRTT0
			}
			rs.Reference = a.references(r.Region, false)
			a.change(Change{Kind: ChangeRegion, Region: r.Region, From: StateNormal, To: rs.State, Reasons: []Reason{regional(k.reason(RuleRegionState, "region", name, "state", rs.State), r.Region)}})
		}
		s.Regions[r.Region] = rs
		return
	}
	// A session-bound report of a region with a complaint.
	if r.SessionID != "" && (r.Timepoint == PainMorning || r.Timepoint == PainAfter || r.Timepoint == PainDuring || r.Timepoint == PainWarmup) {
		base := baselineBefore(s.Pain, r.Region, r.SessionID, r.At)
		status, lasted, persisted := k.sessionPain(s.Pain, r.Region, r.SessionID, base)
		trend := k.weeklyTrendRising(s.Pain, r.Region, r.At)
		if status == painBreach || trend {
			a.breach(&rs, r.Region, day, lasted, persisted)
		}
	}
	if rs.HoldAtRef && a.greenWeek(r.Region, day) {
		rs.HoldAtRef = false
	}
	a.referrals(&rs, r.Region, day)
	s.Regions[r.Region] = rs
}

// breach handles a violated pain rule (PAR-D-18, PAR-D-20, PAR-D-28).
func (a *adapter) breach(rs *RegionState, id string, day time.Time, lasted, persisted bool) {
	k, s := a.k, &a.s
	if n := len(rs.Breaches); n > 0 && rs.Breaches[n-1].Equal(day) {
		return
	}
	rs.Breaches = append(rs.Breaches, day)
	name := k.regions[id].Name
	st := rttStage(rs.State)
	switch {
	case st >= 1 && persisted:
		rs.State = rttState(max(st-1, 1))
		rs.StepSessions, rs.StepSince = 0, day
	case st >= 1 && lasted:
		rs.StepSessions = 0
	default:
		rs.PainDeloadTo = day.AddDate(0, 0, int(k.T.PainDeloadDays))
		rs.HoldAtRef = true
		if rs.Reference == nil {
			rs.Reference = a.references(id, false)
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
	}
	a.change(Change{Kind: ChangeDeload, Region: id, To: DeloadPain, Reasons: []Reason{regional(k.reason(RulePain, "region", name), id)}})
	recent := 0
	for _, t := range rs.Breaches {
		if daysBetween(t, day) <= k.T.BreachWindow {
			recent++
		}
	}
	if float64(recent) >= k.T.BreachCount && rs.Referral != "advise" {
		rs.Referral = "advise"
		a.change(Change{Kind: ChangeReferral, Region: id, Reasons: []Reason{regional(k.reason(RuleReferral, "region", name), id)}})
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
func (a *adapter) references(id string, breakOnly bool) map[string]float64 {
	k := a.k
	week := weekStart(civil(a.at, time.UTC))
	h := k.history(a.s, week)
	out := map[string]float64{}
	for _, acc := range k.rampAccounts(id, breakOnly) {
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
	switch {
	case out.Stop || noStructures && out.Urgency != "" && out.Urgency != UrgencyAdvise:
		rs.State = StateLocked
		s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintStopped, Created: day},
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
	unlock := func(r string) {
		rs, ok := s.Regions[r]
		if !ok || rs.State != StateLocked {
			return
		}
		rs.State, rs.StartFraction = StateRTT1, k.T.RTTStartReferral
		rs.Step, rs.StepSessions, rs.StepSince, rs.Since = k.startStep(rs.StartFraction), 0, day, day
		rs.EnteredVia = "clearance"
		s.Regions[r] = rs
		a.change(Change{Kind: ChangeRegion, Region: r, From: StateLocked, To: StateRTT1,
			Reasons: []Reason{regional(k.reason(RuleRegionState, "region", k.regions[r].Name, "state", StateRTT1), r)}})
	}
	if id == "" {
		s.Constraints = slices.DeleteFunc(s.Constraints, func(c Constraint) bool { return c.Kind == ConstraintStopped || c.Kind == ConstraintLocked })
		s.Screening.Cleared = true
		for _, r := range sortedKeys(s.Regions) {
			unlock(r)
		}
		return
	}
	s.Constraints = slices.DeleteFunc(s.Constraints, func(c Constraint) bool { return c.Kind == ConstraintLocked && c.Region == id })
	unlock(id)
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
	// Pauses from the log (ADAPT-16).
	if s.Break == nil && len(s.History) > 0 {
		last := s.History[len(s.History)-1].Date
		days := daysBetween(last, week)
		if days >= k.T.BreakMid {
			s.Break = &BreakState{Days: days, Since: week, StepSince: week, Logged: true}
			a.change(Change{Kind: ChangeBreak, To: formatNum(days), Reasons: []Reason{k.reason(RuleBreak, "days", days)}})
		}
	} else if b := s.Break; b != nil {
		weeks := math.Floor(daysBetween(b.Since, week) / 7)
		grown := k.breakFactorFor(b.Days) * math.Pow(k.T.BreakGrowth, weeks)
		if (b.Step >= 3 || b.Days < k.T.LayoffDays) && grown >= 1 {
			s.Break = nil
		}
	}
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
