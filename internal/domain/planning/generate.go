package planning

import (
	"errors"
	"math"
	"slices"
	"time"
)

// Plan is one week of planned sessions (spec §5.10).
type Plan struct {
	WeekStart      time.Time          `json:"week_start"`
	RulesetVersion string             `json:"ruleset_version"`
	InputHash      string             `json:"input_hash"`
	Stopped        bool               `json:"stopped"`
	Deload         string             `json:"deload,omitempty"`
	Sessions       []PlannedSession   `json:"sessions"`
	RestDays       []time.Time        `json:"rest_days,omitempty"`
	Exclusions     []Exclusion        `json:"exclusions,omitempty"`
	Hints          []Reason           `json:"hints,omitempty"`
	Realism        []Realism          `json:"realism,omitempty"`
	Monitor        []RegionMonitor    `json:"monitor,omitempty"`
	Headroom       map[string]float64 `json:"headroom,omitempty"`
	Loads          []AccountLoad      `json:"loads,omitempty"`
	Reasons        []Reason           `json:"reasons"`
	Disclaimer     string             `json:"disclaimer"`
}

// PlannedSession is one session of the plan.
type PlannedSession struct {
	Index      int       `json:"index"`
	Date       time.Time `json:"date"`
	Kind       string    `json:"kind"` // full, light, deload
	EstMinutes float64   `json:"est_minutes"`
	Blocks     []Block   `json:"blocks"`
	Reasons    []Reason  `json:"reasons,omitempty"`
}

// Session kinds.
const (
	SessionFull   = "full"
	SessionLight  = "light"
	SessionDeload = "deload"
)

// Block roles in session order (SESS-02).
const (
	BlockWarmup   = "warmup"
	BlockBalance  = "balance"
	BlockMax      = "skill_max"
	BlockVolume   = "skill_volume"
	BlockStrength = "strength"
	BlockEnd      = "end"
)

// Block is a group of items with one purpose.
type Block struct {
	Role    string   `json:"role"`
	Minutes float64  `json:"minutes,omitempty"` // general warm-up or balance time
	Paired  bool     `json:"paired,omitempty"`
	Items   []Item   `json:"items"`
	Reasons []Reason `json:"reasons,omitempty"`
}

// Item is a group of identical planned sets of one exercise.
type Item struct {
	Exercise    string   `json:"exercise"`
	Skill       string   `json:"skill,omitempty"`
	Stimulus    string   `json:"stimulus"`
	Kind        string   `json:"kind"` // working or warmup
	Sets        int      `json:"sets"`
	Reps        int      `json:"reps,omitempty"`
	HoldS       int      `json:"hold_s,omitempty"`
	LoadKg      float64  `json:"load_kg,omitempty"`
	Assist      string   `json:"assist,omitempty"`
	Reserve     int      `json:"reserve"` // RIR or SIR
	RestS       int      `json:"rest_s"`
	Calibration bool     `json:"calibration,omitempty"`
	Offer       bool     `json:"offer,omitempty"` // probe or first attempt: only on active acceptance
	StopRules   []string `json:"stop_rules,omitempty"`
	Class       int      `json:"intensity"`
	Priority    int      `json:"priority"`
	Role        string   `json:"role"`
	Monitor     bool     `json:"monitor,omitempty"`
	Reasons     []Reason `json:"reasons"`
}

// AccountLoad is the week load of one account before and after the caps
// (spec §7.2): Target is what the ladders asked for, Cap the limit and Rule
// the rule that set it.
type AccountLoad struct {
	Account string  `json:"account"`
	Target  float64 `json:"target"`
	Planned float64 `json:"planned"`
	Cap     float64 `json:"cap"`
	Rule    string  `json:"rule"`
}

// Exclusion explains why an exercise is not in the plan.
type Exclusion struct {
	Exercise string `json:"exercise"`
	Reason   Reason `json:"reason"`
}

// RegionMonitor lists the red-flag questions the client asks offline for a
// monitored region (spec §10.5).
type RegionMonitor struct {
	Region    string    `json:"region"`
	Questions []RedFlag `json:"questions"`
}

// ErrOnboardingRequired is SAFE-01.
var ErrOnboardingRequired = errors.New("onboarding required")

// gen carries one generation run.
type gen struct {
	k         *Knowledge
	s         Snapshot
	now       time.Time
	week      time.Time
	today     time.Time
	hist      loadHistory
	exp       string
	minor     bool
	equipment map[string]bool
	hasBands  bool
	plan      *Plan
	ladders   []*active
	slots     []*slot
	fullCount int
	deload    string
	weekIdx   int // weeks since onboarding
	caps      map[string]float64
	capRules  map[string]string
	targets   map[string]float64
	// targetSess is the session load per structure before the caps.
	targetSess []map[string]float64
}

// Generate builds the plan for the week starting at week (a Monday, date at
// 00:00 UTC) as seen at now (spec §5.1). It is deterministic.
func Generate(k *Knowledge, s Snapshot, now, week time.Time) (Plan, error) {
	if !s.Profile.DisclaimerAck || s.Profile.OnboardedAt.IsZero() {
		return Plan{}, ErrOnboardingRequired
	}
	g := &gen{
		k: k, s: s, now: now, week: week, today: civil(now, time.UTC),
		exp:       experience(k, s.Profile),
		minor:     isMinor(k, s.Profile.BirthYear, now),
		equipment: map[string]bool{},
		plan: &Plan{WeekStart: week, RulesetVersion: k.Version, Disclaimer: k.Disclaimer(),
			InputHash: InputHash(k, s, now, week)},
	}
	for _, e := range s.Profile.Equipment {
		g.equipment[e] = true
	}
	g.hasBands = g.equipment["resistance_bands"]
	g.weekIdx = int(math.Floor(daysBetween(weekStart(s.Profile.OnboardedAt), week) / 7))

	// SAFE-07, SAFE-02.
	if g.minor {
		g.plan.Stopped = true
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleMinor))
		return *g.plan, nil
	}
	if hasConstraint(s, ConstraintStopped) || s.Screening.ExertionSymptoms && !s.Screening.Cleared {
		g.plan.Stopped = true
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleStopped))
		return *g.plan, nil
	}
	if s.Screening.AnyYes && !s.Screening.Cleared {
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleScreening))
	}
	if !s.Profile.HealthConsent {
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleNoConsent))
	}
	g.hist = k.history(s, week)
	g.deload = g.deloadKind()
	if g.deload != "" {
		g.plan.Deload = g.deload
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleDeloadWeek, "kind", g.deload))
	}

	g.activeLadders()
	for _, a := range g.ladders {
		g.selectRung(a)
	}
	g.addFeeders()
	g.antagonistHint()
	g.ladders = slices.DeleteFunc(g.ladders, func(a *active) bool { return a.rung == nil })
	g.planDays()
	g.allocate()
	g.buildSessions()
	g.trimLoads()
	g.applyDeload()
	g.trimTime()
	g.consolidate()
	g.finish()
	return *g.plan, nil
}

// antagonistHint explains a missing counter-direction (GOAL-06): when no
// rung of the antagonist ladder can be planned, the user sees why.
func (g *gen) antagonistHint() {
	for _, a := range g.ladders {
		if a.rung != nil || !slices.ContainsFunc(a.reasons, func(r Reason) bool { return r.RuleID == RuleAntagonist }) {
			continue
		}
		for _, rung := range a.skill.Rungs { // the easiest rung explains best
			for _, e := range g.plan.Exclusions {
				if e.Exercise == rung {
					g.plan.Hints = append(g.plan.Hints, a.reasons[0], e.Reason)
					return
				}
			}
		}
	}
}

// addFeeders adds the feeder skills of ladders on their eccentric rung
// (SEL-09).
func (g *gen) addFeeders() {
	have := map[string]bool{}
	for _, a := range g.ladders {
		have[a.skill.Slug] = true
	}
	for _, a := range slices.Clone(g.ladders) {
		if a.stim != StimEccentric {
			continue
		}
		for _, f := range a.skill.Feeders {
			if have[f] || g.k.skills[f] == nil {
				continue
			}
			have[f] = true
			na := &active{skill: g.k.skills[f], role: RoleSupport, priority: a.priority,
				reasons: []Reason{g.k.reason(RuleRepRung, "exercise", a.rung.Name)}}
			g.selectRung(na)
			if na.rung != nil {
				g.ladders = append(g.ladders, na)
			}
		}
	}
}

// deloadKind decides whether this week is a deload week (WEEK-07, ADAPT-14).
func (g *gen) deloadKind() string {
	k, ph := g.k, g.s.Phase
	if !ph.DeloadWeek.IsZero() && ph.DeloadWeek.Equal(g.week) {
		return ph.DeloadKind
	}
	if ph.MesoStart.IsZero() {
		return ""
	}
	weeks := math.Floor(daysBetween(ph.MesoStart, g.week)/7) + 1
	if weeks == k.T.MesoWeeks {
		return DeloadPlanned
	}
	if weeks > k.T.MaxBuildWeeks {
		return DeloadMaxBuild
	}
	return ""
}

// finish drops sessions left without training, fills rest days,
// monitoring questions and hints.
func (g *gen) finish() {
	k := g.k
	// A day with nothing but warm-up and prehab is a planned rest day
	// (WEEK-02); the planner never writes user_training_days (ENT-S-8).
	kept := g.plan.Sessions[:0]
	dropped := 0
	for _, ps := range g.plan.Sessions {
		if hasTraining(ps) {
			ps.Index = len(kept)
			kept = append(kept, ps)
		} else {
			dropped++
		}
	}
	g.plan.Sessions = kept
	if dropped > 0 {
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleSessionKind, "full", g.fullCount, "rest", dropped))
	}
	week := g.weekLoads()
	for _, a := range sortedKeys(g.caps) {
		g.plan.Loads = append(g.plan.Loads, AccountLoad{Account: a, Target: round3(g.targets[a]),
			Planned: round3(week[a]), Cap: round3(g.caps[a]), Rule: g.capRules[a]})
	}
	used := map[time.Weekday]bool{}
	for _, ps := range g.plan.Sessions {
		used[ps.Date.Weekday()] = true
	}
	for d := 0; d < 7; d++ {
		day := g.week.AddDate(0, 0, d)
		if !used[day.Weekday()] {
			g.plan.RestDays = append(g.plan.RestDays, day)
		}
	}
	for _, id := range sortedKeys(g.s.Regions) {
		rs := g.s.Regions[id]
		if rs.State == StateNormal && !rs.Complaint {
			continue
		}
		g.plan.Monitor = append(g.plan.Monitor, RegionMonitor{Region: id, Questions: k.RedFlags(id, g.minor)})
	}
	for _, goal := range g.s.Goals {
		if goal.TargetDate != nil {
			if r, ok := k.RealismFor(g.s, goal, g.now); ok {
				g.plan.Realism = append(g.plan.Realism, r)
			}
		}
	}
	slices.SortStableFunc(g.plan.Exclusions, func(a, b Exclusion) int {
		if a.Exercise < b.Exercise {
			return -1
		}
		if a.Exercise > b.Exercise {
			return 1
		}
		return 0
	})
}

// hasTraining reports whether a session has a working set outside the
// warm-up and prehab.
func hasTraining(ps PlannedSession) bool {
	for _, b := range ps.Blocks {
		if b.Role == BlockWarmup {
			continue
		}
		for _, it := range b.Items {
			if it.Sets > 0 && it.Stimulus != StimPrehab {
				return true
			}
		}
	}
	return false
}
