package progress

import (
	"sort"
	"time"
)

const dateLayout = "2006-01-02"

// StreakRules tune streaks.
type StreakRules struct {
	InitialFreezes int // credits a new athlete starts with
	EarnEvery      int // one credit per this many counted days in a run
	MaxFreezes     int
}

// DefaultStreakRules: two freezes to start, one more per week of streak, at
// most three banked.
var DefaultStreakRules = StreakRules{InitialFreezes: 2, EarnEvery: 7, MaxFreezes: 3}

// Streak is the outcome of walking an athlete's days.
type Streak struct {
	Current       int
	Longest       int
	FreezeCredits int
	// Frozen are the days a freeze bridged.
	Frozen []time.Time
	// LastCounted is the most recent counted day, if any.
	LastCounted *time.Time
	// RunStart is the first day of the current run, when there is one.
	RunStart *time.Time
}

// ComputeStreak walks every day from the first counted day to today.
//
// A day counts if the athlete trained, logged a planned rest day, or was in
// a deload (ADR 0003 §1): rest is training. A gap of uncounted days is
// bridged by freezes only if there are enough credits for all of it; a
// bridged day keeps the run alive without lengthening it. Today never breaks
// a streak — it is not over yet. Freezes are earned automatically, never
// bought.
//
// counted and today are civil dates (only Y-M-D matter). It is a pure
// function of its inputs, so the streak can always be recomputed from
// user_training_days.
func ComputeStreak(counted []time.Time, today time.Time, rules StreakRules) Streak {
	days := map[string]bool{}
	for _, d := range counted {
		d = civil(d)
		if d.After(civil(today)) {
			continue // a future day is a clock error; it cannot count yet
		}
		days[d.Format(dateLayout)] = true
	}
	s := Streak{FreezeCredits: rules.InitialFreezes}
	if len(days) == 0 {
		return s
	}
	sorted := make([]string, 0, len(days))
	for k := range days {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	first, _ := time.Parse(dateLayout, sorted[0])
	last, _ := time.Parse(dateLayout, sorted[len(sorted)-1])
	s.LastCounted = &last

	today = civil(today)
	run := 0
	var runStart time.Time
	for d := first; !d.After(today); {
		if days[d.Format(dateLayout)] {
			if run == 0 {
				runStart = d
			}
			run++
			if rules.EarnEvery > 0 && run%rules.EarnEvery == 0 && s.FreezeCredits < rules.MaxFreezes {
				s.FreezeCredits++
			}
			if run > s.Longest {
				s.Longest = run
			}
			d = d.AddDate(0, 0, 1)
			continue
		}
		// A gap: measure it up to the next counted day, or up to today (which
		// itself never counts against the athlete).
		gapEnd := d
		for !gapEnd.After(today) && !days[gapEnd.Format(dateLayout)] {
			gapEnd = gapEnd.AddDate(0, 0, 1)
		}
		gap := int(gapEnd.Sub(d).Hours() / 24)
		if gapEnd.After(today) {
			gap-- // today is still open
		}
		switch {
		case gap <= 0:
		case run > 0 && gap <= s.FreezeCredits:
			s.FreezeCredits -= gap
			for i := 0; i < gap; i++ {
				s.Frozen = append(s.Frozen, d.AddDate(0, 0, i))
			}
		default:
			run = 0
		}
		d = gapEnd
	}
	s.Current = run
	if run > 0 {
		rs := runStart
		s.RunStart = &rs
	}
	return s
}

func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
