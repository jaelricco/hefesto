package planning_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// adaptAll applies events in order and returns the snapshot and the changes
// of the last one.
func adaptAll(t *testing.T, k *planning.Knowledge, s planning.Snapshot, evs ...planning.Event) (planning.Snapshot, []planning.Change) {
	t.Helper()
	var ch []planning.Change
	for _, ev := range evs {
		var err error
		s, ch, err = planning.Adapt(k, s, ev)
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, ch
}

func consentEvent(at time.Time, c planning.ConsentChange) planning.Event {
	return planning.Event{Kind: planning.EventConsent, At: at, Consent: &c}
}

func constraintsOf(s planning.Snapshot, region string) []string {
	var out []string
	for _, c := range s.Constraints {
		if c.Region == region {
			out = append(out, c.Kind)
		}
	}
	slices.Sort(out)
	return out
}

func hasChangeKind(cs []planning.Change, kind string) bool {
	return slices.ContainsFunc(cs, func(c planning.Change) bool { return c.Kind == kind })
}

// A withdrawal deletes the health data and keeps what protects the user as
// constraints (spec §4.9, §13.4, ENT-S-7).
func TestConsentWithdrawalKeepsProtection(t *testing.T) {
	k := kb(t)
	day := monday.AddDate(0, 0, 2)
	s, _ := start(t, k, persona3()) // elbow_inner in the ramp, with a pain baseline
	if len(s.Pain) == 0 || s.Regions["elbow_inner"].State != planning.StateRTT1 {
		t.Fatalf("persona 3 starts without pain or ramp: %+v", s.Regions)
	}
	// Lock a second region by a red flag.
	s, _ = adaptAll(t, k, s, planning.Event{Kind: planning.EventRedFlags, At: day, Region: "wrist_back_extension",
		Answers: map[string]bool{"RF-03": true}})
	s, ch := adaptAll(t, k, s, consentEvent(day, planning.ConsentChange{Granted: false}))

	if s.Profile.HealthConsent || len(s.Regions) != 0 || s.Pain != nil || s.Screening != (planning.Screening{}) {
		t.Fatalf("health data kept: consent %v, regions %v, pain %v, screening %+v", s.Profile.HealthConsent,
			s.Regions, s.Pain, s.Screening)
	}
	if got := constraintsOf(s, "elbow_inner"); !slices.Equal(got, []string{planning.ConstraintExcluded}) {
		t.Errorf("elbow after the withdrawal: %v, want excluded", got)
	}
	if got := constraintsOf(s, "wrist_back_extension"); !slices.Equal(got, []string{planning.ConstraintLocked}) {
		t.Errorf("wrist after the withdrawal: %v, want locked", got)
	}
	if !hasChangeKind(ch, planning.ChangeConsent) {
		t.Errorf("changes %+v name no consent change", ch)
	}
	p, err := planning.Generate(k, s, now, monday)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(p.Reasons, func(r planning.Reason) bool { return r.RuleID == "SAFE-04" }) {
		t.Error("the plan does not say it plans without consent")
	}
	if violations := planning.CheckInvariants(k, s, now, monday, p); len(violations) > 0 {
		t.Errorf("plan after the withdrawal: %v", violations)
	}

	// The same answer again changes nothing.
	again, ch := adaptAll(t, k, s, consentEvent(day, planning.ConsentChange{Granted: false}))
	if len(ch) != 0 || !slices.Equal(constraintsOf(again, "elbow_inner"), constraintsOf(s, "elbow_inner")) {
		t.Errorf("a second withdrawal changed %+v", ch)
	}
}

// A grant turns an exclusion into stage 0, which has an exit (spec §8.3),
// and takes the screening and the past injuries again.
func TestConsentGrantTracksExcludedRegions(t *testing.T) {
	k := kb(t)
	day := monday.AddDate(0, 0, 1)
	a := persona3()
	a.HealthConsent = false
	s, _ := start(t, k, a)
	if got := constraintsOf(s, "elbow_inner"); !slices.Equal(got, []string{planning.ConstraintExcluded}) {
		t.Fatalf("without consent the elbow is %v, want excluded", got)
	}
	s, ch := adaptAll(t, k, s, consentEvent(day, planning.ConsentChange{Granted: true,
		Screening: []bool{false, true, false, false, false, false}, PastInjuries: []string{"knee"}}))
	if !s.Profile.HealthConsent || !s.Screening.AnyYes {
		t.Fatalf("consent %v, screening %+v", s.Profile.HealthConsent, s.Screening)
	}
	elbow := s.Regions["elbow_inner"]
	if elbow.State != planning.StateRTT0 || !elbow.Complaint || len(constraintsOf(s, "elbow_inner")) != 0 {
		t.Errorf("elbow after the grant: %+v, constraints %v", elbow, constraintsOf(s, "elbow_inner"))
	}
	if !s.Regions["knee"].PriorInjury {
		t.Error("the past injury of the knee is not kept")
	}
	if !hasChangeKind(ch, planning.ChangeConsent) || !hasChangeKind(ch, planning.ChangeRegion) {
		t.Errorf("changes %+v", ch)
	}
	p, err := planning.Generate(k, s, now, monday)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(p.Reasons, func(r planning.Reason) bool { return r.RuleID == "SAFE-03" }) {
		t.Error("a yes in the screening allows tests again")
	}

	// Stage 0 ends as usual: green daily pain, then negative red flags.
	s, ch = adaptAll(t, k, s, planning.Event{Kind: planning.EventPain, At: day.AddDate(0, 0, 1),
		Pain: &planning.PainReport{ID: "p-1", Region: "elbow_inner", Timepoint: planning.PainDaily, NRS: 1, At: day.AddDate(0, 0, 1)}})
	if !hasChangeKind(ch, planning.ChangeRedFlags) {
		t.Fatalf("green daily pain in stage 0 does not ask the red flags: %+v", ch)
	}
	s, _ = adaptAll(t, k, s, planning.Event{Kind: planning.EventRedFlags, At: day.AddDate(0, 0, 1), Region: "elbow_inner",
		Answers: noFlags("elbow_inner")})
	if st := s.Regions["elbow_inner"].State; st != planning.StateRTT1 {
		t.Errorf("after negative red flags the elbow is %s, want rtt_1", st)
	}
}

// Without consent the planner keeps no state and cannot ramp: a cleared
// lock leaves the region out like stage 0, and a stage-0 red flag excludes
// it (SAFE-04).
func TestWithoutConsentRegionsStayOut(t *testing.T) {
	k := kb(t)
	day := monday.AddDate(0, 0, 1)
	a := persona2()
	a.HealthConsent = false
	s, _ := start(t, k, a)
	s, _ = adaptAll(t, k, s, planning.Event{Kind: planning.EventRedFlags, At: day, Region: "knee", Answers: map[string]bool{"RF-03": true}})
	if got := constraintsOf(s, "knee"); !slices.Equal(got, []string{planning.ConstraintLocked}) {
		t.Fatalf("knee after RF-03: %v", got)
	}
	s, _ = adaptAll(t, k, s, planning.Event{Kind: planning.EventClearance, At: day, Region: "knee"})
	if got := constraintsOf(s, "knee"); !slices.Equal(got, []string{planning.ConstraintExcluded}) {
		t.Errorf("knee after the clearance: %v, want excluded", got)
	}
	s, _ = adaptAll(t, k, s, planning.Event{Kind: planning.EventRedFlags, At: day, Region: "elbow_outer", Answers: map[string]bool{"RF-04": true}})
	if got := constraintsOf(s, "elbow_outer"); !slices.Equal(got, []string{planning.ConstraintExcluded}) {
		t.Errorf("elbow after RF-04: %v, want excluded", got)
	}

	// With consent a lock without a state ramps after its clearance.
	s, _ = adaptAll(t, k, s,
		planning.Event{Kind: planning.EventRedFlags, At: day, Region: "shoulder_front", Answers: map[string]bool{"RF-03": true}},
		consentEvent(day, planning.ConsentChange{Granted: true, Screening: make([]bool, 6)}),
		planning.Event{Kind: planning.EventClearance, At: day, Region: "shoulder_front"})
	if st := s.Regions["shoulder_front"].State; st != planning.StateRTT1 {
		t.Errorf("shoulder after the clearance with consent: %s, want rtt_1", st)
	}
}

func TestValidateConsent(t *testing.T) {
	k := kb(t)
	var ve *planning.ValidationError
	cases := []struct {
		c     planning.ConsentChange
		field string
	}{
		{planning.ConsentChange{Granted: true}, "/screening"},
		{planning.ConsentChange{Granted: true, Screening: make([]bool, 6), PastInjuries: []string{"toe"}}, "/past_injuries/0"},
		{planning.ConsentChange{Granted: true, Screening: make([]bool, 6), PastInjuries: []string{"knee", "knee"}}, "/past_injuries/1"},
		{planning.ConsentChange{Screening: make([]bool, 6)}, "/screening"},
		{planning.ConsentChange{PastInjuries: []string{"knee"}}, "/past_injuries"},
	}
	for _, c := range cases {
		if err := k.ValidateConsent(c.c); !errors.As(err, &ve) || ve.Fields[c.field] == "" {
			t.Errorf("%+v: %v, want an error at %s", c.c, err, c.field)
		}
	}
	for _, c := range []planning.ConsentChange{{}, {Granted: true, Screening: make([]bool, 6), PastInjuries: []string{"knee"}}} {
		if err := k.ValidateConsent(c); err != nil {
			t.Errorf("%+v: %v", c, err)
		}
	}
}
