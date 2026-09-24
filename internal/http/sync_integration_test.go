//go:build integration

package http_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// pull fetches one page of changes.
func (a *api) pull(u user, cursor float64, limit int) map[string]any {
	a.t.Helper()
	return a.call("GET", fmt.Sprintf("/v1/sync?cursor=%d&limit=%d", int64(cursor), limit), u.access, nil).ok(200, "SyncPage")
}

// pullAll follows the cursor to the end and returns every unit seen, by id,
// the last copy winning, and the final cursor.
func (a *api) pullAll(u user, cursor float64, limit int) (map[string]map[string]any, float64, int) {
	a.t.Helper()
	seen := map[string]map[string]any{}
	pages := 0
	for {
		p := a.pull(u, cursor, limit)
		pages++
		for _, kind := range []string{"sessions", "blocks", "sets", "bodyweight"} {
			for _, x := range p[kind].([]any) {
				row := x.(map[string]any)
				row["_kind"] = kind
				seen[row["id"].(string)] = row
			}
		}
		next := p["cursor"].(float64)
		if next < cursor {
			a.t.Fatalf("cursor went back from %v to %v", cursor, next)
		}
		cursor = next
		if p["has_more"] != true {
			return seen, cursor, pages
		}
	}
}

func (a *api) push(u user, key string, ops ...map[string]any) res {
	a.t.Helper()
	return a.callWith("POST", "/v1/sync", u.access, map[string]string{"Idempotency-Key": key}, map[string]any{"ops": ops})
}

func results(t *testing.T, m map[string]any) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, r := range m["results"].([]any) {
		out = append(out, r.(map[string]any))
	}
	return out
}

func wantStatuses(t *testing.T, m map[string]any, want ...string) []map[string]any {
	t.Helper()
	rs := results(t, m)
	if len(rs) != len(want) {
		t.Fatalf("%d results, want %d: %v", len(rs), len(want), rs)
	}
	for i, r := range rs {
		if r["status"] != want[i] || r["index"] != float64(i) {
			t.Fatalf("op %d: %v, want %s", i, r, want[i])
		}
	}
	return rs
}

func at(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func TestSyncPullPagesEveryChange(t *testing.T) {
	l := newLogger(t, "pull@example.test")
	pullUp := l.a.exerciseID("pull-up")
	var setIDs []string
	for i := range 3 {
		id := newID()
		l.putSet(id, i, reps(pullUp, 5+i)).ok(201, "SetEntry")
		setIDs = append(setIDs, id)
	}

	seen, cursor, pages := l.a.pullAll(l.u, 0, 2)
	if pages < 3 {
		t.Fatalf("five units in pages of two took %d pages", pages)
	}
	count := map[string]int{}
	for _, r := range seen {
		count[r["_kind"].(string)]++
	}
	if count["sessions"] != 1 || count["blocks"] != 1 || count["sets"] != 3 {
		t.Fatalf("pulled %v", count)
	}
	s := seen[setIDs[1]]
	if s["session_id"] != l.session || s["deleted_at"] != nil || len(s["elements"].([]any)) != 1 {
		t.Fatalf("set %v", s)
	}

	// Nothing new: an empty page that keeps the cursor.
	p := l.a.pull(l.u, cursor, 100)
	if p["cursor"] != cursor || p["has_more"] != false || len(p["sets"].([]any)) != 0 {
		t.Fatalf("empty pull %v", p)
	}

	// An edit and a deletion arrive, and nothing else.
	l.putSet(setIDs[0], 0, reps(pullUp, 9)).ok(200, "SetEntry")
	l.a.call("DELETE", l.path("sets", setIDs[2]), l.u.access, nil).ok(204, "")
	seen, cursor2, _ := l.a.pullAll(l.u, cursor, 100)
	if len(seen) != 2 || cursor2 <= cursor {
		t.Fatalf("after edits: %d units, cursor %v -> %v: %v", len(seen), cursor, cursor2, seen)
	}
	if el := seen[setIDs[0]]["elements"].([]any)[0].(map[string]any); el["reps"] != float64(9) {
		t.Fatalf("edited set %v", seen[setIDs[0]])
	}
	gone := seen[setIDs[2]]
	if gone["deleted_at"] == nil || len(gone["elements"].([]any)) != 0 {
		t.Fatalf("tombstone %v", gone)
	}

	// Another athlete's feed is their own.
	other := l.a.register("pull-other@example.test")
	if seen, _, _ := l.a.pullAll(other, 0, 100); len(seen) != 0 {
		t.Fatalf("another user pulled %v", seen)
	}

	l.a.call("GET", "/v1/sync", l.u.access, nil).problem(400, "bad-request")
	l.a.call("GET", "/v1/sync?cursor=-1", l.u.access, nil).problem(400, "bad-request")
	l.a.call("GET", "/v1/sync?cursor=0&limit=5000", l.u.access, nil).problem(400, "bad-request")
}

// A set can change before its block does; a page holding the set then
// carries the block too, so it applies on its own.
func TestSyncPullBringsParentsForward(t *testing.T) {
	l := newLogger(t, "parents@example.test")
	setID := newID()
	l.putSet(setID, 0, reps(l.a.exerciseID("pull-up"), 5)).ok(201, "SetEntry")
	l.a.call("PUT", l.path("blocks", l.block), l.u.access, map[string]any{"order_index": 0, "notes": "later"}).ok(200, "Block")

	// Feed order: session, set, block (changed last). Page one: session, set.
	p := l.a.pull(l.u, 0, 2)
	if len(p["sets"].([]any)) != 1 || p["has_more"] != true {
		t.Fatalf("page one %v", p)
	}
	blocks := p["blocks"].([]any)
	if len(blocks) != 1 || blocks[0].(map[string]any)["notes"] != "later" {
		t.Fatalf("the set's block was not brought forward: %v", p)
	}
	// It comes again at its own position; applying it twice is harmless.
	p = l.a.pull(l.u, p["cursor"].(float64), 2)
	if len(p["blocks"].([]any)) != 1 || p["has_more"] != false {
		t.Fatalf("page two %v", p)
	}
}

func TestSyncPushAppliesAnOfflineSession(t *testing.T) {
	a := newAPI(t)
	u := a.register("push@example.test")
	pullUp := a.exerciseID("pull-up")
	started := time.Now().Add(-time.Hour)
	session, block := newID(), newID()
	set := func(order, n int) map[string]any {
		return map[string]any{"op": "put", "entity": "set", "id": newID(), "session_id": session, "data": map[string]any{
			"block_id": block, "order_index": order, "completed_at": at(started.Add(time.Duration(order+1) * time.Minute)),
			"elements": []any{reps(pullUp, n)},
		}}
	}
	ops := []map[string]any{
		{"op": "put", "entity": "session", "id": session, "data": map[string]any{
			"started_at": at(started), "timezone": "Europe/Zurich", "title": "Offline", "perceived_fatigue": 6,
		}},
		{"op": "put", "entity": "block", "id": block, "session_id": session, "data": map[string]any{"order_index": 0}},
		set(0, 5), set(1, 6),
		{"op": "put", "entity": "bodyweight", "id": newID(), "data": map[string]any{
			"measured_at": at(started), "timezone": "Europe/Zurich", "bodyweight_kg": 71.5,
		}},
		{"op": "complete", "entity": "session", "id": session, "data": map[string]any{}},
	}
	key := newID()
	r := a.push(u, key, ops...)
	m := r.ok(200, "SyncPushResult")
	rs := wantStatuses(t, m, "applied", "applied", "applied", "applied", "applied", "applied")
	c := rs[5]["completion"].(map[string]any)
	if got := unlockedKeys(c); len(got) != 1 || got[0] != "pull-up/strict-5" {
		t.Fatalf("offline completion unlocked %v", got)
	}

	s := a.call("GET", "/v1/sessions/"+session, u.access, nil).ok(200, "Session")
	if s["status"] != "completed" || s["title"] != "Offline" || s["perceived_fatigue"] != float64(6) ||
		len(s["blocks"].([]any)[0].(map[string]any)["sets"].([]any)) != 2 {
		t.Fatalf("session after push %v", s)
	}

	// The same request again is answered from the first response.
	again := a.push(u, key, ops...)
	if again.header.Get("Idempotent-Replayed") != "true" || !reflect.DeepEqual(again.ok(200, "SyncPushResult"), m) {
		t.Fatalf("replay %d %s %v", again.status, again.body, again.header)
	}
	// The same key with another request is a client bug.
	a.push(u, key, ops[0]).problem(422, "idempotency-key-reuse")
	// A key is required.
	a.callWith("POST", "/v1/sync", u.access, nil, map[string]any{"ops": ops}).problem(400, "bad-request")
	a.push(u, newID(), []map[string]any{}...).problem(422, "validation")

	// What was pushed comes back in the feed, bodyweight included.
	seen, _, _ := a.pullAll(u, 0, 100)
	kinds := map[string]int{}
	for _, row := range seen {
		kinds[row["_kind"].(string)]++
	}
	if kinds["sessions"] != 1 || kinds["blocks"] != 1 || kinds["sets"] != 2 || kinds["bodyweight"] != 1 {
		t.Fatalf("feed after push %v", kinds)
	}
}

func TestSyncPushConflicts(t *testing.T) {
	l := newLogger(t, "conflicts@example.test")
	pullUp := l.a.exerciseID("pull-up")
	setID := newID()
	t2 := time.Now().Add(-time.Minute)
	put := func(n int, clock time.Time) map[string]any {
		return map[string]any{"op": "put", "entity": "set", "id": setID, "session_id": l.session, "data": map[string]any{
			"block_id": l.block, "order_index": 0, "updated_at": at(clock), "elements": []any{reps(pullUp, n)},
		}}
	}
	l.a.push(l.u, newID(), put(8, t2)).ok(200, "SyncPushResult")

	// A copy of the set older by the athlete's clock loses, whenever it arrives.
	rs := wantStatuses(t, l.a.push(l.u, newID(), put(3, t2.Add(-time.Hour))).ok(200, "SyncPushResult"), "superseded")
	if rs[0]["problem"].(map[string]any)["type"] != "https://hefesto.fit/problems/stale-write" {
		t.Fatalf("stale write %v", rs[0])
	}
	s := l.a.call("GET", l.path(), l.u.access, nil).ok(200, "Session")
	if got := s["blocks"].([]any)[0].(map[string]any)["sets"].([]any)[0].(map[string]any)["elements"].([]any)[0].(map[string]any)["reps"]; got != float64(8) {
		t.Fatalf("a stale write changed the set: %v", got)
	}
	// REST follows the same rule when the client sends its clock.
	l.a.call("PUT", l.path("sets", setID), l.u.access, map[string]any{
		"block_id": l.block, "order_index": 0, "updated_at": at(t2.Add(-time.Hour)), "elements": []any{reps(pullUp, 3)},
	}).problem(409, "stale-write")
	// A newer copy wins.
	wantStatuses(t, l.a.push(l.u, newID(), put(9, t2.Add(time.Second))).ok(200, "SyncPushResult"), "applied")

	// Deletions are final: writes to a tombstone are superseded; deleting it
	// again is a no-op that applies.
	l.a.call("DELETE", l.path("sets", setID), l.u.access, nil).ok(204, "")
	del := map[string]any{"op": "delete", "entity": "set", "id": setID, "session_id": l.session}
	wantStatuses(t, l.a.push(l.u, newID(), put(10, time.Now()), del).ok(200, "SyncPushResult"), "superseded", "applied")

	l.a.call("DELETE", l.path(), l.u.access, nil).ok(204, "")
	wantStatuses(t, l.a.push(l.u, newID(),
		map[string]any{"op": "put", "entity": "session", "id": l.session, "data": map[string]any{
			"started_at": at(time.Now()), "timezone": "Europe/Zurich"}},
		map[string]any{"op": "put", "entity": "block", "id": newID(), "session_id": l.session, "data": map[string]any{"order_index": 1}},
	).ok(200, "SyncPushResult"), "superseded", "superseded")
}

func TestSyncPushRejectsBadOpsOneByOne(t *testing.T) {
	l := newLogger(t, "rejects@example.test")
	pullUp := l.a.exerciseID("pull-up")
	bad := map[string]any{"op": "put", "entity": "set", "id": newID(), "session_id": l.session, "data": map[string]any{
		"block_id": l.block, "order_index": 0,
		"elements": []any{map[string]any{"id": newID(), "exercise_id": pullUp, "measure": "reps", "hold_seconds": 10}},
	}}
	good := map[string]any{"op": "put", "entity": "set", "id": newID(), "session_id": l.session, "data": map[string]any{
		"block_id": l.block, "order_index": 1, "elements": []any{reps(pullUp, 5)},
	}}
	rs := wantStatuses(t, l.a.push(l.u, newID(),
		bad,
		map[string]any{"op": "complete", "entity": "block", "id": newID(), "session_id": l.session},
		map[string]any{"op": "put", "entity": "block", "id": newID(), "data": map[string]any{"order_index": 3}},
		map[string]any{"op": "delete", "entity": "set", "id": newID(), "session_id": l.session},
		map[string]any{"op": "put", "entity": "session", "id": newID(), "data": map[string]any{"timezone": "Europe/Zurich"}},
		good,
	).ok(200, "SyncPushResult"), "rejected", "rejected", "rejected", "rejected", "rejected", "applied")

	errs := rs[0]["problem"].(map[string]any)["errors"].(map[string]any)
	found := false
	for k := range errs {
		if len(k) > len("/data/elements/0") && k[:len("/data/elements/0")] == "/data/elements/0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("errors not located in the op's data: %v", errs)
	}
	if rs[3]["problem"].(map[string]any)["type"] != "https://hefesto.fit/problems/not-found" {
		t.Fatalf("delete of an unknown set %v", rs[3])
	}
}

func TestBodyweightViaSync(t *testing.T) {
	a := newAPI(t)
	u := a.register("bodyweight@example.test")
	id := newID()
	// 23:30 UTC is already the next day in Zurich.
	put := map[string]any{"op": "put", "entity": "bodyweight", "id": id, "data": map[string]any{
		"measured_at": "2026-09-20T23:30:00Z", "timezone": "Europe/Zurich", "bodyweight_kg": 70.2, "note": "morning",
	}}
	wantStatuses(t, a.push(u, newID(), put).ok(200, "SyncPushResult"), "applied")
	seen, cursor, _ := a.pullAll(u, 0, 100)
	if b := seen[id]; b["local_date"] != "2026-09-21" || b["bodyweight_kg"] != 70.2 || b["note"] != "morning" {
		t.Fatalf("bodyweight %v", b)
	}
	wantStatuses(t, a.push(u, newID(), map[string]any{"op": "delete", "entity": "bodyweight", "id": id}, put).ok(200, "SyncPushResult"),
		"applied", "superseded")
	seen, _, _ = a.pullAll(u, cursor, 100)
	if seen[id]["deleted_at"] == nil {
		t.Fatalf("bodyweight tombstone %v", seen[id])
	}
	// Another athlete cannot write over it.
	other := a.register("bodyweight-other@example.test")
	wantStatuses(t, a.push(other, newID(), put).ok(200, "SyncPushResult"), "rejected")
}
