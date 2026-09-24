package progress

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// Ties on time are broken by id, so evidence never depends on map order.
func TestEvidenceTieBreakIsDeterministic(t *testing.T) {
	at := time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC)
	v := 12.0
	a, b := uuid.MustParse("00000000-0000-0000-0000-00000000000a"), uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	h := SetHistory{
		{SetEntryID: b, Exercise: "x", Measure: "reps", Value: &v, Assistance: ClassUnassisted, PerformedAt: at},
		{SetEntryID: a, Exercise: "x", Measure: "reps", Value: &v, Assistance: ClassUnassisted, PerformedAt: at},
	}
	c := Criteria{All: []Condition{{Exercise: "x", Measure: "reps", Op: ">=", Value: 10}}}
	for i := 0; i < 20; i++ {
		r, err := Evaluate(c, h, at.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if r.Evidence == nil || *r.Evidence != a {
			t.Fatalf("evidence %v, want %v", r.Evidence, a)
		}
	}

	// The same tie across two conditions.
	c = Criteria{All: []Condition{
		{Exercise: "x", Measure: "reps", Op: ">=", Value: 10},
		{Exercise: "y", Measure: "reps", Op: ">=", Value: 10},
	}}
	h = SetHistory{
		{SetEntryID: b, Exercise: "x", Measure: "reps", Value: &v, Assistance: ClassUnassisted, PerformedAt: at},
		{SetEntryID: a, Exercise: "y", Measure: "reps", Value: &v, Assistance: ClassUnassisted, PerformedAt: at},
	}
	r, err := Evaluate(c, h, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if r.Evidence == nil || *r.Evidence != a {
		t.Fatalf("cross-condition tie: evidence %v, want %v", r.Evidence, a)
	}
}

func TestAttemptsAndPrimary(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	v8, v3 := 8.0, 3.0
	s1, s2 := uuid.New(), uuid.New()
	h := SetHistory{
		{SetEntryID: s1, Exercise: "x", Measure: "reps", Value: &v8, Assistance: ClassUnassisted, PerformedAt: now.AddDate(0, 0, -10)},
		{SetEntryID: s1, Exercise: "x", Measure: "reps", Value: &v3, Assistance: ClassUnassisted, PerformedAt: now.AddDate(0, 0, -10)},
		{SetEntryID: s2, Exercise: "x", Measure: "reps", Value: &v3, Assistance: ClassUnassisted, PerformedAt: now.AddDate(0, 0, -2)},
		{SetEntryID: uuid.New(), Exercise: "x", Measure: "reps", Value: &v8, Assistance: ClassUnassisted, PerformedAt: now.AddDate(0, 0, 2)},
	}
	r, err := Evaluate(Criteria{All: []Condition{{Exercise: "x", Measure: "reps", Op: ">=", Value: 10}}}, h, now)
	if err != nil {
		t.Fatal(err)
	}
	p := r.Primary()
	if p == nil || p.Attempts != 2 || p.LastAttempt == nil || !p.LastAttempt.Equal(now.AddDate(0, 0, -2)) {
		t.Fatalf("primary %+v", p)
	}
	anyOnly, _ := Evaluate(Criteria{Any: []Condition{{Exercise: "x", Measure: "reps", Op: ">=", Value: 1}}}, h, now)
	if anyOnly.Primary() == nil || anyOnly.Primary().Condition.Exercise != "x" {
		t.Fatal("any-only primary")
	}
	if (Result{}).Primary() != nil {
		t.Fatal("self-attest-only criteria have no primary condition")
	}
}
