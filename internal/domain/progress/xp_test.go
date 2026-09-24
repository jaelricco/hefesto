package progress

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSessionXP(t *testing.T) {
	sid := uuid.New()
	base := CompletedSession{ID: sid, FirstOfDay: true, PerformedElements: 12}
	cases := []struct {
		name string
		mod  func(*CompletedSession)
		want int
	}{
		{"a training session", func(*CompletedSession) {}, XPSessionBase},
		{"good rated form earns the bonus", func(s *CompletedSession) { s.FormRatings = []int{4, 5, 4, 4, 3} }, XPSessionBase + XPQualityBonus},
		{"mostly poor form earns no bonus", func(s *CompletedSession) { s.FormRatings = []int{4, 2, 3} }, XPSessionBase},
		{"too few ratings earn no bonus", func(s *CompletedSession) { s.FormRatings = []int{5, 5} }, XPSessionBase},
		{"volume earns nothing extra", func(s *CompletedSession) { s.PerformedElements = 500 }, XPSessionBase},
		{"second session of the day earns nothing", func(s *CompletedSession) { s.FirstOfDay = false }, 0},
		{"rest day earns nothing", func(s *CompletedSession) { s.IsRestDay = true }, 0},
		{"empty session earns nothing", func(s *CompletedSession) { s.PerformedElements = 0 }, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := base
			tc.mod(&s)
			total := 0
			for _, a := range SessionXP(s) {
				total += a.Amount
				if a.RefID != sid || a.Source != XPSessionCompleted {
					t.Fatalf("award %+v", a)
				}
			}
			if total != tc.want {
				t.Fatalf("got %d XP, want %d", total, tc.want)
			}
		})
	}
}

func TestUnlockXP(t *testing.T) {
	lvl := uuid.New()
	if a := UnlockXP(lvl, 7, "auto"); len(a) != 1 || a[0].Amount != XPUnlockBase+7*XPUnlockPerTier || a[0].RefID != lvl {
		t.Fatalf("auto unlock: %+v", a)
	}
	if a := UnlockXP(lvl, 7, "self_attested"); len(a) != 0 {
		t.Fatalf("self-attested unlock earned XP: %+v", a)
	}
}

func TestStreakXP(t *testing.T) {
	user := uuid.New()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if a := StreakXP(user, Streak{Current: 6, RunStart: &start}); len(a) != 0 {
		t.Fatalf("6 days earned %+v", a)
	}
	a := StreakXP(user, Streak{Current: 30, RunStart: &start})
	if len(a) != 2 || a[0].Amount != 20 || a[1].Amount != 50 {
		t.Fatalf("30 days: %+v", a)
	}
	again := StreakXP(user, Streak{Current: 31, RunStart: &start})
	if again[0].RefID != a[0].RefID {
		t.Fatal("milestone ref id is not stable within a run")
	}
	later := start.AddDate(0, 2, 0)
	if b := StreakXP(user, Streak{Current: 7, RunStart: &later}); b[0].RefID == a[0].RefID {
		t.Fatal("a new run cannot earn its milestones again")
	}
	if StreakXP(user, Streak{}) != nil {
		t.Fatal("no run, no award")
	}
}
