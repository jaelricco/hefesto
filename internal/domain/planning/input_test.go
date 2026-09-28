package planning_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// Invalid answers name the field by a JSON pointer into the request body,
// as the API reports them (spec §10.4).
func TestAnswerValidation(t *testing.T) {
	k := kb(t)
	past := monday.AddDate(0, 0, -1)
	cases := []struct {
		name  string
		edit  func(a *planning.Answers)
		field string
	}{
		{"disclaimer", func(a *planning.Answers) { a.DisclaimerAck = false }, "/disclaimer_ack"},
		{"birth year", func(a *planning.Answers) { a.BirthYear = 1900 }, "/birth_year"},
		{"no goal", func(a *planning.Answers) { a.Goals = nil }, "/goals"},
		{"unknown skill", func(a *planning.Answers) { a.Goals[0].Skill = "flag" }, "/goals/0/skill"},
		{"unknown level", func(a *planning.Answers) { a.Goals[0].TargetLevel = "one-arm" }, "/goals/0/target_level"},
		{"skill twice", func(a *planning.Answers) {
			a.Goals = append(a.Goals, planning.Goal{Skill: "planche", TargetLevel: "tuck", Priority: 2})
		}, "/goals/1/skill"},
		{"priority twice", func(a *planning.Answers) {
			a.Goals = append(a.Goals, planning.Goal{Skill: "front-lever", TargetLevel: "full", Priority: 1})
		}, "/goals/1/priority"},
		{"past date", func(a *planning.Answers) { a.Goals[0].TargetDate = &past }, "/goals/0/target_date"},
		{"sessions", func(a *planning.Answers) { a.SessionsPerWeek = 8 }, "/sessions_per_week"},
		{"minutes", func(a *planning.Answers) { a.SessionMinutes = 50 }, "/session_minutes"},
		{"bodyweight", func(a *planning.Answers) { a.BodyweightKg = 20 }, "/bodyweight_kg"},
		{"equipment", func(a *planning.Answers) { a.Equipment = []string{"gym", "trampoline"} }, "/equipment/1"},
		{"day twice", func(a *planning.Answers) { a.PreferredDays = []time.Weekday{time.Monday, time.Monday} }, "/preferred_days/1"},
		{"training level", func(a *planning.Answers) { a.TrainingLevel = "pro" }, "/training_level"},
		{"training age", func(a *planning.Answers) { a.TrainingAge = "10_years" }, "/calisthenics_training_age"},
		{"last training", func(a *planning.Answers) { a.LastRegular = "" }, "/last_regular_training"},
		{"data confidence", func(a *planning.Answers) { a.DataConfidence = "guessed" }, "/data_confidence"},
		{"unknown question", func(a *planning.Answers) { a.Classes["squat_class"] = "0" }, "/classes/squat_class"},
		{"unknown class", func(a *planning.Answers) { a.Classes["push_up_class"] = "31_40" }, "/classes/push_up_class"},
		{"stage level", func(a *planning.Answers) {
			a.Stages["planche"] = planning.StageAnswer{Level: "full-straddle", Class: "4_9"}
		}, "/skill_stages/planche/level"},
		{"stage class", func(a *planning.Answers) {
			a.Stages["planche"] = planning.StageAnswer{Level: "tuck", Class: "30_60"}
		}, "/skill_stages/planche/class"},
		{"pre-break level", func(a *planning.Answers) { a.PreBreak = map[string]string{"planche": "maltese"} }, "/pre_break_level/planche"},
		{"mobility check", func(a *planning.Answers) { a.Mobility = map[string]string{"hip": "yes"} }, "/mobility_checks/hip"},
		{"mobility answer", func(a *planning.Answers) { a.Mobility = map[string]string{"wrist": "maybe"} }, "/mobility_checks/wrist"},
		{"screening", func(a *planning.Answers) { a.Screening = a.Screening[:5] }, "/screening"},
		{"unknown region", func(a *planning.Answers) { a.BodyMap["toe"] = planning.BodyMapEntry{Current: true} }, "/body_map/toe"},
		{"complaint missing", func(a *planning.Answers) { delete(a.Complaints, "elbow_inner") }, "/complaints/elbow_inner"},
		{"complaint not current", func(a *planning.Answers) {
			a.Complaints["knee"] = a.Complaints["elbow_inner"]
		}, "/complaints/knee"},
		{"pain above 10", func(a *planning.Answers) {
			c := a.Complaints["elbow_inner"]
			c.PainDaily = 11
			a.Complaints["elbow_inner"] = c
		}, "/complaints/elbow_inner/pain_daily"},
		{"restriction", func(a *planning.Answers) {
			c := a.Complaints["elbow_inner"]
			c.Restrictions = []string{"overhead", "jumping"}
			a.Complaints["elbow_inner"] = c
		}, "/complaints/elbow_inner/restrictions/1"},
		{"red flags missing", func(a *planning.Answers) { delete(a.RedFlags, "elbow_inner") }, "/red_flags/elbow_inner"},
		{"red flag unanswered", func(a *planning.Answers) { delete(a.RedFlags["elbow_inner"], "RF-07") }, "/red_flags/elbow_inner/RF-07"},
		{"red flag not asked", func(a *planning.Answers) { a.RedFlags["elbow_inner"]["RF-08"] = false }, "/red_flags/elbow_inner/RF-08"},
		{"follow-up after no", func(a *planning.Answers) {
			a.RedFlags["elbow_inner"]["RF-05-displaced"] = false
		}, "/red_flags/elbow_inner/RF-05-displaced"},
		{"follow-up missing", func(a *planning.Answers) { a.RedFlags["elbow_inner"]["RF-05"] = true }, "/red_flags/elbow_inner/RF-05-displaced"},
		{"clarification", func(a *planning.Answers) { a.Clarifications = map[string]bool{"R-9/planche": true} }, "/clarifications/R-9~1planche"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := persona3()
			c.edit(&a)
			_, _, err := planning.Start(k, a, now)
			var ve *planning.ValidationError
			if !errors.As(err, &ve) || !errors.Is(err, planning.ErrInvalidAnswers) {
				t.Fatalf("got %v, want a validation error", err)
			}
			if _, ok := ve.Fields[c.field]; !ok {
				t.Errorf("fields %v do not name %s", ve.Fields, c.field)
			}
		})
	}
	if _, _, err := planning.Start(k, persona3(), now); err != nil {
		t.Fatalf("the unchanged persona fails: %v", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	before, _ := json.Marshal(s)
	u := planning.ProfileUpdate{SessionsPerWeek: 2, SessionMinutes: 45, Equipment: []string{"outdoor_park"}, BodyweightKg: 70,
		PreferredDays: []time.Weekday{time.Tuesday, time.Saturday}, Mobility: map[string]string{"wrist": "partly"},
		MaxAddedLoadKg: 20}
	got, err := planning.UpdateProfile(s, u)
	if err != nil {
		t.Fatal(err)
	}
	p := got.Profile
	if p.SessionsPerWeek != 2 || p.SessionMinutes != 45 || p.BodyweightKg != 70 || p.MaxAddedLoadKg != 20 ||
		p.Mobility["wrist"] != "partly" || !slices.Equal(p.PreferredDays, u.PreferredDays) {
		t.Errorf("profile not applied: %+v", p)
	}
	if want := []string{"floor", "low_bar", "outdoor_park", "parallel_bars", "pull_up_bar", "wall"}; !slices.Equal(p.Equipment, want) {
		t.Errorf("equipment %v, want %v", p.Equipment, want)
	}
	if p.BirthYear != s.Profile.BirthYear || p.HealthConsent != s.Profile.HealthConsent || !reflect.DeepEqual(got.Goals, s.Goals) {
		t.Error("the update changed more than the profile's editable part")
	}
	if after, _ := json.Marshal(s); string(after) != string(before) {
		t.Error("the input snapshot was modified")
	}

	u.SessionMinutes, u.SmallestPlateKg = 50, -1
	_, err = planning.UpdateProfile(s, u)
	var ve *planning.ValidationError
	if !errors.As(err, &ve) || ve.Fields["/session_minutes"] == "" || ve.Fields["/smallest_plate_kg"] == "" {
		t.Errorf("got %v, want errors for session_minutes and smallest_plate_kg", err)
	}
}

func TestSetGoals(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	date := monday.AddDate(0, 6, 0).Add(15 * time.Hour)
	goals := []planning.Goal{
		{Skill: "handstand", TargetLevel: "free-30s", Priority: 2, TargetDate: &date},
		{Skill: "planche", TargetLevel: "full", Priority: 1},
	}
	got, err := planning.SetGoals(k, s, goals, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Goals[0].Skill != "planche" || got.Goals[1].Skill != "handstand" {
		t.Errorf("goals not in priority order: %+v", got.Goals)
	}
	if d := got.Goals[1].TargetDate; d == nil || !d.Equal(monday.AddDate(0, 6, 0)) {
		t.Errorf("target date %v is not the calendar day", d)
	}
	if !reflect.DeepEqual(got.Ladders, s.Ladders) {
		t.Error("ladders changed with the goals")
	}
	if r := k.Realism(got, now); len(r) != 1 || r[0].Skill != "handstand" {
		t.Errorf("realism %+v, want the dated handstand goal", r)
	}
	_, err = planning.SetGoals(k, s, append(goals, planning.Goal{Skill: "l-sit", TargetLevel: "full", Priority: 3},
		planning.Goal{Skill: "front-lever", TargetLevel: "full", Priority: 4}), now)
	var ve *planning.ValidationError
	if !errors.As(err, &ve) || ve.Fields["/goals"] == "" {
		t.Errorf("four goals: got %v", err)
	}
}

// A red-flag answer from the API names every question asked for the region
// (onboarding.md §3.7).
func TestValidateRedFlags(t *testing.T) {
	k := kb(t)
	if err := k.ValidateRedFlags("elbow_inner", noFlags("elbow_inner"), false); err != nil {
		t.Errorf("all answered: %v", err)
	}
	var ve *planning.ValidationError
	err := k.ValidateRedFlags("elbow_inner", map[string]bool{}, false)
	if !errors.As(err, &ve) || ve.Fields["/answers/RF-01"] == "" {
		t.Errorf("none answered: %v", err)
	}
	minor := noFlags("wrist_back_extension")
	if err := k.ValidateRedFlags("wrist_back_extension", minor, true); err == nil {
		t.Error("a minor's wrist is also asked RF-12")
	}
	if err := k.ValidateRedFlags("toe", nil, false); !errors.As(err, &ve) || ve.Fields["/region"] == "" {
		t.Errorf("unknown region: %v", err)
	}
}

func TestRegionStatuses(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona3())
	st := k.RegionStatuses(s, now)
	if len(st) != 1 || st[0].Region.ID != "elbow_inner" || !st[0].Tracked || st[0].State.State != planning.StateRTT1 {
		t.Fatalf("statuses %+v, want elbow_inner in rtt_1", st)
	}
	if len(st[0].RedFlags) != 8 {
		t.Errorf("%d red-flag questions, want the 8 asked at the elbow", len(st[0].RedFlags))
	}

	// Without consent no state is kept; the exclusion still holds.
	a := persona3()
	a.HealthConsent = false
	s, _ = start(t, k, a)
	st = k.RegionStatuses(s, now)
	if len(st) != 1 || st[0].Tracked || !slices.Equal(st[0].Constraints, []string{planning.ConstraintExcluded}) {
		t.Errorf("without consent: %+v", st)
	}
}

func TestCatalogueOrder(t *testing.T) {
	k := kb(t)
	src := k.Sources()
	if len(src) == 0 {
		t.Fatal("no sources")
	}
	for i := 1; i < len(src); i++ {
		a, b := src[i-1].ID, src[i].ID
		if a[0] == b[0] && len(a) > len(b) {
			t.Fatalf("%s before %s: IDs are not in numeric order", a, b)
		}
	}
	if len(k.Rules()) == 0 || len(k.Params()) == 0 || len(k.Skills()) == 0 || len(k.Exercises()) == 0 || len(k.Regions()) == 0 {
		t.Error("an empty catalogue")
	}
}

// The equipment vocabulary of the answers is the knowledge base's.
func TestEquipmentMatchesKnowledgeSchema(t *testing.T) {
	raw, err := os.ReadFile("../../../content/schema/training/common.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs struct {
			Equipment struct {
				Enum []string `json:"enum"`
			} `json:"equipment"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if got := planning.Vocabularies()["equipment"]; !slices.Equal(got, schema.Defs.Equipment.Enum) {
		t.Errorf("equipment %v, knowledge base %v", got, schema.Defs.Equipment.Enum)
	}
}
