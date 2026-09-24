//go:build integration

package http_test

import (
	"fmt"
	"testing"
)

// logger bundles a user and a session with one block, the common starting
// point of the logging tests.
type logger struct {
	a       *api
	u       user
	session string
	block   string
}

func newLogger(t *testing.T, email string) logger {
	a := newAPI(t)
	u := a.register(email)
	l := logger{a: a, u: u, session: newID(), block: newID()}
	a.call("POST", "/v1/sessions", u.access, map[string]any{
		"id": l.session, "started_at": "2026-09-23T21:30:00Z", "timezone": "Europe/Zurich", "title": "Pull day",
	}).ok(201, "Session")
	a.call("PUT", l.path("blocks", l.block), u.access, map[string]any{"order_index": 0}).ok(201, "Block")
	return l
}

func (l logger) path(parts ...string) string {
	p := "/v1/sessions/" + l.session
	for _, s := range parts {
		p += "/" + s
	}
	return p
}

func reps(exercise string, n int) map[string]any {
	return map[string]any{"id": newID(), "exercise_id": exercise, "measure": "reps", "reps": n}
}

func hold(exercise string, s float64) map[string]any {
	return map[string]any{"id": newID(), "exercise_id": exercise, "measure": "hold_seconds", "hold_seconds": s}
}

func (l logger) putSet(id string, order int, elements ...map[string]any) res {
	return l.a.call("PUT", l.path("sets", id), l.u.access, map[string]any{
		"block_id": l.block, "order_index": order, "completed_at": "2026-09-23T21:40:00Z", "rpe": 8,
		"elements": elements,
	})
}

func elementsOf(set map[string]any) []map[string]any {
	var out []map[string]any
	for _, e := range set["elements"].([]any) {
		out = append(out, e.(map[string]any))
	}
	return out
}

func TestCreateSession(t *testing.T) {
	l := newLogger(t, "create@example.test")
	s := l.a.call("GET", l.path(), l.u.access, nil).ok(200, "Session")
	// 21:30 UTC is 23:30 in Zurich: still the 23rd there.
	if s["local_date"] != "2026-09-23" || s["status"] != "draft" || s["title"] != "Pull day" {
		t.Fatalf("session %v", s)
	}
	if blocks := s["blocks"].([]any); len(blocks) != 1 {
		t.Fatalf("blocks %v", blocks)
	}

	l.a.call("POST", "/v1/sessions", l.u.access, map[string]any{
		"id": l.session, "started_at": "2026-09-23T10:00:00Z", "timezone": "Europe/Zurich",
	}).problem(409, "already-exists")
	hasField(t, l.a.call("POST", "/v1/sessions", l.u.access, map[string]any{
		"id": newID(), "started_at": "2026-09-23T10:00:00Z", "timezone": "Middle/Earth",
	}).problem(422, "validation"), "/timezone")
	hasField(t, l.a.call("POST", "/v1/sessions", l.u.access, map[string]any{
		"id": newID(), "started_at": "2026-09-23T10:00:00Z", "timezone": "Europe/Zurich", "template_id": newID(),
	}).problem(422, "validation"), "/template_id")
}

func TestPlainSetAndComboUseTheSamePath(t *testing.T) {
	l := newLogger(t, "combo@example.test")
	pullUp := l.a.exerciseID("pull-up")
	tuck := l.a.exerciseID("front-lever-tuck")
	adv := l.a.exerciseID("front-lever-advanced-tuck")

	plain := l.putSet(newID(), 0, reps(pullUp, 8)).ok(201, "SetEntry")
	if els := elementsOf(plain); len(els) != 1 || els[0]["reps"] != float64(8) || els[0]["assistance_class"] != "unassisted" {
		t.Fatalf("plain set %v", plain)
	}

	comboID := newID()
	combo := l.putSet(comboID, 1, hold(tuck, 5), hold(adv, 3), hold(tuck, 4)).ok(201, "SetEntry")
	els := elementsOf(combo)
	if len(els) != 3 {
		t.Fatalf("combo has %d elements", len(els))
	}
	for i, e := range els {
		if e["order_index"] != float64(i) {
			t.Fatalf("element %d has order_index %v", i, e["order_index"])
		}
	}

	var entries, elements int
	if err := l.a.pool.QueryRow(t.Context(), `SELECT count(DISTINCT set_entry_id), count(*) FROM set_elements WHERE set_entry_id = $1`, comboID).
		Scan(&entries, &elements); err != nil {
		t.Fatal(err)
	}
	if entries != 1 || elements != 3 {
		t.Fatalf("combo stored as %d entries / %d elements", entries, elements)
	}
}

func TestReplaceSetElements(t *testing.T) {
	l := newLogger(t, "replace@example.test")
	tuck := l.a.exerciseID("front-lever-tuck")
	adv := l.a.exerciseID("front-lever-advanced-tuck")

	setID := newID()
	a, b, c := hold(tuck, 5), hold(adv, 3), hold(tuck, 4)
	l.putSet(setID, 0, a, b, c).ok(201, "SetEntry")

	// Drop b, reverse the rest, change a value: same ids keep their rows.
	a["hold_seconds"] = 6
	out := l.putSet(setID, 0, c, a).ok(200, "SetEntry")
	els := elementsOf(out)
	if len(els) != 2 || els[0]["id"] != c["id"] || els[1]["id"] != a["id"] || els[1]["hold_seconds"] != float64(6) {
		t.Fatalf("replaced elements %v", els)
	}
	var tombstones int
	if err := l.a.pool.QueryRow(t.Context(), `SELECT count(*) FROM set_elements WHERE id = $1 AND deleted_at IS NOT NULL`, b["id"]).Scan(&tombstones); err != nil {
		t.Fatal(err)
	}
	if tombstones != 1 {
		t.Fatal("a dropped element was not tombstoned")
	}
}

func TestAssistanceAndLoad(t *testing.T) {
	l := newLogger(t, "assist@example.test")
	pullUp := l.a.exerciseID("pull-up")
	band := newID()
	l.a.call("POST", "/v1/me/bands", l.u.access, map[string]any{
		"id": band, "brand": "Home", "colour_label": "Red", "resistance_min_kg": 5, "resistance_max_kg": 15,
	}).ok(201, "Band")

	setID := newID()
	el := reps(pullUp, 6)
	assist := map[string]any{"id": newID(), "type": "band", "band_id": band, "band_count": 1, "anchor": "under_foot"}
	el["assistance"] = assist
	out := l.putSet(setID, 0, el).ok(201, "SetEntry")
	got := elementsOf(out)[0]
	if got["assistance_class"] != "assisted" || got["assistance"].(map[string]any)["band_id"] != band {
		t.Fatalf("assisted element %v", got)
	}

	// Remove the assistance and add load: now it is a weighted set.
	el["assistance"] = nil
	el["load_kg"] = 10
	got = elementsOf(l.putSet(setID, 0, el).ok(200, "SetEntry"))[0]
	if got["assistance_class"] != "loaded" || got["assistance"] != nil {
		t.Fatalf("loaded element %v", got)
	}

	// A counterweight is assistance, never a negative load.
	el["load_kg"] = -10
	hasField(t, l.putSet(setID, 0, el).problem(422, "validation"), "/elements/0/load_kg")
	el["load_kg"] = 0
	el["assistance"] = map[string]any{"id": newID(), "type": "counterweight", "estimated_assist_kg": 10}
	if c := elementsOf(l.putSet(setID, 0, el).ok(200, "SetEntry"))[0]["assistance_class"]; c != "assisted" {
		t.Fatalf("counterweight class %v", c)
	}

	// Another user's band cannot be used.
	other := l.a.register("band-thief@example.test")
	theirBand := newID()
	l.a.call("POST", "/v1/me/bands", other.access, map[string]any{
		"id": theirBand, "brand": "Theirs", "colour_label": "Red", "resistance_min_kg": 5, "resistance_max_kg": 15,
	}).ok(201, "Band")
	el["assistance"] = map[string]any{"id": newID(), "type": "band", "band_id": theirBand, "band_count": 1}
	hasField(t, l.putSet(setID, 0, el).problem(422, "validation"), "/elements/0/assistance/band_id")
}

func TestSetValidation(t *testing.T) {
	l := newLogger(t, "validate@example.test")
	pullUp := l.a.exerciseID("pull-up")

	cases := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"no elements", map[string]any{"block_id": l.block, "order_index": 0, "elements": []any{}}, "/elements"},
		{"value for another measure", map[string]any{"block_id": l.block, "order_index": 0,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps", "hold_seconds": 5}}}, "/elements/0/hold_seconds"},
		{"bad tempo", map[string]any{"block_id": l.block, "order_index": 0,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps", "tempo": "3-0-X"}}}, "/elements/0/tempo"},
		{"unknown exercise", map[string]any{"block_id": l.block, "order_index": 0,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": newID(), "measure": "reps"}}}, "/elements/0/exercise_id"},
		{"block of another session", map[string]any{"block_id": newID(), "order_index": 0,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps"}}}, "/block_id"},
		{"completed without a value", map[string]any{"block_id": l.block, "order_index": 0, "completed_at": "2026-09-23T21:40:00Z",
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps"}}}, "/elements/0/reps"},
		{"rpe off the half step", map[string]any{"block_id": l.block, "order_index": 0, "rpe": 7.3,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps"}}}, "/rpe"},
		{"unknown field", map[string]any{"block_id": l.block, "order_index": 0, "sets": 3,
			"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps"}}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hasField(t, l.a.call("PUT", l.path("sets", newID()), l.u.access, tc.body).problem(422, "validation"), tc.field)
		})
	}

	// A retired exercise is kept on existing elements but cannot be newly chosen.
	setID := newID()
	el := reps(pullUp, 5)
	l.putSet(setID, 0, el).ok(201, "SetEntry")
	l.a.exec(`UPDATE exercises SET status = 'retired' WHERE slug = 'pull-up'`)
	el["reps"] = 6
	l.putSet(setID, 0, el).ok(200, "SetEntry")
	hasField(t, l.putSet(newID(), 1, reps(pullUp, 5)).problem(422, "validation"), "/elements/0/exercise_id")
}

func TestOrderConflictAndReorder(t *testing.T) {
	l := newLogger(t, "order@example.test")
	pullUp := l.a.exerciseID("pull-up")

	s1, s2, s3 := newID(), newID(), newID()
	l.putSet(s1, 0, reps(pullUp, 8)).ok(201, "SetEntry")
	l.putSet(s2, 1, reps(pullUp, 7)).ok(201, "SetEntry")
	l.putSet(s3, 1, reps(pullUp, 6)).problem(409, "order-conflict")

	// A deleted set frees its position immediately.
	l.a.call("DELETE", l.path("sets", s2), l.u.access, nil).ok(204, "")
	l.putSet(s3, 1, reps(pullUp, 6)).ok(201, "SetEntry")

	// Swap the two live sets in one step.
	sess := l.a.call("POST", l.path("reorder"), l.u.access, map[string]any{
		"sets": []any{map[string]any{"block_id": l.block, "set_ids": []string{s3, s1}}},
	}).ok(200, "Session")
	sets := sess["blocks"].([]any)[0].(map[string]any)["sets"].([]any)
	if sets[0].(map[string]any)["id"] != s3 || sets[1].(map[string]any)["id"] != s1 {
		t.Fatalf("reorder not applied: %v", sets)
	}

	errs := l.a.call("POST", l.path("reorder"), l.u.access, map[string]any{
		"sets": []any{map[string]any{"block_id": l.block, "set_ids": []string{s3}}},
	}).problem(422, "validation")
	hasField(t, errs, "/sets/0/set_ids")

	b2 := newID()
	l.a.call("PUT", l.path("blocks", b2), l.u.access, map[string]any{"order_index": 1, "kind": "superset"}).ok(201, "Block")
	sess = l.a.call("POST", l.path("reorder"), l.u.access, map[string]any{"blocks": []string{b2, l.block}}).ok(200, "Session")
	if sess["blocks"].([]any)[0].(map[string]any)["id"] != b2 {
		t.Fatal("blocks not reordered")
	}
	hasField(t, l.a.call("PUT", l.path("blocks", b2), l.u.access, map[string]any{"order_index": 0, "interval_s": 60}).problem(422, "validation"), "/interval_s")
}

func TestDeleteCascadesAndTombstonesAreFinal(t *testing.T) {
	l := newLogger(t, "delete@example.test")
	pullUp := l.a.exerciseID("pull-up")
	setID := newID()
	l.putSet(setID, 0, reps(pullUp, 8)).ok(201, "SetEntry")

	l.a.call("DELETE", l.path("blocks", l.block), l.u.access, nil).ok(204, "")
	if blocks := l.a.call("GET", l.path(), l.u.access, nil).ok(200, "Session")["blocks"].([]any); len(blocks) != 0 {
		t.Fatalf("deleted block still returned: %v", blocks)
	}
	var live int
	if err := l.a.pool.QueryRow(t.Context(), `SELECT count(*) FROM set_elements WHERE session_id = $1 AND deleted_at IS NULL`, l.session).Scan(&live); err != nil {
		t.Fatal(err)
	}
	if live != 0 {
		t.Fatal("deleting a block left live elements behind")
	}
	// A tombstone is final: the id cannot be written again.
	l.a.call("PUT", l.path("blocks", l.block), l.u.access, map[string]any{"order_index": 0}).problem(404, "not-found")

	l.a.call("DELETE", l.path(), l.u.access, nil).ok(204, "")
	l.a.call("GET", l.path(), l.u.access, nil).problem(404, "not-found")
	l.a.call("DELETE", l.path(), l.u.access, nil).problem(404, "not-found")
}

func TestUpdateSession(t *testing.T) {
	l := newLogger(t, "patch@example.test")
	s := l.a.call("PATCH", l.path(), l.u.access, map[string]any{
		"perceived_fatigue": 7, "bodyweight_kg": 72.5, "ended_at": "2026-09-23T22:30:00Z", "notes": "good",
	}).ok(200, "Session")
	if s["perceived_fatigue"] != float64(7) || s["bodyweight_kg"] != 72.5 || s["notes"] != "good" {
		t.Fatalf("patch not applied: %v", s)
	}
	// null clears; absent leaves alone.
	s = l.a.call("PATCH", l.path(), l.u.access, map[string]any{"perceived_fatigue": nil}).ok(200, "Session")
	if s["perceived_fatigue"] != nil || s["bodyweight_kg"] != 72.5 {
		t.Fatalf("null/absent semantics: %v", s)
	}
	// Moving the start re-derives the athlete's day.
	s = l.a.call("PATCH", l.path(), l.u.access, map[string]any{"started_at": "2026-09-23T22:30:00Z", "ended_at": nil}).ok(200, "Session")
	if s["local_date"] != "2026-09-24" {
		t.Fatalf("local_date %v", s["local_date"])
	}

	hasField(t, l.a.call("PATCH", l.path(), l.u.access, map[string]any{"ended_at": "2026-09-01T00:00:00Z"}).problem(422, "validation"), "/ended_at")
	hasField(t, l.a.call("PATCH", l.path(), l.u.access, map[string]any{"status": "completed"}).problem(422, "validation"), "/status")
	l.a.call("PATCH", l.path(), l.u.access, map[string]any{"status": "abandoned"}).ok(200, "Session")
	l.a.call("PATCH", l.path(), l.u.access, map[string]any{"status": "draft"}).ok(200, "Session")
}

func TestListSessionsPaginates(t *testing.T) {
	a := newAPI(t)
	u := a.register("list@example.test")
	for i := 0; i < 5; i++ {
		a.call("POST", "/v1/sessions", u.access, map[string]any{
			"id": newID(), "started_at": fmt.Sprintf("2026-09-%02dT08:00:00Z", 10+i), "timezone": "Europe/Zurich",
		}).ok(201, "Session")
	}
	page := a.call("GET", "/v1/sessions?limit=2", u.access, nil).ok(200, "SessionPage")
	var seen []string
	for page != nil {
		for _, it := range page["items"].([]any) {
			seen = append(seen, it.(map[string]any)["local_date"].(string))
		}
		c, _ := page["next_cursor"].(string)
		if c == "" {
			break
		}
		page = a.call("GET", "/v1/sessions?limit=2&cursor="+c, u.access, nil).ok(200, "SessionPage")
	}
	want := []string{"2026-09-14", "2026-09-13", "2026-09-12", "2026-09-11", "2026-09-10"}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("pages walked %v, want %v", seen, want)
	}

	ranged := a.call("GET", "/v1/sessions?from=2026-09-11&to=2026-09-12", u.access, nil).ok(200, "SessionPage")
	if n := len(ranged["items"].([]any)); n != 2 {
		t.Fatalf("date range returned %d", n)
	}
	a.call("GET", "/v1/sessions?cursor=garbage", u.access, nil).problem(400, "bad-request")
	a.call("GET", "/v1/sessions?limit=0", u.access, nil).problem(400, "bad-request")
}

func TestLastSetRepeatsTheWholeSet(t *testing.T) {
	l := newLogger(t, "repeat@example.test")
	tuck := l.a.exerciseID("front-lever-tuck")
	adv := l.a.exerciseID("front-lever-advanced-tuck")

	l.a.call("GET", "/v1/me/exercises/"+adv+"/last-set", l.u.access, nil).problem(404, "not-found")

	l.putSet(newID(), 0, hold(tuck, 10)).ok(201, "SetEntry")
	comboID := newID()
	l.putSet(comboID, 1, hold(tuck, 5), hold(adv, 3)).ok(201, "SetEntry")

	last := l.a.call("GET", "/v1/me/exercises/"+adv+"/last-set", l.u.access, nil).ok(200, "LastSet")
	set := last["set"].(map[string]any)
	if set["id"] != comboID || len(set["elements"].([]any)) != 2 || last["local_date"] != "2026-09-23" {
		t.Fatalf("last set %v", last)
	}

	// Planned sets are not "last performed".
	l.a.call("PUT", l.path("sets", newID()), l.u.access, map[string]any{
		"block_id": l.block, "order_index": 2, "is_planned": true,
		"elements": []any{map[string]any{"id": newID(), "exercise_id": adv, "measure": "hold_seconds"}},
	}).ok(201, "SetEntry")
	if got := l.a.call("GET", "/v1/me/exercises/"+adv+"/last-set", l.u.access, nil).ok(200, "LastSet")["set"].(map[string]any)["id"]; got != comboID {
		t.Fatalf("a planned set was returned as last performed: %v", got)
	}
}

func TestUsersCannotReachEachOthersData(t *testing.T) {
	l := newLogger(t, "owner@example.test")
	pullUp := l.a.exerciseID("pull-up")
	el := reps(pullUp, 8)
	setID := newID()
	l.putSet(setID, 0, el).ok(201, "SetEntry")

	intruder := l.a.register("intruder@example.test")
	tok := intruder.access
	l.a.call("GET", l.path(), tok, nil).problem(404, "not-found")
	l.a.call("PATCH", l.path(), tok, map[string]any{"title": "mine now"}).problem(404, "not-found")
	l.a.call("DELETE", l.path(), tok, nil).problem(404, "not-found")
	l.a.call("PUT", l.path("blocks", newID()), tok, map[string]any{"order_index": 0}).problem(404, "not-found")
	l.a.call("DELETE", l.path("sets", setID), tok, nil).problem(404, "not-found")
	l.a.call("POST", l.path("reorder"), tok, map[string]any{"blocks": []string{l.block}}).problem(404, "not-found")
	l.a.call("GET", "/v1/me/exercises/"+pullUp+"/last-set", tok, nil).problem(404, "not-found")

	// Reusing the owner's element id in the intruder's own session is refused,
	// and the owner's row is untouched.
	own := newID()
	ownBlock := newID()
	l.a.call("POST", "/v1/sessions", tok, map[string]any{"id": own, "started_at": "2026-09-23T10:00:00Z", "timezone": "UTC"}).ok(201, "Session")
	l.a.call("PUT", "/v1/sessions/"+own+"/blocks/"+ownBlock, tok, map[string]any{"order_index": 0}).ok(201, "Block")
	stolen := map[string]any{"id": el["id"], "exercise_id": pullUp, "measure": "reps", "reps": 99}
	res := l.a.call("PUT", "/v1/sessions/"+own+"/sets/"+newID(), tok, map[string]any{
		"block_id": ownBlock, "order_index": 0, "elements": []any{stolen},
	})
	if res.status < 400 {
		t.Fatalf("intruder wrote through another user's element id: %d %s", res.status, res.body)
	}
	got := elementsOf(l.a.call("GET", l.path(), l.u.access, nil).ok(200, "Session")["blocks"].([]any)[0].(map[string]any)["sets"].([]any)[0].(map[string]any))
	if got[0]["reps"] != float64(8) {
		t.Fatalf("owner's element changed: %v", got[0])
	}
}
