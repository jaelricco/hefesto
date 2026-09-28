package planning

import (
	"math"
	"slices"
	"time"
)

// cell returns the matrix action of an exercise for a region with a
// complaint (INJ-04). Exercises without a matrix family that still load the
// region at 2 or more count as M.
func (k *Knowledge) cell(region string, ex *Exercise) string {
	if ex.ComplaintFamily != "none" {
		if a, ok := k.matrix[region][ex.ComplaintFamily]; ok {
			return a
		}
	}
	for _, s := range k.regions[region].Structures {
		if float64(k.rEff(ex, s)) >= k.T.SpacingRating {
			return ActionModify
		}
	}
	return ActionNone
}

// maxRating returns the highest rEff of an exercise over a region's
// structures.
func (k *Knowledge) maxRating(region string, ex *Exercise) int {
	best := 0
	for _, s := range k.regions[region].Structures {
		best = max(best, k.rEff(ex, s))
	}
	return best
}

// regionVerdict is what a region says about one exercise.
type regionVerdict struct {
	exclude bool
	modify  bool
	monitor bool
	reason  Reason
}

// judge applies SAFE-05/06, the matrix, supinated variants and professional
// restrictions of every region to one exercise (SEL-03, SEL-04, INJ-04,
// INJ-10). Regions are visited in sorted order so the first reason is
// deterministic.
func (k *Knowledge) judge(s Snapshot, ex *Exercise) regionVerdict {
	var v regionVerdict
	for _, id := range sortedKeys(s.Regions) {
		rs := s.Regions[id]
		name := k.regions[id].Name
		for _, tag := range rs.Restrictions {
			if slices.Contains(ex.Restrictions, tag) {
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleRestriction, "region", name), id)}
			}
		}
		stage := rttStage(rs.State)
		switch {
		case rs.State == StateLocked:
			if k.maxRating(id, ex) >= 1 {
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleRegionLocked, "region", name), id)}
			}
		case stage == 0:
			if k.maxRating(id, ex) >= 2 {
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleRegionRTT0, "region", name), id)}
			}
		case stage >= 1 && rs.Complaint:
			if ex.Supinated && k.regionHas(id, "biceps_distal") {
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleSupinated, "region", name), id)}
			}
			switch k.cell(id, ex) {
			case ActionExclude:
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleMatrix, "region", name, "action", "X"), id)}
			case ActionModify:
				if stage < 5 {
					v.modify = true
					v.monitor = true
					v.reason = regional(k.reason(RuleModify, "region", name), id)
				} else {
					v.monitor = true
				}
			case ActionPain:
				v.monitor = true
			}
		}
	}
	for _, c := range s.Constraints {
		if c.Kind == ConstraintExcluded || c.Kind == ConstraintLocked {
			if k.maxRating(c.Region, ex) >= 2 || (c.Kind == ConstraintLocked && k.maxRating(c.Region, ex) >= 1) {
				return regionVerdict{exclude: true, reason: regional(k.reason(RuleNoConsent, "region", k.regions[c.Region].Name), c.Region)}
			}
		}
	}
	return v
}

func (k *Knowledge) regionHas(region, structure string) bool {
	return slices.Contains(k.regions[region].Structures, structure)
}

// regionsOf returns the regions whose structures an account belongs to.
func (k *Knowledge) regionsOf(a string) []string {
	st := accountStructure(a)
	var out []string
	for _, id := range k.regionIDs {
		if slices.Contains(k.regions[id].Structures, st) {
			out = append(out, id)
		}
	}
	return out
}

// rampAccounts returns the accounts a region's ramp governs (spec §4.6).
func (k *Knowledge) rampAccounts(region string) []string {
	var out []string
	for _, st := range k.regions[region].Structures {
		if st == wrist {
			out = append(out, wrist)
			continue
		}
		out = append(out, account(st, true), account(st, false))
	}
	slices.Sort(out)
	return out
}

// rampFraction is the volume share of a region in the ramp (spec §8.6).
func (k *Knowledge) rampFraction(rs RegionState) float64 {
	steps := []float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}
	switch stage := rttStage(rs.State); {
	case stage <= 0:
		return 0
	case stage >= 3:
		return 1
	default:
		return steps[min(max(rs.Step, 0), len(steps)-1)]
	}
}

// startStep returns the index of a start fraction in the step series.
func (k *Knowledge) startStep(frac float64) int {
	steps := []float64{k.T.RampStep1, k.T.RampStep2, k.T.RampStep3, k.T.RampStep4}
	for i, s := range steps {
		if frac <= s {
			return i
		}
	}
	return len(steps) - 1
}

// FlagOutcome is the result of red-flag answers for one region (INJ-02).
type FlagOutcome struct {
	Stop    bool     `json:"stop"`
	Lock    bool     `json:"lock"`
	RTT0    bool     `json:"rtt0"`
	Urgency string   `json:"urgency,omitempty"`
	Flags   []string `json:"flags,omitempty"`
	Reasons []Reason `json:"reasons,omitempty"`
}

// EvaluateRedFlags turns the answers for one region into actions. RF-05 is
// N only when the joint looks displaced (answer key "RF-05-displaced").
func (k *Knowledge) EvaluateRedFlags(region string, answers map[string]bool, minor bool) FlagOutcome {
	var out FlagOutcome
	rank := map[string]int{UrgencyAdvise: 1, UrgencySoon: 2, UrgencyNow: 3}
	for _, rf := range k.RedFlags(region, minor) {
		if !answers[rf.ID] {
			continue
		}
		urgency, action := rf.Urgency, rf.Action
		if rf.ID == "RF-05" && answers["RF-05-displaced"] {
			urgency, action = UrgencyNow, FlagStop
		}
		out.Flags = append(out.Flags, rf.ID)
		if rank[urgency] > rank[out.Urgency] {
			out.Urgency = urgency
		}
		switch action {
		case FlagStop:
			out.Stop, out.Lock = true, true
		case FlagLock:
			out.Lock = true
		case FlagRTT0:
			out.RTT0 = true
		}
		out.Reasons = append(out.Reasons, regional(k.reason(RuleRedFlagAct, "flag", rf.ID, "urgency", urgency, "advice", rf.Advice), region))
	}
	return out
}

// painStatus classifies the reports around one session for one region
// (spec §8.6).
type painStatus int

const (
	painNone painStatus = iota
	painGreen
	painAcceptable
	painBreach
)

// painVerdict is what the reports around one session say about a region.
type painVerdict struct {
	status    painStatus // thresholds and morning rule (PAR-D-14–PAR-D-16)
	sore      bool       // a value above the pre-session baseline (PAR-S-47)
	nextDay   bool       // the next morning above the baseline (PAR-D-28)
	lasted    bool       // pain lasted over 1 h after the session (PAR-D-28)
	persisted bool       // warm-up pain persisted over 15 min (PAR-D-28)
}

// sessionPain evaluates the reports of a region tied to one session plus
// the following morning. A baseline below 0 means none is known: the
// morning rule (PAR-D-16) is not judged then, and soreness counts from 0
// (PAR-S-47).
func (k *Knowledge) sessionPain(reports []PainReport, region, sessionID string, baseline float64) painVerdict {
	var v painVerdict
	worst := func(st painStatus) {
		if st > v.status {
			v.status = st
		}
	}
	sore := math.Max(baseline, 0)
	for _, r := range reports {
		if r.Region != region || r.SessionID != sessionID {
			continue
		}
		v.lasted = v.lasted || r.LastedOver
		v.persisted = v.persisted || r.Persisted
		switch r.Timepoint {
		case PainDuring, PainAfter, PainWarmup:
			switch {
			case r.NRS > k.T.PainAccept:
				worst(painBreach)
			case r.NRS > k.T.PainGreen:
				worst(painAcceptable)
			default:
				worst(painGreen)
			}
			v.sore = v.sore || r.NRS > sore
		case PainMorning:
			switch {
			case baseline >= 0 && r.NRS > baseline:
				worst(painBreach)
			case r.NRS > k.T.PainGreen:
				worst(painAcceptable)
			default:
				worst(painGreen)
			}
			if r.NRS > sore {
				v.sore, v.nextDay = true, true
			}
		}
	}
	v.sore = v.sore || v.lasted || v.persisted
	return v
}

// weeklyTrendRising reports whether the mean "after" value of the week
// before `now` exceeds the week before it by the trend threshold (PAR-S-42).
func (k *Knowledge) weeklyTrendRising(reports []PainReport, region string, now time.Time) bool {
	mean := func(from, to time.Time) (float64, bool) {
		var sum float64
		var n int
		for _, r := range reports {
			if r.Region == region && r.Timepoint == PainAfter && !r.At.Before(from) && r.At.Before(to) {
				sum += r.NRS
				n++
			}
		}
		if n == 0 {
			return 0, false
		}
		return sum / float64(n), true
	}
	cur, ok1 := mean(now.AddDate(0, 0, -7), now)
	prev, ok2 := mean(now.AddDate(0, 0, -14), now.AddDate(0, 0, -7))
	return ok1 && ok2 && cur-prev >= k.T.PainTrendPoints
}

// baselineBefore returns the pain value before a session: the
// before_session report, else the latest morning or daily report before it,
// else -1: without any baseline the morning rule (PAR-D-16) cannot be
// judged, and only the thresholds apply.
func baselineBefore(reports []PainReport, region, sessionID string, at time.Time) float64 {
	var best PainReport
	found := false
	for _, r := range reports {
		if r.Region != region {
			continue
		}
		if r.Timepoint == PainBefore && r.SessionID == sessionID {
			return r.NRS
		}
		if (r.Timepoint == PainMorning || r.Timepoint == PainDaily) && r.At.Before(at) && (!found || r.At.After(best.At)) {
			best, found = r, true
		}
	}
	if found {
		return best.NRS
	}
	return -1
}
