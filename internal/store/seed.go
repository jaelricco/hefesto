package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/store/dbgen"
)

// SeedOptions tunes a content import.
type SeedOptions struct {
	GitSHA    string // recorded on the content version, if known
	AppliedBy string // who or what ran the import
	DryRun    bool   // do everything, then roll back
}

// SeedResult reports what an import did.
type SeedResult struct {
	Checksum         []byte
	ContentVersionID uuid.UUID // the version now current; new unless Unchanged
	Unchanged        bool      // the database already held exactly this content
	RetiredExercises int64
	RetiredSkills    int64
	RemovedBands     int64
}

// ErrLevelRemoved is returned when content drops a level that the database
// holds. Levels are referenced by user progress, which must never silently
// disappear; retire the whole skill instead, or keep the level.
var ErrLevelRemoved = errors.New("a skill level was removed from content")

// ErrPrerequisiteCycle is returned when the seeded graph has a cycle that
// contentlint did not catch.
var ErrPrerequisiteCycle = errors.New("prerequisite cycle in seeded skill graph")

// seedLockKey serialises concurrent imports (two deploys, or make seed during
// a deploy) on a transaction-scoped advisory lock.
const seedLockKey = 0x68656665 // "hefe"

// SeedContent imports a validated content tree in one transaction.
//
// It is idempotent: when the newest content version already has this tree's
// checksum, nothing is written. Otherwise every upsert is keyed on slug and
// leaves unchanged rows untouched, content that disappeared is retired (never
// deleted, because logs and progress reference it), and a new
// content_versions row records the checksum.
//
// The caller must have run content.Validate; SeedContent trusts the tree.
func SeedContent(ctx context.Context, pool *pgxpool.Pool, tree content.Tree, opts SeedOptions) (SeedResult, error) {
	sum, err := content.Checksum(tree)
	if err != nil {
		return SeedResult{}, err
	}
	res := SeedResult{Checksum: sum}

	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", seedLockKey); err != nil {
			return fmt.Errorf("taking seed lock: %w", err)
		}
		q := dbgen.New(tx)

		latest, err := q.GetLatestContentVersion(ctx)
		switch {
		case err == nil && bytes.Equal(latest.Checksum, sum):
			res.Unchanged = true
			res.ContentVersionID = latest.ID
			return nil
		case err != nil && !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("reading latest content version: %w", err)
		}

		// Level reordering swaps order_index values mid-transaction.
		if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
			return fmt.Errorf("deferring constraints: %w", err)
		}

		s := seeder{q: q, tree: tree, res: &res}
		if err := s.run(ctx, opts); err != nil {
			return err
		}
		if opts.DryRun {
			return errDryRun
		}
		return nil
	})
	if errors.Is(err, errDryRun) {
		return res, nil
	}
	if err != nil {
		return SeedResult{}, fmt.Errorf("seeding content: %w", err)
	}
	return res, nil
}

var errDryRun = errors.New("dry run: rolling back")

type seeder struct {
	q    *dbgen.Queries
	tree content.Tree
	res  *SeedResult

	version   uuid.UUID
	families  map[string]uuid.UUID
	exercises map[string]uuid.UUID
	skillIDs  map[string]uuid.UUID
	levels    map[string]uuid.UUID // "skill/level"
}

func (s *seeder) run(ctx context.Context, opts SeedOptions) error {
	counts, err := json.Marshal(s.tree.Counts())
	if err != nil {
		return fmt.Errorf("encoding item counts: %w", err)
	}
	s.version = uuid.Must(uuid.NewV7())
	if err := s.q.InsertContentVersion(ctx, dbgen.InsertContentVersionParams{
		ID:         s.version,
		Checksum:   s.res.Checksum,
		GitSha:     nonEmpty(opts.GitSHA),
		ItemCounts: counts,
		AppliedBy:  nonEmpty(opts.AppliedBy),
	}); err != nil {
		return fmt.Errorf("recording content version: %w", err)
	}
	s.res.ContentVersionID = s.version

	for _, step := range []func(context.Context) error{
		s.seedFamilies, s.seedExercises, s.seedBands, s.seedSkills, s.seedEdges, s.seedInjuries,
	} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedFamilies(ctx context.Context) error {
	s.families = map[string]uuid.UUID{}
	for i, f := range s.tree.Families {
		id, err := s.q.UpsertFamily(ctx, dbgen.UpsertFamilyParams{
			ID: uuid.Must(uuid.NewV7()), Slug: f.Slug, Name: f.Name, OrderIndex: int16(i), //nolint:gosec // bounded by content size
		})
		if err != nil {
			return fmt.Errorf("family %s: %w", f.Slug, err)
		}
		s.families[f.Slug] = id
	}
	return nil
}

func (s *seeder) seedExercises(ctx context.Context) error {
	s.exercises = map[string]uuid.UUID{}
	slugs := make([]string, 0, len(s.tree.Exercises))
	for _, e := range s.tree.Exercises {
		id, err := s.q.UpsertExercise(ctx, dbgen.UpsertExerciseParams{
			ID:               uuid.Must(uuid.NewV7()),
			Slug:             e.Slug,
			Name:             e.Name,
			Aka:              e.Aka,
			FamilyID:         s.families[e.Family],
			DefaultMeasure:   e.DefaultMeasure,
			LoadSemantics:    e.LoadSemantics,
			IsBodyweight:     *e.IsBodyweight,
			Unilateral:       e.Unilateral,
			TempoApplicable:  *e.TempoApplicable,
			Equipment:        e.Equipment,
			Summary:          e.Summary,
			Cues:             e.Cues,
			CommonFaults:     e.CommonFaults,
			Status:           e.Status,
			ContentVersionID: s.version,
		})
		if err != nil {
			return fmt.Errorf("exercise %s: %w", e.Slug, err)
		}
		s.exercises[e.Slug] = id
		slugs = append(slugs, e.Slug)
	}
	n, err := s.q.RetireExercisesNotIn(ctx, dbgen.RetireExercisesNotInParams{ContentVersionID: s.version, Slugs: slugs})
	if err != nil {
		return fmt.Errorf("retiring exercises: %w", err)
	}
	s.res.RetiredExercises = n
	return nil
}

func (s *seeder) seedBands(ctx context.Context) error {
	keys := make([]string, 0, len(s.tree.Bands))
	for _, b := range s.tree.Bands {
		lo, err := numeric(b.ResistanceMinKg)
		if err != nil {
			return err
		}
		hi, err := numeric(b.ResistanceMaxKg)
		if err != nil {
			return err
		}
		length, err := numericPtr(b.LengthCm)
		if err != nil {
			return err
		}
		thick, err := numericPtr(b.ThicknessMm)
		if err != nil {
			return err
		}
		if err := s.q.UpsertGlobalBand(ctx, dbgen.UpsertGlobalBandParams{
			ID: uuid.Must(uuid.NewV7()), Brand: b.Brand, ColourLabel: b.ColourLabel,
			ResistanceMinKg: lo, ResistanceMaxKg: hi, LengthCm: length, ThicknessMm: thick,
		}); err != nil {
			return fmt.Errorf("band %s %s: %w", b.Brand, b.ColourLabel, err)
		}
		keys = append(keys, b.Brand+"\x1f"+b.ColourLabel)
	}
	n, err := s.q.SoftDeleteGlobalBandsNotIn(ctx, keys)
	if err != nil {
		return fmt.Errorf("removing bands: %w", err)
	}
	s.res.RemovedBands = n
	return nil
}

func (s *seeder) seedSkills(ctx context.Context) error {
	s.skillIDs = map[string]uuid.UUID{}
	s.levels = map[string]uuid.UUID{}
	slugs := make([]string, 0, len(s.tree.Skills))
	for _, sk := range s.tree.Skills {
		params := dbgen.UpsertSkillParams{
			ID:               uuid.Must(uuid.NewV7()),
			Slug:             sk.Slug,
			FamilyID:         s.families[sk.Family],
			Name:             sk.Name,
			Aka:              sk.Aka,
			DifficultyTier:   int16(sk.DifficultyTier), //nolint:gosec // schema bounds it to 1..10
			IsMilestone:      sk.IsMilestone,
			Summary:          sk.Summary,
			PrimaryMuscles:   sk.PrimaryMuscles,
			CommonFaults:     sk.CommonFaults,
			Status:           sk.Status,
			ContentVersionID: s.version,
		}
		if sk.Map != nil {
			var err error
			params.MapConstellation = &sk.Map.Constellation
			if params.MapX, err = numeric(sk.Map.X); err != nil {
				return err
			}
			if params.MapY, err = numeric(sk.Map.Y); err != nil {
				return err
			}
		}
		skillID, err := s.q.UpsertSkill(ctx, params)
		if err != nil {
			return fmt.Errorf("skill %s: %w", sk.Slug, err)
		}
		s.skillIDs[sk.Slug] = skillID
		slugs = append(slugs, sk.Slug)

		if err := s.seedLevels(ctx, sk, skillID); err != nil {
			return err
		}
	}
	n, err := s.q.RetireSkillsNotIn(ctx, dbgen.RetireSkillsNotInParams{ContentVersionID: s.version, Slugs: slugs})
	if err != nil {
		return fmt.Errorf("retiring skills: %w", err)
	}
	s.res.RetiredSkills = n
	return nil
}

func (s *seeder) seedLevels(ctx context.Context, sk content.Skill, skillID uuid.UUID) error {
	existing, err := s.q.ListSkillLevelSlugs(ctx, skillID)
	if err != nil {
		return fmt.Errorf("skill %s levels: %w", sk.Slug, err)
	}
	for _, slug := range existing {
		if !slices.ContainsFunc(sk.Levels, func(l content.Level) bool { return l.Slug == slug }) {
			return fmt.Errorf("%w: %s", ErrLevelRemoved, content.LevelKey(sk.Slug, slug))
		}
	}

	for _, l := range sk.Levels {
		criteria, err := json.Marshal(l.UnlockCriteria)
		if err != nil {
			return fmt.Errorf("level %s criteria: %w", content.LevelKey(sk.Slug, l.Slug), err)
		}
		var weeks *int16
		if l.EstWeeksFromPrev != nil {
			w := int16(*l.EstWeeksFromPrev) //nolint:gosec // authored weeks, small
			weeks = &w
		}
		levelID, err := s.q.UpsertSkillLevel(ctx, dbgen.UpsertSkillLevelParams{
			ID:               uuid.Must(uuid.NewV7()),
			SkillID:          skillID,
			OrderIndex:       int16(l.Order), //nolint:gosec // validated 1..n
			Slug:             l.Slug,
			Name:             l.Name,
			Description:      l.Description,
			UnlockCriteria:   criteria,
			EstWeeksFromPrev: weeks,
		})
		if err != nil {
			return fmt.Errorf("level %s: %w", content.LevelKey(sk.Slug, l.Slug), err)
		}
		s.levels[content.LevelKey(sk.Slug, l.Slug)] = levelID

		if err := s.q.DeleteSkillLevelExercises(ctx, levelID); err != nil {
			return fmt.Errorf("level %s exercises: %w", content.LevelKey(sk.Slug, l.Slug), err)
		}
		for i, e := range l.Exercises {
			if err := s.q.InsertSkillLevelExercise(ctx, dbgen.InsertSkillLevelExerciseParams{
				SkillLevelID: levelID, ExerciseID: s.exercises[e.Slug], Role: e.Role, OrderIndex: int16(i), //nolint:gosec // bounded by content size
			}); err != nil {
				return fmt.Errorf("level %s exercise %s: %w", content.LevelKey(sk.Slug, l.Slug), e.Slug, err)
			}
		}
	}
	return nil
}

func (s *seeder) seedEdges(ctx context.Context) error {
	if err := s.q.DeleteAllSkillEdges(ctx); err != nil {
		return fmt.Errorf("clearing edges: %w", err)
	}
	for _, e := range s.tree.Edges() {
		w, err := numeric(e.Weight)
		if err != nil {
			return err
		}
		if err := s.q.InsertSkillEdge(ctx, dbgen.InsertSkillEdgeParams{
			FromSkillLevelID: s.levels[e.From], ToSkillLevelID: s.levels[e.To], Relation: e.Relation, Weight: w,
		}); err != nil {
			return fmt.Errorf("edge %s -> %s (%s): %w", e.From, e.To, e.Relation, err)
		}
	}
	cyclic, err := s.q.FindPrerequisiteCycles(ctx)
	if err != nil {
		return fmt.Errorf("checking for cycles: %w", err)
	}
	if len(cyclic) > 0 {
		names := make([]string, 0, len(cyclic))
		for key, id := range s.levels {
			if slices.Contains(cyclic, id) {
				names = append(names, key)
			}
		}
		slices.Sort(names)
		return fmt.Errorf("%w: %s", ErrPrerequisiteCycle, strings.Join(names, ", "))
	}
	return nil
}

func (s *seeder) seedInjuries(ctx context.Context) error {
	for _, sk := range s.tree.Skills {
		skillID, err := s.skillID(sk)
		if err != nil {
			return err
		}
		keep := make([]uuid.UUID, 0, len(sk.Injuries))
		for i, in := range sk.Injuries {
			riskID, err := s.q.UpsertInjuryRisk(ctx, dbgen.UpsertInjuryRiskParams{
				ID: uuid.Must(uuid.NewV7()), SkillID: skillID, Region: in.Region, Name: in.Name,
				Description: in.Description, RiskFactors: in.RiskFactors, EarlySigns: in.EarlySigns,
				OrderIndex: int16(i), ContentVersionID: s.version, //nolint:gosec // bounded by content size
			})
			if err != nil {
				return fmt.Errorf("skill %s injury %q: %w", sk.Slug, in.Name, err)
			}
			keep = append(keep, riskID)
			if err := s.q.DeleteInjuryPrehab(ctx, riskID); err != nil {
				return fmt.Errorf("skill %s injury %q prehab: %w", sk.Slug, in.Name, err)
			}
			for j, p := range in.PrehabExercises {
				if err := s.q.InsertInjuryPrehab(ctx, dbgen.InsertInjuryPrehabParams{
					RiskID: riskID, ExerciseID: s.exercises[p.Slug], OrderIndex: int16(j), //nolint:gosec // bounded by content size
				}); err != nil {
					return fmt.Errorf("skill %s injury %q prehab %s: %w", sk.Slug, in.Name, p.Slug, err)
				}
			}
		}
		if err := s.q.DeleteInjuryRisksNotIn(ctx, dbgen.DeleteInjuryRisksNotInParams{SkillID: skillID, Keep: keep}); err != nil {
			return fmt.Errorf("skill %s injuries: %w", sk.Slug, err)
		}
	}
	return nil
}

func (s *seeder) skillID(sk content.Skill) (uuid.UUID, error) {
	id, ok := s.skillIDs[sk.Slug]
	if !ok {
		return uuid.Nil, fmt.Errorf("skill %s was not seeded", sk.Slug)
	}
	return id, nil
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
