package progress

// Criteria is the declarative unlock DSL stored in skill_levels.unlock_criteria
// and authored in content/skills/*.yaml (brief §6).
//
// A level unlocks when every condition in All holds and, if Any is non-empty,
// at least one condition in Any holds. Empty criteria mean the level can only
// be self-attested.
//
// The JSON shape is fixed by content/schema/criteria.schema.json; the
// evaluator arrives in Phase 3.
type Criteria struct {
	All []Condition `json:"all,omitempty" yaml:"all,omitempty"`
	Any []Condition `json:"any,omitempty" yaml:"any,omitempty"`
}

// IsEmpty reports whether the criteria can never be met automatically.
func (c Criteria) IsEmpty() bool { return len(c.All) == 0 && len(c.Any) == 0 }

// Conditions returns All followed by Any.
func (c Criteria) Conditions() []Condition {
	out := make([]Condition, 0, len(c.All)+len(c.Any))
	out = append(out, c.All...)
	return append(out, c.Any...)
}

// Condition is one measurable requirement over logged set elements.
type Condition struct {
	Exercise       string   `json:"exercise" yaml:"exercise"`
	Measure        string   `json:"measure" yaml:"measure"`
	Op             string   `json:"op" yaml:"op"`
	Value          float64  `json:"value" yaml:"value"`
	Assistance     string   `json:"assistance,omitempty" yaml:"assistance,omitempty"`
	MinLoadKg      *float64 `json:"min_load_kg,omitempty" yaml:"min_load_kg,omitempty"`
	MaxLoadKg      *float64 `json:"max_load_kg,omitempty" yaml:"max_load_kg,omitempty"`
	MinFormQuality *int     `json:"min_form_quality,omitempty" yaml:"min_form_quality,omitempty"`
	Occurrences    int      `json:"occurrences,omitempty" yaml:"occurrences,omitempty"`
	WithinDays     *int     `json:"within_days,omitempty" yaml:"within_days,omitempty"`
}

// Defaults applied to a condition that leaves a field out. They are spelled
// out here, once, so that stored criteria are always explicit and the
// evaluator never has to guess.
const (
	DefaultAssistance  = AssistanceNone
	DefaultOccurrences = 1
)

// Assistance filter values for a condition.
const (
	AssistanceNone = "none" // only unassisted elements count
	AssistanceAny  = "any"  // assisted elements count too
)

// WithDefaults returns a copy with every omitted optional field made explicit.
func (c Criteria) WithDefaults() Criteria {
	return Criteria{All: withDefaults(c.All), Any: withDefaults(c.Any)}
}

func withDefaults(in []Condition) []Condition {
	if len(in) == 0 {
		return nil
	}
	out := make([]Condition, len(in))
	for i, cond := range in {
		if cond.Assistance == "" {
			cond.Assistance = DefaultAssistance
		}
		if cond.Occurrences == 0 {
			cond.Occurrences = DefaultOccurrences
		}
		out[i] = cond
	}
	return out
}
