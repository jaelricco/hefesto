package planning

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Files is the knowledge base as authored under content/training/. The
// loader in internal/content decodes the YAML files into it; Build turns it
// into a validated Knowledge.
type Files struct {
	Manifest   Manifest
	Parameters []Param
	Rules      []Rule
	Sources    []Source
	Body       Body
	Exercises  []Exercise
	Skills     []Skill
	Sessions   SessionConfig
	Onboarding OnboardingConfig
}

// Manifest carries the ruleset version (spec §2.5).
type Manifest struct {
	RulesetVersion string `yaml:"ruleset_version"`
	Status         string `yaml:"status"`
}

// Param is one catalogue entry. A research parameter keeps its research ID
// (PAR-A-… to PAR-F-…); values this specification sets are PAR-S-….
type Param struct {
	ID        string             `yaml:"id"`
	Key       string             `yaml:"key"`
	Value     *float64           `yaml:"value,omitempty"`
	Values    map[string]float64 `yaml:"values,omitempty"`
	Unit      string             `yaml:"unit,omitempty"`
	Sources   []string           `yaml:"sources,omitempty"`
	Evidence  string             `yaml:"evidence"`
	Heuristic bool               `yaml:"heuristic,omitempty"`
	Rationale string             `yaml:"rationale,omitempty"`
}

// Rule is one entry of the rule catalogue (spec appendix A). Its logic lives
// in Go; its parameters, text and evidence live here.
type Rule struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Text     string   `yaml:"text"`
	Params   []string `yaml:"params,omitempty"`
	Sources  []string `yaml:"sources,omitempty"`
	Evidence string   `yaml:"evidence"`
}

// Source is one entry of docs/research/00_sources.md.
type Source struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	Year     string `yaml:"year,omitempty"`
	URL      string `yaml:"url,omitempty"`
	Evidence string `yaml:"evidence"`
}

// Body holds structures, regions, load profiles, the complaint matrix, red
// flags and prehab (spec §4.5, §4.6, §8).
type Body struct {
	Structures        []Structure   `yaml:"structures"`
	Regions           []Region      `yaml:"regions"`
	LoadFamilies      []LoadFamily  `yaml:"load_families"`
	ComplaintFamilies []string      `yaml:"complaint_families"`
	Matrix            []MatrixRow   `yaml:"matrix"`
	RedFlags          []RedFlag     `yaml:"red_flags"`
	Prehab            []PrehabEntry `yaml:"prehab"`
	Disclaimer        string        `yaml:"disclaimer"`
	ForbiddenPhrases  []string      `yaml:"forbidden_phrases"`
}

// Structure is a joint or tendon group with its own load accounts.
type Structure struct {
	ID string `yaml:"id"`
	// StraightArmCap marks the structures PAR-D-09 names; all straight-arm
	// accounts use the stricter rate anyway (spec §7.1), the flag documents it.
	StraightArmCap bool `yaml:"straight_arm_cap,omitempty"`
}

// Region is a place on the onboarding body map.
type Region struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Structures []string `yaml:"structures"`
	Draft      bool     `yaml:"draft,omitempty"`
}

// LoadFamily is one row of the ordinal load profile (research 04 table 2).
type LoadFamily struct {
	ID      string         `yaml:"id"`
	Ratings map[string]int `yaml:"ratings"`
	// RefTorque and RefBW describe the heaviest usual stage the ratings refer
	// to; lighter rungs scale by their own ratio (PAR-C-44).
	RefTorque float64  `yaml:"ref_torque,omitempty"`
	RefBW     float64  `yaml:"ref_bw,omitempty"`
	Sources   []string `yaml:"sources,omitempty"`
}

// MatrixRow maps one region to an action per complaint family (research 05 §8).
type MatrixRow struct {
	Region  string              `yaml:"region"`
	Cells   map[string]string   `yaml:"cells"`
	Sources map[string][]string `yaml:"sources,omitempty"`
}

// Matrix actions.
const (
	ActionExclude = "X"
	ActionModify  = "M"
	ActionPain    = "S"
	ActionNone    = "-"
)

// RedFlag is one warning-sign question (research 05 §9).
type RedFlag struct {
	ID string `yaml:"id"`
	// Regions lists where the question is asked; "*" means every region.
	Regions   []string `yaml:"regions"`
	MinorOnly bool     `yaml:"minor_only,omitempty"`
	Urgency   string   `yaml:"urgency"` // N, D or A
	Action    string   `yaml:"action"`  // stop, lock or rtt0
	Question  string   `yaml:"question"`
	Advice    string   `yaml:"advice"`
}

// Red-flag urgencies and actions.
const (
	UrgencyNow    = "N"
	UrgencySoon   = "D"
	UrgencyAdvise = "A"

	FlagStop = "stop"
	FlagLock = "lock"
	FlagRTT0 = "rtt0"
)

// PrehabEntry lists prehab exercises for a region with the wording its
// evidence allows (PAR-D-38).
type PrehabEntry struct {
	Region     string   `yaml:"region"`
	Activation []string `yaml:"activation"`
	Loaded     []string `yaml:"loaded,omitempty"`
	Label      string   `yaml:"label"`
}

// Exercise is a loggable exercise with the metadata the planner needs (spec
// §2.2). In phase 5 the planner keeps its own list; merging it into
// content/exercises comes with the content import.
type Exercise struct {
	Slug            string   `yaml:"slug"`
	Name            string   `yaml:"name"`
	Status          string   `yaml:"status"`
	Pattern         string   `yaml:"pattern"`
	StraightArm     string   `yaml:"straight_arm"`
	Direction       string   `yaml:"direction"`
	LoadFamily      string   `yaml:"load_family"`
	Modifiers       []string `yaml:"load_modifiers,omitempty"`
	ComplaintFamily string   `yaml:"complaint_family"`
	Supinated       bool     `yaml:"supinated_straight_arm,omitempty"`
	Measure         string   `yaml:"measure"`
	HoldClass       string   `yaml:"hold_class,omitempty"`
	Requires        []string `yaml:"requires"`
	OG              float64  `yaml:"og_ordinal"`
	Torque          float64  `yaml:"torque_ratio,omitempty"`
	BW              float64  `yaml:"bw_fraction,omitempty"`
	Assistable      bool     `yaml:"assistable,omitempty"`
	Loadable        bool     `yaml:"loadable,omitempty"`
	GtG             bool     `yaml:"gtg_allowed,omitempty"`
	Eccentric       bool     `yaml:"eccentric,omitempty"`
	Restrictions    []string `yaml:"restriction_tags,omitempty"`
	Sources         []string `yaml:"sources,omitempty"`

	// Filled by Build from the skill that lists the exercise as a rung.
	Skill string `yaml:"-"`
	Rung  int    `yaml:"-"`
}

// Exercise vocabularies.
const (
	MeasureReps = "reps"
	MeasureHold = "hold_seconds"

	ArmNone     = "none"
	ArmStraight = "straight_arm"
	ArmAxial    = "straight_arm_axial"

	HoldSkill     = "skill_static"
	HoldBalance   = "balance"
	HoldCore      = "core"
	HoldEndurance = "endurance"

	DirPush = "push"
	DirPull = "pull"
	DirNone = "none"
)

// Skill is the planning view of a skill: its ladder of exercises (easiest
// first) and its levels with prerequisites (spec §3). Foundation skills are
// the roots of research 02 §4.
type Skill struct {
	Slug           string   `yaml:"slug"`
	Name           string   `yaml:"name"`
	Foundation     bool     `yaml:"foundation,omitempty"`
	LimitingFactor string   `yaml:"limiting_factor"`
	Rungs          []string `yaml:"rungs"`
	Levels         []Level  `yaml:"levels"`
	// Feeders are support skills trained while the ladder sits on its
	// eccentric rung (SEL-09, e.g. rows for the pull-up).
	Feeders []string `yaml:"feeders,omitempty"`
	Sources []string `yaml:"sources,omitempty"`
}

// Limiting factors (PAR-E-28).
const (
	LimitStrength = "strength"
	LimitBalance  = "balance"
	LimitMixed    = "mixed"
)

// Level is a proof of ability: met when unlocked or when the dose value of
// its exercise reaches Threshold (spec §3.4).
type Level struct {
	Slug          string   `yaml:"slug"`
	Exercise      string   `yaml:"exercise"`
	Threshold     float64  `yaml:"threshold"`
	OG            float64  `yaml:"og_ordinal"`
	Prerequisites []string `yaml:"prerequisites,omitempty"`
	Recommended   []Edge   `yaml:"recommended,omitempty"`
	Milestone     bool     `yaml:"milestone,omitempty"`
	// Hint is the soft readiness hint shown while the level's prerequisites
	// are unmet (GOAL-04); it never blocks.
	Hint string `yaml:"hint,omitempty"`
}

// Edge is a weighted recommended edge to "skill/level".
type Edge struct {
	To     string  `yaml:"to"`
	Weight float64 `yaml:"weight"`
}

// SessionConfig holds the session templates and default day patterns.
type SessionConfig struct {
	Templates   []Template       `yaml:"templates"`
	DayPatterns map[int][]string `yaml:"day_patterns"`
}

// Template is a session layout by available minutes (PAR-B-64–67).
type Template struct {
	Minutes      int      `yaml:"minutes"`
	WarmupMin    float64  `yaml:"warmup_min"`
	SkillMin     float64  `yaml:"skill_min"`
	VolumeBlocks bool     `yaml:"volume_blocks"`
	Pairs        int      `yaml:"antagonist_pairs"`
	Accessories  int      `yaml:"accessories"`
	Params       []string `yaml:"params"`
}

// OnboardingConfig maps answer classes to capacity priors (onboarding §3.6,
// §5.2).
type OnboardingConfig struct {
	Questions []ClassQuestion `yaml:"questions"`
	Stages    []StageQuestion `yaml:"stages"`
}

// ClassQuestion is a "how many / how long" question with answer classes.
type ClassQuestion struct {
	Key      string        `yaml:"key"`
	Exercise string        `yaml:"exercise"`
	Classes  []AnswerClass `yaml:"classes"`
}

// AnswerClass is one answer; Hi 0 with Open means "more than Lo".
type AnswerClass struct {
	Key  string  `yaml:"key"`
	Lo   float64 `yaml:"lo"`
	Hi   float64 `yaml:"hi"`
	Open bool    `yaml:"open,omitempty"`
}

// StageQuestion maps a goal skill to the hold or rep classes of its stage
// question (skill_stage_hold_class).
type StageQuestion struct {
	Skill   string        `yaml:"skill"`
	Classes []AnswerClass `yaml:"classes"`
}

// Knowledge is the validated, indexed knowledge base.
type Knowledge struct {
	Version string
	Status  string

	params     map[string]Param
	rules      map[string]Rule
	sources    map[string]Source
	structures []string
	regions    map[string]Region
	regionIDs  []string
	families   map[string]LoadFamily
	complaint  map[string]bool
	matrix     map[string]map[string]string
	matrixSrc  map[string]map[string][]string
	redFlags   []RedFlag
	prehab     map[string]PrehabEntry
	exercises  map[string]*Exercise
	skills     map[string]*Skill
	skillIDs   []string
	templates  []Template
	days       map[int][]time.Weekday
	onboarding OnboardingConfig
	disclaimer string
	forbidden  []string

	T Tuning
}

// Issue is a problem found while building the knowledge base (spec §2.6).
type Issue struct {
	Check string // KB-01 … KB-13
	Where string
	Msg   string
	Warn  bool
}

func (i Issue) String() string {
	sev := "error"
	if i.Warn {
		sev = "warning"
	}
	return fmt.Sprintf("%s: %s: %s: %s", sev, i.Check, i.Where, i.Msg)
}

// HasErrors reports whether any issue is an error (or any issue at all when
// strict).
func HasErrors(issues []Issue, strict bool) bool {
	for _, i := range issues {
		if strict || !i.Warn {
			return true
		}
	}
	return false
}

// Build validates the authored files and returns the indexed knowledge base.
// It never panics; when any issue is an error the returned knowledge must not
// be used for planning.
func Build(f Files) (*Knowledge, []Issue) {
	b := builder{kb: &Knowledge{
		Version:    f.Manifest.RulesetVersion,
		Status:     f.Manifest.Status,
		params:     map[string]Param{},
		rules:      map[string]Rule{},
		sources:    map[string]Source{},
		regions:    map[string]Region{},
		families:   map[string]LoadFamily{},
		complaint:  map[string]bool{},
		matrix:     map[string]map[string]string{},
		matrixSrc:  map[string]map[string][]string{},
		prehab:     map[string]PrehabEntry{},
		exercises:  map[string]*Exercise{},
		skills:     map[string]*Skill{},
		days:       map[int][]time.Weekday{},
		onboarding: f.Onboarding,
		disclaimer: strings.TrimSpace(f.Body.Disclaimer),
		forbidden:  f.Body.ForbiddenPhrases,
	}}
	if f.Manifest.RulesetVersion == "" {
		b.errf("KB-01", "manifest", "ruleset_version is missing")
	}
	b.catalogues(f)
	b.body(f.Body)
	b.exercisesAndSkills(f.Exercises, f.Skills)
	b.sessions(f.Sessions)
	b.onboardingRefs(f.Onboarding)
	b.references()
	b.kb.T, b.issues = resolveTuning(b.kb, b.issues)
	b.language()
	return b.kb, b.issues
}

type builder struct {
	kb     *Knowledge
	issues []Issue
}

func (b *builder) errf(check, where, format string, args ...any) {
	b.issues = append(b.issues, Issue{Check: check, Where: where, Msg: fmt.Sprintf(format, args...)})
}

func (b *builder) warnf(check, where, format string, args ...any) {
	b.issues = append(b.issues, Issue{Check: check, Where: where, Msg: fmt.Sprintf(format, args...), Warn: true})
}

func (b *builder) catalogues(f Files) {
	for _, s := range f.Sources {
		if _, dup := b.kb.sources[s.ID]; dup {
			b.errf("KB-02", "sources", "duplicate source %s", s.ID)
		}
		b.kb.sources[s.ID] = s
	}
	for _, p := range f.Parameters {
		if _, dup := b.kb.params[p.ID]; dup {
			b.errf("KB-03", "parameters", "duplicate parameter %s", p.ID)
		}
		if p.Value == nil && len(p.Values) == 0 {
			b.errf("KB-03", p.ID, "parameter has neither value nor values")
		}
		if !p.Heuristic && len(p.Sources) == 0 {
			b.errf("KB-03", p.ID, "parameter has no sources and is not marked heuristic")
		}
		if p.Heuristic && strings.TrimSpace(p.Rationale) == "" {
			b.errf("KB-03", p.ID, "heuristic parameter needs a rationale")
		}
		b.kb.params[p.ID] = p
	}
	for _, r := range f.Rules {
		if _, dup := b.kb.rules[r.ID]; dup {
			b.errf("KB-02", "rules", "duplicate rule %s", r.ID)
		}
		if strings.TrimSpace(r.Text) == "" {
			b.errf("KB-11", r.ID, "rule has no text")
		}
		b.kb.rules[r.ID] = r
	}
}

func (b *builder) body(body Body) {
	for _, s := range body.Structures {
		if slices.Contains(b.kb.structures, s.ID) {
			b.errf("KB-02", "structures", "duplicate structure %s", s.ID)
		}
		b.kb.structures = append(b.kb.structures, s.ID)
	}
	for _, r := range body.Regions {
		for _, s := range r.Structures {
			if !slices.Contains(b.kb.structures, s) {
				b.errf("KB-02", "region "+r.ID, "unknown structure %s", s)
			}
		}
		b.kb.regions[r.ID] = r
		b.kb.regionIDs = append(b.kb.regionIDs, r.ID)
	}
	for _, lf := range body.LoadFamilies {
		for s, v := range lf.Ratings {
			if !slices.Contains(b.kb.structures, s) {
				b.errf("KB-02", "load family "+lf.ID, "unknown structure %s", s)
			}
			if v < 0 || v > 3 {
				b.errf("KB-09", "load family "+lf.ID, "rating %d for %s is outside 0–3", v, s)
			}
		}
		b.kb.families[lf.ID] = lf
	}
	for _, c := range body.ComplaintFamilies {
		b.kb.complaint[c] = true
	}
	for _, row := range body.Matrix {
		if _, ok := b.kb.regions[row.Region]; !ok {
			b.errf("KB-02", "matrix", "unknown region %s", row.Region)
		}
		b.kb.matrix[row.Region] = row.Cells
		b.kb.matrixSrc[row.Region] = row.Sources
		for fam, act := range row.Cells {
			if !b.kb.complaint[fam] {
				b.errf("KB-02", "matrix "+row.Region, "unknown complaint family %s", fam)
			}
			switch act {
			case ActionExclude, ActionModify, ActionPain, ActionNone:
			default:
				b.errf("KB-09", "matrix "+row.Region, "unknown action %q", act)
			}
		}
	}
	// KB-06: every region with structures has a full matrix row.
	for _, id := range b.kb.regionIDs {
		r := b.kb.regions[id]
		if len(r.Structures) == 0 {
			continue
		}
		for fam := range b.kb.complaint {
			if _, ok := b.kb.matrix[id][fam]; !ok {
				b.errf("KB-06", "matrix "+id, "no cell for complaint family %s", fam)
			}
		}
	}
	has := map[string]map[string]bool{}
	for _, rf := range body.RedFlags {
		switch rf.Urgency {
		case UrgencyNow, UrgencySoon, UrgencyAdvise:
		default:
			b.errf("KB-10", rf.ID, "urgency must be N, D or A")
		}
		switch rf.Action {
		case FlagStop, FlagLock, FlagRTT0:
		default:
			b.errf("KB-10", rf.ID, "unknown action %q", rf.Action)
		}
		for _, r := range rf.Regions {
			if r != "*" {
				if _, ok := b.kb.regions[r]; !ok {
					b.errf("KB-02", rf.ID, "unknown region %s", r)
				}
			}
			if has[r] == nil {
				has[r] = map[string]bool{}
			}
			has[r][rf.ID] = true
		}
		b.kb.redFlags = append(b.kb.redFlags, rf)
	}
	for _, id := range []string{"RF-01", "RF-02", "RF-03", "RF-04", "RF-05", "RF-06", "RF-07", "RF-10"} {
		if !has["*"][id] {
			b.errf("KB-10", "red_flags", "%s must be asked for every region", id)
		}
	}
	for _, p := range body.Prehab {
		b.kb.prehab[p.Region] = p
	}
	if b.kb.disclaimer == "" {
		b.errf("KB-01", "body", "disclaimer is missing")
	}
}

func (b *builder) exercisesAndSkills(exs []Exercise, sks []Skill) {
	for i := range exs {
		e := exs[i]
		where := "exercise " + e.Slug
		if _, dup := b.kb.exercises[e.Slug]; dup {
			b.errf("KB-02", where, "duplicate exercise")
		}
		if e.Pattern == "" || e.LoadFamily == "" || e.ComplaintFamily == "" || e.Measure == "" || e.StraightArm == "" {
			b.errf("KB-07", where, "pattern, load_family, complaint_family, measure and straight_arm are required")
		}
		if _, ok := b.kb.families[e.LoadFamily]; !ok {
			b.errf("KB-02", where, "unknown load family %s", e.LoadFamily)
		}
		if e.ComplaintFamily != "none" && !b.kb.complaint[e.ComplaintFamily] {
			b.errf("KB-02", where, "unknown complaint family %s", e.ComplaintFamily)
		}
		if e.Measure == MeasureHold && e.HoldClass == "" {
			b.errf("KB-07", where, "hold exercises need a hold_class")
		}
		if e.Torque < 0 || e.Torque > 1.0001 || e.BW < 0 || e.BW > 1.0001 {
			b.errf("KB-09", where, "torque_ratio and bw_fraction must lie in [0, 1]")
		}
		b.kb.exercises[e.Slug] = &e
	}
	for i := range sks {
		s := sks[i]
		where := "skill " + s.Slug
		if _, dup := b.kb.skills[s.Slug]; dup {
			b.errf("KB-02", where, "duplicate skill")
		}
		if len(s.Rungs) == 0 || len(s.Levels) == 0 {
			b.errf("KB-08", where, "a skill needs rungs and levels")
		}
		prevOG := -1.0
		for r, slug := range s.Rungs {
			e, ok := b.kb.exercises[slug]
			if !ok {
				b.errf("KB-02", where, "unknown rung exercise %s", slug)
				continue
			}
			if e.Skill != "" {
				b.errf("KB-02", where, "exercise %s is already a rung of %s", slug, e.Skill)
			}
			e.Skill, e.Rung = s.Slug, r
			// KB-05: rungs are monotone in the OG ordinal.
			if e.OG < prevOG {
				b.errf("KB-05", where, "rung %s (OG %.1f) is easier than the rung before it (OG %.1f)", slug, e.OG, prevOG)
			}
			prevOG = e.OG
		}
		for _, l := range s.Levels {
			if !slices.Contains(s.Rungs, l.Exercise) {
				b.errf("KB-08", where+"/"+l.Slug, "level exercise %s is not a rung of the skill", l.Exercise)
			}
		}
		b.kb.skills[s.Slug] = &s
		b.kb.skillIDs = append(b.kb.skillIDs, s.Slug)
	}
	slices.Sort(b.kb.skillIDs)
	b.checkGraph()
}

// checkGraph verifies prerequisite references and acyclicity, including the
// implicit previous level of each skill (KB-04).
func (b *builder) checkGraph() {
	const (
		white = iota
		grey
		black
	)
	colour := map[string]int{}
	var visit func(ref string) bool
	visit = func(ref string) bool {
		switch colour[ref] {
		case grey:
			return false
		case black:
			return true
		}
		colour[ref] = grey
		for _, p := range b.kb.prereqs(ref) {
			if !visit(p) {
				b.errf("KB-04", ref, "prerequisite cycle through %s", p)
				colour[ref] = black
				return false
			}
		}
		colour[ref] = black
		return true
	}
	for _, id := range b.kb.skillIDs {
		s := b.kb.skills[id]
		for _, l := range s.Levels {
			ref := id + "/" + l.Slug
			for _, p := range l.Prerequisites {
				if b.kb.level(p) == nil {
					b.errf("KB-02", ref, "unknown prerequisite %s", p)
				}
			}
			for _, e := range l.Recommended {
				if b.kb.level(e.To) == nil {
					b.errf("KB-02", ref, "unknown recommended level %s", e.To)
				}
			}
		}
	}
	for _, id := range b.kb.skillIDs {
		for _, l := range b.kb.skills[id].Levels {
			visit(id + "/" + l.Slug)
		}
	}
}

func (b *builder) sessions(sc SessionConfig) {
	b.kb.templates = slices.Clone(sc.Templates)
	slices.SortFunc(b.kb.templates, func(x, y Template) int { return cmp.Compare(x.Minutes, y.Minutes) })
	for _, m := range []int{20, 30, 45, 60, 75, 90} {
		if !slices.ContainsFunc(b.kb.templates, func(t Template) bool { return t.Minutes == m }) {
			b.errf("KB-09", "sessions", "no template for %d minutes", m)
		}
	}
	names := map[string]time.Weekday{"mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
		"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday, "sun": time.Sunday}
	for n := 1; n <= 7; n++ {
		days, ok := sc.DayPatterns[n]
		if !ok || len(days) != n {
			b.errf("KB-09", "sessions", "day pattern for %d sessions is missing or has the wrong length", n)
			continue
		}
		for _, d := range days {
			wd, ok := names[d]
			if !ok {
				b.errf("KB-09", "sessions", "unknown weekday %q", d)
				continue
			}
			b.kb.days[n] = append(b.kb.days[n], wd)
		}
	}
}

func (b *builder) onboardingRefs(oc OnboardingConfig) {
	for _, q := range oc.Questions {
		if _, ok := b.kb.exercises[q.Exercise]; !ok {
			b.errf("KB-02", "onboarding "+q.Key, "unknown exercise %s", q.Exercise)
		}
	}
	for _, s := range oc.Stages {
		if _, ok := b.kb.skills[s.Skill]; !ok {
			b.errf("KB-02", "onboarding stage", "unknown skill %s", s.Skill)
		}
	}
}

// references checks that every parameter, rule, prehab and exercise reference
// exists (KB-02) and flags draft content (KB-13).
func (b *builder) references() {
	for _, id := range sortedKeys(b.kb.params) {
		for _, s := range b.kb.params[id].Sources {
			if _, ok := b.kb.sources[s]; !ok {
				b.errf("KB-02", id, "unknown source %s", s)
			}
		}
	}
	for _, id := range sortedKeys(b.kb.rules) {
		r := b.kb.rules[id]
		for _, p := range r.Params {
			if _, ok := b.kb.params[p]; !ok {
				b.errf("KB-02", id, "unknown parameter %s", p)
			}
		}
		for _, s := range r.Sources {
			if _, ok := b.kb.sources[s]; !ok {
				b.errf("KB-02", id, "unknown source %s", s)
			}
		}
	}
	for _, id := range sortedKeys(b.kb.families) {
		for _, s := range b.kb.families[id].Sources {
			if _, ok := b.kb.sources[s]; !ok {
				b.errf("KB-02", "load family "+id, "unknown source %s", s)
			}
		}
	}
	for _, id := range sortedKeys(b.kb.exercises) {
		e := b.kb.exercises[id]
		for _, s := range e.Sources {
			if _, ok := b.kb.sources[s]; !ok {
				b.errf("KB-02", "exercise "+id, "unknown source %s", s)
			}
		}
		if e.Status == "draft_placeholder" {
			b.warnf("KB-13", "exercise "+id, "draft content awaiting review (ENT-10)")
		}
	}
	for _, id := range sortedKeys(b.kb.prehab) {
		p := b.kb.prehab[id]
		for _, s := range append(slices.Clone(p.Activation), p.Loaded...) {
			if _, ok := b.kb.exercises[s]; !ok {
				b.errf("KB-02", "prehab "+id, "unknown exercise %s", s)
			}
		}
	}
	for _, id := range requiredRules {
		if _, ok := b.kb.rules[id]; !ok {
			b.errf("KB-03", "rules", "rule %s used by the planner is missing", id)
		}
	}
}

// language checks rule texts, red-flag texts and the disclaimer against the
// forbidden phrases (KB-12, spec §9.3).
func (b *builder) language() {
	check := func(where, text string) {
		if p := forbiddenIn(b.kb.forbidden, text); p != "" {
			b.errf("KB-12", where, "text contains the forbidden phrase %q", p)
		}
	}
	for _, id := range sortedKeys(b.kb.rules) {
		r := b.kb.rules[id]
		check(id, r.Title+" "+r.Text)
	}
	for _, rf := range b.kb.redFlags {
		check(rf.ID, rf.Question+" "+rf.Advice)
	}
}

func forbiddenIn(phrases []string, text string) string {
	low := strings.ToLower(text)
	for _, p := range phrases {
		if p != "" && strings.Contains(low, strings.ToLower(p)) {
			return p
		}
	}
	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// level returns the level named "skill/level", or nil.
func (k *Knowledge) level(ref string) *Level {
	skill, lvl, ok := strings.Cut(ref, "/")
	if !ok {
		return nil
	}
	s := k.skills[skill]
	if s == nil {
		return nil
	}
	for i := range s.Levels {
		if s.Levels[i].Slug == lvl {
			return &s.Levels[i]
		}
	}
	return nil
}

// prereqs returns the hard prerequisites of a level, including the implicit
// previous level of its own skill.
func (k *Knowledge) prereqs(ref string) []string {
	skill, lvl, _ := strings.Cut(ref, "/")
	s := k.skills[skill]
	if s == nil {
		return nil
	}
	var out []string
	for i, l := range s.Levels {
		if l.Slug != lvl {
			continue
		}
		if i > 0 {
			out = append(out, skill+"/"+s.Levels[i-1].Slug)
		}
		out = append(out, l.Prerequisites...)
	}
	return out
}

// Exercise returns an exercise by slug.
func (k *Knowledge) Exercise(slug string) (*Exercise, bool) {
	e, ok := k.exercises[slug]
	return e, ok
}

// Skill returns a skill by slug.
func (k *Knowledge) Skill(slug string) (*Skill, bool) {
	s, ok := k.skills[slug]
	return s, ok
}

// Rule returns a rule by ID.
func (k *Knowledge) Rule(id string) (Rule, bool) {
	r, ok := k.rules[id]
	return r, ok
}

// Source returns a source by ID.
func (k *Knowledge) Source(id string) (Source, bool) {
	s, ok := k.sources[id]
	return s, ok
}

// Disclaimer is the text every plan and injury payload carries (spec §8.1).
func (k *Knowledge) Disclaimer() string { return k.disclaimer }

// RedFlags returns the questions to ask for a region (spec §8.2).
func (k *Knowledge) RedFlags(region string, minor bool) []RedFlag {
	var out []RedFlag
	for _, rf := range k.redFlags {
		if rf.MinorOnly && !minor {
			continue
		}
		if slices.Contains(rf.Regions, "*") || slices.Contains(rf.Regions, region) {
			out = append(out, rf)
		}
	}
	return out
}
