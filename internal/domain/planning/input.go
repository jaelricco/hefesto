package planning

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// ErrInvalidAnswers wraps validation problems in the answers.
var ErrInvalidAnswers = errors.New("invalid onboarding answers")

// ValidationError lists the invalid fields of an input, keyed by a JSON
// pointer into it (RFC 6901). It wraps ErrInvalidAnswers.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	keys := sortedKeys(e.Fields)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + e.Fields[k]
	}
	return ErrInvalidAnswers.Error() + ": " + strings.Join(parts, "; ")
}

func (e *ValidationError) Unwrap() error { return ErrInvalidAnswers }

// fields collects field errors; the first message for a pointer wins.
type fields map[string]string

func (f fields) add(ptr, msg string) {
	if _, ok := f[ptr]; !ok {
		f[ptr] = msg
	}
}

func (f fields) err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}

// token escapes a map key for a JSON pointer.
func token(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

// Closed vocabularies of the answers (onboarding.md §3). Skills, levels,
// answer classes, regions and red flags come from the knowledge base.
var (
	equipmentKinds = []string{"floor", "wall", "pull_up_bar", "low_bar", "parallel_bars", "parallettes", "rings",
		"resistance_bands", "weight_vest", "dumbbells_or_plates", "dip_belt", "box_or_bench", "gym", "outdoor_park"}
	trainingLevels   = []string{LevelSedentary, LevelRecreational, LevelTrained, LevelHighlyTrained}
	lastRegular      = []string{"current_or_lt_3_weeks", "3_to_6_weeks", "7_to_16_weeks", "17_to_26_weeks", "gt_26_weeks", "never"}
	dataConfidences  = []string{"estimated", "counted_last_4_weeks", "filmed"}
	sessionLengths   = []int{20, 30, 45, 60, 75, 90}
	mobilityChecks   = []string{"wrist", "shoulder", "ankle", "compression"}
	mobilityAnswers  = []string{"yes", "partly", "no"}
	onsets           = []string{"sudden", "gradual"}
	suspicions       = []string{"yes", "no", "unsure"}
	assessments      = []string{"no", "yes_overuse_or_tendon", "yes_tear_or_suspected_tear", "yes_other", "in_progress"}
	restrictionTags  = []string{"support_straight_arm", "hang_pull", "overhead", "wrist_extension_loaded", "supination_loaded", "spine_extension"}
	painTimepoints   = []string{PainBefore, PainWarmup, PainDuring, PainAfter, PainMorning, PainDaily}
	screeningAnswers = 6
)

// Vocabularies returns the closed vocabularies of the answers by field
// name, for the API contract to be checked against.
func Vocabularies() map[string][]string {
	return map[string][]string{
		"equipment":                 slices.Clone(equipmentKinds),
		"training_level":            slices.Clone(trainingLevels),
		"calisthenics_training_age": sortedKeys(trainingMonths),
		"last_regular_training":     slices.Clone(lastRegular),
		"data_confidence":           slices.Clone(dataConfidences),
		"mobility_checks":           slices.Clone(mobilityChecks),
		"mobility_answer":           slices.Clone(mobilityAnswers),
		"onset":                     slices.Clone(onsets),
		"suspected_serious":         slices.Clone(suspicions),
		"professional_assessment":   slices.Clone(assessments),
		"restrictions":              slices.Clone(restrictionTags),
		"timepoint":                 slices.Clone(painTimepoints),
	}
}

// Bounds of the numeric answers (onboarding.md §3.3, §3.5, §3.8). The plate
// bound is a plausibility limit, not a training rule.
const (
	minBirthYear   = 1920
	minBodyweight  = 30
	maxBodyweight  = 250
	maxAddedLoadKg = 100
	maxPlateKg     = 25
	maxPainNRS     = 10
	// maxAhead bounds how far a client's clock may run ahead of the
	// server's; a plausibility limit, not a training rule.
	maxAhead = 24 * time.Hour
)

func validateAnswers(k *Knowledge, a Answers, now time.Time) error {
	f := fields{}
	if !a.DisclaimerAck {
		f.add("/disclaimer_ack", "must be confirmed")
	}
	if a.BirthYear < minBirthYear || a.BirthYear > now.Year() {
		f.add("/birth_year", fmt.Sprintf("must be %d–%d", minBirthYear, now.Year()))
	}
	k.checkGoals(f, a.Goals, now)
	checkAvailability(f, a.SessionsPerWeek, a.SessionMinutes, a.BodyweightKg, a.Equipment, a.PreferredDays)
	checkOneOf(f, "/training_level", a.TrainingLevel, trainingLevels)
	checkOneOf(f, "/calisthenics_training_age", a.TrainingAge, sortedKeys(trainingMonths))
	checkOneOf(f, "/last_regular_training", a.LastRegular, lastRegular)
	checkOneOf(f, "/data_confidence", a.DataConfidence, dataConfidences)
	k.checkClasses(f, a)
	for _, skill := range sortedKeys(a.PreBreak) {
		if k.level(skill+"/"+a.PreBreak[skill]) == nil {
			f.add("/pre_break_level/"+token(skill), "unknown skill or level")
		}
	}
	checkMobility(f, a.Mobility)
	if len(a.Screening) != screeningAnswers {
		f.add("/screening", fmt.Sprintf("all %d questions must be answered", screeningAnswers))
	}
	k.checkBody(f, a, isMinor(k, a.BirthYear, now))
	for _, id := range sortedKeys(a.Clarifications) {
		rule, skill, _ := strings.Cut(id, "/")
		if rule != "R-2" && rule != "R-3" || k.skills[skill] == nil {
			f.add("/clarifications/"+token(id), "not a question the onboarding asks")
		}
	}
	return f.err()
}

// checkGoals checks one to three goals with priorities 1 to n and known
// target levels (onboarding.md §3.2).
func (k *Knowledge) checkGoals(f fields, goals []Goal, now time.Time) {
	if len(goals) < 1 || len(goals) > 3 {
		f.add("/goals", "1 to 3 goals are required")
	}
	today := civil(now, time.UTC)
	skills, prios := map[string]bool{}, map[int]bool{}
	for i, g := range goals {
		ptr := fmt.Sprintf("/goals/%d", i)
		switch {
		case k.skills[g.Skill] == nil:
			f.add(ptr+"/skill", "unknown skill")
		case skills[g.Skill]:
			f.add(ptr+"/skill", "skill is already a goal")
		case k.level(g.Skill+"/"+g.TargetLevel) == nil:
			f.add(ptr+"/target_level", "not a level of this skill")
		}
		skills[g.Skill] = true
		if g.Priority < 1 || g.Priority > len(goals) || prios[g.Priority] {
			f.add(ptr+"/priority", "priorities must be 1 to n, each once")
		}
		prios[g.Priority] = true
		if g.TargetDate != nil && g.TargetDate.Before(today) {
			f.add(ptr+"/target_date", "must not be in the past")
		}
	}
}

// checkAvailability checks the fields a user may change later as well.
func checkAvailability(f fields, sessions, minutes int, bodyweight float64, equipment []string, days []time.Weekday) {
	if sessions < 1 || sessions > 7 {
		f.add("/sessions_per_week", "must be 1–7")
	}
	if !slices.Contains(sessionLengths, minutes) {
		f.add("/session_minutes", "must be 20, 30, 45, 60, 75 or 90")
	}
	if bodyweight < minBodyweight || bodyweight > maxBodyweight {
		f.add("/bodyweight_kg", fmt.Sprintf("must be %d–%d", minBodyweight, maxBodyweight))
	}
	for i, e := range equipment {
		if !slices.Contains(equipmentKinds, e) {
			f.add(fmt.Sprintf("/equipment/%d", i), "unknown equipment")
		}
	}
	seen := map[time.Weekday]bool{}
	for i, d := range days {
		if d < time.Sunday || d > time.Saturday || seen[d] {
			f.add(fmt.Sprintf("/preferred_days/%d", i), "must be distinct weekdays")
		}
		seen[d] = true
	}
}

func checkMobility(f fields, m map[string]string) {
	for _, key := range sortedKeys(m) {
		ptr := "/mobility_checks/" + token(key)
		if !slices.Contains(mobilityChecks, key) {
			f.add(ptr, "unknown check")
		} else {
			checkOneOf(f, ptr, m[key], mobilityAnswers)
		}
	}
}

func checkOneOf(f fields, ptr, v string, allowed []string) {
	if !slices.Contains(allowed, v) {
		f.add(ptr, "must be one of "+strings.Join(allowed, ", "))
	}
}

// checkClasses checks the answer classes and the stages against the
// questions of the knowledge base; "unknown" is always an answer.
func (k *Knowledge) checkClasses(f fields, a Answers) {
	for _, key := range sortedKeys(a.Classes) {
		i := slices.IndexFunc(k.onboarding.Questions, func(q ClassQuestion) bool { return q.Key == key })
		if i < 0 {
			f.add("/classes/"+token(key), "unknown question")
			continue
		}
		checkClass(f, "/classes/"+token(key), a.Classes[key], k.onboarding.Questions[i].Classes)
	}
	for _, skill := range sortedKeys(a.Stages) {
		ptr, st := "/skill_stages/"+token(skill), a.Stages[skill]
		sk := k.skills[skill]
		if sk == nil {
			f.add(ptr, "unknown skill")
			continue
		}
		if st.Level == "none" || st.Level == "unknown" {
			continue
		}
		if !slices.ContainsFunc(sk.Levels, func(l Level) bool { return l.Slug == st.Level }) {
			f.add(ptr+"/level", "not a level of this skill, none or unknown")
			continue
		}
		i := slices.IndexFunc(k.onboarding.Stages, func(q StageQuestion) bool { return q.Skill == skill })
		if i < 0 {
			f.add(ptr, "no stage question for this skill")
			continue
		}
		checkClass(f, ptr+"/class", st.Class, k.onboarding.Stages[i].Classes)
	}
}

func checkClass(f fields, ptr, key string, classes []AnswerClass) {
	if key == "unknown" || slices.ContainsFunc(classes, func(c AnswerClass) bool { return c.Key == key }) {
		return
	}
	keys := make([]string, 0, len(classes)+1)
	for _, c := range classes {
		keys = append(keys, c.Key)
	}
	f.add(ptr, "must be one of "+strings.Join(append(keys, "unknown"), ", "))
}

// checkBody checks the body map, the complaints and the red flags: a
// current region needs its complaint and an answer to every red-flag
// question asked for it (onboarding.md §3.7).
func (k *Knowledge) checkBody(f fields, a Answers, minor bool) {
	for _, id := range sortedKeys(a.BodyMap) {
		if _, ok := k.regions[id]; !ok {
			f.add("/body_map/"+token(id), "unknown region")
			continue
		}
		if !a.BodyMap[id].Current {
			continue
		}
		if _, ok := a.Complaints[id]; !ok {
			f.add("/complaints/"+token(id), "required for a current complaint")
		}
		if _, ok := a.RedFlags[id]; !ok {
			f.add("/red_flags/"+token(id), "required for a current complaint")
		}
	}
	for _, id := range sortedKeys(a.Complaints) {
		ptr, c := "/complaints/"+token(id), a.Complaints[id]
		if !a.BodyMap[id].Current {
			f.add(ptr, "the body map does not mark this region as current")
			continue
		}
		for name, v := range map[string]float64{"pain_daily": c.PainDaily, "pain_training": c.PainTraining} {
			if v < 0 || v > maxPainNRS {
				f.add(ptr+"/"+name, "must be 0–10")
			}
		}
		if c.DurationWeeks < 0 {
			f.add(ptr+"/duration_weeks", "must not be negative")
		}
		checkOneOf(f, ptr+"/onset", c.Onset, onsets)
		checkOneOf(f, ptr+"/suspected_serious", c.Suspected, suspicions)
		checkOneOf(f, ptr+"/professional_assessment", c.Assessment, assessments)
		for i, r := range c.Restrictions {
			checkOneOf(f, fmt.Sprintf("%s/restrictions/%d", ptr, i), r, restrictionTags)
		}
	}
	for _, id := range sortedKeys(a.RedFlags) {
		ptr := "/red_flags/" + token(id)
		if !a.BodyMap[id].Current {
			f.add(ptr, "the body map does not mark this region as current")
			continue
		}
		k.checkRedFlags(f, ptr, id, a.RedFlags[id], minor)
	}
}

// checkRedFlags checks the answers of one region: every question asked
// there is answered, and a follow-up exactly when its question was answered
// yes.
func (k *Knowledge) checkRedFlags(f fields, ptr, region string, answers map[string]bool, minor bool) {
	asked := map[string]bool{}
	for _, rf := range k.RedFlags(region, minor) {
		asked[rf.ID] = true
		if _, ok := answers[rf.ID]; !ok {
			f.add(ptr+"/"+token(rf.ID), "must be answered")
		}
		if rf.Followup == nil {
			continue
		}
		asked[rf.Followup.ID] = true
		_, answered := answers[rf.Followup.ID]
		switch {
		case answers[rf.ID] && !answered:
			f.add(ptr+"/"+token(rf.Followup.ID), "must be answered after "+rf.ID+" yes")
		case !answers[rf.ID] && answered:
			f.add(ptr+"/"+token(rf.Followup.ID), "is asked only after "+rf.ID+" yes")
		}
	}
	for _, id := range sortedKeys(answers) {
		if !asked[id] {
			f.add(ptr+"/"+token(id), "not asked for this region")
		}
	}
}

// ValidateRedFlags checks the answers to the red-flag questions of a region
// before they are applied (spec §8.2); pointers are below /answers.
func (k *Knowledge) ValidateRedFlags(region string, answers map[string]bool, minor bool) error {
	f := fields{}
	if _, ok := k.regions[region]; !ok {
		f.add("/region", "unknown region")
		return f.err()
	}
	k.checkRedFlags(f, "/answers", region, answers, minor)
	return f.err()
}

// ValidatePain checks a pain report before it is applied (spec §8.6).
func (k *Knowledge) ValidatePain(r PainReport, now time.Time) error {
	f := fields{}
	if _, ok := k.regions[r.Region]; !ok {
		f.add("/region", "unknown region")
	}
	checkOneOf(f, "/timepoint", r.Timepoint, painTimepoints)
	if r.NRS < 0 || r.NRS > maxPainNRS {
		f.add("/nrs", "must be 0–10")
	}
	if r.At.IsZero() || r.At.After(now.Add(maxAhead)) {
		f.add("/at", "must not be in the future")
	}
	return f.err()
}

// ProfileUpdate is the part of the profile a user may change after the
// onboarding: availability, equipment, body weight, preferred days,
// mobility checks and the added-load limits (onboarding.md §3.3–3.4,
// §3.8). Consent, birth year and the training background stay as given.
type ProfileUpdate struct {
	SessionsPerWeek int
	SessionMinutes  int
	Equipment       []string
	BodyweightKg    float64
	PreferredDays   []time.Weekday
	Mobility        map[string]string
	MaxAddedLoadKg  float64 // 0: no limit
	SmallestPlateKg float64 // 0: PAR-S-37
}

// UpdateProfile applies a profile change. Nothing else in the snapshot
// changes; the plan is generated anew from it (WEEK-08).
func UpdateProfile(s Snapshot, u ProfileUpdate) (Snapshot, error) {
	f := fields{}
	checkAvailability(f, u.SessionsPerWeek, u.SessionMinutes, u.BodyweightKg, u.Equipment, u.PreferredDays)
	checkMobility(f, u.Mobility)
	if u.MaxAddedLoadKg < 0 || u.MaxAddedLoadKg > maxAddedLoadKg {
		f.add("/max_added_load_kg", fmt.Sprintf("must be 0–%d", maxAddedLoadKg))
	}
	if u.SmallestPlateKg < 0 || u.SmallestPlateKg > maxPlateKg {
		f.add("/smallest_plate_kg", fmt.Sprintf("must be 0–%d", maxPlateKg))
	}
	if err := f.err(); err != nil {
		return Snapshot{}, err
	}
	out := s.clone()
	p := &out.Profile
	p.SessionsPerWeek, p.SessionMinutes = u.SessionsPerWeek, u.SessionMinutes
	p.Equipment = expandEquipment(u.Equipment)
	p.BodyweightKg = u.BodyweightKg
	p.PreferredDays = slices.Clone(u.PreferredDays)
	p.Mobility = cloneMap(u.Mobility)
	p.MaxAddedLoadKg, p.SmallestPlateKg = u.MaxAddedLoadKg, u.SmallestPlateKg
	return out.canonical(), nil
}

// SetGoals replaces the goals (onboarding.md §3.2); target dates count as
// calendar days. Ladders stay, so a skill that returns as a goal keeps its
// rung. The plan is generated anew from it (WEEK-08).
func SetGoals(k *Knowledge, s Snapshot, goals []Goal, now time.Time) (Snapshot, error) {
	f := fields{}
	k.checkGoals(f, goals, now)
	if err := f.err(); err != nil {
		return Snapshot{}, err
	}
	out := s.clone()
	out.Goals = make([]Goal, len(goals))
	for i, g := range goals {
		if g.TargetDate != nil {
			d := civil(*g.TargetDate, time.UTC)
			g.TargetDate = &d
		}
		out.Goals[i] = g
	}
	slices.SortFunc(out.Goals, func(x, y Goal) int { return x.Priority - y.Priority })
	return out.canonical(), nil
}

// Realism returns the realism check of every dated goal (GOAL-05).
func (k *Knowledge) Realism(s Snapshot, now time.Time) []Realism {
	var out []Realism
	for _, g := range s.Goals {
		if r, ok := k.RealismFor(s, g, now); ok {
			out = append(out, r)
		}
	}
	return out
}
