package planning

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Answers is the onboarding as the client sends it (onboarding.md §3).
type Answers struct {
	BirthYear        int                        `json:"birth_year"`
	HealthConsent    bool                       `json:"health_data_consent"`
	DisclaimerAck    bool                       `json:"disclaimer_ack"`
	Goals            []Goal                     `json:"goals"`
	SessionsPerWeek  int                        `json:"sessions_per_week"`
	SessionMinutes   int                        `json:"session_minutes"`
	Equipment        []string                   `json:"equipment"`
	BodyweightKg     float64                    `json:"bodyweight_kg"`
	TrainingLevel    string                     `json:"training_level"`
	TrainingAge      string                     `json:"calisthenics_training_age"`
	LastRegular      string                     `json:"last_regular_training"`
	PreBreak         map[string]string          `json:"pre_break_level,omitempty"` // skill → level slug
	Classes          map[string]string          `json:"classes"`                   // question key → class key
	Stages           map[string]StageAnswer     `json:"skill_stages,omitempty"`    // skill → stage
	Mobility         map[string]string          `json:"mobility_checks,omitempty"`
	DataConfidence   string                     `json:"data_confidence"`
	ExertionSymptoms bool                       `json:"exertion_symptoms"`
	Screening        []bool                     `json:"screening"`
	BodyMap          map[string]BodyMapEntry    `json:"body_map,omitempty"`
	Complaints       map[string]Complaint       `json:"complaints,omitempty"`
	RedFlags         map[string]map[string]bool `json:"red_flags,omitempty"` // region → flag → yes
	Clarifications   map[string]bool            `json:"clarifications,omitempty"`
	PreferredDays    []time.Weekday             `json:"preferred_days,omitempty"`
}

// StageAnswer is the stage a user reports for a goal skill.
type StageAnswer struct {
	Level string `json:"level"` // level slug, "none" or "unknown"
	Class string `json:"class"` // hold or rep class key
}

// BodyMapEntry marks a region as currently affected and/or injured in the
// last 12 months.
type BodyMapEntry struct {
	Current bool `json:"current"`
	Past12  bool `json:"past_12_months"`
}

// Complaint details a current complaint (onboarding §3.7).
type Complaint struct {
	PainDaily     float64  `json:"pain_daily"`
	PainTraining  float64  `json:"pain_training"`
	Onset         string   `json:"onset"`
	DurationWeeks float64  `json:"duration_weeks"`
	Suspected     string   `json:"suspected_serious"` // yes, no, unsure
	Assessment    string   `json:"professional_assessment"`
	Restrictions  []string `json:"restrictions,omitempty"`
}

// Question is a clarification the onboarding asks (at most two, §5.5).
type Question struct {
	ID    string `json:"id"`
	Rule  string `json:"rule"`
	Skill string `json:"skill,omitempty"`
	Text  string `json:"text"`
}

// OnboardingResult is what Start returns besides the snapshot.
type OnboardingResult struct {
	Status    string     `json:"status"` // needs_answers, blocked, complete
	Questions []Question `json:"questions,omitempty"`
	Realism   []Realism  `json:"realism,omitempty"`
	Hints     []Reason   `json:"hints,omitempty"`
	Reasons   []Reason   `json:"reasons,omitempty"`
	// Disclaimer travels with every result, because red-flag advice and
	// referral hints may be part of it (spec §9.3).
	Disclaimer string `json:"disclaimer"`
}

// Onboarding statuses.
const (
	OnboardingNeedsAnswers = "needs_answers"
	OnboardingBlocked      = "blocked"
	OnboardingComplete     = "complete"
)

// ErrInvalidAnswers wraps validation problems in the answers.
var ErrInvalidAnswers = errors.New("invalid onboarding answers")

var equipmentImplied = map[string][]string{
	"outdoor_park": {"pull_up_bar", "low_bar", "parallel_bars"},
	"gym":          {"pull_up_bar", "parallel_bars", "dumbbells_or_plates", "box_or_bench"},
}

var trainingMonths = map[string]float64{
	"lt_6_months": 0, "6_to_12_months": 6, "1_to_4_years": 12, "gt_4_years": 48,
}

// Start turns onboarding answers into the start state (onboarding §7).
func Start(k *Knowledge, a Answers, now time.Time) (Snapshot, OnboardingResult, error) {
	if err := validateAnswers(k, a, now); err != nil {
		return Snapshot{}, OnboardingResult{}, err
	}
	today := civil(now, time.UTC)
	s := Snapshot{
		Goals:      slices.Clone(a.Goals),
		Capacities: map[string]Estimate{},
		Ladders:    map[string]LadderState{},
		Regions:    map[string]RegionState{},
		Unlocked:   map[string]bool{},
		Headroom:   map[string]float64{},
		Entry:      map[string]bool{},
		Phase:      Phase{MesoStart: weekStart(today)},
	}
	slices.SortFunc(s.Goals, func(x, y Goal) int { return x.Priority - y.Priority })
	s.Profile = Profile{
		BirthYear:       a.BirthYear,
		SessionsPerWeek: a.SessionsPerWeek,
		SessionMinutes:  a.SessionMinutes,
		Equipment:       expandEquipment(a.Equipment),
		BodyweightKg:    a.BodyweightKg,
		TrainingLevel:   a.TrainingLevel,
		TrainingMonths:  trainingMonths[a.TrainingAge],
		LastRegular:     a.LastRegular,
		HealthConsent:   a.HealthConsent,
		DisclaimerAck:   a.DisclaimerAck,
		OnboardedAt:     today,
		PreferredDays:   slices.Clone(a.PreferredDays),
		Mobility:        cloneMap(a.Mobility),
	}
	res := OnboardingResult{Status: OnboardingComplete, Disclaimer: k.Disclaimer()}

	// Safety first (O-4): exertion symptoms, screening, red flags.
	s.Screening = Screening{ExertionSymptoms: a.ExertionSymptoms, AnyYes: slices.Contains(a.Screening, true)}
	if a.ExertionSymptoms {
		s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintStopped, Created: today})
		res.Reasons = append(res.Reasons, k.reason(RuleStopped))
	}
	minor := isMinor(k, a.BirthYear, now)
	k.startRegions(&s, a, minor, today, &res)
	if hasConstraint(s, ConstraintStopped) {
		res.Status = OnboardingBlocked
	}

	// Capacities from the answer classes.
	widen := a.LastRegular == "7_to_16_weeks" || a.LastRegular == "17_to_26_weeks" || a.LastRegular == "gt_26_weeks"
	unknown := map[string]bool{} // capacities from "don't know" answers
	for _, q := range k.onboarding.Questions {
		key, ok := a.Classes[q.Key]
		if !ok {
			continue
		}
		if key == "unknown" {
			unknown[CapKey(q.Exercise, AssistNone)] = true
		}
		ex := k.exercises[q.Exercise]
		e, ok := k.classPrior(ex, q.Classes, key, a.DataConfidence, today)
		if !ok {
			continue
		}
		s.Capacities[CapKey(ex.Slug, AssistNone)] = e
	}

	// Stages of goal skills, with plausibility checks (onboarding §5.5). At
	// most two questions; an unanswered or declined question resolves
	// conservatively, so the start state is safe even before the answers.
	var questions []Question
	for _, g := range s.Goals {
		st, ok := a.Stages[g.Skill]
		if !ok || st.Level == "none" || st.Level == "unknown" || st.Level == "" {
			continue
		}
		sk := k.skills[g.Skill]
		claimedIdx := slices.IndexFunc(sk.Levels, func(l Level) bool { return l.Slug == st.Level })
		if claimedIdx < 0 {
			continue
		}
		if q, rule := k.plausibility(s, a, sk, claimedIdx); q != nil {
			yes, answered := a.Clarifications[q.ID]
			if !answered && len(questions) < 2 {
				questions = append(questions, *q)
			}
			if rule == "R-3" {
				widen = true
			}
			if !answered || !yes {
				if rule == "R-3" {
					claimedIdx--
				}
				claimedIdx = k.plausibleLevel(s, sk, claimedIdx)
				widen = true
				res.Reasons = append(res.Reasons, k.reason(RuleEntryRung, "skill", sk.Name))
			}
		}
		if claimedIdx < 0 {
			continue
		}
		lvl := &sk.Levels[claimedIdx]
		ex := k.exercises[lvl.Exercise]
		for _, sq := range k.onboarding.Stages {
			if sq.Skill != g.Skill {
				continue
			}
			if e, ok := k.classPrior(ex, sq.Classes, st.Class, a.DataConfidence, today); ok {
				s.Capacities[CapKey(ex.Slug, AssistNone)] = e
				unknown[CapKey(ex.Slug, AssistNone)] = st.Class == "unknown"
			}
		}
		s.Ladders[g.Skill] = LadderState{Claimed: ex.Slug, Status: StatusClaimed, Since: today}
	}
	if len(questions) > 2 {
		questions = questions[:2]
	}
	if len(questions) > 0 && res.Status != OnboardingBlocked {
		res.Status = OnboardingNeedsAnswers
		res.Questions = questions
	}
	if widen {
		for key, e := range s.Capacities {
			e.Sigma = float64(e.Sigma * k.T.Widening)
			s.Capacities[key] = e
		}
	}

	// Pause from the onboarding (spec §6.11, PAR-S-41).
	if days, ok := breakDays(k, a.LastRegular); ok && days > 0 {
		s.Break = &BreakState{Days: days, StraightDays: days, Since: weekStart(today), StepSince: today}
		for skill, lvl := range a.PreBreak {
			if l := k.level(skill + "/" + lvl); l != nil {
				ls := s.Ladders[skill]
				ls.CapRung = l.Exercise
				if ls.Claimed == "" {
					ls.Claimed, ls.Status, ls.Since = l.Exercise, StatusClaimed, today
				}
				s.Ladders[skill] = ls
			}
		}
	}

	// Entry ramp for current trainers (LOAD-04b): only a stated capacity
	// qualifies, not a "don't know" (spec §7.4).
	if a.LastRegular == "current_or_lt_3_weeks" && !minor {
		for key, e := range s.Capacities {
			if e.Mu <= 0 || unknown[key] {
				continue
			}
			ex := k.exercises[strings.SplitN(key, "|", 2)[0]]
			for acc := range k.setLoad(ex, KindWorking, 0, s.Profile.BodyweightKg) {
				if !accountHasComplaint(k, s, acc) {
					s.Entry[acc] = true
				}
			}
		}
	}

	// Realism for dated goals (GOAL-05).
	for _, g := range s.Goals {
		if g.TargetDate != nil {
			if r, ok := k.RealismFor(s, g, now); ok {
				res.Realism = append(res.Realism, r)
			}
		}
	}
	if s.Profile.HealthConsent {
		return s, res, nil
	}
	// Without consent nothing health-related is kept beyond the constraints.
	s.Regions = map[string]RegionState{}
	s.Screening = Screening{}
	res.Reasons = append(res.Reasons, k.reason(RuleNoConsent))
	return s, res, nil
}

func validateAnswers(k *Knowledge, a Answers, now time.Time) error {
	var probs []string
	if !a.DisclaimerAck {
		probs = append(probs, "disclaimer_ack must be confirmed")
	}
	if a.BirthYear < 1920 || a.BirthYear > now.Year() {
		probs = append(probs, "birth_year out of range")
	}
	if len(a.Goals) < 1 || len(a.Goals) > 3 {
		probs = append(probs, "1 to 3 goals are required")
	}
	seen := map[int]bool{}
	for _, g := range a.Goals {
		if k.level(g.Skill+"/"+g.TargetLevel) == nil {
			probs = append(probs, fmt.Sprintf("unknown goal %s/%s", g.Skill, g.TargetLevel))
		}
		if g.Priority < 1 || g.Priority > len(a.Goals) || seen[g.Priority] {
			probs = append(probs, "goal priorities must be 1..n without gaps")
		}
		seen[g.Priority] = true
	}
	if a.SessionsPerWeek < 1 || a.SessionsPerWeek > 7 {
		probs = append(probs, "sessions_per_week must be 1–7")
	}
	if !slices.Contains([]int{20, 30, 45, 60, 75, 90}, a.SessionMinutes) {
		probs = append(probs, "session_minutes must be 20, 30, 45, 60, 75 or 90")
	}
	if a.BodyweightKg < 30 || a.BodyweightKg > 250 {
		probs = append(probs, "bodyweight_kg must be 30–250")
	}
	if len(probs) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidAnswers, strings.Join(probs, "; "))
	}
	return nil
}

func expandEquipment(in []string) []string {
	set := map[string]bool{"floor": true, "wall": true}
	for _, e := range in {
		set[e] = true
		for _, imp := range equipmentImplied[e] {
			set[imp] = true
		}
	}
	return sortedKeys(set)
}

func isMinor(k *Knowledge, birthYear int, now time.Time) bool {
	return float64(now.Year()-birthYear) <= k.T.MinorAge
}

func hasConstraint(s Snapshot, kind string) bool {
	return slices.ContainsFunc(s.Constraints, func(c Constraint) bool { return c.Kind == kind })
}

func accountHasComplaint(k *Knowledge, s Snapshot, acc string) bool {
	for _, r := range k.regionsOf(acc) {
		if rs, ok := s.Regions[r]; ok && (rs.Complaint || rs.PriorInjury || rs.State != StateNormal) {
			return true
		}
	}
	return false
}

// classPrior returns the prior for an answer class, with "unknown" treated
// as the lowest non-zero class and a wide σ (PAR-S-45).
func (k *Knowledge) classPrior(ex *Exercise, classes []AnswerClass, key, confidence string, at time.Time) (Estimate, bool) {
	if key == "unknown" {
		for _, c := range classes {
			if c.Hi > 0 || c.Open {
				e := k.selfReport(ex, c, "estimated", at)
				e.Sigma = math.Max(e.Sigma, float64(k.T.UnknownFrac*e.Mu))
				return e, true
			}
		}
		return Estimate{}, false
	}
	for _, c := range classes {
		if c.Key == key {
			return k.selfReport(ex, c, confidence, at), true
		}
	}
	return Estimate{}, false
}

// plausibility checks a claimed stage (R-2, R-3). It returns the question to
// ask and the rule that raised it.
func (k *Knowledge) plausibility(s Snapshot, a Answers, sk *Skill, idx int) (*Question, string) {
	if idx < 0 {
		return nil, ""
	}
	lvl := sk.Levels[idx]
	name := k.exercises[lvl.Exercise].Name
	// R-3: sedentary or recreational with an advanced stage (OG ≥ 6).
	if (a.TrainingLevel == LevelSedentary || a.TrainingLevel == LevelRecreational) && lvl.OG >= k.T.IntermediateOG {
		return &Question{ID: "R-3/" + sk.Slug, Rule: "R-3", Skill: sk.Slug,
			Text: fmt.Sprintf("Hältst du die Stufe «%s» ohne Band und mit gestreckten Armen?", name)}, "R-3"
	}
	// R-2: a stage whose foundations look implausible.
	if !k.foundationsPlausible(s, sk, idx) {
		prev := sk.Levels[max(idx-1, 0)]
		return &Question{ID: "R-2/" + sk.Slug, Rule: "R-2", Skill: sk.Slug,
			Text: fmt.Sprintf("Kannst du die Stufe «%s» sauber halten bzw. ausführen?", k.exercises[prev.Exercise].Name)}, "R-2"
	}
	return nil, ""
}

// foundationsPlausible reports whether every hard prerequisite outside the
// skill's own chain has an estimate of at least PAR-S-46 × its threshold.
// A prerequisite without an estimate counts as plausible (nothing to
// contradict).
func (k *Knowledge) foundationsPlausible(s Snapshot, sk *Skill, idx int) bool {
	for _, p := range k.allPrereqs(sk.Slug + "/" + sk.Levels[idx].Slug) {
		pl := k.level(p)
		if pl == nil {
			continue
		}
		if e, ok := s.Capacities[CapKey(pl.Exercise, AssistNone)]; ok && e.Mu < float64(k.T.PlausibleFrac*pl.Threshold) {
			return false
		}
	}
	return true
}

// plausibleLevel is the conservative resolution of R-2: the highest level at
// or below idx whose foundations are plausible, or -1.
func (k *Knowledge) plausibleLevel(s Snapshot, sk *Skill, idx int) int {
	for ; idx >= 0; idx-- {
		if k.foundationsPlausible(s, sk, idx) {
			return idx
		}
	}
	return -1
}

// allPrereqs returns every hard prerequisite reachable from a level,
// excluding the implicit chain of its own skill.
func (k *Knowledge) allPrereqs(ref string) []string {
	seen := map[string]bool{}
	var walk func(r string)
	walk = func(r string) {
		l := k.level(r)
		if l == nil {
			return
		}
		for _, p := range l.Prerequisites {
			if !seen[p] {
				seen[p] = true
				walk(p)
			}
		}
		for _, p := range k.prereqs(r) {
			if strings.HasPrefix(p, strings.SplitN(r, "/", 2)[0]+"/") {
				walk(p)
			}
		}
	}
	walk(ref)
	return sortedKeys(seen)
}

// breakDays maps last_regular_training to a pause length (PAR-S-41).
func breakDays(k *Knowledge, class string) (float64, bool) {
	switch class {
	case "current_or_lt_3_weeks", "never", "":
		return 0, true
	case "3_to_6_weeks":
		return k.T.BreakMonth, true
	case "7_to_16_weeks":
		return k.T.BreakQuarter, true
	case "17_to_26_weeks", "gt_26_weeks":
		return k.T.BreakHalf, true
	}
	return 0, false
}

// startRegions applies the body map, complaints and red flags (onboarding
// §3.7, spec §8.2–8.3).
func (k *Knowledge) startRegions(s *Snapshot, a Answers, minor bool, today time.Time, res *OnboardingResult) {
	for _, id := range sortedKeys(a.BodyMap) {
		bm := a.BodyMap[id]
		if _, ok := k.regions[id]; !ok {
			continue
		}
		rs := RegionState{State: StateNormal, Since: today, StepSince: today, PriorInjury: bm.Past12}
		if !bm.Current {
			s.Regions[id] = rs
			continue
		}
		c := a.Complaints[id]
		rs.Complaint, rs.ComplaintAt, rs.EnteredVia = true, today, "onboarding"
		rs.Restrictions = slices.Clone(c.Restrictions)
		out := k.EvaluateRedFlags(id, a.RedFlags[id], minor)
		serious := c.Suspected == "yes" || c.Assessment == "yes_tear_or_suspected_tear"
		unsure := c.Suspected == "unsure" || c.Assessment == "in_progress"
		switch {
		case out.Stop:
			rs.State = StateLocked
			s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintStopped, Region: id, Created: today},
				Constraint{Kind: ConstraintLocked, Region: id, Created: today})
		case out.Lock || len(k.regions[id].Structures) == 0 && (out.Urgency != "" || serious):
			rs.State = StateLocked
			s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintLocked, Region: id, Created: today})
			if len(k.regions[id].Structures) == 0 {
				s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintStopped, Region: id, Created: today})
			}
		case serious:
			rs.State = StateLocked
			s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintLocked, Region: id, Created: today})
		case out.RTT0 || unsure || c.PainDaily > k.T.PainGreen:
			rs.State = StateRTT0
		default:
			rs.State = StateRTT1
			rs.StartFraction = k.T.RTTStart
			// After a professional assessment, or with training pain above
			// the acceptable limit, the ramp starts at a quarter (PAR-D-33,
			// PAR-D-15).
			if c.Assessment == "yes_overuse_or_tendon" || c.Assessment == "yes_other" || c.PainTraining > k.T.PainAccept {
				rs.StartFraction = k.T.RTTStartReferral
			}
			rs.Step = k.startStep(rs.StartFraction)
		}
		// The daily pain is the first baseline of the monitoring (PAR-D-13,
		// PAR-D-16); stored only with consent to health data.
		if a.HealthConsent {
			s.Pain = append(s.Pain, PainReport{Region: id, Timepoint: PainDaily, NRS: c.PainDaily, At: today})
		}
		if c.DurationWeeks > 4 {
			rs.Referral = "advise"
			res.Hints = append(res.Hints, regional(k.reason(RuleReferral, "region", k.regions[id].Name), id))
		}
		res.Reasons = append(res.Reasons, out.Reasons...)
		res.Reasons = append(res.Reasons, regional(k.reason(RuleRegionState, "region", k.regions[id].Name, "state", rs.State), id))
		if !a.HealthConsent && rs.State != StateLocked && !out.Stop {
			s.Constraints = append(s.Constraints, Constraint{Kind: ConstraintExcluded, Region: id, Created: today})
		}
		s.Regions[id] = rs
	}
}
