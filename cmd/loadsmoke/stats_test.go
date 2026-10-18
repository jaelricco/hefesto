package main

import (
	"strings"
	"testing"
	"time"
)

func TestPercentileIsNearestRank(t *testing.T) {
	var lat []time.Duration
	for i := 1; i <= 100; i++ {
		lat = append(lat, time.Duration(i)*time.Millisecond)
	}
	for p, want := range map[int]time.Duration{50: 50 * time.Millisecond, 95: 95 * time.Millisecond, 99: 99 * time.Millisecond, 100: 100 * time.Millisecond} {
		if got := percentile(lat, p); got != want {
			t.Errorf("p%d = %s, want %s", p, got, want)
		}
	}
	if got := percentile([]time.Duration{7 * time.Millisecond}, 95); got != 7*time.Millisecond {
		t.Errorf("single sample p95 = %s", got)
	}
	if got := percentile(nil, 95); got != 0 {
		t.Errorf("empty p95 = %s", got)
	}
}

func TestReportFailsOnErrorsAndBudgets(t *testing.T) {
	b := budgets{classRead: 100 * time.Millisecond}

	r := &recorder{}
	for range 20 {
		r.add(sample{route: "GET /a", class: classRead, latency: 10 * time.Millisecond, ok: true})
		r.add(sample{route: "POST /login", class: classAuth, latency: 2 * time.Second, ok: true})
	}
	rep := r.report(time.Second, b)
	if !rep.Passed() {
		t.Fatalf("fast reads and a slow unbudgeted class should pass: %+v", rep)
	}

	// Anything over budget at p95 fails, even with no errors.
	for range 5 {
		r.add(sample{route: "GET /a", class: classRead, latency: 500 * time.Millisecond, ok: true})
	}
	if rep = r.report(time.Second, b); rep.Passed() || !rep.Routes[0].OverBudget {
		t.Fatalf("p95 over budget should fail: %+v", rep.Routes)
	}

	// One error fails the run, and its message is kept.
	r = &recorder{}
	r.add(sample{route: "GET /a", class: classRead, latency: time.Millisecond, ok: true})
	r.add(sample{route: "GET /a", class: classRead, latency: time.Millisecond})
	r.fail("GET /a: status 500")
	rep = r.report(time.Second, b)
	if rep.Passed() || rep.Errors != 1 || rep.Routes[0].Errors != 1 {
		t.Fatalf("an error should fail the run: %+v", rep)
	}
	var out strings.Builder
	if err := rep.write(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "status 500") || !strings.Contains(out.String(), "GET /a") {
		t.Fatalf("report %q", out.String())
	}

	// A run that made no requests proves nothing.
	if (&recorder{}).report(time.Second, b).Passed() {
		t.Fatal("an empty run should not pass")
	}
}
