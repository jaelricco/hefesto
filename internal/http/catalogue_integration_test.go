//go:build integration

package http_test

import (
	"net/http"
	"testing"
)

func names(m map[string]any) []string {
	var out []string
	for _, it := range m["items"].([]any) {
		out = append(out, it.(map[string]any)["slug"].(string))
	}
	return out
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func TestExerciseCatalogue(t *testing.T) {
	a := newAPI(t)
	u := a.register("catalogue@example.test")

	all := a.call("GET", "/v1/exercises", u.access, nil)
	m := all.ok(200, "ExerciseList")
	if n := len(m["items"].([]any)); n != 7 {
		t.Fatalf("got %d exercises, want the 7 seeded", n)
	}
	etag := all.header.Get("ETag")
	if etag == "" || etag != `"`+m["content_version"].(string)+`"` {
		t.Fatalf("ETag %q does not carry the content version %v", etag, m["content_version"])
	}

	// Unchanged catalogue: 304 and no body.
	req, _ := http.NewRequest("GET", a.srv.URL+"/v1/exercises", nil)
	req.Header.Set("Authorization", "Bearer "+u.access)
	req.Header.Set("If-None-Match", etag)
	resp, err := a.srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("If-None-Match: status %d", resp.StatusCode)
	}

	hs := names(a.call("GET", "/v1/exercises?family=handstand", u.access, nil).ok(200, "ExerciseList"))
	if len(hs) != 1 || hs[0] != "wall-handstand-hold" {
		t.Fatalf("family filter: %v", hs)
	}
	fuzzy := names(a.call("GET", "/v1/exercises?q=frnt%20lever", u.access, nil).ok(200, "ExerciseList"))
	if !contains(fuzzy, "front-lever-tuck") {
		t.Fatalf("fuzzy search missed front-lever-tuck: %v", fuzzy)
	}
	dumbbell := names(a.call("GET", "/v1/exercises?equipment=dumbbell", u.access, nil).ok(200, "ExerciseList"))
	if len(dumbbell) != 1 || dumbbell[0] != "pronated-curl-eccentric" {
		t.Fatalf("equipment filter: %v", dumbbell)
	}
	a.call("GET", "/v1/exercises?family=Not_A_Slug", u.access, nil).problem(400, "bad-request")

	ex := a.call("GET", "/v1/exercises/pull-up", u.access, nil).ok(200, "Exercise")
	if ex["default_measure"] != "reps" || ex["status"] != "draft_placeholder" {
		t.Fatalf("pull-up: %v", ex)
	}
	a.call("GET", "/v1/exercises/no-such-exercise", u.access, nil).problem(404, "not-found")

	// Retired exercises disappear from the catalogue.
	a.exec(`UPDATE exercises SET status = 'retired' WHERE slug = 'pull-up'`)
	a.call("GET", "/v1/exercises/pull-up", u.access, nil).problem(404, "not-found")
	if contains(names(a.call("GET", "/v1/exercises", u.access, nil).ok(200, "ExerciseList")), "pull-up") {
		t.Fatal("retired exercise listed")
	}
}

func TestBands(t *testing.T) {
	a := newAPI(t)
	u := a.register("bands@example.test")
	other := a.register("other-bands@example.test")

	a.exec(`INSERT INTO bands (id, brand, colour_label, resistance_min_kg, resistance_max_kg)
	        VALUES ('01920000-0000-7000-8000-00000000b0b0', 'Catalogue', 'Blue', 10, 25)`)

	id := newID()
	b := a.call("POST", "/v1/me/bands", u.access, map[string]any{
		"id": id, "brand": "Home", "colour_label": "Red", "resistance_min_kg": 5, "resistance_max_kg": 15,
	}).ok(201, "Band")
	if b["owner"] != "mine" {
		t.Fatalf("owner %v", b["owner"])
	}
	a.call("POST", "/v1/me/bands", u.access, map[string]any{
		"id": id, "brand": "Home", "colour_label": "Green", "resistance_min_kg": 5, "resistance_max_kg": 15,
	}).problem(409, "already-exists")
	hasField(t, a.call("POST", "/v1/me/bands", u.access, map[string]any{
		"id": newID(), "brand": "Home", "colour_label": "Black", "resistance_min_kg": 30, "resistance_max_kg": 15,
	}).problem(422, "validation"), "/resistance_min_kg")

	mine := a.call("GET", "/v1/bands", u.access, nil).ok(200, "BandList")["items"].([]any)
	theirs := a.call("GET", "/v1/bands", other.access, nil).ok(200, "BandList")["items"].([]any)
	if len(mine) != 2 || len(theirs) != 1 {
		t.Fatalf("bands visible: mine %d (want catalogue + own), theirs %d (want catalogue)", len(mine), len(theirs))
	}

	a.call("DELETE", "/v1/me/bands/"+id, other.access, nil).problem(404, "not-found")
	a.call("DELETE", "/v1/me/bands/01920000-0000-7000-8000-00000000b0b0", u.access, nil).problem(404, "not-found")
	a.call("DELETE", "/v1/me/bands/"+id, u.access, nil).ok(204, "")
	if n := len(a.call("GET", "/v1/bands", u.access, nil).ok(200, "BandList")["items"].([]any)); n != 1 {
		t.Fatalf("deleted band still listed (%d bands)", n)
	}
}
