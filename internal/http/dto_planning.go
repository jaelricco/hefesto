package http

import (
	"cmp"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	core "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/planning"
)

// DTOs of the planning tag. The core's types keep their own JSON for the
// snapshot and the stored plan; these are the API's, as api/openapi.yaml
// declares them: every list present, absent values null, ISO weekdays.

// ------------------------------------------------------------------ input

type goalIn struct {
	Skill       string  `json:"skill"`
	TargetLevel string  `json:"target_level"`
	Priority    int     `json:"priority"`
	TargetDate  *string `json:"target_date"`
}

func goalsToCore(in []goalIn) ([]core.Goal, error) {
	out := make([]core.Goal, len(in))
	for i, g := range in {
		out[i] = core.Goal{Skill: g.Skill, TargetLevel: g.TargetLevel, Priority: g.Priority}
		if g.TargetDate != nil {
			d, err := time.Parse(dateLayout, *g.TargetDate)
			if err != nil {
				return nil, errBadRequest{fmt.Sprintf("goals[%d].target_date is not a date (YYYY-MM-DD)", i)}
			}
			out[i].TargetDate = &d
		}
	}
	return out, nil
}

// weekdaysToCore maps ISO weekdays (1 = Monday … 7 = Sunday) to Go's.
func weekdaysToCore(days []int) []time.Weekday {
	if days == nil {
		return nil
	}
	out := make([]time.Weekday, len(days))
	for i, d := range days {
		out[i] = time.Weekday(d % 7)
	}
	return out
}

func weekdaysOut(days []time.Weekday) []int {
	out := make([]int, len(days))
	for i, d := range days {
		out[i] = int(d)
		if d == time.Sunday {
			out[i] = 7
		}
	}
	return out
}

type stageIn struct {
	Level string `json:"level"`
	Class string `json:"class"`
}

type bodyMapIn struct {
	Current bool `json:"current"`
	Past12  bool `json:"past_12_months"`
}

type complaintIn struct {
	PainDaily     float64  `json:"pain_daily"`
	PainTraining  float64  `json:"pain_training"`
	Onset         string   `json:"onset"`
	DurationWeeks float64  `json:"duration_weeks"`
	Suspected     string   `json:"suspected_serious"`
	Assessment    string   `json:"professional_assessment"`
	Restrictions  []string `json:"restrictions"`
}

type onboardingIn struct {
	BirthYear        int                        `json:"birth_year"`
	HealthConsent    bool                       `json:"health_data_consent"`
	DisclaimerAck    bool                       `json:"disclaimer_ack"`
	Goals            []goalIn                   `json:"goals"`
	SessionsPerWeek  int                        `json:"sessions_per_week"`
	SessionMinutes   int                        `json:"session_minutes"`
	Equipment        []string                   `json:"equipment"`
	BodyweightKg     float64                    `json:"bodyweight_kg"`
	TrainingLevel    string                     `json:"training_level"`
	TrainingAge      string                     `json:"calisthenics_training_age"`
	LastRegular      string                     `json:"last_regular_training"`
	PreBreak         map[string]string          `json:"pre_break_level"`
	Classes          map[string]string          `json:"classes"`
	Stages           map[string]stageIn         `json:"skill_stages"`
	Mobility         map[string]string          `json:"mobility_checks"`
	DataConfidence   string                     `json:"data_confidence"`
	ExertionSymptoms bool                       `json:"exertion_symptoms"`
	Screening        []bool                     `json:"screening"`
	BodyMap          map[string]bodyMapIn       `json:"body_map"`
	Complaints       map[string]complaintIn     `json:"complaints"`
	RedFlags         map[string]map[string]bool `json:"red_flags"`
	Clarifications   map[string]bool            `json:"clarifications"`
	PreferredDays    []int                      `json:"preferred_days"`
}

func (in onboardingIn) toCore() (core.Answers, error) {
	goals, err := goalsToCore(in.Goals)
	if err != nil {
		return core.Answers{}, err
	}
	a := core.Answers{
		BirthYear: in.BirthYear, HealthConsent: in.HealthConsent, DisclaimerAck: in.DisclaimerAck, Goals: goals,
		SessionsPerWeek: in.SessionsPerWeek, SessionMinutes: in.SessionMinutes, Equipment: in.Equipment,
		BodyweightKg: in.BodyweightKg, TrainingLevel: in.TrainingLevel, TrainingAge: in.TrainingAge,
		LastRegular: in.LastRegular, PreBreak: in.PreBreak, Classes: in.Classes, Mobility: in.Mobility,
		DataConfidence: in.DataConfidence, ExertionSymptoms: in.ExertionSymptoms, Screening: in.Screening,
		RedFlags: in.RedFlags, Clarifications: in.Clarifications, PreferredDays: weekdaysToCore(in.PreferredDays),
	}
	if a.Classes == nil {
		a.Classes = map[string]string{}
	}
	if in.Stages != nil {
		a.Stages = map[string]core.StageAnswer{}
		for k, v := range in.Stages {
			a.Stages[k] = core.StageAnswer{Level: v.Level, Class: v.Class}
		}
	}
	if in.BodyMap != nil {
		a.BodyMap = map[string]core.BodyMapEntry{}
		for k, v := range in.BodyMap {
			a.BodyMap[k] = core.BodyMapEntry{Current: v.Current, Past12: v.Past12}
		}
	}
	if in.Complaints != nil {
		a.Complaints = map[string]core.Complaint{}
		for k, v := range in.Complaints {
			a.Complaints[k] = core.Complaint{PainDaily: v.PainDaily, PainTraining: v.PainTraining, Onset: v.Onset,
				DurationWeeks: v.DurationWeeks, Suspected: v.Suspected, Assessment: v.Assessment, Restrictions: v.Restrictions}
		}
	}
	return a, nil
}

type profileIn struct {
	SessionsPerWeek int               `json:"sessions_per_week"`
	SessionMinutes  int               `json:"session_minutes"`
	Equipment       []string          `json:"equipment"`
	BodyweightKg    float64           `json:"bodyweight_kg"`
	PreferredDays   []int             `json:"preferred_days"`
	Mobility        map[string]string `json:"mobility_checks"`
	MaxAddedLoadKg  float64           `json:"max_added_load_kg"`
	SmallestPlateKg float64           `json:"smallest_plate_kg"`
}

func (in profileIn) toCore() core.ProfileUpdate {
	return core.ProfileUpdate{SessionsPerWeek: in.SessionsPerWeek, SessionMinutes: in.SessionMinutes,
		Equipment: in.Equipment, BodyweightKg: in.BodyweightKg, PreferredDays: weekdaysToCore(in.PreferredDays),
		Mobility: in.Mobility, MaxAddedLoadKg: in.MaxAddedLoadKg, SmallestPlateKg: in.SmallestPlateKg}
}

type goalsIn struct {
	Goals []goalIn `json:"goals"`
}

type eventIn struct {
	ID uuid.UUID `json:"id"`
}

type consentIn struct {
	ID           uuid.UUID `json:"id"`
	Granted      bool      `json:"granted"`
	Screening    []bool    `json:"screening"`
	PastInjuries []string  `json:"past_injuries"`
}

type plannedSessionStartIn struct {
	ID        uuid.UUID  `json:"id"`
	StartedAt *time.Time `json:"started_at"`
	Timezone  string     `json:"timezone"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type redFlagsIn struct {
	ID      uuid.UUID       `json:"id"`
	Answers map[string]bool `json:"answers"`
}

type painIn struct {
	ID          uuid.UUID  `json:"id"`
	Region      string     `json:"region"`
	Timepoint   string     `json:"timepoint"`
	NRS         float64    `json:"nrs"`
	At          time.Time  `json:"at"`
	LastedOver  bool       `json:"lasted_over_1h"`
	Persisted   bool       `json:"persisted_over_15min"`
	SuddenSharp bool       `json:"sudden_sharp"`
	SessionID   *uuid.UUID `json:"session_id"`
}

func (in painIn) toCore() core.PainReport {
	r := core.PainReport{Region: in.Region, Timepoint: in.Timepoint, NRS: in.NRS, At: in.At.UTC(),
		LastedOver: in.LastedOver, Persisted: in.Persisted, SuddenSharp: in.SuddenSharp}
	if in.SessionID != nil {
		r.SessionID = in.SessionID.String()
	}
	return r
}

// ----------------------------------------------------------------- output

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dateOut(t time.Time) string { return t.UTC().Format(dateLayout) }

func dateOrNil(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	d := dateOut(t)
	return &d
}

type reasonOut struct {
	RuleID   string            `json:"rule_id"`
	Params   []string          `json:"params"`
	Sources  []string          `json:"sources"`
	Evidence string            `json:"evidence"`
	Text     string            `json:"text"`
	Args     map[string]string `json:"args"`
	Region   *string           `json:"region"`
}

// reasonFrom keeps the sources off a reason tied to a region: next to a
// complaint a study title would read like a diagnosis (EXPL-07). The full
// trail stays in plan_decisions.
func reasonFrom(r core.Reason) reasonOut {
	out := reasonOut{RuleID: r.RuleID, Params: nonNil(r.Params), Sources: nonNil(r.Sources), Evidence: r.Evidence,
		Text: r.Text, Args: r.Args, Region: strOrNil(r.Region)}
	if out.Args == nil {
		out.Args = map[string]string{}
	}
	if r.Region != "" {
		out.Sources = []string{}
	}
	return out
}

func reasonsFrom(rs []core.Reason) []reasonOut {
	out := make([]reasonOut, len(rs))
	for i, r := range rs {
		out[i] = reasonFrom(r)
	}
	return out
}

type realismOut struct {
	Skill         string    `json:"skill"`
	TargetLevel   string    `json:"target_level"`
	LowerWeeks    float64   `json:"lower_weeks"`
	ShownFrom     float64   `json:"shown_from_weeks"`
	ShownTo       float64   `json:"shown_to_weeks"`
	WeeksToDate   float64   `json:"weeks_to_date"`
	Unrealistic   bool      `json:"unrealistic"`
	Milestone     *string   `json:"milestone"`
	MilestoneFrom *float64  `json:"milestone_from_weeks"`
	MilestoneTo   *float64  `json:"milestone_to_weeks"`
	Reason        reasonOut `json:"reason"`
}

func realismFrom(rs []core.Realism) []realismOut {
	out := make([]realismOut, len(rs))
	for i, r := range rs {
		out[i] = realismOut{Skill: r.Skill, TargetLevel: r.TargetLevel, LowerWeeks: r.LowerWeeks, ShownFrom: r.ShownFrom,
			ShownTo: r.ShownTo, WeeksToDate: r.WeeksToDate, Unrealistic: r.Unrealistic, Milestone: strOrNil(r.Milestone),
			Reason: reasonFrom(r.Reason)}
		if r.Milestone != "" {
			from, to := r.MilestoneFrom, r.MilestoneTo
			out[i].MilestoneFrom, out[i].MilestoneTo = &from, &to
		}
	}
	return out
}

type redFlagFollowupOut struct {
	ID       string `json:"id"`
	Urgency  string `json:"urgency"`
	Action   string `json:"action"`
	Question string `json:"question"`
	Advice   string `json:"advice"`
}

type redFlagOut struct {
	ID        string              `json:"id"`
	Urgency   string              `json:"urgency"`
	Action    string              `json:"action"`
	Question  string              `json:"question"`
	Advice    string              `json:"advice"`
	MinorOnly bool                `json:"minor_only"`
	Followup  *redFlagFollowupOut `json:"followup"`
}

// redFlagsFrom leaves out the sources: red flags belong to a region
// (EXPL-07).
func redFlagsFrom(fs []core.RedFlag) []redFlagOut {
	out := make([]redFlagOut, len(fs))
	for i, f := range fs {
		out[i] = redFlagOut{ID: f.ID, Urgency: f.Urgency, Action: f.Action, Question: f.Question, Advice: f.Advice,
			MinorOnly: f.MinorOnly}
		if f.Followup != nil {
			out[i].Followup = &redFlagFollowupOut{ID: f.Followup.ID, Urgency: f.Followup.Urgency, Action: f.Followup.Action,
				Question: f.Followup.Question, Advice: f.Followup.Advice}
		}
	}
	return out
}

type planItemOut struct {
	ID           string      `json:"id"`
	Exercise     string      `json:"exercise"`
	ExerciseName string      `json:"exercise_name"`
	Skill        *string     `json:"skill"`
	Stimulus     string      `json:"stimulus"`
	Kind         string      `json:"kind"`
	Sets         int         `json:"sets"`
	Reps         *int        `json:"reps"`
	HoldS        *int        `json:"hold_s"`
	LoadKg       *float64    `json:"load_kg"`
	Assist       string      `json:"assist"`
	Reserve      int         `json:"reserve"`
	RestS        int         `json:"rest_s"`
	Calibration  bool        `json:"calibration"`
	Offer        bool        `json:"offer"`
	StopRules    []string    `json:"stop_rules"`
	Intensity    string      `json:"intensity"`
	Role         string      `json:"role"`
	Monitor      bool        `json:"monitor"`
	Reasons      []reasonOut `json:"reasons"`
}

type planBlockOut struct {
	Role    string        `json:"role"`
	Minutes *float64      `json:"minutes"`
	Paired  bool          `json:"paired"`
	Items   []planItemOut `json:"items"`
	Reasons []reasonOut   `json:"reasons"`
}

type plannedSessionOut struct {
	ID               string         `json:"id"`
	Status           string         `json:"status"`
	WorkoutSessionID *string        `json:"workout_session_id"`
	Index            int            `json:"index"`
	Date             string         `json:"date"`
	Kind             string         `json:"kind"`
	EstMinutes       float64        `json:"est_minutes"`
	Blocks           []planBlockOut `json:"blocks"`
	Reasons          []reasonOut    `json:"reasons"`
}

type exclusionOut struct {
	Exercise     string    `json:"exercise"`
	ExerciseName string    `json:"exercise_name"`
	Reason       reasonOut `json:"reason"`
}

type monitorOut struct {
	Region    string       `json:"region"`
	Questions []redFlagOut `json:"questions"`
}

type accountLoadOut struct {
	Account string  `json:"account"`
	Target  float64 `json:"target"`
	Planned float64 `json:"planned"`
	Cap     float64 `json:"cap"`
	Rule    string  `json:"rule"`
}

type planOut struct {
	ID             string              `json:"id"`
	WeekStart      string              `json:"week_start"`
	RulesetVersion string              `json:"ruleset_version"`
	InputHash      string              `json:"input_hash"`
	Stopped        bool                `json:"stopped"`
	Deload         *string             `json:"deload"`
	Sessions       []plannedSessionOut `json:"sessions"`
	RestDays       []string            `json:"rest_days"`
	Exclusions     []exclusionOut      `json:"exclusions"`
	Hints          []reasonOut         `json:"hints"`
	Realism        []realismOut        `json:"realism"`
	Monitor        []monitorOut        `json:"monitor"`
	Loads          []accountLoadOut    `json:"loads"`
	Reasons        []reasonOut         `json:"reasons"`
	Disclaimer     string              `json:"disclaimer"`
}

// exerciseName names an exercise of the knowledge base; a slug it no longer
// has (a plan of an older ruleset) is shown as the slug.
func exerciseName(k *core.Knowledge, slug string) string {
	if ex, ok := k.Exercise(slug); ok {
		return ex.Name
	}
	return slug
}

func itemFrom(k *core.Knowledge, it core.Item) planItemOut {
	out := planItemOut{ID: it.ID, Exercise: it.Exercise, ExerciseName: exerciseName(k, it.Exercise), Skill: strOrNil(it.Skill),
		Stimulus: it.Stimulus, Kind: it.Kind, Sets: it.Sets, Assist: core.AssistNone, Reserve: it.Reserve, RestS: it.RestS,
		Calibration: it.Calibration, Offer: it.Offer, StopRules: nonNil(it.StopRules), Intensity: it.Intensity(),
		Role: it.Role, Monitor: it.Monitor, Reasons: reasonsFrom(it.Reasons)}
	if it.Reps > 0 {
		reps := it.Reps
		out.Reps = &reps
	}
	if it.HoldS > 0 {
		hold := it.HoldS
		out.HoldS = &hold
	}
	if it.LoadKg > 0 {
		load := it.LoadKg
		out.LoadKg = &load
	}
	if it.Assist != "" {
		out.Assist = it.Assist
	}
	return out
}

func plannedSessionFrom(k *core.Knowledge, s core.PlannedSession) plannedSessionOut {
	out := plannedSessionOut{ID: s.ID, Status: cmp.Or(s.Status, planning.SessionPlanned), WorkoutSessionID: strOrNil(s.WorkoutSessionID),
		Index: s.Index, Date: dateOut(s.Date), Kind: s.Kind, EstMinutes: s.EstMinutes,
		Blocks: make([]planBlockOut, len(s.Blocks)), Reasons: reasonsFrom(s.Reasons)}
	for i, b := range s.Blocks {
		bo := planBlockOut{Role: b.Role, Paired: b.Paired, Items: make([]planItemOut, len(b.Items)), Reasons: reasonsFrom(b.Reasons)}
		if b.Minutes > 0 {
			m := b.Minutes
			bo.Minutes = &m
		}
		for j, it := range b.Items {
			bo.Items[j] = itemFrom(k, it)
		}
		out.Blocks[i] = bo
	}
	return out
}

func planFrom(k *core.Knowledge, p core.Plan) planOut {
	out := planOut{ID: p.ID, WeekStart: dateOut(p.WeekStart), RulesetVersion: p.RulesetVersion, InputHash: p.InputHash,
		Stopped: p.Stopped, Deload: strOrNil(p.Deload), Sessions: make([]plannedSessionOut, len(p.Sessions)),
		RestDays: make([]string, len(p.RestDays)), Exclusions: make([]exclusionOut, len(p.Exclusions)),
		Hints: reasonsFrom(p.Hints), Realism: realismFrom(p.Realism), Monitor: make([]monitorOut, len(p.Monitor)),
		Loads: make([]accountLoadOut, len(p.Loads)), Reasons: reasonsFrom(p.Reasons), Disclaimer: p.Disclaimer}
	for i, s := range p.Sessions {
		out.Sessions[i] = plannedSessionFrom(k, s)
	}
	for i, d := range p.RestDays {
		out.RestDays[i] = dateOut(d)
	}
	for i, e := range p.Exclusions {
		out.Exclusions[i] = exclusionOut{Exercise: e.Exercise, ExerciseName: exerciseName(k, e.Exercise), Reason: reasonFrom(e.Reason)}
	}
	for i, m := range p.Monitor {
		out.Monitor[i] = monitorOut{Region: m.Region, Questions: redFlagsFrom(m.Questions)}
	}
	for i, l := range p.Loads {
		out.Loads[i] = accountLoadOut{Account: l.Account, Target: l.Target, Planned: l.Planned, Cap: l.Cap, Rule: l.Rule}
	}
	return out
}

type plannedSessionDetailOut struct {
	PlanID         string            `json:"plan_id"`
	WeekStart      string            `json:"week_start"`
	RulesetVersion string            `json:"ruleset_version"`
	Session        plannedSessionOut `json:"session"`
	Disclaimer     string            `json:"disclaimer"`
}

type onboardingQuestionOut struct {
	ID    string  `json:"id"`
	Rule  string  `json:"rule"`
	Skill *string `json:"skill"`
	Text  string  `json:"text"`
}

type onboardingResultOut struct {
	Status     string                  `json:"status"`
	Questions  []onboardingQuestionOut `json:"questions"`
	Realism    []realismOut            `json:"realism"`
	Hints      []reasonOut             `json:"hints"`
	Reasons    []reasonOut             `json:"reasons"`
	Disclaimer string                  `json:"disclaimer"`
	Plan       *planOut                `json:"plan,omitempty"`
}

func onboardingResultFrom(k *core.Knowledge, res core.OnboardingResult, p *core.Plan) onboardingResultOut {
	out := onboardingResultOut{Status: res.Status, Questions: make([]onboardingQuestionOut, len(res.Questions)),
		Realism: realismFrom(res.Realism), Hints: reasonsFrom(res.Hints), Reasons: reasonsFrom(res.Reasons),
		Disclaimer: res.Disclaimer}
	for i, q := range res.Questions {
		out.Questions[i] = onboardingQuestionOut{ID: q.ID, Rule: q.Rule, Skill: strOrNil(q.Skill), Text: q.Text}
	}
	if p != nil {
		po := planFrom(k, *p)
		out.Plan = &po
	}
	return out
}

type mobilityOut struct {
	Wrist       string `json:"wrist,omitempty"`
	Shoulder    string `json:"shoulder,omitempty"`
	Ankle       string `json:"ankle,omitempty"`
	Compression string `json:"compression,omitempty"`
}

type profileOut struct {
	BirthYear       int         `json:"birth_year"`
	IsMinor         bool        `json:"is_minor"`
	SessionsPerWeek int         `json:"sessions_per_week"`
	SessionMinutes  int         `json:"session_minutes"`
	Equipment       []string    `json:"equipment"`
	BodyweightKg    float64     `json:"bodyweight_kg"`
	TrainingLevel   string      `json:"training_level"`
	TrainingMonths  float64     `json:"training_months"`
	LastRegular     string      `json:"last_regular_training"`
	HealthConsent   bool        `json:"health_data_consent"`
	DisclaimerAck   bool        `json:"disclaimer_ack"`
	OnboardedAt     string      `json:"onboarded_at"`
	PreferredDays   []int       `json:"preferred_days"`
	Mobility        mobilityOut `json:"mobility_checks"`
	MaxAddedLoadKg  *float64    `json:"max_added_load_kg"`
	SmallestPlateKg *float64    `json:"smallest_plate_kg"`
}

func profileFrom(p core.Profile, minor bool) profileOut {
	out := profileOut{BirthYear: p.BirthYear, IsMinor: minor, SessionsPerWeek: p.SessionsPerWeek, SessionMinutes: p.SessionMinutes,
		Equipment: nonNil(p.Equipment), BodyweightKg: p.BodyweightKg, TrainingLevel: p.TrainingLevel,
		TrainingMonths: p.TrainingMonths, LastRegular: p.LastRegular, HealthConsent: p.HealthConsent,
		DisclaimerAck: p.DisclaimerAck, OnboardedAt: dateOut(p.OnboardedAt), PreferredDays: weekdaysOut(p.PreferredDays),
		Mobility: mobilityOut{Wrist: p.Mobility["wrist"], Shoulder: p.Mobility["shoulder"], Ankle: p.Mobility["ankle"],
			Compression: p.Mobility["compression"]}}
	if p.MaxAddedLoadKg > 0 {
		v := p.MaxAddedLoadKg
		out.MaxAddedLoadKg = &v
	}
	if p.SmallestPlateKg > 0 {
		v := p.SmallestPlateKg
		out.SmallestPlateKg = &v
	}
	return out
}

type goalOut struct {
	Skill       string  `json:"skill"`
	TargetLevel string  `json:"target_level"`
	Priority    int     `json:"priority"`
	TargetDate  *string `json:"target_date"`
}

type goalListOut struct {
	Goals   []goalOut    `json:"goals"`
	Realism []realismOut `json:"realism"`
}

func goalListFrom(goals []core.Goal, realism []core.Realism) goalListOut {
	out := goalListOut{Goals: make([]goalOut, len(goals)), Realism: realismFrom(realism)}
	for i, g := range goals {
		out.Goals[i] = goalOut{Skill: g.Skill, TargetLevel: g.TargetLevel, Priority: g.Priority}
		if g.TargetDate != nil {
			out.Goals[i].TargetDate = dateOrNil(*g.TargetDate)
		}
	}
	return out
}

type changeOut struct {
	Kind    string      `json:"kind"`
	Skill   *string     `json:"skill"`
	Region  *string     `json:"region"`
	Session *string     `json:"session_id"`
	From    *string     `json:"from"`
	To      *string     `json:"to"`
	Reasons []reasonOut `json:"reasons"`
}

func changesFrom(cs []core.Change) []changeOut {
	out := make([]changeOut, len(cs))
	for i, c := range cs {
		out[i] = changeOut{Kind: c.Kind, Skill: strOrNil(c.Skill), Region: strOrNil(c.Region), Session: strOrNil(c.Session),
			From: strOrNil(c.From), To: strOrNil(c.To), Reasons: reasonsFrom(c.Reasons)}
	}
	return out
}

type eventResultOut struct {
	Changes  []changeOut `json:"changes"`
	Replayed bool        `json:"replayed"`
}

func eventResultFrom(o planning.Outcome) eventResultOut {
	return eventResultOut{Changes: changesFrom(o.Changes), Replayed: o.Replayed}
}

type decisionOut struct {
	ID         uuid.UUID   `json:"id"`
	Trigger    string      `json:"trigger"`
	SourceID   string      `json:"source_id"`
	OccurredAt time.Time   `json:"occurred_at"`
	Changes    []changeOut `json:"changes"`
}

type decisionPageOut struct {
	Items      []decisionOut `json:"items"`
	NextCursor *string       `json:"next_cursor"`
}

type painReportOut struct {
	ID          string    `json:"id"`
	Region      string    `json:"region"`
	Timepoint   string    `json:"timepoint"`
	NRS         float64   `json:"nrs"`
	At          time.Time `json:"at"`
	LastedOver  bool      `json:"lasted_over_1h"`
	Persisted   bool      `json:"persisted_over_15min"`
	SuddenSharp bool      `json:"sudden_sharp"`
	SessionID   *string   `json:"session_id"`
}

type painPageOut struct {
	Items      []painReportOut `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}

func painReportFrom(r core.PainReport) painReportOut {
	return painReportOut{ID: r.ID, Region: r.Region, Timepoint: r.Timepoint, NRS: r.NRS, At: r.At.UTC(),
		LastedOver: r.LastedOver, Persisted: r.Persisted, SuddenSharp: r.SuddenSharp, SessionID: strOrNil(r.SessionID)}
}

type regionStatusOut struct {
	Region        string       `json:"region"`
	Name          string       `json:"name"`
	Tracked       bool         `json:"tracked"`
	State         *string      `json:"state"`
	EnteredVia    *string      `json:"entered_via"`
	Since         *string      `json:"since"`
	StartFraction *float64     `json:"start_fraction"`
	Complaint     bool         `json:"complaint"`
	PriorInjury   bool         `json:"prior_injury"`
	Restrictions  []string     `json:"restrictions"`
	Referral      *string      `json:"referral"`
	RestUntil     *string      `json:"rest_until"`
	Constraints   []string     `json:"constraints"`
	RedFlags      []redFlagOut `json:"red_flags"`
}

type regionOverviewOut struct {
	Regions    []regionStatusOut `json:"regions"`
	Disclaimer string            `json:"disclaimer"`
}

func regionOverviewFrom(k *core.Knowledge, st []core.RegionStatus) regionOverviewOut {
	out := regionOverviewOut{Regions: make([]regionStatusOut, len(st)), Disclaimer: k.Disclaimer()}
	for i, s := range st {
		ro := regionStatusOut{Region: s.Region.ID, Name: s.Region.Name, Tracked: s.Tracked, Restrictions: []string{},
			Constraints: nonNil(s.Constraints), RedFlags: redFlagsFrom(s.RedFlags)}
		if s.Tracked {
			rs := s.State
			ro.State, ro.EnteredVia, ro.Since = strOrNil(rs.State), strOrNil(rs.EnteredVia), dateOrNil(rs.Since)
			ro.Complaint, ro.PriorInjury, ro.Referral = rs.Complaint, rs.PriorInjury, strOrNil(rs.Referral)
			ro.Restrictions, ro.RestUntil = nonNil(rs.Restrictions), dateOrNil(rs.RestUntil)
			if rs.StartFraction > 0 {
				f := rs.StartFraction
				ro.StartFraction = &f
			}
		}
		out.Regions[i] = ro
	}
	return out
}

type capacityOut struct {
	Exercise     string    `json:"exercise"`
	ExerciseName string    `json:"exercise_name"`
	Measure      string    `json:"measure"`
	Assist       string    `json:"assist"`
	Mu           float64   `json:"mu"`
	Sigma        float64   `json:"sigma"`
	Confidence   string    `json:"confidence"`
	Origin       string    `json:"origin"`
	EstimatedAt  time.Time `json:"estimated_at"`
	Observations int       `json:"observations"`
}

type capacityListOut struct {
	Items []capacityOut `json:"items"`
}

func capacitiesFrom(k *core.Knowledge, caps map[string]core.Estimate) capacityListOut {
	keys := make([]string, 0, len(caps))
	for key := range caps {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := capacityListOut{Items: make([]capacityOut, 0, len(keys))}
	for _, key := range keys {
		e := caps[key]
		slug, assist, _ := strings.Cut(key, "|")
		measure := core.MeasureReps
		if ex, ok := k.Exercise(slug); ok {
			measure = ex.Measure
		}
		out.Items = append(out.Items, capacityOut{Exercise: slug, ExerciseName: exerciseName(k, slug), Measure: measure,
			Assist: assist, Mu: e.Mu, Sigma: e.Sigma, Confidence: k.Confidence(e), Origin: e.Origin,
			EstimatedAt: e.At.UTC(), Observations: e.N})
	}
	return out
}

// --------------------------------------------------------------- catalogue

type plannerRuleOut struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Params   []string `json:"params"`
	Sources  []string `json:"sources"`
	Evidence string   `json:"evidence"`
}

type plannerRuleListOut struct {
	RulesetVersion string           `json:"ruleset_version"`
	Items          []plannerRuleOut `json:"items"`
}

func rulesFrom(k *core.Knowledge) plannerRuleListOut {
	rules := k.Rules()
	out := plannerRuleListOut{RulesetVersion: k.Version, Items: make([]plannerRuleOut, len(rules))}
	for i, r := range rules {
		out.Items[i] = plannerRuleOut{ID: r.ID, Title: r.Title, Text: r.Text, Params: nonNil(r.Params),
			Sources: nonNil(r.Sources), Evidence: r.Evidence}
	}
	return out
}

type plannerSourceOut struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Authors  *string `json:"authors"`
	Year     *string `json:"year"`
	URL      *string `json:"url"`
	Type     *string `json:"type"`
	Evidence string  `json:"evidence"`
}

type plannerSourceListOut struct {
	RulesetVersion string             `json:"ruleset_version"`
	Items          []plannerSourceOut `json:"items"`
}

func sourcesFrom(k *core.Knowledge) plannerSourceListOut {
	sources := k.Sources()
	out := plannerSourceListOut{RulesetVersion: k.Version, Items: make([]plannerSourceOut, len(sources))}
	for i, s := range sources {
		out.Items[i] = plannerSourceOut{ID: s.ID, Title: s.Title, Authors: strOrNil(s.Authors), Year: strOrNil(s.Year),
			URL: strOrNil(s.URL), Type: strOrNil(s.Type), Evidence: s.Evidence}
	}
	return out
}

type plannerParameterOut struct {
	ID         string             `json:"id"`
	Key        string             `json:"key"`
	Text       *string            `json:"text"`
	Value      *float64           `json:"value"`
	Values     map[string]float64 `json:"values"`
	Unit       *string            `json:"unit"`
	Sources    []string           `json:"sources"`
	Evidence   string             `json:"evidence"`
	Heuristic  bool               `json:"heuristic"`
	Definition bool               `json:"definition"`
	Rationale  *string            `json:"rationale"`
}

type plannerParameterListOut struct {
	RulesetVersion string                `json:"ruleset_version"`
	Items          []plannerParameterOut `json:"items"`
}

func parametersFrom(k *core.Knowledge) plannerParameterListOut {
	params := k.Params()
	out := plannerParameterListOut{RulesetVersion: k.Version, Items: make([]plannerParameterOut, len(params))}
	for i, p := range params {
		out.Items[i] = plannerParameterOut{ID: p.ID, Key: p.Key, Text: strOrNil(p.Text), Value: p.Value, Values: p.Values,
			Unit: strOrNil(p.Unit), Sources: nonNil(p.Sources), Evidence: p.Evidence, Heuristic: p.Heuristic,
			Definition: p.Definition, Rationale: strOrNil(p.Rationale)}
	}
	return out
}

type answerClassOut struct {
	Key  string   `json:"key"`
	Lo   float64  `json:"lo"`
	Hi   *float64 `json:"hi"`
	Open bool     `json:"open"`
}

func classesFrom(cs []core.AnswerClass) []answerClassOut {
	out := make([]answerClassOut, len(cs))
	for i, c := range cs {
		out[i] = answerClassOut{Key: c.Key, Lo: c.Lo, Open: c.Open}
		if !c.Open {
			hi := c.Hi
			out[i].Hi = &hi
		}
	}
	return out
}

type classQuestionOut struct {
	Key      string           `json:"key"`
	Exercise string           `json:"exercise"`
	Classes  []answerClassOut `json:"classes"`
}

type plannerLevelOut struct {
	Slug      string  `json:"slug"`
	Exercise  string  `json:"exercise"`
	Threshold float64 `json:"threshold"`
	Milestone bool    `json:"milestone"`
}

type plannerSkillOut struct {
	Slug         string            `json:"slug"`
	Name         string            `json:"name"`
	Foundation   bool              `json:"foundation"`
	Levels       []plannerLevelOut `json:"levels"`
	StageClasses []answerClassOut  `json:"stage_classes"`
}

type plannerExerciseOut struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	Measure  string   `json:"measure"`
	Requires []string `json:"requires"`
	Skill    *string  `json:"skill"`
	Draft    bool     `json:"draft"`
}

type plannerRegionOut struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	RedFlags []redFlagOut `json:"red_flags"`
}

type plannerCatalogueOut struct {
	RulesetVersion string               `json:"ruleset_version"`
	Disclaimer     string               `json:"disclaimer"`
	Skills         []plannerSkillOut    `json:"skills"`
	Exercises      []plannerExerciseOut `json:"exercises"`
	Regions        []plannerRegionOut   `json:"regions"`
	ClassQuestions []classQuestionOut   `json:"class_questions"`
}

// catalogueFrom lists every red-flag question of a region, the ones for
// minors marked; the client asks those only of minors.
func catalogueFrom(k *core.Knowledge) plannerCatalogueOut {
	ob := k.Onboarding()
	stages := map[string][]core.AnswerClass{}
	for _, s := range ob.Stages {
		stages[s.Skill] = s.Classes
	}
	out := plannerCatalogueOut{RulesetVersion: k.Version, Disclaimer: k.Disclaimer()}
	for _, s := range k.Skills() {
		so := plannerSkillOut{Slug: s.Slug, Name: s.Name, Foundation: s.Foundation, Levels: make([]plannerLevelOut, len(s.Levels)),
			StageClasses: classesFrom(stages[s.Slug])}
		for i, l := range s.Levels {
			so.Levels[i] = plannerLevelOut{Slug: l.Slug, Exercise: l.Exercise, Threshold: l.Threshold, Milestone: l.Milestone}
		}
		out.Skills = append(out.Skills, so)
	}
	for _, e := range k.Exercises() {
		out.Exercises = append(out.Exercises, plannerExerciseOut{Slug: e.Slug, Name: e.Name, Measure: e.Measure,
			Requires: nonNil(e.Requires), Skill: strOrNil(e.Skill), Draft: e.Status == "draft_placeholder"})
	}
	for _, r := range k.Regions() {
		out.Regions = append(out.Regions, plannerRegionOut{ID: r.ID, Name: r.Name, RedFlags: redFlagsFrom(k.RedFlags(r.ID, true))})
	}
	for _, q := range ob.Questions {
		out.ClassQuestions = append(out.ClassQuestions, classQuestionOut{Key: q.Key, Exercise: q.Exercise, Classes: classesFrom(q.Classes)})
	}
	out.Skills, out.Exercises, out.Regions, out.ClassQuestions = nonNil(out.Skills), nonNil(out.Exercises), nonNil(out.Regions),
		nonNil(out.ClassQuestions)
	return out
}
