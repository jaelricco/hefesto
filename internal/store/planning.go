package store

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/planning"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// Planner stores the training planner's state, plans and decisions in the
// tables of migration 00009 (ADR 0013). It implements planning.Transactor:
// every call of the planning service runs in one transaction that holds the
// user's planner lock.
type Planner struct {
	pool *pgxpool.Pool
}

// NewPlanner wraps a pool.
func NewPlanner(pool *pgxpool.Pool) *Planner { return &Planner{pool: pool} }

// ErrPlannerUser is a store call for another user than the transaction's.
var ErrPlannerUser = errors.New("planner store is bound to another user")

// InTx runs fn in a transaction holding the planner lock of userID. The
// stores fn receives are bound to that user and transaction.
func (p *Planner) InTx(ctx context.Context, userID uuid.UUID, fn func(planning.Stores) error) error {
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
			return fmt.Errorf("deferring constraints: %w", err)
		}
		q := dbgen.New(tx)
		if err := q.LockPlanner(ctx, userID); err != nil {
			return fmt.Errorf("locking the planner: %w", err)
		}
		t := &plannerTx{q: q, user: userID}
		return fn(planning.Stores{Snapshots: t, Plans: t, Decisions: t, Sessions: t})
	})
	if err != nil {
		return fmt.Errorf("planner transaction: %w", err)
	}
	return nil
}

// plannerTx is the store of one transaction. It remembers the rows it read
// or wrote, so a save writes only what changed.
type plannerTx struct {
	q    *dbgen.Queries
	user uuid.UUID
	prev *plannerRows
}

var (
	_ planning.SnapshotStore = (*plannerTx)(nil)
	_ planning.PlanStore     = (*plannerTx)(nil)
	_ planning.DecisionLog   = (*plannerTx)(nil)
	_ planning.SessionLog    = (*plannerTx)(nil)
)

func (t *plannerTx) check(userID uuid.UUID) error {
	if userID != t.user {
		return fmt.Errorf("%w: %s, not %s", ErrPlannerUser, t.user, userID)
	}
	return nil
}

// ---------------------------------------------------------------- snapshot

// plannerRows is a snapshot in the form of the table rows. Keyed parts are
// maps by primary key; the history and the pain reports are ID sets,
// because they only grow.
type plannerRows struct {
	profile     *dbgen.UpsertTrainingProfileParams
	goals       []dbgen.UserGoal // active goals as stored, with their IDs
	screening   *dbgen.UpsertScreeningParams
	constraints []dbgen.PlanningConstraint // active constraints as stored
	regions     map[string]dbgen.UpsertRegionStatusParams
	capacities  map[string]dbgen.UpsertCapacityEstimateParams
	ladders     map[string]dbgen.UpsertLadderStateParams
	state       *dbgen.UpsertPlannerStateParams
	brk         *dbgen.UpsertTrainingBreakParams
	pain        map[uuid.UUID]bool
	sessions    map[uuid.UUID]bool
}

// Snapshot reads the planner state of a user; ok is false before the
// onboarding.
func (t *plannerTx) Snapshot(ctx context.Context, userID uuid.UUID) (domain.Snapshot, bool, error) {
	if err := t.check(userID); err != nil {
		return domain.Snapshot{}, false, err
	}
	s, rows, ok, err := t.read(ctx)
	if err != nil || !ok {
		return domain.Snapshot{}, ok, err
	}
	t.prev = rows
	return s, true, nil
}

// read loads the snapshot and its rows.
func (t *plannerTx) read(ctx context.Context) (domain.Snapshot, *plannerRows, bool, error) {
	q, user := t.q, t.user
	prof, err := q.GetTrainingProfile(ctx, user)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Snapshot{}, emptyRows(), false, nil
	}
	if err != nil {
		return domain.Snapshot{}, nil, false, fmt.Errorf("reading training profile: %w", err)
	}
	profile, err := profileFromRow(prof)
	if err != nil {
		return domain.Snapshot{}, nil, false, err
	}
	s := domain.Snapshot{Profile: profile}

	goals, err := q.ListActiveGoals(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading goals: %w", err)
	}
	for _, g := range goals {
		s.Goals = append(s.Goals, domain.Goal{Skill: g.Skill, TargetLevel: g.TargetLevel,
			Priority: int(deref(g.Priority)), TargetDate: utcPtr(g.TargetDate)})
	}

	switch sc, err := q.GetScreening(ctx, user); {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return s, nil, false, fmt.Errorf("reading screening: %w", err)
	default:
		s.Screening = domain.Screening{ExertionSymptoms: sc.ExertionSymptoms, AnyYes: sc.AnyYes, Cleared: sc.Cleared}
	}

	constraints, err := q.ListActiveConstraints(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading constraints: %w", err)
	}
	for _, c := range constraints {
		s.Constraints = append(s.Constraints, domain.Constraint{Kind: c.Kind, Region: deref(c.Region), Created: c.CreatedAt.UTC()})
	}

	regions, err := q.ListRegionStatus(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading regions: %w", err)
	}
	s.Regions = make(map[string]domain.RegionState, len(regions))
	for _, r := range regions {
		rs, err := regionFromRow(r)
		if err != nil {
			return s, nil, false, err
		}
		s.Regions[r.Region] = rs
	}

	pain, err := q.ListPainReports(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading pain reports: %w", err)
	}
	for _, r := range pain {
		s.Pain = append(s.Pain, domain.PainReport{ID: r.ID.String(), Region: r.Region, Timepoint: r.Timepoint, NRS: r.Nrs,
			At: r.ReportedAt.UTC(), LastedOver: r.LastedOver1h, Persisted: r.PersistedOver15min, SuddenSharp: r.SuddenSharp,
			SessionID: uuidString(r.SessionID)})
	}

	caps, err := q.ListCapacityEstimates(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading capacities: %w", err)
	}
	s.Capacities = make(map[string]domain.Estimate, len(caps))
	for _, c := range caps {
		s.Capacities[domain.CapKey(c.Exercise, c.Assistance)] = domain.Estimate{Mu: c.Mu, Sigma: c.Sigma, Origin: c.Origin,
			At: utc(c.ObservedAt), Seen: utc(c.SeenAt), N: int(c.NObs), Pending: c.Pending}
	}

	ladders, err := q.ListLadderStates(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading ladders: %w", err)
	}
	s.Ladders = make(map[string]domain.LadderState, len(ladders))
	for _, l := range ladders {
		s.Ladders[l.Skill] = domain.LadderState{Rung: deref(l.Rung), Status: deref(l.Status), Since: utc(l.Since),
			Exposures: int(l.Exposures), Claimed: deref(l.Claimed), CapRung: deref(l.CapRung), ProbeOffer: l.ProbeOffer,
			RepTarget: l.RepTarget, LoadKg: l.LoadKg, EccS: l.EccS, LastUp: utc(l.LastUp), CapTo: deref(l.CapTo),
			CapUntil: utc(l.CapUntil)}
	}

	switch st, err := q.GetPlannerState(ctx, user); {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return s, nil, false, fmt.Errorf("reading planner state: %w", err)
	default:
		s.Phase = domain.Phase{MesoStart: utc(st.MesoStart), LastDeload: utc(st.LastDeload), DeloadWeek: utc(st.DeloadWeek),
			DeloadKind: deref(st.DeloadKind), DeloadNext: deref(st.DeloadNext), Calibrate: nilIfEmpty(st.Calibrate)}
		if err := decodeAll(map[string]any{"headroom": &s.Headroom, "entry": &s.Entry, "unlocked": &s.Unlocked},
			map[string][]byte{"headroom": st.Headroom, "entry": st.Entry, "unlocked": st.Unlocked}); err != nil {
			return s, nil, false, err
		}
	}

	switch b, err := q.GetTrainingBreak(ctx, user); {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return s, nil, false, fmt.Errorf("reading break: %w", err)
	default:
		bs := &domain.BreakState{Days: b.Days, StraightDays: b.StraightDays, Since: utc(b.Since), Step: int(b.Step),
			StepSince: utc(b.StepSince), StepSessions: int(b.StepSessions), Logged: b.Logged}
		if err := decodeAll(map[string]any{"reference": &bs.Reference, "base": &bs.Base},
			map[string][]byte{"reference": b.Reference, "base": b.Base}); err != nil {
			return s, nil, false, err
		}
		bs.Reference, bs.Base = nilIfEmptyMap(bs.Reference), nilIfEmptyMap(bs.Base)
		s.Break = bs
	}

	sessions, err := q.ListPlannerSessions(ctx, user)
	if err != nil {
		return s, nil, false, fmt.Errorf("reading planner sessions: %w", err)
	}
	for _, h := range sessions {
		ls := domain.LoggedSession{ID: h.ID.String(), Date: h.LocalDate.UTC(), Deload: h.Deload, Fatigue: h.Fatigue}
		if err := json.Unmarshal(h.Sets, &ls.Sets); err != nil {
			return s, nil, false, fmt.Errorf("decoding sets of session %s: %w", h.ID, err)
		}
		s.History = append(s.History, ls)
	}

	rows, err := rowsOf(user, s)
	if err != nil {
		return s, nil, false, err
	}
	rows.goals, rows.constraints = goals, constraints
	return s, rows, true, nil
}

// SaveSnapshot writes a snapshot. Rows that did not change since the read
// are not written. The history only gains rows; so do the pain reports,
// until a withdrawn consent drops them.
func (t *plannerTx) SaveSnapshot(ctx context.Context, userID uuid.UUID, s domain.Snapshot) error {
	if err := t.check(userID); err != nil {
		return err
	}
	if t.prev == nil {
		_, rows, _, err := t.read(ctx)
		if err != nil {
			return err
		}
		t.prev = rows
	}
	next, err := rowsOf(t.user, s)
	if err != nil {
		return err
	}
	prev, q, user := t.prev, t.q, t.user

	if !reflect.DeepEqual(prev.profile, next.profile) {
		if err := q.UpsertTrainingProfile(ctx, *next.profile); err != nil {
			return fmt.Errorf("writing training profile: %w", err)
		}
	}
	goals, err := t.saveGoals(ctx, prev.goals, s.Goals)
	if err != nil {
		return err
	}
	next.goals = goals
	switch {
	case reflect.DeepEqual(prev.screening, next.screening):
	case next.screening == nil:
		if err := q.DeleteScreening(ctx, user); err != nil {
			return fmt.Errorf("deleting screening: %w", err)
		}
	default:
		if err := q.UpsertScreening(ctx, *next.screening); err != nil {
			return fmt.Errorf("writing screening: %w", err)
		}
	}
	constraints, err := t.saveConstraints(ctx, prev.constraints, s.Constraints)
	if err != nil {
		return err
	}
	next.constraints = constraints

	if err := saveKeyed(ctx, prev.regions, next.regions, q.UpsertRegionStatus, func(ctx context.Context, region string) error {
		return q.DeleteRegionStatus(ctx, dbgen.DeleteRegionStatusParams{UserID: user, Region: region})
	}); err != nil {
		return fmt.Errorf("writing regions: %w", err)
	}
	if err := saveKeyed(ctx, prev.capacities, next.capacities, q.UpsertCapacityEstimate, func(ctx context.Context, key string) error {
		ex, assist, _ := strings.Cut(key, "|")
		return q.DeleteCapacityEstimate(ctx, dbgen.DeleteCapacityEstimateParams{UserID: user, Exercise: ex, Assistance: assist})
	}); err != nil {
		return fmt.Errorf("writing capacities: %w", err)
	}
	if err := saveKeyed(ctx, prev.ladders, next.ladders, q.UpsertLadderState, func(ctx context.Context, skill string) error {
		return q.DeleteLadderState(ctx, dbgen.DeleteLadderStateParams{UserID: user, Skill: skill})
	}); err != nil {
		return fmt.Errorf("writing ladders: %w", err)
	}
	if !reflect.DeepEqual(prev.state, next.state) {
		if err := q.UpsertPlannerState(ctx, *next.state); err != nil {
			return fmt.Errorf("writing planner state: %w", err)
		}
	}
	switch {
	case reflect.DeepEqual(prev.brk, next.brk):
	case next.brk == nil:
		if err := q.DeleteTrainingBreak(ctx, user); err != nil {
			return fmt.Errorf("deleting break: %w", err)
		}
	default:
		if err := q.UpsertTrainingBreak(ctx, *next.brk); err != nil {
			return fmt.Errorf("writing break: %w", err)
		}
	}
	if err := t.savePain(ctx, prev.pain, s.Pain); err != nil {
		return err
	}
	if err := t.saveSessions(ctx, prev.sessions, s.History); err != nil {
		return err
	}
	t.prev = next
	return nil
}

// saveKeyed upserts the rows that are new or changed and deletes the rows
// that are gone.
func saveKeyed[P any](ctx context.Context, prev, next map[string]P, upsert func(context.Context, P) error,
	del func(context.Context, string) error) error {
	for _, k := range slices.Sorted(maps.Keys(next)) {
		if old, ok := prev[k]; ok && reflect.DeepEqual(old, next[k]) {
			continue
		}
		if err := upsert(ctx, next[k]); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(prev)) {
		if _, ok := next[k]; !ok {
			if err := del(ctx, k); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
	}
	return nil
}

// saveGoals keeps a stored goal whose skill and level stay a goal, drops
// the others and inserts new ones. Priorities may swap: the unique
// constraint is checked at commit.
func (t *plannerTx) saveGoals(ctx context.Context, prev []dbgen.UserGoal, next []domain.Goal) ([]dbgen.UserGoal, error) {
	q, user := t.q, t.user
	var out []dbgen.UserGoal
	used := map[uuid.UUID]bool{}
	for _, g := range next {
		pri := int16(g.Priority) //nolint:gosec // 1–3, checked by the database
		date := utcPtr(g.TargetDate)
		i := slices.IndexFunc(prev, func(p dbgen.UserGoal) bool {
			return !used[p.ID] && p.Skill == g.Skill && p.TargetLevel == g.TargetLevel
		})
		if i < 0 {
			row := dbgen.UserGoal{ID: uuid.Must(uuid.NewV7()), UserID: user, Skill: g.Skill, TargetLevel: g.TargetLevel,
				Priority: &pri, TargetDate: date, Status: "active"}
			if err := q.InsertGoal(ctx, dbgen.InsertGoalParams{ID: row.ID, UserID: user, Skill: g.Skill,
				TargetLevel: g.TargetLevel, Priority: &pri, TargetDate: date}); err != nil {
				return nil, fmt.Errorf("inserting goal %s: %w", g.Skill, err)
			}
			out = append(out, row)
			continue
		}
		row := prev[i]
		used[row.ID] = true
		if deref(row.Priority) != pri || !reflect.DeepEqual(row.TargetDate, date) {
			if err := q.UpdateGoal(ctx, dbgen.UpdateGoalParams{ID: row.ID, UserID: user, Priority: &pri, TargetDate: date}); err != nil {
				return nil, fmt.Errorf("updating goal %s: %w", g.Skill, err)
			}
			row.Priority, row.TargetDate = &pri, date
		}
		out = append(out, row)
	}
	for _, p := range prev {
		if !used[p.ID] {
			if err := q.DropGoal(ctx, dbgen.DropGoalParams{ID: p.ID, UserID: user}); err != nil {
				return nil, fmt.Errorf("dropping goal %s: %w", p.Skill, err)
			}
		}
	}
	return out, nil
}

// saveConstraints keeps the stored constraints when the list is unchanged;
// otherwise it lifts them and stores the new list, which keeps its order.
func (t *plannerTx) saveConstraints(ctx context.Context, prev []dbgen.PlanningConstraint, next []domain.Constraint) ([]dbgen.PlanningConstraint, error) {
	same := len(prev) == len(next)
	for i := 0; same && i < len(next); i++ {
		same = prev[i].Kind == next[i].Kind && deref(prev[i].Region) == next[i].Region && prev[i].CreatedAt.Equal(next[i].Created)
	}
	if same {
		return prev, nil
	}
	q, user := t.q, t.user
	for _, p := range prev {
		if err := q.ClearConstraint(ctx, dbgen.ClearConstraintParams{ID: p.ID, UserID: user}); err != nil {
			return nil, fmt.Errorf("lifting constraint %s: %w", p.Kind, err)
		}
	}
	pos, err := q.NextConstraintPosition(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("numbering constraints: %w", err)
	}
	var out []dbgen.PlanningConstraint
	for _, c := range next {
		row := dbgen.PlanningConstraint{ID: uuid.Must(uuid.NewV7()), UserID: user, Kind: c.Kind, Region: strPtr(c.Region),
			CreatedAt: c.Created.UTC(), Position: pos}
		if err := q.InsertConstraint(ctx, dbgen.InsertConstraintParams{ID: row.ID, UserID: user, Kind: row.Kind,
			Region: row.Region, CreatedAt: row.CreatedAt, Position: pos}); err != nil {
			return nil, fmt.Errorf("storing constraint %s: %w", c.Kind, err)
		}
		out = append(out, row)
		pos++
	}
	return out, nil
}

// savePain stores the reports the table does not have yet, in the order of
// the snapshot.
func (t *plannerTx) savePain(ctx context.Context, have map[uuid.UUID]bool, reports []domain.PainReport) error {
	q, user := t.q, t.user
	if err := t.deleteWithdrawnPain(ctx, have, reports); err != nil {
		return err
	}
	var pos int32 = -1
	for _, r := range reports {
		id, err := uuid.Parse(r.ID)
		if err != nil {
			return fmt.Errorf("pain report %q: the ID must be a UUID: %w", r.ID, err)
		}
		if have[id] {
			continue
		}
		if pos < 0 {
			if pos, err = q.NextPainPosition(ctx, user); err != nil {
				return fmt.Errorf("numbering pain reports: %w", err)
			}
		}
		session, err := parseOptionalUUID(r.SessionID)
		if err != nil {
			return fmt.Errorf("pain report %s: session: %w", r.ID, err)
		}
		n, err := q.InsertPainReport(ctx, dbgen.InsertPainReportParams{ID: id, UserID: user, Region: r.Region,
			Timepoint: r.Timepoint, Nrs: r.NRS, LastedOver1h: r.LastedOver, PersistedOver15min: r.Persisted,
			SuddenSharp: r.SuddenSharp, SessionID: session, ReportedAt: r.At.UTC(), Position: pos})
		if err != nil {
			return fmt.Errorf("storing pain report %s: %w", r.ID, err)
		}
		if n == 0 {
			// The user's own reports are in have; this ID is another user's.
			return fmt.Errorf("pain report %s: %w", r.ID, ErrAlreadyExists)
		}
		pos++
		have[id] = true
	}
	return nil
}

// deleteWithdrawnPain deletes the stored reports the snapshot no longer has.
// Pain reports only grow, except when the consent is withdrawn and the core
// drops them all (spec §4.9).
func (t *plannerTx) deleteWithdrawnPain(ctx context.Context, have map[uuid.UUID]bool, reports []domain.PainReport) error {
	keep := make([]uuid.UUID, 0, len(reports))
	kept := map[uuid.UUID]bool{}
	for _, r := range reports {
		if id, err := uuid.Parse(r.ID); err == nil {
			keep, kept[id] = append(keep, id), true
		}
	}
	gone := false
	for id := range have {
		if !kept[id] {
			gone = true
			delete(have, id)
		}
	}
	if !gone {
		return nil
	}
	if err := t.q.DeletePainReportsExcept(ctx, dbgen.DeletePainReportsExceptParams{UserID: t.user, Keep: keep}); err != nil {
		return fmt.Errorf("deleting pain reports: %w", err)
	}
	return nil
}

// saveSessions stores the sessions the history does not have yet.
func (t *plannerTx) saveSessions(ctx context.Context, have map[uuid.UUID]bool, history []domain.LoggedSession) error {
	q, user := t.q, t.user
	var pos int32 = -1
	for _, h := range history {
		id, err := uuid.Parse(h.ID)
		if err != nil {
			return fmt.Errorf("session %q: the ID must be the workout session's UUID: %w", h.ID, err)
		}
		if have[id] {
			continue
		}
		if pos < 0 {
			if pos, err = q.NextPlannerSessionPosition(ctx, user); err != nil {
				return fmt.Errorf("numbering sessions: %w", err)
			}
		}
		sets := h.Sets
		if sets == nil {
			sets = []domain.LoggedSet{}
		}
		raw, err := json.Marshal(sets)
		if err != nil {
			return fmt.Errorf("encoding sets of session %s: %w", h.ID, err)
		}
		if err := q.InsertPlannerSession(ctx, dbgen.InsertPlannerSessionParams{ID: id, UserID: user,
			LocalDate: h.Date.UTC(), Deload: h.Deload, Fatigue: h.Fatigue, Sets: raw, Position: pos}); err != nil {
			return fmt.Errorf("storing session %s: %w", h.ID, err)
		}
		pos++
		have[id] = true
	}
	return nil
}

// rowsOf converts a snapshot into table rows. It allocates everything it
// returns, so later changes to the snapshot do not reach the rows.
func rowsOf(user uuid.UUID, s domain.Snapshot) (*plannerRows, error) {
	p := s.Profile
	mobility, err := encodeMap(nilIfEmptyMap(p.Mobility))
	if err != nil {
		return nil, err
	}
	days := make([]int16, len(p.PreferredDays))
	for i, d := range p.PreferredDays {
		days[i] = int16((int(d)+6)%7 + 1) //nolint:gosec // 1–7
	}
	r := emptyRows()
	r.profile = &dbgen.UpsertTrainingProfileParams{UserID: user, BirthYear: int16(p.BirthYear), //nolint:gosec // checked by the database
		SessionsPerWeek: int16(p.SessionsPerWeek), SessionMinutes: int16(p.SessionMinutes), //nolint:gosec // checked by the database
		Equipment: slices.Clone(p.Equipment), BodyweightKg: p.BodyweightKg, TrainingLevel: p.TrainingLevel,
		TrainingMonths: p.TrainingMonths, LastRegularTraining: strPtr(p.LastRegular), HealthDataConsent: p.HealthConsent,
		DisclaimerAck: p.DisclaimerAck, OnboardedAt: p.OnboardedAt.UTC(), PreferredDays: days, Mobility: mobility,
		MaxAddedLoadKg: p.MaxAddedLoadKg, SmallestPlateKg: p.SmallestPlateKg}
	if r.profile.Equipment == nil {
		r.profile.Equipment = []string{}
	}
	if s.Screening != (domain.Screening{}) || p.HealthConsent {
		r.screening = &dbgen.UpsertScreeningParams{UserID: user, ExertionSymptoms: s.Screening.ExertionSymptoms,
			AnyYes: s.Screening.AnyYes, Cleared: s.Screening.Cleared}
	}
	for id, rs := range s.Regions {
		ref, err := encodeMap(nilIfEmptyMap(rs.Reference))
		if err != nil {
			return nil, err
		}
		breaches := make([]time.Time, len(rs.Breaches))
		for i, b := range rs.Breaches {
			breaches[i] = b.UTC()
		}
		restrictions := slices.Clone(rs.Restrictions)
		if restrictions == nil {
			restrictions = []string{}
		}
		r.regions[id] = dbgen.UpsertRegionStatusParams{UserID: user, Region: id, State: rs.State, EnteredVia: strPtr(rs.EnteredVia),
			Since: timePtr(rs.Since), StartFraction: rs.StartFraction, Step: int16(rs.Step), StepSince: timePtr(rs.StepSince), //nolint:gosec // 0–3, checked by the database
			StepSessions: int32(rs.StepSessions), Reference: ref, PriorInjury: rs.PriorInjury, Complaint: rs.Complaint, //nolint:gosec // small counter
			ComplaintAt: timePtr(rs.ComplaintAt), Restrictions: restrictions, Breaches: breaches,
			PainDeloadTo: timePtr(rs.PainDeloadTo), RestUntil: timePtr(rs.RestUntil), HoldAtRef: rs.HoldAtRef,
			Referral: strPtr(rs.Referral)}
	}
	for key, e := range s.Capacities {
		ex, assist, ok := strings.Cut(key, "|")
		if !ok {
			return nil, fmt.Errorf("capacity key %q has no assistance part", key)
		}
		var pending *float64
		if e.Pending != nil {
			v := *e.Pending
			pending = &v
		}
		r.capacities[key] = dbgen.UpsertCapacityEstimateParams{UserID: user, Exercise: ex, Assistance: assist, Mu: e.Mu,
			Sigma: e.Sigma, Origin: e.Origin, ObservedAt: timePtr(e.At), SeenAt: timePtr(e.Seen), NObs: int32(e.N), Pending: pending} //nolint:gosec // observation count
	}
	for skill, l := range s.Ladders {
		r.ladders[skill] = dbgen.UpsertLadderStateParams{UserID: user, Skill: skill, Rung: strPtr(l.Rung), Status: strPtr(l.Status),
			Since: timePtr(l.Since), Exposures: int32(l.Exposures), Claimed: strPtr(l.Claimed), CapRung: strPtr(l.CapRung), //nolint:gosec // exposure count
			ProbeOffer: l.ProbeOffer, RepTarget: l.RepTarget, LoadKg: l.LoadKg, EccS: l.EccS, LastUp: timePtr(l.LastUp),
			CapTo: strPtr(l.CapTo), CapUntil: timePtr(l.CapUntil)}
	}
	ph := s.Phase
	calibrate := slices.Clone(ph.Calibrate)
	if calibrate == nil {
		calibrate = []string{}
	}
	headroom, err := encodeMap(s.Headroom)
	if err != nil {
		return nil, err
	}
	entry, err := encodeMap(s.Entry)
	if err != nil {
		return nil, err
	}
	unlocked, err := encodeMap(s.Unlocked)
	if err != nil {
		return nil, err
	}
	r.state = &dbgen.UpsertPlannerStateParams{UserID: user, MesoStart: timePtr(ph.MesoStart), LastDeload: timePtr(ph.LastDeload),
		DeloadWeek: timePtr(ph.DeloadWeek), DeloadKind: strPtr(ph.DeloadKind), DeloadNext: strPtr(ph.DeloadNext),
		Calibrate: calibrate, Headroom: headroom, Entry: entry, Unlocked: unlocked}
	if b := s.Break; b != nil {
		ref, err := encodeMap(nilIfEmptyMap(b.Reference))
		if err != nil {
			return nil, err
		}
		base, err := encodeMap(nilIfEmptyMap(b.Base))
		if err != nil {
			return nil, err
		}
		r.brk = &dbgen.UpsertTrainingBreakParams{UserID: user, Days: b.Days, StraightDays: b.StraightDays, Since: timePtr(b.Since),
			Step: int16(b.Step), StepSince: timePtr(b.StepSince), StepSessions: int32(b.StepSessions), Logged: b.Logged, //nolint:gosec // 0–3 and a small counter
			Reference: ref, Base: base}
	}
	for _, rep := range s.Pain {
		if id, err := uuid.Parse(rep.ID); err == nil {
			r.pain[id] = true
		}
	}
	for _, h := range s.History {
		if id, err := uuid.Parse(h.ID); err == nil {
			r.sessions[id] = true
		}
	}
	return r, nil
}

// emptyRows is the row form of a user without planner state.
func emptyRows() *plannerRows {
	return &plannerRows{
		regions:    map[string]dbgen.UpsertRegionStatusParams{},
		capacities: map[string]dbgen.UpsertCapacityEstimateParams{},
		ladders:    map[string]dbgen.UpsertLadderStateParams{},
		pain:       map[uuid.UUID]bool{},
		sessions:   map[uuid.UUID]bool{},
	}
}

func profileFromRow(p dbgen.UserTrainingProfile) (domain.Profile, error) {
	var days []time.Weekday
	for _, d := range p.PreferredDays {
		days = append(days, time.Weekday(int(d)%7))
	}
	out := domain.Profile{BirthYear: int(p.BirthYear), SessionsPerWeek: int(p.SessionsPerWeek), SessionMinutes: int(p.SessionMinutes),
		Equipment: p.Equipment, BodyweightKg: p.BodyweightKg, TrainingLevel: p.TrainingLevel, TrainingMonths: p.TrainingMonths,
		LastRegular: deref(p.LastRegularTraining), HealthConsent: p.HealthDataConsent, DisclaimerAck: p.DisclaimerAck,
		OnboardedAt: p.OnboardedAt.UTC(), PreferredDays: days, MaxAddedLoadKg: p.MaxAddedLoadKg, SmallestPlateKg: p.SmallestPlateKg}
	if len(p.Mobility) > 0 {
		if err := json.Unmarshal(p.Mobility, &out.Mobility); err != nil {
			return out, fmt.Errorf("decoding mobility: %w", err)
		}
	}
	return out, nil
}

func regionFromRow(r dbgen.UserRegionStatus) (domain.RegionState, error) {
	rs := domain.RegionState{State: r.State, EnteredVia: deref(r.EnteredVia), Since: utc(r.Since), StartFraction: r.StartFraction,
		Step: int(r.Step), StepSince: utc(r.StepSince), StepSessions: int(r.StepSessions), PriorInjury: r.PriorInjury,
		Complaint: r.Complaint, ComplaintAt: utc(r.ComplaintAt), Restrictions: nilIfEmpty(r.Restrictions),
		PainDeloadTo: utc(r.PainDeloadTo), RestUntil: utc(r.RestUntil), HoldAtRef: r.HoldAtRef, Referral: deref(r.Referral)}
	for _, b := range r.Breaches {
		rs.Breaches = append(rs.Breaches, b.UTC())
	}
	if len(r.Reference) > 0 {
		if err := json.Unmarshal(r.Reference, &rs.Reference); err != nil {
			return rs, fmt.Errorf("decoding reference of region %s: %w", r.Region, err)
		}
		rs.Reference = nilIfEmptyMap(rs.Reference)
	}
	return rs, nil
}

// ------------------------------------------------------------------- plans

// ActivePlan returns the active plan of a week.
func (t *plannerTx) ActivePlan(ctx context.Context, userID uuid.UUID, week time.Time) (domain.Plan, bool, error) {
	if err := t.check(userID); err != nil {
		return domain.Plan{}, false, err
	}
	row, err := t.q.GetActivePlan(ctx, dbgen.GetActivePlanParams{UserID: userID, WeekStart: dateOf(week.UTC())})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plan{}, false, nil
	}
	if err != nil {
		return domain.Plan{}, false, fmt.Errorf("reading plan: %w", err)
	}
	var p domain.Plan
	if err := json.Unmarshal(row.Payload, &p); err != nil {
		return domain.Plan{}, false, fmt.Errorf("decoding plan %s: %w", row.ID, err)
	}
	if err := t.fillStates(ctx, row.ID, &p); err != nil {
		return domain.Plan{}, false, err
	}
	return p, true, nil
}

// fillStates sets whether each session of a plan was started, and its log
// session. The payload holds what the core planned; the start is a row.
func (t *plannerTx) fillStates(ctx context.Context, planID uuid.UUID, p *domain.Plan) error {
	rows, err := t.q.ListPlannedSessionStates(ctx, dbgen.ListPlannedSessionStatesParams{PlanID: planID, UserID: t.user})
	if err != nil {
		return fmt.Errorf("reading planned sessions of plan %s: %w", planID, err)
	}
	for _, r := range rows {
		i := int(r.OrderIndex)
		if i >= len(p.Sessions) {
			return fmt.Errorf("plan %s has no session %d", planID, i)
		}
		ps := &p.Sessions[i]
		ps.Status, ps.WorkoutSessionID = r.Status, ""
		switch {
		case r.WorkoutSessionID != nil && r.SessionLive:
			ps.WorkoutSessionID = r.WorkoutSessionID.String()
		case r.Status == planning.SessionStarted || r.Status == planning.SessionCompleted:
			// The log session was deleted: the session can start again.
			ps.Status = planning.SessionPlanned
		}
	}
	return nil
}

// SavePlan stores a plan as the active one of its week; the one it replaces
// stays as superseded.
func (t *plannerTx) SavePlan(ctx context.Context, userID uuid.UUID, p domain.Plan) error {
	if err := t.check(userID); err != nil {
		return err
	}
	q, week := t.q, dateOf(p.WeekStart.UTC())
	// The payload is what the core planned; the start of a session lives in
	// its row.
	stored := p
	stored.Sessions = slices.Clone(p.Sessions)
	for i := range stored.Sessions {
		stored.Sessions[i].Status, stored.Sessions[i].WorkoutSessionID = "", ""
	}
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encoding plan: %w", err)
	}
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return fmt.Errorf("plan id %q: %w", p.ID, err)
	}
	if err := q.SupersedeActivePlan(ctx, dbgen.SupersedeActivePlanParams{UserID: userID, WeekStart: week}); err != nil {
		return fmt.Errorf("superseding plan: %w", err)
	}
	if err := q.InsertPlan(ctx, dbgen.InsertPlanParams{ID: id, UserID: userID, WeekStart: week,
		RulesetVersion: p.RulesetVersion, InputHash: p.InputHash, Payload: payload}); err != nil {
		return fmt.Errorf("storing plan: %w", err)
	}
	for i, ps := range p.Sessions {
		sid, err := uuid.Parse(ps.ID)
		if err != nil {
			return fmt.Errorf("planned session id %q: %w", ps.ID, err)
		}
		status := cmp.Or(ps.Status, planning.SessionPlanned)
		workout, err := parseOptionalUUID(ps.WorkoutSessionID)
		if err != nil {
			return fmt.Errorf("planned session %s: log session: %w", ps.ID, err)
		}
		if err := q.InsertPlannedSession(ctx, dbgen.InsertPlannedSessionParams{ID: sid, UserID: userID,
			PlanID: id, OrderIndex: int32(i), ScheduledDate: dateOf(ps.Date.UTC()), Kind: ps.Kind, EstMinutes: ps.EstMinutes, //nolint:gosec // a week has at most seven sessions
			Status: status, WorkoutSessionID: workout}); err != nil {
			return fmt.Errorf("storing planned session %d: %w", i, err)
		}
	}
	return nil
}

// PlannedSession finds a session of one of the user's active plans.
func (t *plannerTx) PlannedSession(ctx context.Context, userID, sessionID uuid.UUID) (domain.Plan, int, bool, error) {
	if err := t.check(userID); err != nil {
		return domain.Plan{}, 0, false, err
	}
	row, err := t.q.GetActivePlannedSession(ctx, dbgen.GetActivePlannedSessionParams{ID: sessionID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Plan{}, 0, false, nil
	}
	if err != nil {
		return domain.Plan{}, 0, false, fmt.Errorf("reading planned session: %w", err)
	}
	var p domain.Plan
	if err := json.Unmarshal(row.Payload, &p); err != nil {
		return domain.Plan{}, 0, false, fmt.Errorf("decoding plan of session %s: %w", sessionID, err)
	}
	if err := t.fillStates(ctx, row.PlanID, &p); err != nil {
		return domain.Plan{}, 0, false, err
	}
	return p, int(row.OrderIndex), true, nil
}

// --------------------------------------------------------------- decisions

// Recorded returns the recorded event of a trigger and source.
func (t *plannerTx) Recorded(ctx context.Context, userID uuid.UUID, trigger, sourceID string) (planning.Decision, bool, error) {
	if err := t.check(userID); err != nil {
		return planning.Decision{}, false, err
	}
	row, err := t.q.GetDecision(ctx, dbgen.GetDecisionParams{UserID: userID, Trigger: trigger, SourceID: sourceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return planning.Decision{}, false, nil
	}
	if err != nil {
		return planning.Decision{}, false, fmt.Errorf("reading decision log: %w", err)
	}
	d, err := decisionFromRow(row)
	return d, err == nil, err
}

// Record stores an event and its changes.
func (t *plannerTx) Record(ctx context.Context, userID uuid.UUID, d planning.Decision) error {
	if err := t.check(userID); err != nil {
		return err
	}
	cs := d.Changes
	if cs == nil {
		cs = []domain.Change{}
	}
	raw, err := json.Marshal(cs)
	if err != nil {
		return fmt.Errorf("encoding changes: %w", err)
	}
	if err := t.q.InsertDecision(ctx, dbgen.InsertDecisionParams{ID: d.ID, UserID: userID, Trigger: d.Trigger,
		SourceID: d.SourceID, OccurredAt: d.At, Changes: raw}); err != nil {
		return fmt.Errorf("recording decision: %w", err)
	}
	return nil
}

// ListDecisions returns recorded events newest first, after the cursor.
func (t *plannerTx) ListDecisions(ctx context.Context, userID uuid.UUID, after *planning.Cursor, limit int) ([]planning.Decision, error) {
	if err := t.check(userID); err != nil {
		return nil, err
	}
	arg := dbgen.ListDecisionsParams{UserID: userID, PageLimit: int32(min(max(limit, 0), maxPage))} //nolint:gosec // bounded above
	if after != nil {
		arg.CursorAt, arg.CursorID = &after.At, &after.ID
	}
	rows, err := t.q.ListDecisions(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("listing decisions: %w", err)
	}
	out := make([]planning.Decision, 0, len(rows))
	for _, row := range rows {
		d, err := decisionFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// maxPage bounds one page of a list.
const maxPage = 1000

func decisionFromRow(row dbgen.PlanDecision) (planning.Decision, error) {
	var cs []domain.Change
	if err := json.Unmarshal(row.Changes, &cs); err != nil {
		return planning.Decision{}, fmt.Errorf("decoding decision %s: %w", row.ID, err)
	}
	if len(cs) == 0 {
		cs = nil
	}
	return planning.Decision{ID: row.ID, Trigger: row.Trigger, SourceID: row.SourceID, At: row.OccurredAt.UTC(), Changes: cs}, nil
}

// ----------------------------------------------------------------- helpers

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// timePtr maps the zero time, which the core uses for "never", to NULL.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func utc(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.UTC()
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func parseOptionalUUID(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%q is not a UUID: %w", s, err)
	}
	return &id, nil
}

func nilIfEmpty[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return s
}

func nilIfEmptyMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) == 0 {
		return nil
	}
	return m
}

// encodeMap stores a nil map as NULL and any other map as a JSON object.
func encodeMap[V any](m map[string]V) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encoding map: %w", err)
	}
	return raw, nil
}

// decodeAll decodes each non-NULL column into its target map.
func decodeAll(targets map[string]any, cols map[string][]byte) error {
	for name, raw := range cols {
		if raw == nil {
			continue
		}
		if err := json.Unmarshal(raw, targets[name]); err != nil {
			return fmt.Errorf("decoding %s: %w", name, err)
		}
	}
	return nil
}
