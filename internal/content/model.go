// Package content loads, validates and fingerprints the authored content tree
// in content/: families, exercises, bands and skills.
//
// It is shared by cmd/contentlint, which reports, and cmd/seed, which refuses
// to touch the database unless the tree is clean. The field-by-field
// reference for authors is docs/CONTENT_AUTHORING.md.
package content

import (
	"cmp"
	"slices"

	"github.com/jaelricco/hefesto/internal/domain/progress"
)

// Status values shared by exercises and skills.
const (
	StatusActive           = "active"
	StatusDraftPlaceholder = "draft_placeholder"
	StatusRetired          = "retired"
)

// Relation values for skill edges.
const (
	RelationPrerequisite = "prerequisite"
	RelationRecommended  = "recommended"
	RelationAlternative  = "alternative"
	RelationAntagonist   = "antagonist"
)

// RolePrimaryTest is the exercise role that tests a level.
const RolePrimaryTest = "primary_test"

// Tree is a fully loaded content directory.
type Tree struct {
	Families  []Family   `json:"families"`
	Exercises []Exercise `json:"exercises"`
	Bands     []Band     `json:"bands"`
	Skills    []Skill    `json:"skills"`
}

// Family is one entry of the shared push/pull/core/... vocabulary.
type Family struct {
	Slug string `json:"slug" yaml:"slug"`
	Name string `json:"name" yaml:"name"`
}

// Exercise is something an athlete performs and logs.
type Exercise struct {
	Slug            string   `json:"slug" yaml:"slug"`
	Status          string   `json:"status" yaml:"status"`
	Name            string   `json:"name" yaml:"name"`
	Aka             []string `json:"aka" yaml:"aka"`
	Family          string   `json:"family" yaml:"family"`
	DefaultMeasure  string   `json:"default_measure" yaml:"default_measure"`
	LoadSemantics   string   `json:"load_semantics" yaml:"load_semantics"`
	IsBodyweight    *bool    `json:"is_bodyweight" yaml:"is_bodyweight"`
	Unilateral      bool     `json:"unilateral" yaml:"unilateral"`
	TempoApplicable *bool    `json:"tempo_applicable" yaml:"tempo_applicable"`
	Equipment       []string `json:"equipment" yaml:"equipment"`
	Summary         string   `json:"summary" yaml:"summary"`
	Cues            []string `json:"cues" yaml:"cues"`
	CommonFaults    []string `json:"common_faults" yaml:"common_faults"`

	File string `json:"-" yaml:"-"`
}

// Band is a global resistance-band catalogue entry.
type Band struct {
	Brand           string   `json:"brand" yaml:"brand"`
	ColourLabel     string   `json:"colour_label" yaml:"colour_label"`
	ResistanceMinKg float64  `json:"resistance_min_kg" yaml:"resistance_min_kg"`
	ResistanceMaxKg float64  `json:"resistance_max_kg" yaml:"resistance_max_kg"`
	LengthCm        *float64 `json:"length_cm" yaml:"length_cm"`
	ThicknessMm     *float64 `json:"thickness_mm" yaml:"thickness_mm"`

	File string `json:"-" yaml:"-"`
}

// Skill is a node on the map.
type Skill struct {
	Slug           string    `json:"slug" yaml:"slug"`
	Status         string    `json:"status" yaml:"status"`
	Name           string    `json:"name" yaml:"name"`
	Family         string    `json:"family" yaml:"family"`
	DifficultyTier int       `json:"difficulty_tier" yaml:"difficulty_tier"`
	IsMilestone    bool      `json:"is_milestone" yaml:"is_milestone"`
	Aka            []string  `json:"aka" yaml:"aka"`
	Summary        string    `json:"summary" yaml:"summary"`
	PrimaryMuscles []string  `json:"primary_muscles" yaml:"primary_muscles"`
	CommonFaults   []string  `json:"common_faults" yaml:"common_faults"`
	Map            *MapPoint `json:"map" yaml:"map"`
	Levels         []Level   `json:"levels" yaml:"levels"`
	Injuries       []Injury  `json:"injuries" yaml:"injuries"`

	File string `json:"-" yaml:"-"`
}

// MapPoint places a skill on the constellation map.
type MapPoint struct {
	Constellation string  `json:"constellation" yaml:"constellation"`
	X             float64 `json:"x" yaml:"x"`
	Y             float64 `json:"y" yaml:"y"`
}

// Level is one ordered step of a skill.
type Level struct {
	Slug             string            `json:"slug" yaml:"slug"`
	Name             string            `json:"name" yaml:"name"`
	Order            int               `json:"order" yaml:"order"`
	Description      string            `json:"description" yaml:"description"`
	EstWeeksFromPrev *int              `json:"est_weeks_from_prev" yaml:"est_weeks_from_prev"`
	Exercises        []LevelExercise   `json:"exercises" yaml:"exercises"`
	UnlockCriteria   progress.Criteria `json:"unlock_criteria" yaml:"unlock_criteria"`
	Prerequisites    []LevelRef        `json:"prerequisites" yaml:"prerequisites"`
	Links            []LevelLink       `json:"links" yaml:"links"`
}

// LevelExercise attaches an exercise to a level with a role.
type LevelExercise struct {
	Slug string `json:"slug" yaml:"slug"`
	Role string `json:"role" yaml:"role"`
}

// LevelRef names a level of another (or the same) skill.
type LevelRef struct {
	Skill  string   `json:"skill" yaml:"skill"`
	Level  string   `json:"level" yaml:"level"`
	Weight *float64 `json:"weight" yaml:"weight"`
}

// LevelLink is a non-prerequisite edge into a level.
type LevelLink struct {
	Skill    string   `json:"skill" yaml:"skill"`
	Level    string   `json:"level" yaml:"level"`
	Weight   *float64 `json:"weight" yaml:"weight"`
	Relation string   `json:"relation" yaml:"relation"`
}

// Injury is educational injury-risk content attached to a skill.
type Injury struct {
	Region          string        `json:"region" yaml:"region"`
	Name            string        `json:"name" yaml:"name"`
	Description     string        `json:"description" yaml:"description"`
	RiskFactors     []string      `json:"risk_factors" yaml:"risk_factors"`
	EarlySigns      []string      `json:"early_signs" yaml:"early_signs"`
	PrehabExercises []ExerciseRef `json:"prehab_exercises" yaml:"prehab_exercises"`
	Disclaimer      string        `json:"disclaimer" yaml:"disclaimer"`
}

// ExerciseRef names an exercise by slug.
type ExerciseRef struct {
	Slug string `json:"slug" yaml:"slug"`
}

// Edge is a resolved skill-graph edge between two levels, addressed by
// "skill/level" keys. Edges derives it from the tree.
type Edge struct {
	From     string
	To       string
	Relation string
	Weight   float64
}

// LevelKey is the canonical "skill/level" address of a level.
func LevelKey(skill, level string) string { return skill + "/" + level }

// Edges returns every edge the tree implies, in a deterministic order:
//
//   - an implicit prerequisite from each level to the next level of the same
//     skill (a skill's levels are a ladder; authors do not repeat that);
//   - each declared prerequisite;
//   - each declared link.
//
// The seed materialises exactly these edges, and contentlint checks exactly
// these for cycles.
func (t Tree) Edges() []Edge {
	var out []Edge
	for _, s := range t.Skills {
		levels := s.sortedLevels()
		for i, l := range levels {
			to := LevelKey(s.Slug, l.Slug)
			if i > 0 {
				out = append(out, Edge{From: LevelKey(s.Slug, levels[i-1].Slug), To: to, Relation: RelationPrerequisite, Weight: 1})
			}
			for _, p := range l.Prerequisites {
				out = append(out, Edge{From: LevelKey(p.Skill, p.Level), To: to, Relation: RelationPrerequisite, Weight: weightOr1(p.Weight)})
			}
			for _, k := range l.Links {
				out = append(out, Edge{From: LevelKey(k.Skill, k.Level), To: to, Relation: k.Relation, Weight: weightOr1(k.Weight)})
			}
		}
	}
	return out
}

func (s Skill) sortedLevels() []Level {
	out := slices.Clone(s.Levels)
	slices.SortStableFunc(out, func(a, b Level) int { return cmp.Compare(a.Order, b.Order) })
	return out
}

func weightOr1(w *float64) float64 {
	if w == nil {
		return 1
	}
	return *w
}
