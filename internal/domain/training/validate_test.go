package training

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func element(m Measure) Element {
	return Element{ID: uuid.New(), ExerciseID: uuid.New(), Measure: m}
}

func fields(t *testing.T, err error) FieldErrors {
	t.Helper()
	if err == nil {
		return nil
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a ValidationError, got %T: %v", err, err)
	}
	return ve.Fields
}

func TestPlainSetAndComboShareOnePath(t *testing.T) {
	plain := SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.Reps = ptr(8); return e }()}}
	if err := ValidateSet(&plain); err != nil {
		t.Fatalf("plain set: %v", err)
	}

	hold := func() Element { e := element(MeasureHoldSeconds); e.HoldSeconds = ptr(3.0); return e }
	press := func() Element { e := element(MeasureReps); e.Reps = ptr(1); return e }
	combo := SetEntry{Elements: []Element{hold(), press(), hold()}}
	if err := ValidateSet(&combo); err != nil {
		t.Fatalf("combo: %v", err)
	}
	for i, e := range combo.Elements {
		if e.OrderIndex != i {
			t.Errorf("element %d has order_index %d", i, e.OrderIndex)
		}
	}
}

func TestValidateSet(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name  string
		set   SetEntry
		field string
	}{
		{"no elements", SetEntry{}, "/elements"},
		{"too many elements", SetEntry{Elements: make([]Element, MaxElementsPerSet+1)}, "/elements"},
		{"planned but completed", SetEntry{IsPlanned: true, CompletedAt: &now, Elements: []Element{element(MeasureNone)}}, "/completed_at"},
		{"rpe not a half step", SetEntry{RPE: ptr(7.3), Elements: []Element{element(MeasureNone)}}, "/rpe"},
		{"value for another measure", SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.HoldSeconds = ptr(5.0); return e }()}}, "/elements/0/hold_seconds"},
		{"bad tempo", SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.Tempo = ptr("3-0-X-1"); return e }()}}, "/elements/0/tempo"},
		{"negative load", SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.LoadKg = -10; return e }()}}, "/elements/0/load_kg"},
		{"rom note without partial rom", SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.ROMNote = ptr("half"); return e }()}}, "/elements/0/rom_note"},
		{"form quality out of range", SetEntry{Elements: []Element{func() Element { e := element(MeasureReps); e.FormQuality = ptr(6); return e }()}}, "/elements/0/form_quality"},
		{"completed set missing a value", SetEntry{CompletedAt: &now, Elements: []Element{element(MeasureReps)}}, "/elements/0/reps"},
		{"band without band", SetEntry{Elements: []Element{func() Element {
			e := element(MeasureReps)
			e.Assistance = &Assistance{ID: uuid.New(), Type: AssistanceTypeBand}
			return e
		}()}}, "/elements/0/assistance/band_id"},
		{"band count without band", SetEntry{Elements: []Element{func() Element {
			e := element(MeasureReps)
			e.Assistance = &Assistance{ID: uuid.New(), Type: "partner", BandCount: 2}
			return e
		}()}}, "/elements/0/assistance/band_count"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fields(t, ValidateSet(&tc.set))
			if _, ok := got[tc.field]; !ok {
				t.Fatalf("expected an error at %s, got %v", tc.field, got)
			}
		})
	}

	dup := element(MeasureNone)
	s := SetEntry{Elements: []Element{dup, dup}}
	if _, ok := fields(t, ValidateSet(&s))["/elements/1/id"]; !ok {
		t.Fatal("duplicate element ids were accepted")
	}
}

func TestClassOf(t *testing.T) {
	band := &Assistance{Type: AssistanceTypeBand}
	cases := []struct {
		e    Element
		want AssistanceClass
	}{
		{Element{}, Unassisted},
		{Element{LoadKg: 10}, Loaded},
		{Element{Assistance: band}, Assisted},
		{Element{LoadKg: 10, Assistance: band}, Assisted},
	}
	for _, tc := range cases {
		if got := ClassOf(tc.e); got != tc.want {
			t.Errorf("ClassOf(%+v) = %s, want %s", tc.e, got, tc.want)
		}
	}

	s := SetEntry{Elements: []Element{{ID: uuid.New(), Measure: MeasureNone, LoadKg: 20}}}
	if err := ValidateSet(&s); err != nil {
		t.Fatal(err)
	}
	if s.Elements[0].AssistanceClass != Loaded {
		t.Fatal("ValidateSet did not derive the assistance class")
	}
}

func TestLocalDateStaysOnTheAthletesDay(t *testing.T) {
	zurich, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	// 23:30 in Zurich is the next day in UTC.
	instant := time.Date(2026, 9, 23, 21, 30, 0, 0, time.UTC)
	if got := LocalDate(instant, zurich).Format("2006-01-02"); got != "2026-09-23" {
		t.Fatalf("got %s", got)
	}
	if got := LocalDate(instant.Add(time.Hour), zurich).Format("2006-01-02"); got != "2026-09-24" {
		t.Fatalf("got %s", got)
	}
}

func TestValidateBlockAndSession(t *testing.T) {
	if _, ok := fields(t, ValidateBlock(Block{Kind: "straight", IntervalS: ptr(60)}))["/interval_s"]; !ok {
		t.Error("interval on a straight block was accepted")
	}
	if err := ValidateBlock(Block{Kind: BlockKindEMOM, IntervalS: ptr(60)}); err != nil {
		t.Error(err)
	}

	start := time.Now()
	before := start.Add(-time.Minute)
	if _, ok := fields(t, ValidateSession(Session{StartedAt: start, EndedAt: &before}))["/ended_at"]; !ok {
		t.Error("session ending before it starts was accepted")
	}
	if _, ok := fields(t, ValidateSession(Session{StartedAt: start, PerceivedFatigue: ptr(11)}))["/perceived_fatigue"]; !ok {
		t.Error("fatigue 11 was accepted")
	}
}

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{StatusDraft, StatusDraft, true},
		{StatusDraft, StatusAbandoned, true},
		{StatusAbandoned, StatusDraft, true},
		{StatusDraft, StatusCompleted, false},
		{StatusCompleted, StatusDraft, false},
		{StatusCompleted, StatusAbandoned, false},
	}
	for _, tc := range cases {
		if got := CanTransition(tc.from, tc.to); got != tc.ok {
			t.Errorf("%s -> %s: got %v", tc.from, tc.to, got)
		}
	}
}

func TestValidateOrder(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	if err := ValidateOrder("/blocks", []uuid.UUID{b, a}, []uuid.UUID{a, b}); err != nil {
		t.Fatal(err)
	}
	for name, ids := range map[string][]uuid.UUID{
		"missing one": {a},
		"duplicate":   {a, a},
		"stranger":    {a, c},
	} {
		if ValidateOrder("/blocks", ids, []uuid.UUID{a, b}) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
