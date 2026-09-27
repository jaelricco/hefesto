package planning

import (
	"math"
	"slices"
	"strings"
	"time"
)

// Load accounts (spec §4.5): every structure except the wrist has a
// straight-arm and a bent-arm account; the wrist has one.
const wrist = "wrist"

func account(structure string, straight bool) string {
	if structure == wrist {
		return wrist
	}
	if straight {
		return structure + "/SA"
	}
	return structure + "/BA"
}

func accountStructure(a string) string {
	s, _, _ := strings.Cut(a, "/")
	return s
}

func isStraightAccount(a string) bool { return a == wrist || strings.HasSuffix(a, "/SA") }

// capRate is the weekly increase c of an account (spec §7.1).
func (k *Knowledge) capRate(a string) float64 {
	switch {
	case a == wrist:
		return k.T.CapWrist
	case strings.HasSuffix(a, "/SA"):
		return k.T.CapStraight
	default:
		return k.T.CapBent
	}
}

// rEff is the effective 0–3 rating of an exercise on a structure (spec §4.5).
func (k *Knowledge) rEff(ex *Exercise, structure string) int {
	r := k.families[ex.LoadFamily].Ratings[structure]
	for _, m := range ex.Modifiers {
		switch {
		case m == "neutral_grip" && structure == wrist:
			r--
		case m == "rings" && (structure == "biceps_distal" || structure == "biceps_long_head_anterior_shoulder"):
			r++
		case m == "wide_or_supinated_grip" && structure == "shoulder_overhead":
			r++
		}
	}
	return min(3, max(0, r))
}

// scale is k(e) of the load unit (PAR-C-44).
func (k *Knowledge) scale(ex *Exercise, loadKg, bw float64) float64 {
	fam := k.families[ex.LoadFamily]
	s := 1.0
	switch {
	case ex.Torque > 0 && fam.RefTorque > 0:
		s = ex.Torque / fam.RefTorque
	case ex.BW > 0 && fam.RefBW > 0:
		s = ex.BW / fam.RefBW
	}
	if loadKg > 0 && bw > 0 {
		s = float64(s * (bw + loadKg) / bw)
	}
	return s
}

// setLoad returns the load units of one set on every account it touches.
func (k *Knowledge) setLoad(ex *Exercise, kind string, loadKg, bw float64) map[string]float64 {
	w := 1.0
	if kind == KindWarmup {
		w = k.T.WarmupWeight
	}
	sc := k.scale(ex, loadKg, bw)
	straight := ex.StraightArm != ArmNone
	out := map[string]float64{}
	for _, s := range k.structures {
		r := k.rEff(ex, s)
		if r == 0 {
			continue
		}
		u := float64(w * math.Min(3, float64(float64(r)*sc)) / 3)
		out[account(s, straight)] += u
	}
	return out
}

// loadHistory aggregates the log (spec §4.5).
type loadHistory struct {
	weekly     map[time.Time]map[string]float64 // ISO week start → account → W
	deloadWeek map[time.Time]bool
	sessionMax map[string]float64 // structure → largest session load in the spike window
	lastLoad   map[string]time.Time
	hardAt     map[string][]hardMark // structure → dates of hard or moderate stimuli
}

type hardMark struct {
	day   time.Time
	class int
}

func (k *Knowledge) history(s Snapshot, today time.Time) loadHistory {
	h := loadHistory{
		weekly:     map[time.Time]map[string]float64{},
		deloadWeek: map[time.Time]bool{},
		sessionMax: map[string]float64{},
		lastLoad:   map[string]time.Time{},
		hardAt:     map[string][]hardMark{},
	}
	bw := s.Profile.BodyweightKg
	for _, sess := range s.History {
		if !sess.Date.Before(today) {
			continue
		}
		wk := weekStart(sess.Date)
		if sess.Deload {
			h.deloadWeek[wk] = true
		}
		perStructure := map[string]float64{}
		for _, set := range sess.Sets {
			ex, ok := k.exercises[set.Exercise]
			if !ok {
				continue
			}
			for a, u := range k.setLoad(ex, set.Kind, set.LoadKg, bw) {
				if h.weekly[wk] == nil {
					h.weekly[wk] = map[string]float64{}
				}
				h.weekly[wk][a] += u
				perStructure[accountStructure(a)] += u
				if u > 0 && sess.Date.After(h.lastLoad[a]) {
					h.lastLoad[a] = sess.Date
				}
			}
			cls := k.loggedClass(ex, set)
			if cls > classLight {
				for _, st := range k.structures {
					if float64(k.rEff(ex, st)) >= k.T.SpacingRating {
						h.hardAt[st] = append(h.hardAt[st], hardMark{sess.Date, cls})
					}
				}
			}
		}
		if daysBetween(sess.Date, today) <= k.T.SpikeDays {
			for st, v := range perStructure {
				h.sessionMax[st] = math.Max(h.sessionMax[st], v)
			}
		}
	}
	return h
}

// reference is R(a): the mean over up to three completed weeks with load,
// without week-level deloads, within the monitoring window (PAR-S-14).
func (k *Knowledge) reference(h loadHistory, a string, week time.Time) (float64, int) {
	var vals []float64
	for i := 1; i <= int(k.T.WindowWeeks) && len(vals) < 3; i++ {
		wk := week.AddDate(0, 0, -7*i)
		if h.deloadWeek[wk] {
			continue
		}
		if v := h.weekly[wk][a]; v > 0 {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals)), len(vals)
}

// Intensity classes for spacing (spec §7.5).
const (
	classLight = iota
	classModerate
	classHard
)

// loggedClass infers the spacing class of a logged set.
func (k *Knowledge) loggedClass(ex *Exercise, set LoggedSet) int {
	if set.Kind == KindWarmup {
		return classLight
	}
	if set.Kind == KindTest || ex.Eccentric {
		return classHard
	}
	if ex.StraightArm == ArmStraight && ex.HoldClass == HoldSkill {
		return classHard
	}
	if ex.StraightArm == ArmStraight {
		return classModerate
	}
	if ex.Measure == MeasureReps && ex.Pattern != "prehab" && ex.Pattern != "mobility" {
		if set.Reserve != nil && *set.Reserve <= 1 {
			return classHard
		}
		return classModerate
	}
	return classLight
}

// structuresAt returns the structures an exercise loads at or above the
// spacing threshold (PAR-S-07), sorted.
func (k *Knowledge) structuresAt(ex *Exercise) []string {
	var out []string
	for _, s := range k.structures {
		if float64(k.rEff(ex, s)) >= k.T.SpacingRating {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out
}
