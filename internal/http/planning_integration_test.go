//go:build integration

package http_test

import (
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	lhttp "github.com/jaelricco/hefesto/internal/http"
	"github.com/jaelricco/hefesto/internal/planning"
	"github.com/jaelricco/hefesto/internal/store"
	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

// plannerMonday is the planner's clock in these tests: plans depend on the
// day, so the tests fix it.
var plannerMonday = time.Date(2026, time.September, 28, 7, 0, 0, 0, time.UTC)

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now() time.Time { return c.t }

// withPlanner gives the server the planner on its database, with the
// repository's knowledge base and a fixed clock.
func withPlanner(clock *fixedClock) option {
	return func(t *testing.T, a *api, deps *lhttp.RouterDeps) {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		deps.Planner = &planning.Service{
			Knowledge: planning.LoadContentKnowledge(filepath.Join(pgtest.RepoRoot(), "content"), false, log),
			Store:     store.NewPlanner(a.pool), Clock: clock, Log: log,
		}
	}
}

// noFlags answers every red-flag question of an adult's arm region with no.
func noFlags(yes ...string) map[string]any {
	out := map[string]any{}
	for _, id := range []string{"RF-01", "RF-02", "RF-03", "RF-04", "RF-05", "RF-06", "RF-07", "RF-10"} {
		out[id] = false
	}
	for _, id := range yes {
		out[id] = true
	}
	return out
}

// onboarding is an advanced athlete with a current complaint at the inner
// elbow and the planche as goal.
func onboarding() map[string]any {
	return map[string]any{
		"birth_year": 1995, "health_data_consent": true, "disclaimer_ack": true,
		"goals":             []any{map[string]any{"skill": "planche", "target_level": "full", "priority": 1, "target_date": "2027-09-27"}},
		"sessions_per_week": 3, "session_minutes": 60, "equipment": []any{"gym", "rings", "resistance_bands"},
		"bodyweight_kg": 75, "training_level": "trained", "calisthenics_training_age": "1_to_4_years",
		"last_regular_training": "current_or_lt_3_weeks", "data_confidence": "estimated",
		"classes":           map[string]any{"push_up_class": "21_30", "pull_up_class": "12_15", "dip_class": "13_20"},
		"skill_stages":      map[string]any{"planche": map[string]any{"level": "tuck", "class": "10_19"}},
		"mobility_checks":   map[string]any{"wrist": "yes"},
		"exertion_symptoms": false, "screening": []any{false, false, false, false, false, false},
		"body_map": map[string]any{"elbow_inner": map[string]any{"current": true, "past_12_months": false}},
		"complaints": map[string]any{"elbow_inner": map[string]any{"pain_daily": 1.5, "pain_training": 3.5, "onset": "gradual",
			"duration_weeks": 2, "suspected_serious": "no", "professional_assessment": "no"}},
		"red_flags":      map[string]any{"elbow_inner": noFlags()},
		"preferred_days": []any{1, 3, 5},
	}
}

func onboarded(t *testing.T, a *api, email string) user {
	t.Helper()
	u := a.register(email)
	a.call("POST", "/v1/me/onboarding", u.access, onboarding()).ok(200, "OnboardingResult")
	return u
}

func TestPlannerWithoutKnowledgeIsUnavailable(t *testing.T) {
	a := newAPI(t) // no planner
	u := a.register("noplanner@example.com")
	a.call("GET", "/v1/me/plan", u.access, nil).problem(503, "planning-unavailable")
	a.call("GET", "/v1/planner/rules", "", nil).problem(503, "planning-unavailable")
	a.call("POST", "/v1/me/onboarding", u.access, onboarding()).problem(503, "planning-unavailable")
}

func TestPlannerCatalogueIsPublicAndCached(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	for path, schema := range map[string]string{
		"/v1/planner/rules": "PlannerRuleList", "/v1/planner/sources": "PlannerSourceList",
		"/v1/planner/parameters": "PlannerParameterList", "/v1/planner/catalogue": "PlannerCatalogue",
	} {
		r := a.call("GET", path, "", nil)
		body := r.ok(200, schema)
		etag := r.header.Get("ETag")
		if etag != `"`+body["ruleset_version"].(string)+`"` {
			t.Errorf("%s: ETag %q, ruleset %v", path, etag, body["ruleset_version"])
		}
		a.callWith("GET", path, "", map[string]string{"If-None-Match": etag}).ok(304, "")
	}
	cat := a.call("GET", "/v1/planner/catalogue", "", nil).ok(200, "PlannerCatalogue")
	for _, r := range cat["regions"].([]any) {
		for _, f := range r.(map[string]any)["red_flags"].([]any) {
			if f.(map[string]any)["question"] == "" {
				t.Fatal("a red flag without its question")
			}
		}
	}
}

func TestOnboardingFlow(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := a.register("onboarding@example.com")

	a.call("GET", "/v1/me/plan", u.access, nil).problem(409, "onboarding-required")
	a.call("GET", "/v1/me/training-profile", u.access, nil).problem(409, "onboarding-required")

	// Invalid answers name their fields.
	bad := onboarding()
	bad["goals"] = []any{map[string]any{"skill": "planche", "target_level": "one-arm", "priority": 1}}
	bad["red_flags"] = map[string]any{"elbow_inner": map[string]any{"RF-01": false}}
	errs := a.call("POST", "/v1/me/onboarding", u.access, bad).problem(422, "validation")
	hasField(t, errs, "/goals/0/target_level")
	hasField(t, errs, "/red_flags/elbow_inner/RF-07")
	errs = a.call("POST", "/v1/me/onboarding", u.access, map[string]any{"birth_year": 1995}).problem(422, "validation")
	if len(errs) == 0 {
		t.Fatal("a body missing required answers passed the schema")
	}

	// An implausible stage asks back and stores nothing.
	ask := onboarding()
	ask["training_level"] = "sedentary"
	ask["skill_stages"] = map[string]any{"planche": map[string]any{"level": "full", "class": "4_9"}}
	res := a.call("POST", "/v1/me/onboarding", u.access, ask).ok(200, "OnboardingResult")
	if res["status"] != "needs_answers" || res["plan"] != nil || len(res["questions"].([]any)) == 0 {
		t.Fatalf("clarification: %v", res)
	}
	a.call("GET", "/v1/me/plan", u.access, nil).problem(409, "onboarding-required")

	// With an Idempotency-Key a retry gets the first response.
	key := map[string]string{"Idempotency-Key": "onboarding-" + newID()}
	first := a.callWith("POST", "/v1/me/onboarding", u.access, key, onboarding())
	body := first.ok(200, "OnboardingResult")
	if body["status"] != "complete" || body["plan"] == nil {
		t.Fatalf("onboarding: %v", body)
	}
	if body["disclaimer"] == "" {
		t.Error("no disclaimer in the result")
	}
	again := a.callWith("POST", "/v1/me/onboarding", u.access, key, onboarding())
	// The stored response is jsonb: the same document, not the same bytes.
	if !reflect.DeepEqual(again.ok(200, "OnboardingResult"), body) || again.header.Get("Idempotent-Replayed") != "true" {
		t.Error("the retry is not the first response")
	}
	a.callWith("POST", "/v1/me/onboarding", u.access, key, ask).problem(422, "idempotency-key-reuse")
	a.call("POST", "/v1/me/onboarding", u.access, onboarding()).problem(409, "already-onboarded")

	// The plan names every exercise and keeps sources off regional reasons
	// (EXPL-07).
	plan := body["plan"].(map[string]any)
	for _, s := range plan["sessions"].([]any) {
		for _, b := range s.(map[string]any)["blocks"].([]any) {
			for _, it := range b.(map[string]any)["items"].([]any) {
				item := it.(map[string]any)
				if item["exercise_name"] == "" {
					t.Errorf("item %v without a name", item["exercise"])
				}
				for _, r := range item["reasons"].([]any) {
					reason := r.(map[string]any)
					if reason["region"] != nil && len(reason["sources"].([]any)) > 0 {
						t.Errorf("regional reason %v shows sources", reason["rule_id"])
					}
				}
			}
		}
	}
}

func TestTrainingPlanEndpoints(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := onboarded(t, a, "plan@example.com")

	r := a.call("GET", "/v1/me/plan", u.access, nil)
	plan := r.ok(200, "TrainingPlan")
	if r.header.Get("ETag") != `"`+plan["id"].(string)+`"` {
		t.Errorf("ETag %q is not the plan %v", r.header.Get("ETag"), plan["id"])
	}
	a.callWith("GET", "/v1/me/plan", u.access, map[string]string{"If-None-Match": r.header.Get("ETag")}).ok(304, "")
	a.call("GET", "/v1/me/plan?week=2026-09-30", u.access, nil).ok(200, "TrainingPlan")
	a.call("GET", "/v1/me/plan?week=2026-09-21", u.access, nil).problem(404, "not-found")
	a.call("GET", "/v1/me/plan?week=monday", u.access, nil).problem(400, "bad-request")

	sessions := plan["sessions"].([]any)
	if len(sessions) == 0 {
		t.Fatal("no sessions planned")
	}
	sid := sessions[0].(map[string]any)["id"].(string)
	detail := a.call("GET", "/v1/me/plan/sessions/"+sid, u.access, nil).ok(200, "PlannedSessionDetail")
	if detail["plan_id"] != plan["id"] {
		t.Errorf("session of plan %v, want %v", detail["plan_id"], plan["id"])
	}
	a.call("GET", "/v1/me/plan/sessions/"+newID(), u.access, nil).problem(404, "not-found")
	a.call("GET", "/v1/me/plan/sessions/nope", u.access, nil).problem(400, "bad-request")
	other := a.register("other-plan@example.com")
	a.call("GET", "/v1/me/plan/sessions/"+sid, other.access, nil).problem(404, "not-found")

	regen := a.call("POST", "/v1/me/plan/regenerate", u.access, nil).ok(200, "TrainingPlan")
	if regen["id"] == plan["id"] || regen["input_hash"] != plan["input_hash"] {
		t.Errorf("regenerated %v (%v) from %v (%v)", regen["id"], regen["input_hash"], plan["id"], plan["input_hash"])
	}
	a.call("GET", "/v1/me/plan/sessions/"+sid, u.access, nil).problem(404, "not-found")
}

func TestTrainingProfileAndGoals(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := onboarded(t, a, "profile@example.com")

	p := a.call("GET", "/v1/me/training-profile", u.access, nil).ok(200, "TrainingProfile")
	if p["is_minor"] != false || p["max_added_load_kg"] != nil {
		t.Errorf("profile %v", p)
	}
	days := p["preferred_days"].([]any)
	if len(days) != 3 || days[0] != float64(1) || days[2] != float64(5) {
		t.Errorf("preferred days %v, want ISO 1, 3, 5", days)
	}
	p = a.call("PUT", "/v1/me/training-profile", u.access, map[string]any{
		"sessions_per_week": 2, "session_minutes": 45, "equipment": []any{"outdoor_park"}, "bodyweight_kg": 74,
		"preferred_days": []any{2, 7}, "max_added_load_kg": 20,
	}).ok(200, "TrainingProfile")
	if p["sessions_per_week"] != float64(2) || p["max_added_load_kg"] != float64(20) || p["mobility_checks"].(map[string]any)["wrist"] != nil {
		t.Errorf("updated profile %v", p)
	}
	if got := p["preferred_days"].([]any); len(got) != 2 || got[1] != float64(7) {
		t.Errorf("Sunday is %v, want 7", got)
	}
	plan := a.call("GET", "/v1/me/plan", u.access, nil).ok(200, "TrainingPlan")
	if n := len(plan["sessions"].([]any)); n > 2 {
		t.Errorf("%d sessions after cutting to two a week", n)
	}
	errs := a.call("PUT", "/v1/me/training-profile", u.access, map[string]any{
		"sessions_per_week": 3, "session_minutes": 50, "equipment": []any{}, "bodyweight_kg": 74,
	}).problem(422, "validation")
	hasField(t, errs, "/session_minutes")

	g := a.call("GET", "/v1/me/goals", u.access, nil).ok(200, "TrainingGoalList")
	if len(g["goals"].([]any)) != 1 || len(g["realism"].([]any)) != 1 {
		t.Errorf("goals %v", g)
	}
	g = a.call("PUT", "/v1/me/goals", u.access, map[string]any{"goals": []any{
		map[string]any{"skill": "handstand", "target_level": "free-30s", "priority": 2},
		map[string]any{"skill": "front-lever", "target_level": "full", "priority": 1},
	}}).ok(200, "TrainingGoalList")
	if first := g["goals"].([]any)[0].(map[string]any); first["skill"] != "front-lever" || first["target_date"] != nil {
		t.Errorf("goals in priority order: %v", g["goals"])
	}
	errs = a.call("PUT", "/v1/me/goals", u.access, map[string]any{"goals": []any{
		map[string]any{"skill": "planche", "target_level": "full", "priority": 2},
	}}).problem(422, "validation")
	hasField(t, errs, "/goals/0/priority")
}

func TestPainRegionsAndClearances(t *testing.T) {
	clock := &fixedClock{plannerMonday}
	a := newAPI(t, withPlanner(clock))
	u := onboarded(t, a, "pain@example.com")

	regions := a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")
	list := regions["regions"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["region"] != "elbow_inner" || list[0].(map[string]any)["tracked"] != true {
		t.Fatalf("regions %v", list)
	}

	// Pain asks the red-flag questions; a repeat is answered from the log.
	report := map[string]any{"id": newID(), "region": "elbow_inner", "timepoint": "during", "nrs": 6,
		"at": plannerMonday.Format(time.RFC3339)}
	out := a.call("POST", "/v1/me/pain-reports", u.access, report).ok(200, "PlanEventResult")
	if out["replayed"] != false || !hasChange(out, "ask_red_flags") {
		t.Fatalf("pain report: %v", out)
	}
	again := a.call("POST", "/v1/me/pain-reports", u.access, report).ok(200, "PlanEventResult")
	if again["replayed"] != true || !hasChange(again, "ask_red_flags") {
		t.Fatalf("repeat: %v", again)
	}
	unknown := map[string]any{"id": newID(), "region": "toe", "timepoint": "during", "nrs": 2, "at": plannerMonday.Format(time.RFC3339)}
	hasField(t, a.call("POST", "/v1/me/pain-reports", u.access, unknown).problem(422, "validation"), "/region")
	hasField(t, a.call("POST", "/v1/me/pain-reports", u.access, map[string]any{"id": newID()}).problem(422, "validation"), "")

	clock.t = plannerMonday.Add(time.Hour)
	second := map[string]any{"id": newID(), "region": "elbow_inner", "timepoint": "after", "nrs": 1,
		"at": clock.t.Format(time.RFC3339), "session_id": newID()}
	a.call("POST", "/v1/me/pain-reports", u.access, second).ok(200, "PlanEventResult")
	pg := a.call("GET", "/v1/me/pain-reports?limit=1", u.access, nil).ok(200, "PainReportPage")
	items := pg["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != second["id"] || pg["next_cursor"] == nil {
		t.Fatalf("first page %v", pg)
	}
	pg = a.call("GET", "/v1/me/pain-reports?limit=10&cursor="+pg["next_cursor"].(string), u.access, nil).ok(200, "PainReportPage")
	if n := len(pg["items"].([]any)); n != 2 || pg["next_cursor"] != nil { // the report and the onboarding's baseline
		t.Fatalf("second page has %d reports: %v", n, pg)
	}

	// Red flags: every question answered, the region known.
	a.call("POST", "/v1/me/regions/toe/red-flags", u.access, map[string]any{"id": newID(), "answers": noFlags()}).problem(404, "not-found")
	a.call("POST", "/v1/me/regions/Elbow/red-flags", u.access, map[string]any{"id": newID(), "answers": noFlags()}).problem(400, "bad-request")
	hasField(t, a.call("POST", "/v1/me/regions/elbow_inner/red-flags", u.access,
		map[string]any{"id": newID(), "answers": map[string]any{"RF-01": false}}).problem(422, "validation"), "/answers/RF-02")
	out = a.call("POST", "/v1/me/regions/elbow_inner/red-flags", u.access,
		map[string]any{"id": newID(), "answers": noFlags("RF-03")}).ok(200, "PlanEventResult")
	if !hasChange(out, "region_state") {
		t.Fatalf("a D red flag does not lock the region: %v", out)
	}
	regions = a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")
	elbow := regions["regions"].([]any)[0].(map[string]any)
	if elbow["state"] != "locked" || len(elbow["constraints"].([]any)) != 1 {
		t.Fatalf("elbow after RF-03: %v", elbow)
	}
	a.call("POST", "/v1/me/regions/elbow_inner/clearance", u.access, map[string]any{"id": newID()}).ok(200, "PlanEventResult")
	regions = a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")
	if elbow = regions["regions"].([]any)[0].(map[string]any); elbow["state"] == "locked" {
		t.Fatalf("elbow after the clearance: %v", elbow)
	}

	// Exertion symptoms stop training until the screening clearance.
	out = a.call("POST", "/v1/me/symptoms", u.access, map[string]any{"id": newID()}).ok(200, "PlanEventResult")
	if !hasChange(out, "training_stopped") {
		t.Fatalf("symptoms: %v", out)
	}
	if p := a.call("GET", "/v1/me/plan", u.access, nil).ok(200, "TrainingPlan"); p["stopped"] != true || len(p["sessions"].([]any)) != 0 {
		t.Fatalf("plan after symptoms: stopped %v, %d sessions", p["stopped"], len(p["sessions"].([]any)))
	}
	a.call("POST", "/v1/me/screening/clearance", u.access, map[string]any{"id": newID()}).ok(200, "PlanEventResult")
	if p := a.call("GET", "/v1/me/plan", u.access, nil).ok(200, "TrainingPlan"); p["stopped"] != false {
		t.Fatal("still stopped after the clearance")
	}

	// The decision log pages newest first.
	d := a.call("GET", "/v1/me/plan/decisions?limit=2", u.access, nil).ok(200, "PlanDecisionPage")
	first := d["items"].([]any)
	if len(first) != 2 || first[0].(map[string]any)["trigger"] != "clearance" || d["next_cursor"] == nil {
		t.Fatalf("decisions %v", d)
	}
	d = a.call("GET", "/v1/me/plan/decisions?limit=100&cursor="+d["next_cursor"].(string), u.access, nil).ok(200, "PlanDecisionPage")
	if n := len(d["items"].([]any)); n != 4 || d["next_cursor"] != nil {
		t.Fatalf("rest of the log: %d decisions, %v", n, d)
	}
	a.call("GET", "/v1/me/plan/decisions?cursor=x", u.access, nil).problem(400, "bad-request")

	caps := a.call("GET", "/v1/me/capacity", u.access, nil).ok(200, "CapacityList")
	if len(caps["items"].([]any)) == 0 {
		t.Fatal("no capacities")
	}
}

func TestPainReportsNeedConsent(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := a.register("noconsent@example.com")
	body := onboarding()
	body["health_data_consent"] = false
	a.call("POST", "/v1/me/onboarding", u.access, body).ok(200, "OnboardingResult")
	a.call("POST", "/v1/me/pain-reports", u.access, map[string]any{"id": newID(), "region": "elbow_inner",
		"timepoint": "during", "nrs": 2, "at": plannerMonday.Format(time.RFC3339)}).problem(409, "consent-required")
	regions := a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")
	elbow := regions["regions"].([]any)[0].(map[string]any)
	if elbow["tracked"] != false || elbow["state"] != nil || elbow["constraints"].([]any)[0] != "region_excluded" {
		t.Fatalf("without consent: %v", elbow)
	}
	pg := a.call("GET", "/v1/me/pain-reports", u.access, nil).ok(200, "PainReportPage")
	if len(pg["items"].([]any)) != 0 {
		t.Fatalf("pain kept without consent: %v", pg)
	}
}

func hasChange(result map[string]any, kind string) bool {
	for _, c := range result["changes"].([]any) {
		if c.(map[string]any)["kind"] == kind {
			return true
		}
	}
	return false
}

// A withdrawal deletes the health data and keeps the protection; a grant
// asks the screening again and tracks the excluded region in stage 0.
func TestHealthConsentWithdrawAndGrant(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := onboarded(t, a, "consent@example.com")
	a.call("POST", "/v1/me/pain-reports", u.access, map[string]any{"id": newID(), "region": "elbow_inner",
		"timepoint": "daily", "nrs": 2, "at": plannerMonday.Format(time.RFC3339)}).ok(200, "PlanEventResult")

	withdraw := map[string]any{"id": newID(), "granted": false}
	out := a.call("POST", "/v1/me/health-consent", u.access, withdraw).ok(200, "PlanEventResult")
	if !hasChange(out, "consent_changed") {
		t.Fatalf("withdrawal: %v", out)
	}
	if again := a.call("POST", "/v1/me/health-consent", u.access, withdraw).ok(200, "PlanEventResult"); again["replayed"] != true {
		t.Errorf("repeat: %v", again)
	}
	if p := a.call("GET", "/v1/me/training-profile", u.access, nil).ok(200, "TrainingProfile"); p["health_data_consent"] != false {
		t.Errorf("profile after the withdrawal: %v", p["health_data_consent"])
	}
	elbow := a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")["regions"].([]any)[0].(map[string]any)
	if elbow["tracked"] != false || elbow["constraints"].([]any)[0] != "region_excluded" {
		t.Errorf("elbow after the withdrawal: %v", elbow)
	}
	if pg := a.call("GET", "/v1/me/pain-reports", u.access, nil).ok(200, "PainReportPage"); len(pg["items"].([]any)) != 0 {
		t.Errorf("pain reports kept: %v", pg["items"])
	}
	a.call("POST", "/v1/me/pain-reports", u.access, map[string]any{"id": newID(), "region": "elbow_inner",
		"timepoint": "daily", "nrs": 1, "at": plannerMonday.Format(time.RFC3339)}).problem(409, "consent-required")

	hasField(t, a.call("POST", "/v1/me/health-consent", u.access, map[string]any{"id": newID(), "granted": true}).
		problem(422, "validation"), "/screening")
	hasField(t, a.call("POST", "/v1/me/health-consent", u.access, map[string]any{"id": newID(), "granted": false,
		"screening": []any{false, false, false, false, false, false}}).problem(422, "validation"), "/screening")
	hasField(t, a.call("POST", "/v1/me/health-consent", u.access, map[string]any{"id": newID(), "granted": true,
		"screening": []any{false, false, false, false, false, false}, "past_injuries": []any{"toe"}}).problem(422, "validation"),
		"/past_injuries/0")

	out = a.call("POST", "/v1/me/health-consent", u.access, map[string]any{"id": newID(), "granted": true,
		"screening": []any{false, false, false, false, false, false}, "past_injuries": []any{"knee"}}).ok(200, "PlanEventResult")
	if !hasChange(out, "consent_changed") || !hasChange(out, "region_state") {
		t.Fatalf("grant: %v", out)
	}
	regions := a.call("GET", "/v1/me/regions", u.access, nil).ok(200, "RegionOverview")["regions"].([]any)
	states := map[string]any{}
	for _, r := range regions {
		states[r.(map[string]any)["region"].(string)] = r.(map[string]any)["state"]
	}
	if states["elbow_inner"] != "rtt_0" || states["knee"] != "normal" {
		t.Errorf("regions after the grant: %v", states)
	}
	a.call("POST", "/v1/me/pain-reports", u.access, map[string]any{"id": newID(), "region": "elbow_inner",
		"timepoint": "daily", "nrs": 1, "at": plannerMonday.Format(time.RFC3339)}).ok(200, "PlanEventResult")
	d := a.call("GET", "/v1/me/plan/decisions?limit=100", u.access, nil).ok(200, "PlanDecisionPage")
	consents := 0
	for _, it := range d["items"].([]any) {
		if it.(map[string]any)["trigger"] == "consent" {
			consents++
		}
	}
	if consents != 2 {
		t.Errorf("%d consent changes in the log, want 2", consents)
	}
}

// A planned session starts as a draft in the log (spec §10.2); its sets are
// performed through the log API as usual and keep their plan item.
func TestStartPlannedSession(t *testing.T) {
	a := newAPI(t, withPlanner(&fixedClock{plannerMonday}))
	u := onboarded(t, a, "start@example.com")
	pr := a.call("GET", "/v1/me/plan", u.access, nil)
	plan := pr.ok(200, "TrainingPlan")
	sessions := plan["sessions"].([]any)
	if len(sessions) < 2 {
		t.Fatalf("%d sessions planned", len(sessions))
	}
	first := sessions[0].(map[string]any)
	if first["status"] != "planned" || first["workout_session_id"] != nil {
		t.Fatalf("a new session: %v %v", first["status"], first["workout_session_id"])
	}
	sid := first["id"].(string)
	items := map[string]bool{}
	for _, b := range first["blocks"].([]any) {
		for _, it := range b.(map[string]any)["items"].([]any) {
			items[it.(map[string]any)["id"].(string)] = true
		}
	}
	start := func(id string) map[string]any {
		return map[string]any{"id": id, "timezone": "Europe/Zurich", "started_at": plannerMonday.Add(10 * time.Hour).Format(time.RFC3339)}
	}

	logID := newID()
	r := a.call("POST", "/v1/me/plan/sessions/"+sid+"/start", u.access, start(logID))
	sess := r.ok(201, "Session")
	if r.header.Get("Location") != "/v1/sessions/"+logID || sess["id"] != logID || sess["planned_session_id"] != sid ||
		sess["status"] != "draft" || sess["local_date"] != "2026-09-28" {
		t.Fatalf("started session %v (Location %q)", sess, r.header.Get("Location"))
	}
	var set map[string]any
	for _, b := range sess["blocks"].([]any) {
		for _, s := range b.(map[string]any)["sets"].([]any) {
			s := s.(map[string]any)
			if s["is_planned"] != true || !items[s["planned_item_id"].(string)] {
				t.Fatalf("set %v", s)
			}
			if set == nil {
				set = s
			}
		}
	}
	if set == nil {
		t.Fatal("no planned sets")
	}

	// A second start answers with the first session.
	again := a.call("POST", "/v1/me/plan/sessions/"+sid+"/start", u.access, start(newID())).ok(200, "Session")
	if again["id"] != logID {
		t.Errorf("second start gave %v, want %v", again["id"], logID)
	}
	// The plan shows the start, and its ETag changes with it.
	ar := a.callWith("GET", "/v1/me/plan", u.access, map[string]string{"If-None-Match": pr.header.Get("ETag")})
	now := ar.ok(200, "TrainingPlan")["sessions"].([]any)[0].(map[string]any)
	if now["status"] != "started" || now["workout_session_id"] != logID {
		t.Errorf("planned session after the start: %v %v", now["status"], now["workout_session_id"])
	}
	a.callWith("GET", "/v1/me/plan", u.access, map[string]string{"If-None-Match": ar.header.Get("ETag")}).ok(304, "")

	// Performing a planned set goes through the log API and keeps its item.
	el := set["elements"].([]any)[0].(map[string]any)
	el["reps"], el["hold_seconds"] = nil, nil
	if el["measure"] == "reps" {
		el["reps"] = 3
	} else {
		el["hold_seconds"] = 5
	}
	delete(el, "assistance_class")
	delete(el, "order_index")
	done := a.call("PUT", "/v1/sessions/"+logID+"/sets/"+set["id"].(string), u.access, map[string]any{
		"block_id": set["block_id"], "order_index": set["order_index"], "kind": set["kind"], "is_planned": false,
		"completed_at": plannerMonday.Add(10*time.Hour + 5*time.Minute).Format(time.RFC3339),
		"elements":     []any{map[string]any{"id": el["id"], "exercise_id": el["exercise_id"], "measure": el["measure"], "reps": el["reps"], "hold_seconds": el["hold_seconds"]}},
	}).ok(200, "SetEntry")
	if done["is_planned"] != false || done["planned_item_id"] != set["planned_item_id"] {
		t.Errorf("performed set %v", done)
	}
	pulled := a.call("GET", "/v1/sync?cursor=0&limit=1000", u.access, nil).ok(200, "SyncPage")
	if s := pulled["sessions"].([]any); len(s) != 1 || s[0].(map[string]any)["planned_session_id"] != sid {
		t.Errorf("synced sessions %v", s)
	}

	// Errors: an unknown or another user's session, a bad time zone, a
	// taken ID, and a stop.
	a.call("POST", "/v1/me/plan/sessions/"+newID()+"/start", u.access, start(newID())).problem(404, "not-found")
	other := onboarded(t, a, "start-other@example.com")
	a.call("POST", "/v1/me/plan/sessions/"+sid+"/start", other.access, start(newID())).problem(404, "not-found")
	next := sessions[1].(map[string]any)["id"].(string)
	a.call("POST", "/v1/me/plan/sessions/"+next+"/start", u.access, map[string]any{"id": newID(), "timezone": "Mars/Base"}).
		problem(422, "validation")
	a.call("POST", "/v1/me/plan/sessions/"+next+"/start", u.access, start(logID)).problem(409, "already-exists")
	a.call("POST", "/v1/me/symptoms", u.access, map[string]any{"id": newID()}).ok(200, "PlanEventResult")
	a.call("POST", "/v1/me/plan/sessions/"+next+"/start", u.access, start(newID())).problem(409, "training-stopped")
}
