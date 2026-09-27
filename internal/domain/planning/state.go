package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Snapshot is everything the planner knows about one user at one moment
// (spec §4). It is a value: the core never mutates it; Adapt returns a new
// one.
type Snapshot struct {
	Profile     Profile                `json:"profile"`
	Goals       []Goal                 `json:"goals"`
	Capacities  map[string]Estimate    `json:"capacities"` // key: CapKey(exercise, assist)
	Ladders     map[string]LadderState `json:"ladders"`    // key: skill slug
	Regions     map[string]RegionState `json:"regions"`    // key: region ID
	Screening   Screening              `json:"screening"`
	Constraints []Constraint           `json:"constraints"`
	Phase       Phase                  `json:"phase"`
	History     []LoggedSession        `json:"history"` // completed sessions, oldest first
	Pain        []PainReport           `json:"pain"`
	Unlocked    map[string]bool        `json:"unlocked"` // "skill/level"
	Headroom    map[string]float64     `json:"headroom"` // per load account, in sets (PAR-S-35)
	Entry       map[string]bool        `json:"entry"`    // accounts eligible for LOAD-04b
	Break       *BreakState            `json:"break,omitempty"`
}

// BreakState is a return after a pause (spec §6.11). Straight-arm and wrist
// accounts ramp from 25 % (PAR-D-29, PAR-D-33); the others follow the
// stream-B ramp. It is the "entered_via = break" case of the region
// automaton, kept per account rather than per region because a pause
// affects every region at once.
type BreakState struct {
	Days         float64   `json:"days"`  // length of the pause
	Since        time.Time `json:"since"` // first week back
	Step         int       `json:"step"`
	StepSince    time.Time `json:"step_since"`
	StepSessions int       `json:"step_sessions"`
	Logged       bool      `json:"logged"` // the pre-break reference comes from logs
}

// Profile is the onboarding profile with its derived fields (spec §4.2).
type Profile struct {
	BirthYear       int               `json:"birth_year"`
	SessionsPerWeek int               `json:"sessions_per_week"`
	SessionMinutes  int               `json:"session_minutes"`
	Equipment       []string          `json:"equipment"` // expanded, sorted
	BodyweightKg    float64           `json:"bodyweight_kg"`
	TrainingLevel   string            `json:"training_level"`
	TrainingMonths  float64           `json:"training_months"` // lower bound, advanced by logs
	LastRegular     string            `json:"last_regular_training"`
	HealthConsent   bool              `json:"health_consent"`
	DisclaimerAck   bool              `json:"disclaimer_ack"`
	OnboardedAt     time.Time         `json:"onboarded_at"`
	PreferredDays   []time.Weekday    `json:"preferred_days,omitempty"`
	Mobility        map[string]string `json:"mobility,omitempty"`
	MaxAddedLoadKg  float64           `json:"max_added_load_kg,omitempty"`
	SmallestPlateKg float64           `json:"smallest_plate_kg,omitempty"`
}

// Training levels (PAR-F-48 scale, onboarding §3.5).
const (
	LevelSedentary     = "sedentary"
	LevelRecreational  = "recreational"
	LevelTrained       = "trained"
	LevelHighlyTrained = "highly_trained"
)

// Goal is one of up to three goals (onboarding §3.2).
type Goal struct {
	Skill       string     `json:"skill"`
	TargetLevel string     `json:"target_level"`
	Priority    int        `json:"priority"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
}

// Estimate is a capacity as a normal distribution (spec §4.3).
type Estimate struct {
	Mu      float64   `json:"mu"`
	Sigma   float64   `json:"sigma"`
	Origin  string    `json:"origin"` // self_report, test, log, derived
	At      time.Time `json:"at"`
	N       int       `json:"n"`
	Pending *float64  `json:"pending,omitempty"` // last contradicting observation (ADAPT-03)
}

// Capacity origins.
const (
	OriginSelf    = "self_report"
	OriginLog     = "log"
	OriginTest    = "test"
	OriginDerived = "derived"
)

// Assistance keys of a capacity.
const (
	AssistNone = "none"
	AssistBand = "band"
)

// CapKey is the capacity map key.
func CapKey(exercise, assist string) string { return exercise + "|" + assist }

// LadderState is the working rung of a skill (spec §4.4).
type LadderState struct {
	Rung       string    `json:"rung"`
	Status     string    `json:"status"` // claimed or calibrated
	Since      time.Time `json:"since"`
	Exposures  int       `json:"exposures"`
	Claimed    string    `json:"claimed,omitempty"`  // stage the user named in the onboarding
	CapRung    string    `json:"cap_rung,omitempty"` // highest rung during a break ramp (pre_break_level)
	ProbeOffer bool      `json:"probe_offer,omitempty"`
	RepTarget  float64   `json:"rep_target,omitempty"` // double progression (ADAPT-07)
	LoadKg     float64   `json:"load_kg,omitempty"`    // added load (ADAPT-09)
	EccS       float64   `json:"ecc_s,omitempty"`      // current eccentric duration (ADAPT-10)
	LastUp     time.Time `json:"last_up,omitempty"`
	// CapTo caps the working rung until CapUntil (pain deload, regression).
	CapTo    string    `json:"cap_to,omitempty"`
	CapUntil time.Time `json:"cap_until"`
}

// Ladder statuses.
const (
	StatusClaimed    = "claimed"
	StatusCalibrated = "calibrated"
)

// RegionState is the tolerance state of a body-map region (spec §4.6, §8.3).
type RegionState struct {
	State         string             `json:"state"`
	EnteredVia    string             `json:"entered_via,omitempty"`
	Since         time.Time          `json:"since"`
	StartFraction float64            `json:"start_fraction,omitempty"`
	Step          int                `json:"step,omitempty"` // index into the PAR-D-25 steps
	StepSince     time.Time          `json:"step_since"`
	StepSessions  int                `json:"step_sessions,omitempty"`
	BreakOnly     bool               `json:"break_only,omitempty"` // ramp only for SA and wrist accounts
	Reference     map[string]float64 `json:"reference,omitempty"`  // logged pre-complaint R per account
	PriorInjury   bool               `json:"prior_injury,omitempty"`
	Complaint     bool               `json:"complaint,omitempty"`
	ComplaintAt   time.Time          `json:"complaint_at"`
	Restrictions  []string           `json:"restrictions,omitempty"`
	Breaches      []time.Time        `json:"breaches,omitempty"`
	PainDeloadTo  time.Time          `json:"pain_deload_to"`
	HoldAtRef     bool               `json:"hold_at_ref,omitempty"` // cap 1.0 × R until a green week (PAR-S-40)
	Referral      string             `json:"referral,omitempty"`    // soft, advise
}

// Region states.
const (
	StateNormal = "normal"
	StateLocked = "locked"
	StateRTT0   = "rtt_0"
	StateRTT1   = "rtt_1"
	StateRTT2   = "rtt_2"
	StateRTT3   = "rtt_3"
	StateRTT4   = "rtt_4"
	StateRTT5   = "rtt_5"
)

// rttStage returns the ramp stage 0–5, or -1 for normal and locked.
func rttStage(state string) int {
	switch state {
	case StateRTT0:
		return 0
	case StateRTT1:
		return 1
	case StateRTT2:
		return 2
	case StateRTT3:
		return 3
	case StateRTT4:
		return 4
	case StateRTT5:
		return 5
	}
	return -1
}

func rttState(stage int) string {
	return [...]string{StateRTT0, StateRTT1, StateRTT2, StateRTT3, StateRTT4, StateRTT5}[stage]
}

// Screening is the health screening outcome (onboarding §3.7).
type Screening struct {
	ExertionSymptoms bool `json:"exertion_symptoms"`
	AnyYes           bool `json:"any_yes"`
	Cleared          bool `json:"cleared"`
}

// Constraint is a stored safety constraint without answers or values
// (spec §4.7, ENT-S-7).
type Constraint struct {
	Kind    string    `json:"kind"` // plan_stopped, region_locked, region_excluded
	Region  string    `json:"region,omitempty"`
	Created time.Time `json:"created"`
}

// Constraint kinds.
const (
	ConstraintStopped  = "plan_stopped"
	ConstraintLocked   = "region_locked"
	ConstraintExcluded = "region_excluded"
)

// Phase is the mesocycle state (spec §4.7).
type Phase struct {
	MesoStart  time.Time `json:"meso_start"`
	LastDeload time.Time `json:"last_deload"`
	DeloadWeek time.Time `json:"deload_week"`           // week that is a deload week
	DeloadKind string    `json:"deload_kind,omitempty"` // its kind
	DeloadNext string    `json:"deload_next,omitempty"` // kind scheduled for the next week
	Calibrate  []string  `json:"calibrate,omitempty"`   // capacity keys due for a calibration set
}

// Deload kinds.
const (
	DeloadPlanned    = "planned"
	DeloadStagnation = "stagnation"
	DeloadFatigue    = "fatigue"
	DeloadMaxBuild   = "max_build"
	DeloadPain       = "pain"
)

// LoggedSession is one completed session from the log.
type LoggedSession struct {
	ID      string      `json:"id"`
	Date    time.Time   `json:"date"` // local calendar date at 00:00 UTC
	Deload  bool        `json:"deload,omitempty"`
	Fatigue *float64    `json:"fatigue,omitempty"` // perceived_fatigue 1–10
	Sets    []LoggedSet `json:"sets"`
}

// LoggedSet is one set element as logged, in the order performed.
type LoggedSet struct {
	ID        string   `json:"id"`
	Exercise  string   `json:"exercise"`
	Kind      string   `json:"kind"` // working, warmup, test, backoff, cluster, drop
	Assist    string   `json:"assist"`
	LoadKg    float64  `json:"load_kg,omitempty"`
	Value     float64  `json:"value"`             // reps or seconds
	Reserve   *float64 `json:"reserve,omitempty"` // RIR or SIR
	Form      *float64 `json:"form,omitempty"`
	Failed    bool     `json:"failed,omitempty"`
	Partial   bool     `json:"partial,omitempty"`
	Eccentric bool     `json:"eccentric,omitempty"`
}

// Set kinds.
const (
	KindWorking = "working"
	KindWarmup  = "warmup"
	KindTest    = "test"
)

// PainReport is one NRS report (spec §8.6).
type PainReport struct {
	Region      string    `json:"region"`
	Timepoint   string    `json:"timepoint"` // before_session, warmup, during, after, next_morning, daily
	NRS         float64   `json:"nrs"`
	At          time.Time `json:"at"`
	LastedOver  bool      `json:"lasted_over_1h,omitempty"`
	Persisted   bool      `json:"persisted_over_15min,omitempty"`
	SuddenSharp bool      `json:"sudden_sharp,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
}

// Pain timepoints.
const (
	PainBefore  = "before_session"
	PainWarmup  = "warmup"
	PainDuring  = "during"
	PainAfter   = "after"
	PainMorning = "next_morning"
	PainDaily   = "daily"
)

// InputHash is the SHA-256 of the canonical JSON of everything a plan
// depends on (spec §1.3). encoding/json sorts map keys, which makes the
// encoding canonical for this struct.
func InputHash(k *Knowledge, s Snapshot, now, week time.Time) string {
	payload := struct {
		Version  string    `json:"ruleset_version"`
		Snapshot Snapshot  `json:"snapshot"`
		Now      time.Time `json:"now"`
		Week     time.Time `json:"week_start"`
	}{k.Version, s, now.UTC(), week.UTC()}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// clone returns a deep copy of the snapshot's maps and slices that Adapt
// changes, so callers' values are never mutated.
func (s Snapshot) clone() Snapshot {
	out := s
	out.Capacities = cloneMap(s.Capacities)
	out.Ladders = cloneMap(s.Ladders)
	out.Regions = map[string]RegionState{}
	for k, v := range s.Regions {
		v.Reference = cloneMap(v.Reference)
		v.Breaches = append([]time.Time(nil), v.Breaches...)
		v.Restrictions = append([]string(nil), v.Restrictions...)
		out.Regions[k] = v
	}
	out.Constraints = append([]Constraint(nil), s.Constraints...)
	out.Phase.Calibrate = append([]string(nil), s.Phase.Calibrate...)
	out.Unlocked = cloneMap(s.Unlocked)
	out.Headroom = cloneMap(s.Headroom)
	out.Entry = cloneMap(s.Entry)
	out.Pain = append([]PainReport(nil), s.Pain...)
	out.History = append([]LoggedSession(nil), s.History...)
	if s.Break != nil {
		b := *s.Break
		out.Break = &b
	}
	return out
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// civil returns midnight UTC of the calendar date of t in loc.
func civil(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// daysBetween returns whole days from a to b (dates at 00:00 UTC).
func daysBetween(a, b time.Time) float64 {
	return b.Sub(a).Hours() / 24
}

// weekStart returns the Monday of the ISO week of d.
func weekStart(d time.Time) time.Time {
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return d.AddDate(0, 0, -(wd - 1))
}
