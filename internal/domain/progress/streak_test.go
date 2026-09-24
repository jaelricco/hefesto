package progress

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.Parse(dateLayout, s) // literals; a typo shows up as year 1
	return t
}

func days(ss ...string) []time.Time {
	out := make([]time.Time, len(ss))
	for i, s := range ss {
		out[i] = day(s)
	}
	return out
}

func span(from string, n int) []time.Time {
	out := make([]time.Time, n)
	for i := range out {
		out[i] = day(from).AddDate(0, 0, i)
	}
	return out
}

func TestComputeStreak(t *testing.T) {
	rules := StreakRules{InitialFreezes: 2, EarnEvery: 7, MaxFreezes: 3}
	cases := []struct {
		name            string
		counted         []time.Time
		today           string
		current, longst int
		credits         int
		frozen          int
	}{
		{"nothing logged", nil, "2026-09-30", 0, 0, 2, 0},
		{"three days up to today", days("2026-09-28", "2026-09-29", "2026-09-30"), "2026-09-30", 3, 3, 2, 0},
		{"today not logged yet does not break it", days("2026-09-28", "2026-09-29"), "2026-09-30", 2, 2, 2, 0},
		{"one missed day is bridged by a freeze", days("2026-09-27", "2026-09-29", "2026-09-30"), "2026-09-30", 3, 3, 1, 1},
		{"two missed days use both freezes", days("2026-09-26", "2026-09-29"), "2026-09-30", 2, 2, 0, 2},
		{"a gap longer than the credits breaks the run and keeps them", days("2026-09-20", "2026-09-21", "2026-09-25"), "2026-09-25", 1, 2, 2, 0},
		{"an open gap before today is bridged too", days("2026-09-27"), "2026-09-30", 1, 1, 0, 2},
		{"an open gap too long ends the streak", days("2026-09-20"), "2026-09-30", 0, 1, 2, 0},
		{"a week earns a credit, capped", span("2026-09-01", 14), "2026-09-14", 14, 14, 3, 0},
		{"future days do not count", days("2026-09-30", "2026-10-01"), "2026-09-30", 1, 1, 2, 0},
		{"duplicates count once", days("2026-09-30", "2026-09-30"), "2026-09-30", 1, 1, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := ComputeStreak(tc.counted, day(tc.today), rules)
			if s.Current != tc.current || s.Longest != tc.longst || s.FreezeCredits != tc.credits || len(s.Frozen) != tc.frozen {
				t.Fatalf("got current=%d longest=%d credits=%d frozen=%d; want %d %d %d %d",
					s.Current, s.Longest, s.FreezeCredits, len(s.Frozen), tc.current, tc.longst, tc.credits, tc.frozen)
			}
			if s.Current > 0 && s.RunStart == nil {
				t.Fatal("a live run has no start")
			}
		})
	}
}

// Planned rest is the point: a week of training three days and resting four,
// all logged, is a seven-day streak.
func TestRestDaysKeepTheStreak(t *testing.T) {
	s := ComputeStreak(span("2026-09-01", 7), day("2026-09-07"), DefaultStreakRules)
	if s.Current != 7 {
		t.Fatalf("current %d", s.Current)
	}
}
