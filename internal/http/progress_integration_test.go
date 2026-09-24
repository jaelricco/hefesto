//go:build integration

package http_test

import (
	"testing"
	"time"
)

// trainer logs and completes whole sessions for one user, on days relative
// to today, because the evaluator and the streak both measure from now.
type trainer struct {
	t   *testing.T
	a   *api
	u   user
	loc *time.Location
}

func newTrainer(t *testing.T, email string) trainer {
	t.Helper()
	a := newAPI(t)
	loc, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}
	return trainer{t: t, a: a, u: a.register(email), loc: loc}
}

// day is when a session daysAgo days back starts: local noon, or half an hour
// ago for today so nothing lands in the future.
func (tr trainer) day(daysAgo int) time.Time {
	now := time.Now().In(tr.loc)
	if daysAgo == 0 {
		return now.Add(-30 * time.Minute)
	}
	d := now.AddDate(0, 0, -daysAgo)
	return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, tr.loc)
}

// log creates a session with one block holding the given sets, each a list
// of elements, and leaves it a draft. It returns the session and set ids.
func (tr trainer) log(at time.Time, restDay bool, sets ...[]map[string]any) (string, []string) {
	tr.t.Helper()
	session, block := newID(), newID()
	tr.a.call("POST", "/v1/sessions", tr.u.access, map[string]any{
		"id": session, "started_at": at.UTC().Format(time.RFC3339), "timezone": "Europe/Zurich", "is_rest_day": restDay,
	}).ok(201, "Session")
	var ids []string
	if len(sets) == 0 {
		return session, ids
	}
	tr.a.call("PUT", "/v1/sessions/"+session+"/blocks/"+block, tr.u.access, map[string]any{"order_index": 0}).ok(201, "Block")
	for i, els := range sets {
		id := newID()
		tr.a.call("PUT", "/v1/sessions/"+session+"/sets/"+id, tr.u.access, map[string]any{
			"block_id": block, "order_index": i, "completed_at": at.Add(time.Duration(i+1) * time.Minute).UTC().Format(time.RFC3339),
			"elements": els,
		}).ok(201, "SetEntry")
		ids = append(ids, id)
	}
	return session, ids
}

func (tr trainer) complete(session string) map[string]any {
	tr.t.Helper()
	return tr.a.call("POST", "/v1/sessions/"+session+"/complete", tr.u.access, map[string]any{}).ok(200, "CompletionResult")
}

// train logs a session and completes it.
func (tr trainer) train(daysAgo int, sets ...[]map[string]any) map[string]any {
	tr.t.Helper()
	s, _ := tr.log(tr.day(daysAgo), false, sets...)
	return tr.complete(s)
}

// states returns the skill map's states keyed by "skill/level".
func (tr trainer) states() map[string]map[string]any {
	tr.t.Helper()
	m := tr.a.call("GET", "/v1/me/skill-map", tr.u.access, nil).ok(200, "SkillMap")
	names := map[string]string{}
	for _, s := range m["skills"].([]any) {
		sk := s.(map[string]any)
		for _, l := range sk["levels"].([]any) {
			lv := l.(map[string]any)
			names[lv["id"].(string)] = sk["slug"].(string) + "/" + lv["slug"].(string)
		}
	}
	out := map[string]map[string]any{}
	for _, s := range m["states"].([]any) {
		st := s.(map[string]any)
		out[names[st["level_id"].(string)]] = st
	}
	return out
}

func (tr trainer) levelID(key string) string {
	tr.t.Helper()
	st, ok := tr.states()[key]
	if !ok {
		tr.t.Fatalf("no level %s", key)
	}
	return st["level_id"].(string)
}

func (tr trainer) wantStates(want map[string]string) {
	tr.t.Helper()
	got := tr.states()
	for k, w := range want {
		if got[k]["state"] != w {
			tr.t.Errorf("%s: state %v, want %s", k, got[k]["state"], w)
		}
	}
}

func awards(c map[string]any) map[string]float64 {
	out := map[string]float64{}
	for _, a := range c["xp_awarded"].([]any) {
		aw := a.(map[string]any)
		out[aw["source"].(string)] += aw["amount"].(float64)
	}
	return out
}

func unlockedKeys(c map[string]any) []string {
	var out []string
	for _, u := range c["unlocked"].([]any) {
		un := u.(map[string]any)
		out = append(out, un["skill_slug"].(string)+"/"+un["level_slug"].(string))
	}
	return out
}

func streakOf(m map[string]any) map[string]any { return m["streak"].(map[string]any) }

func TestCompletionUnlocksFromTheLog(t *testing.T) {
	tr := newTrainer(t, "unlock@example.test")
	pullUp := tr.a.exerciseID("pull-up")
	tr.wantStates(map[string]string{
		"pull-up/strict-5": "available", "front-lever/tuck": "locked", "front-lever/advanced-tuck": "locked",
		"handstand/wall": "available",
	})

	// One qualifying set; the criteria want two occurrences.
	c := tr.train(1, []map[string]any{reps(pullUp, 5)})
	if len(unlockedKeys(c)) != 0 || c["already_completed"] != false {
		t.Fatalf("first session unlocked something: %v", c)
	}
	if got := awards(c); got["session_completed"] != 10 || len(got) != 1 {
		t.Fatalf("first session xp %v", got)
	}
	if st := tr.states()["pull-up/strict-5"]; st["state"] != "in_progress" || st["best_value"] != float64(5) || st["best_unit"] != "reps" {
		t.Fatalf("after one set: %v", st)
	}

	// An assisted set does not count towards `assistance: none`, and a draft
	// session is not evidence at all.
	assisted := reps(pullUp, 8)
	assisted["assistance"] = map[string]any{"id": newID(), "type": "counterweight", "estimated_assist_kg": 10}
	tr.train(1, []map[string]any{assisted})
	tr.log(tr.day(0), false, []map[string]any{reps(pullUp, 9)})
	tr.wantStates(map[string]string{"pull-up/strict-5": "in_progress"})

	// The second qualifying set, today, unlocks it.
	session, sets := tr.log(tr.day(0), false, []map[string]any{reps(pullUp, 6)})
	c = tr.complete(session)
	if got := unlockedKeys(c); len(got) != 1 || got[0] != "pull-up/strict-5" {
		t.Fatalf("unlocked %v", got)
	}
	u := c["unlocked"].([]any)[0].(map[string]any)
	if u["verification"] != "auto" || u["evidence_set_entry_id"] != sets[0] {
		t.Fatalf("unlock %v", u)
	}
	tuck := tr.levelID("front-lever/tuck")
	if na := c["newly_available"].([]any); len(na) != 1 || na[0] != tuck {
		t.Fatalf("newly available %v, want [%s]", na, tuck)
	}
	// The first session today earns session XP; the unlock earns 20 + 5×tier 2.
	if got := awards(c); got["session_completed"] != 10 || got["skill_unlocked"] != 30 {
		t.Fatalf("xp %v", got)
	}
	total := c["xp_total"].(float64)
	if total != 50 { // 10 + 10 on the earlier day (the day's first only) + 30
		t.Fatalf("xp total %v", total)
	}
	tr.wantStates(map[string]string{
		"pull-up/strict-5": "unlocked", "front-lever/tuck": "available", "front-lever/advanced-tuck": "locked",
	})
	if st := tr.states()["pull-up/strict-5"]; st["verification"] != "auto" || st["first_achieved_at"] == nil {
		t.Fatalf("unlocked state %v", st)
	}

	// Completing again is idempotent: the same unlock, no new XP.
	again := tr.complete(session)
	if again["already_completed"] != true || len(unlockedKeys(again)) != 1 || len(awards(again)) != 0 || again["xp_total"] != total {
		t.Fatalf("repeat completion %v", again)
	}

	// A second session the same day earns no session XP.
	c = tr.train(0, []map[string]any{reps(pullUp, 3)})
	if got := awards(c); got["session_completed"] != 0 {
		t.Fatalf("second session of the day xp %v", got)
	}

	// Unlocks are never revoked: deleting the evidence changes nothing.
	tr.a.call("DELETE", "/v1/sessions/"+session, tr.u.access, nil).ok(204, "")
	tr.wantStates(map[string]string{"pull-up/strict-5": "unlocked"})
}

func TestUnlockChainAndPrerequisiteGating(t *testing.T) {
	tr := newTrainer(t, "chain@example.test")
	tuck := tr.a.exerciseID("front-lever-tuck")

	// Tuck holds that meet the criteria do not unlock a level whose
	// prerequisite is still locked; the athlete's progress is still shown.
	tr.train(0, []map[string]any{hold(tuck, 20)}, []map[string]any{hold(tuck, 18)})
	if st := tr.states()["front-lever/tuck"]; st["state"] != "locked" || st["best_value"] != float64(20) {
		t.Fatalf("gated level %v", st)
	}

	// Unlocking the prerequisite lets the next completion unlock the tuck from
	// the evidence already logged.
	pullUp := tr.a.exerciseID("pull-up")
	c := tr.train(0, []map[string]any{reps(pullUp, 5)}, []map[string]any{reps(pullUp, 5)})
	got := unlockedKeys(c)
	if len(got) != 2 || got[0] != "pull-up/strict-5" || got[1] != "front-lever/tuck" {
		t.Fatalf("unlocked %v", got)
	}
	// Advanced tuck has no criteria: available, but only by self-attest.
	tr.wantStates(map[string]string{"front-lever/tuck": "unlocked", "front-lever/advanced-tuck": "available"})
}

func TestSelfAttest(t *testing.T) {
	tr := newTrainer(t, "attest@example.test")
	wall := tr.levelID("handstand/wall")
	adv := tr.levelID("front-lever/advanced-tuck")

	m := tr.a.call("POST", "/v1/me/skills/"+wall+"/attest", tr.u.access, map[string]any{}).ok(200, "AttestResult")
	u := m["unlock"].(map[string]any)
	if u["verification"] != "self_attested" || u["level_id"] != wall || u["evidence_set_entry_id"] != nil {
		t.Fatalf("attest %v", m)
	}
	// Attesting again returns the same unlock.
	again := tr.a.call("POST", "/v1/me/skills/"+wall+"/attest", tr.u.access, map[string]any{}).ok(200, "AttestResult")
	if got := again["unlock"].(map[string]any)["occurred_at"]; got != u["occurred_at"] {
		t.Fatalf("second attest occurred_at %v, first %v", got, u["occurred_at"])
	}
	// Self-attested unlocks earn no XP.
	if p := tr.a.call("GET", "/v1/me/progress", tr.u.access, nil).ok(200, "Progress"); p["xp_total"] != float64(0) {
		t.Fatalf("progress %v", p)
	}
	if st := tr.states()["handstand/wall"]; st["state"] != "unlocked" || st["verification"] != "self_attested" {
		t.Fatalf("state %v", st)
	}

	tr.a.call("POST", "/v1/me/skills/"+adv+"/attest", tr.u.access, map[string]any{}).problem(409, "prerequisites-unmet")
	tr.a.call("POST", "/v1/me/skills/"+newID()+"/attest", tr.u.access, map[string]any{}).problem(404, "not-found")
	tr.a.call("POST", "/v1/me/skills/"+wall+"/attest", tr.u.access, map[string]any{"note": "x"}).problem(422, "validation")
}

func TestRestDaysKeepTheStreak(t *testing.T) {
	tr := newTrainer(t, "rest@example.test")
	pullUp := tr.a.exerciseID("pull-up")

	tr.train(2, []map[string]any{reps(pullUp, 3)})
	rest, _ := tr.log(tr.day(1), true)
	c := tr.complete(rest)
	if len(awards(c)) != 0 || len(unlockedKeys(c)) != 0 {
		t.Fatalf("rest day earned something: %v", c)
	}
	if s := streakOf(c); s["current_days"] != float64(2) {
		t.Fatalf("streak after rest day %v", s)
	}
	// Today is not over, so not having trained yet breaks nothing.
	p := tr.a.call("GET", "/v1/me/progress", tr.u.access, nil).ok(200, "Progress")
	if s := streakOf(p); s["current_days"] != float64(2) || s["last_counted_date"] != tr.day(1).Format("2006-01-02") {
		t.Fatalf("progress streak %v", s)
	}
	if p["xp_total"] != float64(10) {
		t.Fatalf("xp %v", p["xp_total"])
	}
}

func TestStreakFreezeAndMilestone(t *testing.T) {
	tr := newTrainer(t, "streak@example.test")
	pullUp := tr.a.exerciseID("pull-up")

	// A missed day is bridged by a freeze: it keeps the run without adding to it.
	tr.train(3, []map[string]any{reps(pullUp, 3)})
	c := tr.train(1, []map[string]any{reps(pullUp, 3)})
	if s := streakOf(c); s["current_days"] != float64(2) || s["freeze_credits"] != float64(1) {
		t.Fatalf("bridged streak %v", s)
	}

	other := newTrainer(t, "milestone@example.test")
	otherPullUp := other.a.exerciseID("pull-up")
	var last map[string]any
	for d := 6; d >= 0; d-- {
		last = other.train(d, []map[string]any{reps(otherPullUp, 3)})
	}
	if s := streakOf(last); s["current_days"] != float64(7) {
		t.Fatalf("streak %v", s)
	}
	if got := awards(last); got["streak_milestone"] != 20 {
		t.Fatalf("milestone xp %v", got)
	}
}

func TestCompletionErrors(t *testing.T) {
	tr := newTrainer(t, "complete-errors@example.test")
	s, _ := tr.log(tr.day(0), false)
	tr.a.call("PATCH", "/v1/sessions/"+s, tr.u.access, map[string]any{"status": "abandoned"}).ok(200, "Session")
	tr.a.call("POST", "/v1/sessions/"+s+"/complete", tr.u.access, map[string]any{}).problem(422, "validation")
	tr.a.call("POST", "/v1/sessions/"+newID()+"/complete", tr.u.access, map[string]any{}).problem(404, "not-found")
	tr.a.call("POST", "/v1/sessions/"+s+"/complete", tr.u.access, map[string]any{"perceived_fatigue": 11}).problem(422, "validation")

	// Another athlete cannot complete my session.
	mine, _ := tr.log(tr.day(0), false)
	other := tr.a.register("intruder@example.test")
	tr.a.call("POST", "/v1/sessions/"+mine+"/complete", other.access, map[string]any{}).problem(404, "not-found")

	c := tr.a.call("POST", "/v1/sessions/"+mine+"/complete", tr.u.access, map[string]any{"perceived_fatigue": 6}).ok(200, "CompletionResult")
	sess := c["session"].(map[string]any)
	if sess["status"] != "completed" || sess["completed_at"] == nil || sess["perceived_fatigue"] != float64(6) {
		t.Fatalf("session %v", sess)
	}
	// An empty session earns nothing but still counts for the streak.
	if len(awards(c)) != 0 || streakOf(c)["current_days"] != float64(1) {
		t.Fatalf("empty session %v", c)
	}
}

func TestSkillGraphAndDetail(t *testing.T) {
	tr := newTrainer(t, "graph@example.test")
	r := tr.a.call("GET", "/v1/skills", tr.u.access, nil)
	g := r.ok(200, "SkillGraph")
	if n := len(g["skills"].([]any)); n != 3 {
		t.Fatalf("skills %d", n)
	}
	etag := r.header.Get("ETag")
	if etag == "" || etag == `""` {
		t.Fatalf("etag %q", etag)
	}
	req := tr.a.callWith("GET", "/v1/skills", tr.u.access, map[string]string{"If-None-Match": etag})
	if req.status != 304 {
		t.Fatalf("conditional get: %d", req.status)
	}

	// The tuck requires strict pull-ups: an edge in the graph.
	strict, tuck := tr.levelID("pull-up/strict-5"), tr.levelID("front-lever/tuck")
	found := false
	for _, e := range g["edges"].([]any) {
		ed := e.(map[string]any)
		if ed["from_level_id"] == strict && ed["to_level_id"] == tuck && ed["relation"] == "prerequisite" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no prerequisite edge strict-5 -> tuck in %v", g["edges"])
	}

	d := tr.a.call("GET", "/v1/skills/front-lever", tr.u.access, nil).ok(200, "SkillDetail")
	if d["injury_disclaimer"].(map[string]any)["code"] != "educational_only" {
		t.Fatalf("detail %v", d)
	}
	inj := d["injuries"].([]any)
	if len(inj) != 1 || inj[0].(map[string]any)["disclaimer"] != "educational_only" ||
		len(inj[0].(map[string]any)["prehab_exercises"].([]any)) != 1 {
		t.Fatalf("injuries %v", inj)
	}
	tr.a.call("GET", "/v1/skills/no-such-skill", tr.u.access, nil).problem(404, "not-found")
	tr.a.call("GET", "/v1/skills/Bad_Slug", tr.u.access, nil).problem(400, "bad-request")
	tr.a.call("GET", "/v1/me/skill-map", "", nil).problem(401, "unauthorized")
}
