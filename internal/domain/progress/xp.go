package progress

import (
	"strconv"

	"github.com/google/uuid"
)

// XP sources, as stored in user_xp_events.source.
const (
	XPSessionCompleted = "session_completed"
	XPSkillUnlocked    = "skill_unlocked"
	XPStreakMilestone  = "streak_milestone"
)

// XP amounts. XP rewards quality and progression, never volume (ADR 0003):
// there is no XP per set, per rep or per minute, and a second session on the
// same day earns nothing.
const (
	XPSessionBase     = 10
	XPQualityBonus    = 5
	XPUnlockBase      = 20
	XPUnlockPerTier   = 5
	qualityMinRated   = 3
	qualityGoodRating = 4
)

// StreakMilestones maps a streak length in days to its one-off award.
var StreakMilestones = []struct{ Days, XP int }{{7, 20}, {30, 50}, {100, 100}, {365, 250}}

// Award is one XP event. RefType/RefID make it unique: awarding the same
// thing twice is impossible at the database level.
type Award struct {
	Source  string
	Amount  int
	RefType string
	RefID   uuid.UUID
}

// CompletedSession is what session XP looks at.
type CompletedSession struct {
	ID        uuid.UUID
	IsRestDay bool
	// FirstOfDay: no other session of the athlete's local day was completed
	// before this one.
	FirstOfDay bool
	// PerformedElements counts elements with a value, from non-planned sets.
	PerformedElements int
	// FormRatings are the form_quality values the athlete recorded.
	FormRatings []int
}

// SessionXP awards a completed training session: a base amount, plus a
// bonus when the athlete rated their form and most of it was good. Rest-day
// sessions keep the streak but earn nothing, and so does an empty session.
func SessionXP(s CompletedSession) []Award {
	if s.IsRestDay || !s.FirstOfDay || s.PerformedElements == 0 {
		return nil
	}
	amount := XPSessionBase
	if n := len(s.FormRatings); n >= qualityMinRated {
		good := 0
		for _, r := range s.FormRatings {
			if r >= qualityGoodRating {
				good++
			}
		}
		if good*5 >= n*4 { // at least 80 %
			amount += XPQualityBonus
		}
	}
	return []Award{{Source: XPSessionCompleted, Amount: amount, RefType: "session", RefID: s.ID}}
}

// UnlockXP awards an unlock proved by logged evidence. Self-attested unlocks
// earn nothing: XP recognises what the log shows.
func UnlockXP(levelID uuid.UUID, difficultyTier int, verification string) []Award {
	if verification != "auto" {
		return nil
	}
	return []Award{{Source: XPSkillUnlocked, Amount: XPUnlockBase + XPUnlockPerTier*difficultyTier, RefType: "skill_level", RefID: levelID}}
}

// StreakXP awards every milestone the current run has reached. The ref id
// is derived from the user, the run's first day and the milestone, so a
// milestone is awarded once per run and again in a later run.
func StreakXP(userID uuid.UUID, s Streak) []Award {
	if s.RunStart == nil {
		return nil
	}
	var out []Award
	for _, m := range StreakMilestones {
		if s.Current >= m.Days {
			key := userID.String() + "/" + s.RunStart.Format(dateLayout) + "/" + strconv.Itoa(m.Days)
			out = append(out, Award{
				Source: XPStreakMilestone, Amount: m.XP, RefType: "streak",
				RefID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("hefesto:streak:"+key)),
			})
		}
	}
	return out
}
