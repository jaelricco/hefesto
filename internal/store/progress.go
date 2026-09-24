package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jaelricco/hefesto/internal/domain/progress"
	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// ErrPrerequisitesUnmet is returned when a level is attested before its
// prerequisites are unlocked.
var ErrPrerequisitesUnmet = errors.New("prerequisites are not unlocked")

// staleAfter is how long an unlocked skill can go unpractised before the map
// shows it as stale. The hint never changes the unlock.
const staleAfter = 90 * 24 * time.Hour

// ------------------------------------------------------------------- graph

// Graph is the skill graph as the API and the evaluator need it.
type Graph struct {
	ContentVersion string
	Skills         []GraphSkill             // non-retired skills, by slug
	Levels         map[uuid.UUID]GraphLevel // every level, retired skills' included
	Edges          []GraphEdge
}

// GraphSkill is a skill with its levels in order.
type GraphSkill struct {
	ID             uuid.UUID
	Slug           string
	Name           string
	Family         string
	DifficultyTier int
	IsMilestone    bool
	Status         string
	Aka            []string
	Summary        string
	PrimaryMuscles []string
	CommonFaults   []string
	Map            *MapPoint
	Levels         []GraphLevel
}

// MapPoint places a skill on the map.
type MapPoint struct {
	Constellation string
	X, Y          float64
}

// GraphLevel is one level of a skill.
type GraphLevel struct {
	ID             uuid.UUID
	SkillID        uuid.UUID
	SkillSlug      string
	SkillName      string
	SkillStatus    string
	DifficultyTier int
	Order          int
	Slug           string
	Name           string
	Description    string
	Criteria       progress.Criteria
	RawCriteria    []byte
	EstWeeks       *int
	Exercises      []LevelExercise
	Prerequisites  []uuid.UUID
}

// LevelExercise attaches an exercise to a level.
type LevelExercise struct {
	ExerciseID uuid.UUID
	Slug       string
	Role       string
}

// GraphEdge is an edge between two levels.
type GraphEdge struct {
	From, To uuid.UUID
	Relation string
	Weight   float64
}

func (g Graph) nodes() []progress.LevelNode {
	out := make([]progress.LevelNode, 0, len(g.Levels))
	for _, l := range g.Levels {
		out = append(out, progress.LevelNode{ID: l.ID, Prerequisites: l.Prerequisites})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	return out
}

// SkillGraph loads the graph.
func (s *Store) SkillGraph(ctx context.Context) (Graph, error) {
	return loadGraph(ctx, s.q)
}

func loadGraph(ctx context.Context, q *dbgen.Queries) (Graph, error) {
	g := Graph{Levels: map[uuid.UUID]GraphLevel{}}
	if v, err := q.GetLatestContentVersion(ctx); err == nil {
		g.ContentVersion = fmt.Sprintf("%x", v.Checksum)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return g, fmt.Errorf("reading content version: %w", err)
	}

	levels, err := q.ListGraphLevels(ctx)
	if err != nil {
		return g, fmt.Errorf("reading levels: %w", err)
	}
	exercises, err := q.ListGraphLevelExercises(ctx)
	if err != nil {
		return g, fmt.Errorf("reading level exercises: %w", err)
	}
	edges, err := q.ListGraphEdges(ctx)
	if err != nil {
		return g, fmt.Errorf("reading edges: %w", err)
	}
	skills, err := q.ListGraphSkills(ctx)
	if err != nil {
		return g, fmt.Errorf("reading skills: %w", err)
	}

	byLevel := map[uuid.UUID][]LevelExercise{}
	for _, e := range exercises {
		byLevel[e.SkillLevelID] = append(byLevel[e.SkillLevelID], LevelExercise{ExerciseID: e.ExerciseID, Slug: e.Slug, Role: e.Role})
	}
	prereqs := map[uuid.UUID][]uuid.UUID{}
	for _, e := range edges {
		g.Edges = append(g.Edges, GraphEdge{From: e.FromSkillLevelID, To: e.ToSkillLevelID, Relation: e.Relation, Weight: numericToFloat(e.Weight)})
		if e.Relation == "prerequisite" {
			prereqs[e.ToSkillLevelID] = append(prereqs[e.ToSkillLevelID], e.FromSkillLevelID)
		}
	}
	bySkill := map[uuid.UUID][]GraphLevel{}
	for _, l := range levels {
		var c progress.Criteria
		if err := json.Unmarshal(l.UnlockCriteria, &c); err != nil {
			return g, fmt.Errorf("level %s/%s criteria: %w", l.SkillSlug, l.Slug, err)
		}
		gl := GraphLevel{
			ID: l.ID, SkillID: l.SkillID, SkillSlug: l.SkillSlug, SkillName: l.SkillName, SkillStatus: l.SkillStatus,
			DifficultyTier: int(l.DifficultyTier), Order: int(l.OrderIndex), Slug: l.Slug, Name: l.Name,
			Description: l.Description, Criteria: c.WithDefaults(), RawCriteria: l.UnlockCriteria,
			EstWeeks: intPtr16(l.EstWeeksFromPrev), Exercises: nonNilLevelExercises(byLevel[l.ID]),
			Prerequisites: prereqs[l.ID],
		}
		g.Levels[l.ID] = gl
		bySkill[l.SkillID] = append(bySkill[l.SkillID], gl)
	}
	for _, sk := range skills {
		gs := GraphSkill{
			ID: sk.ID, Slug: sk.Slug, Name: sk.Name, Family: sk.FamilySlug, DifficultyTier: int(sk.DifficultyTier),
			IsMilestone: sk.IsMilestone, Status: sk.Status, Aka: sk.Aka, Summary: sk.Summary,
			PrimaryMuscles: sk.PrimaryMuscles, CommonFaults: sk.CommonFaults, Levels: bySkill[sk.ID],
		}
		if sk.MapConstellation != nil {
			gs.Map = &MapPoint{Constellation: *sk.MapConstellation, X: numericToFloat(sk.MapX), Y: numericToFloat(sk.MapY)}
		}
		if gs.Levels == nil {
			gs.Levels = []GraphLevel{}
		}
		g.Skills = append(g.Skills, gs)
	}
	if g.Edges == nil {
		g.Edges = []GraphEdge{}
	}
	return g, nil
}

func nonNilLevelExercises(e []LevelExercise) []LevelExercise {
	if e == nil {
		return []LevelExercise{}
	}
	return e
}

// Injury is an educational injury note on a skill.
type Injury struct {
	Region, Name, Description string
	RiskFactors, EarlySigns   []string
	Prehab                    []LevelExercise // Role unused
	Disclaimer                string
}

// SkillDetail returns one non-retired skill with its injury notes.
func (s *Store) SkillDetail(ctx context.Context, slug string) (GraphSkill, []Injury, error) {
	sk, err := s.q.GetGraphSkill(ctx, slug)
	if err != nil {
		return GraphSkill{}, nil, translate(err)
	}
	g, err := loadGraph(ctx, s.q)
	if err != nil {
		return GraphSkill{}, nil, err
	}
	var skill GraphSkill
	for _, gs := range g.Skills {
		if gs.ID == sk.ID {
			skill = gs
		}
	}
	risks, err := s.q.ListSkillInjuries(ctx, sk.ID)
	if err != nil {
		return GraphSkill{}, nil, fmt.Errorf("reading injuries: %w", err)
	}
	prehab, err := s.q.ListInjuryPrehab(ctx, sk.ID)
	if err != nil {
		return GraphSkill{}, nil, fmt.Errorf("reading prehab: %w", err)
	}
	byRisk := map[uuid.UUID][]LevelExercise{}
	for _, p := range prehab {
		byRisk[p.RiskID] = append(byRisk[p.RiskID], LevelExercise{ExerciseID: p.ExerciseID, Slug: p.Slug})
	}
	injuries := make([]Injury, len(risks))
	for i, r := range risks {
		injuries[i] = Injury{
			Region: r.Region, Name: r.Name, Description: r.Description, RiskFactors: r.RiskFactors,
			EarlySigns: r.EarlySigns, Prehab: nonNilLevelExercises(byRisk[r.ID]), Disclaimer: r.Disclaimer,
		}
	}
	return skill, injuries, nil
}

// --------------------------------------------------------------- user state

// LevelState is a user's standing on one level.
type LevelState struct {
	LevelID         uuid.UUID
	State           progress.State
	BestValue       *float64
	BestUnit        *string
	BestAt          *time.Time
	FirstAchievedAt *time.Time
	Verification    *string
	StaleSince      *time.Time
	Attempts        int
}

// storedStates reads a user's rows and turns them into the state machine's
// inputs: what is unlocked, and what the athlete has started.
func storedStates(ctx context.Context, q *dbgen.Queries, userID uuid.UUID) (map[uuid.UUID]dbgen.UserSkillState, map[uuid.UUID]progress.State, map[uuid.UUID]progress.Result, error) {
	rows, err := q.ListUserSkillStates(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading skill states: %w", err)
	}
	stored := make(map[uuid.UUID]dbgen.UserSkillState, len(rows))
	current := map[uuid.UUID]progress.State{}
	started := map[uuid.UUID]progress.Result{}
	for _, r := range rows {
		stored[r.SkillLevelID] = r
		current[r.SkillLevelID] = progress.State(r.State)
		if r.State == string(progress.InProgress) || r.AttemptsCount > 0 {
			started[r.SkillLevelID] = progress.Result{Started: true}
		}
	}
	return stored, current, started, nil
}

// SkillMap returns the graph and the user's state on every level of it.
// Availability is derived from what is unlocked now, so a level added to the
// content becomes available without waiting for the next session.
func (s *Store) SkillMap(ctx context.Context, userID uuid.UUID) (Graph, []LevelState, error) {
	g, err := loadGraph(ctx, s.q)
	if err != nil {
		return g, nil, err
	}
	stored, current, started, err := storedStates(ctx, s.q, userID)
	if err != nil {
		return g, nil, err
	}
	next, _, err := progress.Advance(g.nodes(), current, started)
	if err != nil {
		return g, nil, fmt.Errorf("deriving states: %w", err)
	}
	var out []LevelState
	for _, sk := range g.Skills {
		for _, l := range sk.Levels {
			ls := LevelState{LevelID: l.ID, State: next[l.ID]}
			if r, ok := stored[l.ID]; ok {
				ls.BestValue, ls.BestUnit, ls.BestAt = numericPtrToFloat(r.BestValue), r.BestUnit, r.BestAt
				ls.FirstAchievedAt, ls.StaleSince, ls.Attempts = r.FirstAchievedAt, r.StaleSince, int(r.AttemptsCount)
				if r.FirstAchievedAt != nil {
					v := r.Verification
					ls.Verification = &v
				}
			}
			out = append(out, ls)
		}
	}
	return g, out, nil
}

// ------------------------------------------------------------------ unlocks

// Unlock describes an unlocked level for the celebration.
type Unlock struct {
	LevelID      uuid.UUID
	SkillSlug    string
	SkillName    string
	LevelSlug    string
	LevelName    string
	Verification string
	OccurredAt   time.Time
	Evidence     *uuid.UUID
}

func unlockFrom(g Graph, ev dbgen.SkillUnlockEvent) Unlock {
	l := g.Levels[ev.SkillLevelID]
	return Unlock{
		LevelID: ev.SkillLevelID, SkillSlug: l.SkillSlug, SkillName: l.SkillName, LevelSlug: l.Slug,
		LevelName: l.Name, Verification: ev.Verification, OccurredAt: ev.OccurredAt, Evidence: ev.EvidenceSetEntryID,
	}
}

// Completion is what completing a session did.
type Completion struct {
	Session          training.Session
	AlreadyCompleted bool
	Unlocked         []Unlock
	NewlyAvailable   []uuid.UUID
	XPAwarded        []progress.Award
	XPTotal          int
	Streak           progress.Streak
}

// CompleteInput is the optional detail sent with a completion.
type CompleteInput struct {
	CompletedAt      *time.Time
	EndedAt          *time.Time
	PerceivedFatigue *int
	BodyweightKg     *float64
}

// CompleteSession completes a draft session and runs everything that hangs
// off it, in one transaction serialised per user: the unlock evaluation over
// the athlete's completed sessions, skill states and unlock events, XP, the
// training day and the streak.
func (s *Store) CompleteSession(ctx context.Context, w Writer, sessionID uuid.UUID, in CompleteInput) (Completion, error) {
	var out Completion
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if err := q.LockUserProgress(ctx, w.UserID.String()); err != nil {
			return fmt.Errorf("locking progress: %w", err)
		}
		row, err := q.GetSessionForUpdate(ctx, dbgen.GetSessionForUpdateParams{ID: sessionID, UserID: w.UserID})
		if err != nil {
			return err //nolint:wrapcheck // translated by tx
		}
		user, err := q.GetUserByID(ctx, w.UserID)
		if err != nil {
			return fmt.Errorf("reading user: %w", err)
		}
		loc, err := time.LoadLocation(user.Timezone)
		if err != nil {
			loc = time.UTC
		}
		g, err := loadGraph(ctx, q)
		if err != nil {
			return err
		}
		now := time.Now().Truncate(time.Microsecond) // what Postgres stores, so a repeat reads back the same

		switch row.Status {
		case training.StatusCompleted:
			out, err = repeatCompletion(ctx, q, g, w.UserID, row, loc, now)
			return err
		case training.StatusAbandoned:
			return training.FieldErrors{"": "an abandoned session cannot be completed; set its status back to draft first"}.Err()
		}

		sess := sessionFromRow(row)
		completedAt := now
		if in.CompletedAt != nil {
			completedAt = *in.CompletedAt
		}
		sess.CompletedAt = &completedAt
		switch {
		case in.EndedAt != nil:
			sess.EndedAt = in.EndedAt
		case sess.EndedAt == nil:
			sess.EndedAt = &completedAt
		}
		if in.PerceivedFatigue != nil {
			sess.PerceivedFatigue = in.PerceivedFatigue
		}
		if in.BodyweightKg != nil {
			sess.BodyweightKg = in.BodyweightKg
		}
		if err := training.ValidateSession(sess); err != nil {
			return err //nolint:wrapcheck // domain validation error
		}
		if sess.CompletedAt.Before(sess.StartedAt) {
			return training.FieldErrors{"/completed_at": "a session cannot be completed before it starts"}.Err()
		}
		bw, err := numericPtr(sess.BodyweightKg)
		if err != nil {
			return err
		}
		row, err = q.MarkSessionCompleted(ctx, dbgen.MarkSessionCompletedParams{
			ID: sessionID, UserID: w.UserID, CompletedAt: sess.CompletedAt, EndedAt: sess.EndedAt,
			PerceivedFatigue: int16Ptr(sess.PerceivedFatigue), BodyweightKg: bw, ClientID: w.DeviceID, UpdatedAt: w.At,
		})
		if err != nil {
			return fmt.Errorf("completing session: %w", err)
		}

		firstOfDay, err := q.CompletedSessionsOnDay(ctx, dbgen.CompletedSessionsOnDayParams{
			UserID: w.UserID, LocalDate: row.LocalDate, ExceptID: sessionID,
		})
		if err != nil {
			return fmt.Errorf("counting sessions: %w", err)
		}
		if err := q.MarkTrainingDay(ctx, dbgen.MarkTrainingDayParams{
			UserID: w.UserID, LocalDate: row.LocalDate, HadSession: !row.IsRestDay, PlannedRest: row.IsRestDay,
		}); err != nil {
			return fmt.Errorf("marking training day: %w", err)
		}

		var awards []progress.Award
		if !row.IsRestDay {
			perf, err := q.SessionPerformance(ctx, dbgen.SessionPerformanceParams{SessionID: sessionID, UserID: w.UserID})
			if err != nil {
				return fmt.Errorf("reading session performance: %w", err)
			}
			ratings := make([]int, len(perf.FormRatings))
			for i, r := range perf.FormRatings {
				ratings[i] = int(r)
			}
			awards = append(awards, progress.SessionXP(progress.CompletedSession{
				ID: sessionID, FirstOfDay: firstOfDay == 0, PerformedElements: int(perf.PerformedElements), FormRatings: ratings,
			})...)

			unlocked, newly, err := evaluateUnlocks(ctx, q, g, w.UserID, &sessionID, now)
			if err != nil {
				return err
			}
			out.Unlocked, out.NewlyAvailable = unlocked, newly
			for _, u := range unlocked {
				awards = append(awards, progress.UnlockXP(u.LevelID, g.Levels[u.LevelID].DifficultyTier, u.Verification)...)
			}
		}

		streak, err := recomputeStreak(ctx, q, w.UserID, loc, now)
		if err != nil {
			return err
		}
		out.Streak = streak
		awards = append(awards, progress.StreakXP(w.UserID, streak)...)

		out.XPAwarded, err = recordXP(ctx, q, w.UserID, awards, now)
		if err != nil {
			return err
		}
		total, err := q.TotalXP(ctx, w.UserID)
		if err != nil {
			return fmt.Errorf("reading xp: %w", err)
		}
		out.XPTotal = int(total)
		out.Session, err = loadTree(ctx, q, w.UserID, sessionFromRow(row))
		return err
	})
	return out, err
}

// repeatCompletion answers a second completion of the same session with what
// the first one did, changing nothing.
func repeatCompletion(ctx context.Context, q *dbgen.Queries, g Graph, userID uuid.UUID, row dbgen.WorkoutSession, loc *time.Location, now time.Time) (Completion, error) {
	out := Completion{AlreadyCompleted: true, Unlocked: []Unlock{}, NewlyAvailable: []uuid.UUID{}, XPAwarded: []progress.Award{}}
	events, err := q.ListUnlockEventsForSession(ctx, dbgen.ListUnlockEventsForSessionParams{UserID: userID, SessionID: &row.ID})
	if err != nil {
		return out, fmt.Errorf("reading unlock events: %w", err)
	}
	for _, ev := range events {
		out.Unlocked = append(out.Unlocked, unlockFrom(g, ev))
	}
	total, err := q.TotalXP(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("reading xp: %w", err)
	}
	out.XPTotal = int(total)
	days, err := q.CountedDays(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("reading training days: %w", err)
	}
	out.Streak = progress.ComputeStreak(civilDates(days), training.LocalDate(now, loc), progress.DefaultStreakRules)
	out.Session, err = loadTree(ctx, q, userID, sessionFromRow(row))
	return out, err
}

// evaluateUnlocks runs every level's criteria against the athlete's history,
// advances the state machine, and persists states and unlock events. It
// returns the levels it unlocked and those that became available.
func evaluateUnlocks(ctx context.Context, q *dbgen.Queries, g Graph, userID uuid.UUID, sessionID *uuid.UUID, now time.Time) ([]Unlock, []uuid.UUID, error) {
	stored, current, started, err := storedStates(ctx, q, userID)
	if err != nil {
		return nil, nil, err
	}
	before, _, err := progress.Advance(g.nodes(), current, started)
	if err != nil {
		return nil, nil, fmt.Errorf("deriving states: %w", err)
	}

	slugSet := map[string]bool{}
	for _, l := range g.Levels {
		for _, c := range l.Criteria.Conditions() {
			slugSet[c.Exercise] = true
		}
	}
	slugs := make([]string, 0, len(slugSet))
	for sl := range slugSet {
		slugs = append(slugs, sl)
	}
	history, err := loadHistory(ctx, q, userID, slugs)
	if err != nil {
		return nil, nil, err
	}

	results := make(map[uuid.UUID]progress.Result, len(g.Levels))
	for id, l := range g.Levels {
		r, err := progress.Evaluate(l.Criteria, history, now)
		if err != nil {
			return nil, nil, fmt.Errorf("level %s/%s: %w", l.SkillSlug, l.Slug, err)
		}
		if !r.Started && started[id].Started {
			r.Started = true
		}
		results[id] = r
	}
	next, unlockedIDs, err := progress.Advance(g.nodes(), current, results)
	if err != nil {
		return nil, nil, fmt.Errorf("advancing states: %w", err)
	}

	isNew := map[uuid.UUID]bool{}
	for _, id := range unlockedIDs {
		isNew[id] = true
	}
	for id, l := range g.Levels {
		r := results[id]
		prev, had := stored[id]
		if next[id] == progress.Locked && !had && (r.Primary() == nil || r.Primary().Attempts == 0) {
			continue // nothing to record
		}
		p := dbgen.UpsertUserSkillStateParams{
			UserID: userID, SkillLevelID: id, State: string(next[id]), Verification: "auto",
		}
		if had {
			p.BestValue, p.BestUnit, p.BestAt, p.StaleSince, p.AttemptsCount = prev.BestValue, prev.BestUnit, prev.BestAt, prev.StaleSince, prev.AttemptsCount
			// Postgres checks the proposed row before ON CONFLICT applies, so
			// an unlocked row must be sent with its achievement.
			p.FirstAchievedAt, p.EvidenceSetEntryID = prev.FirstAchievedAt, prev.EvidenceSetEntryID
			if prev.FirstAchievedAt != nil {
				p.Verification = prev.Verification
			}
		}
		if pc := r.Primary(); pc != nil {
			p.AttemptsCount = int32(pc.Attempts) //nolint:gosec // bounded by the athlete's history
			if pc.Best != nil && !sameBest(prev, had, *pc.Best, pc.Condition.Measure) {
				bv, err := numeric(*pc.Best)
				if err != nil {
					return nil, nil, err
				}
				unit := pc.Condition.Measure
				p.BestValue, p.BestUnit, p.BestAt = bv, &unit, &now
			}
			p.StaleSince = nil
			if next[id] == progress.Unlocked && pc.LastAttempt != nil && now.Sub(*pc.LastAttempt) > staleAfter {
				st := pc.LastAttempt.Add(staleAfter)
				p.StaleSince = &st
			}
		}
		if isNew[id] {
			p.FirstAchievedAt, p.EvidenceSetEntryID = &now, r.Evidence
			if err := q.InsertUnlockEvent(ctx, dbgen.InsertUnlockEventParams{
				ID: uuid.Must(uuid.NewV7()), UserID: userID, SkillLevelID: id, OccurredAt: now, SessionID: sessionID,
				EvidenceSetEntryID: r.Evidence, Verification: "auto", CriteriaSnapshot: l.RawCriteria,
			}); err != nil {
				return nil, nil, fmt.Errorf("recording unlock: %w", err)
			}
		}
		if err := q.UpsertUserSkillState(ctx, p); err != nil {
			return nil, nil, fmt.Errorf("saving skill state: %w", err)
		}
	}

	unlocked := make([]Unlock, 0, len(unlockedIDs))
	for _, id := range unlockedIDs {
		l := g.Levels[id]
		unlocked = append(unlocked, Unlock{
			LevelID: id, SkillSlug: l.SkillSlug, SkillName: l.SkillName, LevelSlug: l.Slug, LevelName: l.Name,
			Verification: "auto", OccurredAt: now, Evidence: results[id].Evidence,
		})
	}
	return unlocked, newlyAvailable(g, before, next), nil
}

// sameBest reports whether the stored best already equals value in unit, in
// which case best_at keeps the moment it was first reached.
func sameBest(prev dbgen.UserSkillState, had bool, value float64, unit string) bool {
	old := numericPtrToFloat(prev.BestValue)
	return had && old != nil && *old == value && prev.BestUnit != nil && *prev.BestUnit == unit
}

// newlyAvailable lists levels that were locked and no longer are, other than
// by being unlocked outright.
func newlyAvailable(g Graph, before, after map[uuid.UUID]progress.State) []uuid.UUID {
	out := []uuid.UUID{}
	for _, sk := range g.Skills {
		for _, l := range sk.Levels {
			if before[l.ID] == progress.Locked && (after[l.ID] == progress.Available || after[l.ID] == progress.InProgress) {
				out = append(out, l.ID)
			}
		}
	}
	return out
}

func loadHistory(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, slugs []string) (progress.SetHistory, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	rows, err := q.ObservationsForExercises(ctx, dbgen.ObservationsForExercisesParams{UserID: userID, Slugs: slugs})
	if err != nil {
		return nil, fmt.Errorf("reading history: %w", err)
	}
	h := make(progress.SetHistory, len(rows))
	for i, r := range rows {
		var v *float64
		switch r.Measure {
		case "reps":
			if r.Reps != nil {
				f := float64(*r.Reps)
				v = &f
			}
		case "hold_seconds":
			v = numericPtrToFloat(r.HoldSeconds)
		case "distance_m":
			v = numericPtrToFloat(r.DistanceM)
		}
		h[i] = progress.Observation{
			SetEntryID: r.SetEntryID, SessionID: r.SessionID, Exercise: r.Exercise, Measure: r.Measure, Value: v,
			LoadKg: numericToFloat(r.LoadKg), Assistance: r.AssistanceClass, FormQuality: intPtr16(r.FormQuality),
			Failed: r.Failed, PartialROM: r.IsPartialRom, EccentricOnly: r.IsEccentricOnly, PerformedAt: r.PerformedAt,
		}
	}
	return h, nil
}

// recomputeStreak walks the athlete's counted days as of their local today,
// rewrites which days a freeze bridged, and stores the result.
func recomputeStreak(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, loc *time.Location, now time.Time) (progress.Streak, error) {
	days, err := q.CountedDays(ctx, userID)
	if err != nil {
		return progress.Streak{}, fmt.Errorf("reading training days: %w", err)
	}
	st := progress.ComputeStreak(civilDates(days), training.LocalDate(now, loc), progress.DefaultStreakRules)

	if err := q.ClearFreezes(ctx, userID); err != nil {
		return st, fmt.Errorf("clearing freezes: %w", err)
	}
	if err := q.DeleteEmptyFreezeDays(ctx, userID); err != nil {
		return st, fmt.Errorf("clearing freeze days: %w", err)
	}
	if len(st.Frozen) > 0 {
		frozen := make([]time.Time, len(st.Frozen))
		copy(frozen, st.Frozen)
		pd := make([]pgDate, len(frozen))
		for i, d := range frozen {
			pd[i] = dateOf(d)
		}
		if err := q.MarkFreezeDays(ctx, dbgen.MarkFreezeDaysParams{UserID: userID, Days: pd}); err != nil {
			return st, fmt.Errorf("marking freezes: %w", err)
		}
	}
	var last pgDate
	if st.LastCounted != nil {
		last = dateOf(*st.LastCounted)
	}
	if err := q.UpsertStreak(ctx, dbgen.UpsertStreakParams{
		UserID: userID, CurrentDays: int32(st.Current), LongestDays: int32(st.Longest), //nolint:gosec // day counts
		LastCountedDate: last, FreezeCredits: int16(st.FreezeCredits), //nolint:gosec // capped by the rules
	}); err != nil {
		return st, fmt.Errorf("saving streak: %w", err)
	}
	return st, nil
}

// recordXP inserts awards, returning the ones that were new. An award that
// already exists (the same session, level or milestone) is skipped by the
// unique constraint, so repeating a completion never pays twice.
func recordXP(ctx context.Context, q *dbgen.Queries, userID uuid.UUID, awards []progress.Award, now time.Time) ([]progress.Award, error) {
	out := []progress.Award{}
	for _, a := range awards {
		refType, refID := a.RefType, a.RefID
		_, err := q.InsertXPEvent(ctx, dbgen.InsertXPEventParams{
			ID: uuid.Must(uuid.NewV7()), UserID: userID, Source: a.Source, Amount: int32(a.Amount), //nolint:gosec // small constants
			RefType: &refType, RefID: &refID, OccurredAt: now,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("recording xp: %w", err)
		}
		out = append(out, a)
	}
	return out, nil
}

// Progress is a user's XP and streak.
type Progress struct {
	XPTotal int
	Streak  progress.Streak
}

// UserProgress computes XP and the streak as of now. The streak is derived
// from the training days on every read, so it is right even on a day the
// athlete has not opened the app.
func (s *Store) UserProgress(ctx context.Context, userID uuid.UUID) (Progress, error) {
	user, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return Progress{}, translate(err)
	}
	loc, err := time.LoadLocation(user.Timezone)
	if err != nil {
		loc = time.UTC
	}
	total, err := s.q.TotalXP(ctx, userID)
	if err != nil {
		return Progress{}, fmt.Errorf("reading xp: %w", err)
	}
	days, err := s.q.CountedDays(ctx, userID)
	if err != nil {
		return Progress{}, fmt.Errorf("reading training days: %w", err)
	}
	return Progress{
		XPTotal: int(total),
		Streak:  progress.ComputeStreak(civilDates(days), training.LocalDate(time.Now(), loc), progress.DefaultStreakRules),
	}, nil
}

// AttestLevel records a self-attested unlock. The level's prerequisites must
// be unlocked; attesting an unlocked level returns the existing unlock.
func (s *Store) AttestLevel(ctx context.Context, userID, levelID uuid.UUID) (Unlock, []uuid.UUID, error) {
	var (
		out   Unlock
		newly []uuid.UUID
	)
	err := s.tx(ctx, func(q *dbgen.Queries) error {
		if err := q.LockUserProgress(ctx, userID.String()); err != nil {
			return fmt.Errorf("locking progress: %w", err)
		}
		g, err := loadGraph(ctx, q)
		if err != nil {
			return err
		}
		l, ok := g.Levels[levelID]
		if !ok || l.SkillStatus == "retired" {
			return ErrNotFound
		}
		stored, current, started, err := storedStates(ctx, q, userID)
		if err != nil {
			return err
		}
		before, _, err := progress.Advance(g.nodes(), current, started)
		if err != nil {
			return fmt.Errorf("deriving states: %w", err)
		}
		switch before[levelID] {
		case progress.Unlocked:
			ev, err := q.GetUnlockEvent(ctx, dbgen.GetUnlockEventParams{UserID: userID, SkillLevelID: levelID})
			if err != nil {
				return fmt.Errorf("reading unlock: %w", err)
			}
			out, newly = unlockFrom(g, ev), []uuid.UUID{}
			return nil
		case progress.Locked:
			return ErrPrerequisitesUnmet
		}

		now := time.Now().Truncate(time.Microsecond) // what Postgres stores, so a repeat reads back the same
		if err := q.InsertUnlockEvent(ctx, dbgen.InsertUnlockEventParams{
			ID: uuid.Must(uuid.NewV7()), UserID: userID, SkillLevelID: levelID, OccurredAt: now,
			Verification: "self_attested", CriteriaSnapshot: l.RawCriteria,
		}); err != nil {
			return fmt.Errorf("recording unlock: %w", err)
		}
		p := dbgen.UpsertUserSkillStateParams{
			UserID: userID, SkillLevelID: levelID, State: string(progress.Unlocked),
			FirstAchievedAt: &now, Verification: "self_attested",
		}
		if prev, ok := stored[levelID]; ok {
			p.BestValue, p.BestUnit, p.BestAt, p.AttemptsCount = prev.BestValue, prev.BestUnit, prev.BestAt, prev.AttemptsCount
		}
		if err := q.UpsertUserSkillState(ctx, p); err != nil {
			return fmt.Errorf("saving skill state: %w", err)
		}
		current[levelID] = progress.Unlocked
		after, _, err := progress.Advance(g.nodes(), current, started)
		if err != nil {
			return fmt.Errorf("deriving states: %w", err)
		}
		// A next level with logged evidence unlocks at the next session
		// completion; availability itself is derived when the map is read.
		newly = newlyAvailable(g, before, after)
		out = Unlock{
			LevelID: levelID, SkillSlug: l.SkillSlug, SkillName: l.SkillName, LevelSlug: l.Slug, LevelName: l.Name,
			Verification: "self_attested", OccurredAt: now,
		}
		return nil
	})
	return out, newly, err
}

func civilDates(ds []pgDate) []time.Time {
	out := make([]time.Time, 0, len(ds))
	for _, d := range ds {
		if d.Valid {
			out = append(out, d.Time)
		}
	}
	return out
}
