package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/domain/progress"
	"github.com/jaelricco/hefesto/internal/store"
)

// injuryDisclaimer travels with every injury note (ADR 0003): the payload
// itself says what the content is not.
var injuryDisclaimer = disclaimerOut{
	Code: "educational_only",
	Text: "Educational information about training-related injury risk, not medical advice. " +
		"It cannot diagnose or treat anything. If you have pain or an injury, see a qualified clinician.",
}

type disclaimerOut struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

type mapPointOut struct {
	Constellation string  `json:"constellation"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
}

type levelExerciseOut struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Slug       string    `json:"slug"`
	Role       string    `json:"role"`
}

type skillLevelOut struct {
	ID             uuid.UUID          `json:"id"`
	Slug           string             `json:"slug"`
	Name           string             `json:"name"`
	Order          int                `json:"order"`
	Description    string             `json:"description"`
	EstWeeks       *int               `json:"est_weeks_from_prev"`
	UnlockCriteria progress.Criteria  `json:"unlock_criteria"`
	Exercises      []levelExerciseOut `json:"exercises"`
}

type skillOut struct {
	ID             uuid.UUID       `json:"id"`
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Family         string          `json:"family"`
	DifficultyTier int             `json:"difficulty_tier"`
	IsMilestone    bool            `json:"is_milestone"`
	Status         string          `json:"status"`
	Aka            []string        `json:"aka"`
	Summary        string          `json:"summary"`
	PrimaryMuscles []string        `json:"primary_muscles"`
	CommonFaults   []string        `json:"common_faults"`
	Map            *mapPointOut    `json:"map"`
	Levels         []skillLevelOut `json:"levels"`
}

func skillFrom(s store.GraphSkill) skillOut {
	out := skillOut{
		ID: s.ID, Slug: s.Slug, Name: s.Name, Family: s.Family, DifficultyTier: s.DifficultyTier,
		IsMilestone: s.IsMilestone, Status: s.Status, Aka: s.Aka, Summary: s.Summary,
		PrimaryMuscles: s.PrimaryMuscles, CommonFaults: s.CommonFaults, Levels: make([]skillLevelOut, len(s.Levels)),
	}
	if s.Map != nil {
		out.Map = &mapPointOut{Constellation: s.Map.Constellation, X: s.Map.X, Y: s.Map.Y}
	}
	for i, l := range s.Levels {
		lo := skillLevelOut{
			ID: l.ID, Slug: l.Slug, Name: l.Name, Order: l.Order, Description: l.Description,
			EstWeeks: l.EstWeeks, UnlockCriteria: l.Criteria, Exercises: make([]levelExerciseOut, len(l.Exercises)),
		}
		for j, e := range l.Exercises {
			lo.Exercises[j] = levelExerciseOut{ExerciseID: e.ExerciseID, Slug: e.Slug, Role: e.Role}
		}
		out.Levels[i] = lo
	}
	return out
}

type edgeOut struct {
	From     uuid.UUID `json:"from_level_id"`
	To       uuid.UUID `json:"to_level_id"`
	Relation string    `json:"relation"`
	Weight   float64   `json:"weight"`
}

type skillGraphOut struct {
	ContentVersion string     `json:"content_version"`
	Skills         []skillOut `json:"skills"`
	Edges          []edgeOut  `json:"edges"`
}

func graphFrom(g store.Graph) skillGraphOut {
	out := skillGraphOut{ContentVersion: g.ContentVersion, Skills: make([]skillOut, len(g.Skills)), Edges: make([]edgeOut, len(g.Edges))}
	for i, s := range g.Skills {
		out.Skills[i] = skillFrom(s)
	}
	for i, e := range g.Edges {
		out.Edges[i] = edgeOut{From: e.From, To: e.To, Relation: e.Relation, Weight: e.Weight}
	}
	return out
}

type injuryOut struct {
	Region          string      `json:"region"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	RiskFactors     []string    `json:"risk_factors"`
	EarlySigns      []string    `json:"early_signs"`
	PrehabExercises []prehabOut `json:"prehab_exercises"`
	Disclaimer      string      `json:"disclaimer"`
}

type prehabOut struct {
	ExerciseID uuid.UUID `json:"exercise_id"`
	Slug       string    `json:"slug"`
}

// skillDetailOut is a skill with its injury notes. skillOut is embedded so
// the two stay one shape.
type skillDetailOut struct {
	skillOut
	Injuries         []injuryOut   `json:"injuries"`
	InjuryDisclaimer disclaimerOut `json:"injury_disclaimer"`
}

type levelStateOut struct {
	LevelID         uuid.UUID  `json:"level_id"`
	State           string     `json:"state"`
	BestValue       *float64   `json:"best_value"`
	BestUnit        *string    `json:"best_unit"`
	BestAt          *time.Time `json:"best_at"`
	FirstAchievedAt *time.Time `json:"first_achieved_at"`
	Verification    *string    `json:"verification"`
	StaleSince      *time.Time `json:"stale_since"`
	AttemptsCount   int        `json:"attempts_count"`
}

type skillMapOut struct {
	skillGraphOut
	States []levelStateOut `json:"states"`
}

type unlockOut struct {
	LevelID      uuid.UUID  `json:"level_id"`
	SkillSlug    string     `json:"skill_slug"`
	SkillName    string     `json:"skill_name"`
	LevelSlug    string     `json:"level_slug"`
	LevelName    string     `json:"level_name"`
	Verification string     `json:"verification"`
	OccurredAt   time.Time  `json:"occurred_at"`
	Evidence     *uuid.UUID `json:"evidence_set_entry_id"`
}

func unlockFrom(u store.Unlock) unlockOut {
	return unlockOut{
		LevelID: u.LevelID, SkillSlug: u.SkillSlug, SkillName: u.SkillName, LevelSlug: u.LevelSlug,
		LevelName: u.LevelName, Verification: u.Verification, OccurredAt: utc(u.OccurredAt), Evidence: u.Evidence,
	}
}

type xpAwardOut struct {
	Source string `json:"source"`
	Amount int    `json:"amount"`
}

type streakOut struct {
	CurrentDays     int     `json:"current_days"`
	LongestDays     int     `json:"longest_days"`
	FreezeCredits   int     `json:"freeze_credits"`
	LastCountedDate *string `json:"last_counted_date"`
}

func streakFrom(s progress.Streak) streakOut {
	out := streakOut{CurrentDays: s.Current, LongestDays: s.Longest, FreezeCredits: s.FreezeCredits}
	if s.LastCounted != nil {
		d := s.LastCounted.Format(dateLayout)
		out.LastCountedDate = &d
	}
	return out
}

type completeIn struct {
	CompletedAt      *time.Time `json:"completed_at"`
	EndedAt          *time.Time `json:"ended_at"`
	PerceivedFatigue *int       `json:"perceived_fatigue"`
	BodyweightKg     *float64   `json:"bodyweight_kg"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

type completionOut struct {
	Session          sessionOut   `json:"session"`
	AlreadyCompleted bool         `json:"already_completed"`
	Unlocked         []unlockOut  `json:"unlocked"`
	NewlyAvailable   []uuid.UUID  `json:"newly_available"`
	XPAwarded        []xpAwardOut `json:"xp_awarded"`
	XPTotal          int          `json:"xp_total"`
	Streak           streakOut    `json:"streak"`
}

type attestIn struct{}

type attestOut struct {
	Unlock         unlockOut   `json:"unlock"`
	NewlyAvailable []uuid.UUID `json:"newly_available"`
}

type progressOut struct {
	XPTotal int       `json:"xp_total"`
	Streak  streakOut `json:"streak"`
}

// nonNil makes a nil slice serialise as [] rather than null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (in completeIn) toStore() store.CompleteInput {
	return store.CompleteInput{
		CompletedAt: in.CompletedAt, EndedAt: in.EndedAt, PerceivedFatigue: in.PerceivedFatigue, BodyweightKg: in.BodyweightKg,
	}
}

func completionFrom(c store.Completion) completionOut {
	out := completionOut{
		Session: sessionFrom(c.Session), AlreadyCompleted: c.AlreadyCompleted,
		Unlocked: make([]unlockOut, len(c.Unlocked)), NewlyAvailable: nonNil(c.NewlyAvailable),
		XPAwarded: make([]xpAwardOut, len(c.XPAwarded)), XPTotal: c.XPTotal, Streak: streakFrom(c.Streak),
	}
	for i, u := range c.Unlocked {
		out.Unlocked[i] = unlockFrom(u)
	}
	for i, a := range c.XPAwarded {
		out.XPAwarded[i] = xpAwardOut{Source: a.Source, Amount: a.Amount}
	}
	return out
}
