package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// catalogue is what every athlete needs from the exercise list: one exercise
// logged in reps and one held, and the ETag for conditional reads.
type catalogue struct {
	reps, hold string
	etag       string
}

type exerciseList struct {
	Items []struct {
		ID             string `json:"id"`
		DefaultMeasure string `json:"default_measure"`
	} `json:"items"`
}

type authResponse struct {
	AccessToken string `json:"access_token"`
}

type syncPage struct {
	Cursor  int64 `json:"cursor"`
	HasMore bool  `json:"has_more"`
}

type pushResult struct {
	Results []struct {
		Index  int    `json:"index"`
		Status string `json:"status"`
	} `json:"results"`
}

func newID() string { return uuid.Must(uuid.NewV7()).String() }

// athlete is one virtual user: registered once, then training in a loop.
type athlete struct {
	c      *client
	n      int
	cat    catalogue
	cursor int64
	iter   int
}

// register creates the athlete's account. Registration hashes a password
// with argon2 on purpose, so it is reported but has no latency budget.
func (a *athlete) register(ctx context.Context, run string) error {
	var out authResponse
	_, err := a.c.do(ctx, call{
		method: "POST", path: "/v1/auth/register", route: "POST /v1/auth/register", class: classAuth,
		body: map[string]any{
			"email":    fmt.Sprintf("loadsmoke-%s-%d@example.test", run, a.n),
			"password": "loadsmoke correct horse battery",
			"timezone": "Europe/Zurich",
			"device":   map[string]any{"id": newID(), "platform": "ios"},
		},
		want: []int{http.StatusCreated}, out: &out,
	})
	if err != nil {
		return err
	}
	if out.AccessToken == "" {
		return errors.New("register: no access token")
	}
	a.c.token = out.AccessToken
	return nil
}

// loadCatalogue reads the exercise list once, before the measured run.
func loadCatalogue(ctx context.Context, c *client) (catalogue, error) {
	var list exerciseList
	req, err := http.NewRequestWithContext(ctx, "GET", c.base+"/v1/exercises", nil)
	if err != nil {
		return catalogue{}, fmt.Errorf("catalogue: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return catalogue{}, fmt.Errorf("catalogue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return catalogue{}, fmt.Errorf("catalogue: status %d", resp.StatusCode)
	}
	if err := decode(resp, &list); err != nil {
		return catalogue{}, fmt.Errorf("catalogue: %w", err)
	}
	cat := catalogue{etag: resp.Header.Get("ETag")}
	for _, e := range list.Items {
		switch {
		case e.DefaultMeasure == "reps" && cat.reps == "":
			cat.reps = e.ID
		case e.DefaultMeasure == "hold_seconds" && cat.hold == "":
			cat.hold = e.ID
		}
	}
	if cat.reps == "" || cat.hold == "" {
		return catalogue{}, errors.New("catalogue: the seeded content needs an exercise logged in reps and one held in seconds")
	}
	return cat, nil
}

// train is one pass of what the app does around a workout. Odd passes log
// through the sync endpoint in one batch, as the app's outbox does; even
// passes use the REST endpoints one write at a time. Both then pull the
// changes and read the map, as the app does after every sync.
func (a *athlete) train(ctx context.Context) error {
	a.iter++
	var err error
	if a.iter%2 == 1 {
		err = a.logOffline(ctx)
	} else {
		err = a.logOnline(ctx)
	}
	if err != nil {
		return err
	}
	return a.refresh(ctx)
}

func (a *athlete) sets() [][]map[string]any {
	reps := func(n int) map[string]any {
		return map[string]any{"id": newID(), "exercise_id": a.cat.reps, "measure": "reps", "reps": n}
	}
	hold := func(s float64) map[string]any {
		return map[string]any{"id": newID(), "exercise_id": a.cat.hold, "measure": "hold_seconds", "hold_seconds": s}
	}
	// Three straight sets and a combo: one code path for both.
	return [][]map[string]any{{reps(8)}, {reps(7)}, {reps(6)}, {hold(8), hold(5), hold(6)}}
}

func setBody(block string, i int, elements []map[string]any, at time.Time) map[string]any {
	return map[string]any{
		"block_id": block, "order_index": i, "rpe": 8, "rest_after_actual_s": 120,
		"completed_at": at.Add(time.Duration(i) * 3 * time.Minute).Format(time.RFC3339),
		"elements":     elements,
	}
}

// logOffline sends a whole session as one sync batch, the way the app drains
// its outbox after training without a connection.
func (a *athlete) logOffline(ctx context.Context) error {
	session, block := newID(), newID()
	start := time.Now().UTC().Add(-time.Hour)
	ops := []map[string]any{
		{"op": "put", "entity": "session", "id": session, "data": map[string]any{
			"started_at": start.Format(time.RFC3339), "timezone": "Europe/Zurich", "title": "Load smoke",
		}},
		{"op": "put", "entity": "block", "id": block, "session_id": session, "data": map[string]any{"order_index": 0}},
	}
	for i, els := range a.sets() {
		ops = append(ops, map[string]any{
			"op": "put", "entity": "set", "id": newID(), "session_id": session, "data": setBody(block, i, els, start),
		})
	}
	ops = append(ops, map[string]any{"op": "complete", "entity": "session", "id": session, "data": map[string]any{"perceived_fatigue": 7}})

	var out pushResult
	if _, err := a.c.do(ctx, call{
		method: "POST", path: "/v1/sync", route: "POST /v1/sync", class: classComplete,
		headers: map[string]string{"Idempotency-Key": newID()},
		body:    map[string]any{"ops": ops}, want: []int{http.StatusOK}, out: &out,
	}); err != nil {
		return err
	}
	if len(out.Results) != len(ops) {
		return a.fail("POST /v1/sync: %d results for %d ops", len(out.Results), len(ops))
	}
	for _, r := range out.Results {
		if r.Status != "applied" {
			return a.fail("POST /v1/sync: op %d %s", r.Index, r.Status)
		}
	}
	return nil
}

// logOnline logs a session one request at a time.
func (a *athlete) logOnline(ctx context.Context) error {
	session, block := newID(), newID()
	start := time.Now().UTC().Add(-time.Hour)
	sp := "/v1/sessions/" + session
	steps := []call{
		{method: "POST", path: "/v1/sessions", route: "POST /v1/sessions", class: classWrite,
			body: map[string]any{"id": session, "started_at": start.Format(time.RFC3339), "timezone": "Europe/Zurich", "title": "Load smoke"},
			want: []int{http.StatusCreated}},
		{method: "PUT", path: sp + "/blocks/" + block, route: "PUT /v1/sessions/{id}/blocks/{id}", class: classWrite,
			body: map[string]any{"order_index": 0}, want: []int{http.StatusCreated}},
	}
	for i, els := range a.sets() {
		steps = append(steps, call{method: "PUT", path: sp + "/sets/" + newID(), route: "PUT /v1/sessions/{id}/sets/{id}", class: classWrite,
			body: setBody(block, i, els, start), want: []int{http.StatusCreated}})
	}
	steps = append(steps,
		call{method: "GET", path: sp, route: "GET /v1/sessions/{id}", class: classRead, want: []int{http.StatusOK}},
		call{method: "POST", path: sp + "/complete", route: "POST /v1/sessions/{id}/complete", class: classComplete,
			body: map[string]any{"perceived_fatigue": 7}, want: []int{http.StatusOK}},
	)
	for _, s := range steps {
		if _, err := a.c.do(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// refresh is what the app reads after a sync.
func (a *athlete) refresh(ctx context.Context) error {
	for {
		var page syncPage
		if _, err := a.c.do(ctx, call{
			method: "GET", path: "/v1/sync?limit=200&cursor=" + strconv.FormatInt(a.cursor, 10),
			route: "GET /v1/sync", class: classRead, want: []int{http.StatusOK}, out: &page,
		}); err != nil {
			return err
		}
		a.cursor = page.Cursor
		if !page.HasMore {
			break
		}
	}
	reads := []call{
		{method: "GET", path: "/v1/me/skill-map", route: "GET /v1/me/skill-map", class: classRead, want: []int{http.StatusOK}},
		{method: "GET", path: "/v1/me/progress", route: "GET /v1/me/progress", class: classRead, want: []int{http.StatusOK}},
		{method: "GET", path: "/v1/sessions?limit=20", route: "GET /v1/sessions", class: classRead, want: []int{http.StatusOK}},
		{method: "GET", path: "/v1/exercises", route: "GET /v1/exercises (If-None-Match)", class: classRead,
			headers: map[string]string{"If-None-Match": a.cat.etag}, want: []int{http.StatusNotModified, http.StatusOK}},
	}
	for _, r := range reads {
		if _, err := a.c.do(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (a *athlete) fail(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	a.c.rec.fail(msg)
	return errors.New(msg)
}
