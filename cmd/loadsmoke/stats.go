package main

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
	"text/tabwriter"
	"time"
)

// class groups routes that share a latency budget.
type class string

const (
	classRead     class = "read"
	classWrite    class = "write"
	classComplete class = "complete" // evaluates unlocks, awards XP
	classAuth     class = "auth"     // argon2 by design; never budgeted
)

// budgets are the p95 each class must stay under. A zero budget is not checked.
type budgets map[class]time.Duration

// sample is one request as the athlete saw it.
type sample struct {
	route   string
	class   class
	latency time.Duration
	ok      bool
}

// recorder collects samples from every athlete.
type recorder struct {
	mu       sync.Mutex
	samples  []sample
	errs     []string
	failures int
}

func (r *recorder) add(s sample) {
	r.mu.Lock()
	r.samples = append(r.samples, s)
	r.mu.Unlock()
}

// fail records why a request counted as an error. Only the first few are kept.
func (r *recorder) fail(msg string) {
	r.mu.Lock()
	r.failures++
	if len(r.errs) < 20 {
		r.errs = append(r.errs, msg)
	}
	r.mu.Unlock()
}

// routeStats summarises one route.
type routeStats struct {
	Route         string
	Class         class
	Count, Errors int
	P50, P95, P99 time.Duration
	Max           time.Duration
	Budget        time.Duration
	OverBudget    bool
}

// report is the outcome of a run.
type report struct {
	Routes   []routeStats
	Requests int
	Errors   int
	Elapsed  time.Duration
	Samples  []string // first error messages
}

func (r report) Passed() bool {
	if r.Errors > 0 || r.Requests == 0 {
		return false
	}
	for _, s := range r.Routes {
		if s.OverBudget {
			return false
		}
	}
	return true
}

func (r *recorder) report(elapsed time.Duration, b budgets) report {
	r.mu.Lock()
	defer r.mu.Unlock()
	byRoute := map[string][]sample{}
	for _, s := range r.samples {
		byRoute[s.route] = append(byRoute[s.route], s)
	}
	out := report{Elapsed: elapsed, Samples: slices.Clone(r.errs), Errors: r.failures}
	for route, ss := range byRoute {
		lat := make([]time.Duration, 0, len(ss))
		errs := 0
		for _, s := range ss {
			if s.ok {
				lat = append(lat, s.latency)
			} else {
				errs++
			}
		}
		slices.Sort(lat)
		st := routeStats{
			Route: route, Class: ss[0].class, Count: len(ss), Errors: errs,
			P50: percentile(lat, 50), P95: percentile(lat, 95), P99: percentile(lat, 99),
			Budget: b[ss[0].class],
		}
		if len(lat) > 0 {
			st.Max = lat[len(lat)-1]
		}
		st.OverBudget = st.Budget > 0 && st.P95 > st.Budget
		out.Routes = append(out.Routes, st)
		out.Requests += len(ss)
	}
	sort.Slice(out.Routes, func(i, j int) bool { return out.Routes[i].Route < out.Routes[j].Route })
	return out
}

// percentile is the nearest-rank percentile of sorted latencies.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := (p*len(sorted) + 99) / 100 // ceil(p/100 * n)
	if rank < 1 {
		rank = 1
	}
	return sorted[rank-1]
}

func (r report) write(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	if _, err := fmt.Fprintln(tw, "route\tclass\tcount\terrors\tp50\tp95\tp99\tmax\tp95 budget\t"); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	for _, s := range r.Routes {
		budget := "-"
		if s.Budget > 0 {
			budget = ms(s.Budget)
			if s.OverBudget {
				budget += " OVER"
			}
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t\n",
			s.Route, s.Class, s.Count, s.Errors, ms(s.P50), ms(s.P95), ms(s.P99), ms(s.Max), budget); err != nil {
			return fmt.Errorf("writing report: %w", err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	rps := 0.0
	if r.Elapsed > 0 {
		rps = float64(r.Requests) / r.Elapsed.Seconds()
	}
	if _, err := fmt.Fprintf(w, "\n%d requests in %s (%.1f/s), %d errors\n", r.Requests, r.Elapsed.Round(time.Millisecond), rps, r.Errors); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	for _, e := range r.Samples {
		if _, err := fmt.Fprintf(w, "  error: %s\n", e); err != nil {
			return fmt.Errorf("writing report: %w", err)
		}
	}
	return nil
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}
