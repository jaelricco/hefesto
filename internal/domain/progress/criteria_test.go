package progress

import "testing"

func TestWithDefaultsMakesOmittedFieldsExplicit(t *testing.T) {
	five := 5
	in := Criteria{
		All: []Condition{{Exercise: "a", Measure: "reps", Op: ">=", Value: 5}},
		Any: []Condition{{Exercise: "b", Measure: "reps", Op: ">", Value: 1, Assistance: AssistanceAny, Occurrences: 3, WithinDays: &five}},
	}
	got := in.WithDefaults()

	if c := got.All[0]; c.Assistance != AssistanceNone || c.Occurrences != 1 {
		t.Errorf("omitted fields not defaulted: %+v", c)
	}
	if c := got.Any[0]; c.Assistance != AssistanceAny || c.Occurrences != 3 || c.WithinDays != &five {
		t.Errorf("explicit fields changed: %+v", c)
	}
	if in.All[0].Assistance != "" {
		t.Error("WithDefaults mutated its receiver")
	}
	if len(got.Conditions()) != 2 {
		t.Errorf("Conditions: got %d, want 2", len(got.Conditions()))
	}
}

func TestEmptyCriteria(t *testing.T) {
	var c Criteria
	if !c.IsEmpty() || !c.WithDefaults().IsEmpty() {
		t.Fatal("empty criteria must stay empty: the level is self-attest only")
	}
}
