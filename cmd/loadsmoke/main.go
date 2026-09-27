// Command loadsmoke is a load smoke test: a few dozen virtual athletes log
// sessions, sync and read the skill map against a running API, then it
// reports latency per route and fails on any error or a p95 over budget.
//
// It is a smoke test, not a benchmark: it answers "does the API hold up
// under a realistic handful of concurrent athletes", on a fresh stack in CI
// or against staging before a release. Never point it at production; it
// creates accounts.
//
//	go run ./cmd/loadsmoke -base-url http://localhost:8080 -athletes 25 -duration 60s
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type options struct {
	baseURL  string
	athletes int
	duration time.Duration
	think    time.Duration
	timeout  time.Duration
	budgets  budgets
}

func main() {
	os.Exit(run())
}

func run() int {
	o := options{budgets: budgets{}}
	var read, write, complete time.Duration
	flag.StringVar(&o.baseURL, "base-url", "http://localhost:8080", "the API to load")
	flag.IntVar(&o.athletes, "athletes", 25, "concurrent virtual athletes")
	flag.DurationVar(&o.duration, "duration", time.Minute, "how long athletes keep training after registering")
	flag.DurationVar(&o.think, "think", 250*time.Millisecond, "pause between an athlete's workouts")
	flag.DurationVar(&o.timeout, "timeout", 10*time.Second, "per-request timeout")
	flag.DurationVar(&read, "p95-read", 300*time.Millisecond, "p95 budget for reads (0 disables)")
	flag.DurationVar(&write, "p95-write", 400*time.Millisecond, "p95 budget for single writes (0 disables)")
	flag.DurationVar(&complete, "p95-complete", 1500*time.Millisecond, "p95 budget for completing a session and sync batches (0 disables)")
	flag.Parse()
	o.budgets[classRead], o.budgets[classWrite], o.budgets[classComplete] = read, write, complete

	if o.athletes < 1 || o.duration <= 0 {
		fmt.Fprintln(os.Stderr, "loadsmoke: -athletes must be at least 1 and -duration positive")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rep, err := load(ctx, o)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsmoke: %v\n", err)
		return 1
	}
	if err := rep.write(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "loadsmoke: %v\n", err)
		return 1
	}
	if !rep.Passed() {
		fmt.Println("FAIL")
		return 1
	}
	fmt.Println("PASS")
	return 0
}

// load registers every athlete, then lets them train concurrently for the
// configured duration. Registration is reported but excluded from the
// elapsed time, since it is deliberately slow (argon2).
func load(ctx context.Context, o options) (report, error) {
	rec := &recorder{}
	transport := &http.Transport{MaxIdleConns: o.athletes * 2, MaxIdleConnsPerHost: o.athletes * 2}
	hc := &http.Client{Timeout: o.timeout, Transport: transport}
	runID := strconv.FormatInt(time.Now().UnixNano(), 36)

	if err := ready(ctx, hc, o.baseURL); err != nil {
		return report{}, err
	}

	athletes := make([]*athlete, o.athletes)
	var cat catalogue
	for i := range athletes {
		a := &athlete{c: &client{base: o.baseURL, http: hc, rec: rec}, n: i}
		if err := a.register(ctx, runID); err != nil {
			return report{}, fmt.Errorf("registering athlete %d: %w", i, err)
		}
		if i == 0 {
			var err error
			if cat, err = loadCatalogue(ctx, a.c); err != nil {
				return report{}, err
			}
		}
		a.cat = cat
		athletes[i] = a
	}
	fmt.Printf("registered %d athletes, training for %s\n\n", o.athletes, o.duration)

	runCtx, cancel := context.WithTimeout(ctx, o.duration)
	defer cancel()
	start := time.Now()
	var wg sync.WaitGroup
	for _, a := range athletes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for runCtx.Err() == nil {
				// A failed workout is already recorded; the athlete carries on.
				_ = a.train(runCtx)
				select {
				case <-time.After(o.think):
				case <-runCtx.Done():
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	if ctx.Err() != nil {
		return report{}, fmt.Errorf("interrupted: %w", ctx.Err())
	}
	return rec.report(elapsed, o.budgets), nil
}

// ready waits up to 30 seconds for the API to report ready.
func ready(ctx context.Context, hc *http.Client, base string) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		req, err := http.NewRequestWithContext(ctx, "GET", base+"/readyz", nil)
		if err != nil {
			return fmt.Errorf("readiness: %w", err)
		}
		resp, err := hc.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			err = fmt.Errorf("status %d", resp.StatusCode)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("API at %s is not ready: %w", base, err)
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return errors.Join(ctx.Err(), err)
		}
	}
}
