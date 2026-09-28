package planning_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

func TestTiredCheckIn(t *testing.T) {
	k := kb(t)
	f := func(v float64) *float64 { return &v }
	n := func(v int) *int { return &v }
	for _, c := range []struct {
		in    planning.CheckIn
		tired bool
	}{
		{planning.CheckIn{}, false},
		{planning.CheckIn{SleepH: f(6)}, true}, // PAR-E-41: ≤ 6 h
		{planning.CheckIn{SleepH: f(6.5)}, false},
		{planning.CheckIn{Fatigue: n(8)}, true}, // PAR-S-28: ≥ 8
		{planning.CheckIn{Fatigue: n(7)}, false},
		{planning.CheckIn{SleepH: f(8), Fatigue: n(9)}, true},
	} {
		if got := k.Tired(c.in); got != c.tired {
			t.Errorf("%+v: tired %v, want %v", c.in, got, c.tired)
		}
	}
	var ve *planning.ValidationError
	if err := planning.ValidateCheckIn(planning.CheckIn{SleepH: f(25), Fatigue: n(0)}); !errors.As(err, &ve) ||
		ve.Fields["/check_in/sleep_hours"] == "" || ve.Fields["/check_in/fatigue"] == "" {
		t.Errorf("out of range: %v", err)
	}
	if err := planning.ValidateCheckIn(planning.CheckIn{SleepH: f(0), Fatigue: n(10)}); err != nil {
		t.Errorf("in range: %v", err)
	}
}

// A tired check-in turns the max block into technique; strength stays.
func TestTechnique(t *testing.T) {
	k := kb(t)
	ps := planning.PlannedSession{ID: "ps", Blocks: []planning.Block{
		{Role: planning.BlockWarmup, Items: []planning.Item{
			{ID: "ramp", Exercise: "planche-lean", Stimulus: planning.StimSkill, Kind: planning.KindWarmup, Sets: 1, HoldS: 5, Reserve: 4},
		}},
		{Role: planning.BlockMax, Items: []planning.Item{
			{ID: "hold", Exercise: "planche-tuck", Stimulus: planning.StimSkill, Kind: planning.KindWorking, Sets: 4, HoldS: 10, Reserve: 4, RestS: 180},
			{ID: "probe", Exercise: "planche-advanced-tuck", Stimulus: planning.StimSkill, Kind: planning.KindWorking, Sets: 1, HoldS: 3, Offer: true},
			{ID: "limit", Exercise: "front-lever-tuck", Stimulus: planning.StimSkill, Kind: planning.KindWorking, Sets: 3, HoldS: 2},
			{ID: "reps", Exercise: "pull-up", Stimulus: planning.StimSkillReps, Kind: planning.KindWorking, Sets: 3, Reps: 5, Reserve: 2},
			{ID: "lean", Exercise: "planche-lean", Stimulus: planning.StimConditioning, Kind: planning.KindWorking, Sets: 4, HoldS: 21, Reserve: 9},
		}},
		{Role: planning.BlockStrength, Items: []planning.Item{
			{ID: "dip", Exercise: "dip", Stimulus: planning.StimStrength, Kind: planning.KindWorking, Sets: 3, Reps: 8, Reserve: 2},
		}},
	}}
	out, changed := k.Technique(ps)
	if !changed {
		t.Fatal("nothing changed")
	}
	maxB := out.Blocks[1]
	if len(maxB.Items) != 3 || maxB.Items[0].ID != "hold" || maxB.Items[1].ID != "reps" || maxB.Items[2].ID != "lean" {
		t.Fatalf("max block %+v, want the hold as technique and the rep skill", maxB.Items)
	}
	// d = 10 + 4 = 14: h = min(0.5 · 14, 10) = 7, three tries, reserve 7.
	if h := maxB.Items[0]; h.Stimulus != planning.StimTechnique || h.HoldS != 7 || h.Sets != 3 || h.Reserve != 7 ||
		h.Intensity() != "moderate" || h.Calibration {
		t.Errorf("technique %+v", h)
	}
	// A conditioning hold too: d = 30, h = min(15, 10) = 10, reserve 20.
	if l := maxB.Items[2]; l.Stimulus != planning.StimTechnique || l.HoldS != 10 || l.Sets != 3 || l.Reserve != 20 {
		t.Errorf("conditioning as technique %+v", l)
	}
	if !reflect.DeepEqual(maxB.Items[1], ps.Blocks[1].Items[3]) || !reflect.DeepEqual(out.Blocks[2], ps.Blocks[2]) ||
		!reflect.DeepEqual(out.Blocks[0], ps.Blocks[0]) {
		t.Error("rep skill, strength or warm-up changed")
	}
	if n := len(out.Reasons); n != 1 || out.Reasons[0].RuleID != planning.RuleCheckin {
		t.Errorf("session reasons %+v", out.Reasons)
	}
	if again, more := k.Technique(out); more || !reflect.DeepEqual(again, out) {
		t.Errorf("a second application changed %v", more)
	}
	light := planning.PlannedSession{Blocks: []planning.Block{ps.Blocks[2]}}
	if same, changed := k.Technique(light); changed || !reflect.DeepEqual(same, light) {
		t.Error("a session without a max block changed")
	}
}
