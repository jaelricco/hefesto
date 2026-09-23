//go:build integration

package store_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/store"
	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

// contentDir copies the repository's content tree into a temp dir the test
// may edit.
func contentDir(t *testing.T) string {
	t.Helper()
	src := filepath.Join(pgtest.RepoRoot(), "content")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func edit(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("%s does not contain %q", rel, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, new, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func load(t *testing.T, dir string) content.Tree {
	t.Helper()
	tree, issues, err := content.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	issues = append(issues, content.Validate(tree)...)
	if content.HasErrors(issues, false) {
		t.Fatalf("content invalid: %v", issues)
	}
	return tree
}

func seed(t *testing.T, pool *pgxpool.Pool, dir string) store.SeedResult {
	t.Helper()
	res, err := store.SeedContent(context.Background(), pool, load(t, dir), store.SeedOptions{AppliedBy: "test"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return res
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func TestSeedImportsTheRepositoryContent(t *testing.T) {
	pool := pgtest.New(t)
	res := seed(t, pool, contentDir(t))
	if res.Unchanged {
		t.Fatal("first seed reported unchanged")
	}

	for table, want := range map[string]int{
		"families": 7, "exercises": 7, "skills": 3, "skill_levels": 4,
		"skill_edges": 2, "skill_injury_risks": 1, "injury_prehab_exercises": 1,
		"content_versions": 1,
	} {
		if got := count(t, pool, "SELECT count(*) FROM "+table); got != want {
			t.Errorf("%s: got %d rows, want %d", table, got, want)
		}
	}
	// The implicit ladder edge and the declared cross-skill prerequisite.
	if got := count(t, pool, `
		SELECT count(*) FROM skill_edges e
		JOIN skill_levels f ON f.id = e.from_skill_level_id JOIN skills fs ON fs.id = f.skill_id
		JOIN skill_levels t ON t.id = e.to_skill_level_id   JOIN skills ts ON ts.id = t.skill_id
		WHERE (fs.slug, f.slug, ts.slug, t.slug) IN (('pull-up','strict-5','front-lever','tuck'),
		                                             ('front-lever','tuck','front-lever','advanced-tuck'))
		  AND e.relation = 'prerequisite'`); got != 2 {
		t.Errorf("expected both prerequisite edges, found %d", got)
	}
	// Criteria are stored with defaults made explicit.
	if got := count(t, pool, `SELECT count(*) FROM skill_levels WHERE unlock_criteria @> '{"all":[{"assistance":"none","occurrences":2}]}'`); got != 2 {
		t.Errorf("criteria with explicit defaults: got %d levels, want 2", got)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	first := seed(t, pool, dir)

	var before time.Time
	if err := pool.QueryRow(context.Background(), `SELECT max(updated_at) FROM exercises`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	second := seed(t, pool, dir)
	if !second.Unchanged || second.ContentVersionID != first.ContentVersionID {
		t.Fatalf("re-seeding unchanged content was not a no-op: %+v", second)
	}
	var after time.Time
	if err := pool.QueryRow(context.Background(), `SELECT max(updated_at) FROM exercises`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !after.Equal(before) {
		t.Fatal("re-seeding bumped updated_at")
	}
	if got := count(t, pool, `SELECT count(*) FROM content_versions`); got != 1 {
		t.Fatalf("content_versions: got %d, want 1", got)
	}
}

func TestSeedUpdatesOnlyWhatChanged(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	first := seed(t, pool, dir)

	edit(t, dir, "exercises/pull-up.yaml", "name: Pull-up", "name: Pull-up (strict)")
	second := seed(t, pool, dir)
	if second.Unchanged || second.ContentVersionID == first.ContentVersionID {
		t.Fatal("changed content did not produce a new version")
	}
	if got := count(t, pool, `SELECT count(*) FROM exercises WHERE content_version_id = $1`, second.ContentVersionID); got != 1 {
		t.Fatalf("rows stamped with the new version: got %d, want exactly the edited one", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM exercises WHERE slug = 'pull-up' AND name = 'Pull-up (strict)'`); got != 1 {
		t.Fatal("edit not applied")
	}
}

func TestSeedRetiresRemovedContent(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	seed(t, pool, dir)

	// Drop the handstand skill and its only exercise.
	if err := os.Remove(filepath.Join(dir, "skills/handstand.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "exercises/wall-handstand-hold.yaml")); err != nil {
		t.Fatal(err)
	}
	res := seed(t, pool, dir)
	if res.RetiredSkills != 1 || res.RetiredExercises != 1 {
		t.Fatalf("retired %d skills, %d exercises; want 1, 1", res.RetiredSkills, res.RetiredExercises)
	}
	if got := count(t, pool, `SELECT count(*) FROM skills WHERE slug = 'handstand' AND status = 'retired'`); got != 1 {
		t.Fatal("removed skill was not retired")
	}
	// Retired, not deleted: its level is still there for user progress to point at.
	if got := count(t, pool, `SELECT count(*) FROM skill_levels l JOIN skills s ON s.id = l.skill_id WHERE s.slug = 'handstand'`); got != 1 {
		t.Fatal("retiring a skill deleted its levels")
	}
}

func TestSeedRefusesToRemoveALevel(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	first := seed(t, pool, dir)

	b, err := os.ReadFile(filepath.Join(dir, "skills/front-lever.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	cut := s[strings.Index(s, "  - slug: advanced-tuck"):strings.Index(s, "injuries:")]
	if err := os.WriteFile(filepath.Join(dir, "skills/front-lever.yaml"), []byte(strings.Replace(s, cut, "", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = store.SeedContent(context.Background(), pool, load(t, dir), store.SeedOptions{})
	if !errors.Is(err, store.ErrLevelRemoved) {
		t.Fatalf("expected ErrLevelRemoved, got %v", err)
	}
	// The whole import rolled back.
	if got := count(t, pool, `SELECT count(*) FROM content_versions WHERE id <> $1`, first.ContentVersionID); got != 0 {
		t.Fatal("a failed seed left a content version behind")
	}
}

func TestSeedReordersLevels(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	seed(t, pool, dir)

	edit(t, dir, "skills/front-lever.yaml", "order: 1", "order: X")
	edit(t, dir, "skills/front-lever.yaml", "order: 2", "order: 1")
	edit(t, dir, "skills/front-lever.yaml", "order: X", "order: 2")
	seed(t, pool, dir)
	if got := count(t, pool, `SELECT count(*) FROM skill_levels WHERE slug = 'advanced-tuck' AND order_index = 1`); got != 1 {
		t.Fatal("levels were not reordered")
	}
}

func TestSeedRevertRecordsANewVersion(t *testing.T) {
	pool := pgtest.New(t)
	dir := contentDir(t)
	a := seed(t, pool, dir)
	edit(t, dir, "exercises/pull-up.yaml", "name: Pull-up", "name: Pull-up B")
	seed(t, pool, dir)
	edit(t, dir, "exercises/pull-up.yaml", "name: Pull-up B", "name: Pull-up")
	back := seed(t, pool, dir)
	if back.Unchanged {
		t.Fatal("reverting to earlier content was treated as unchanged")
	}
	if string(back.Checksum) != string(a.Checksum) {
		t.Fatal("reverted content has a different checksum")
	}
	if got := count(t, pool, `SELECT count(*) FROM exercises WHERE slug = 'pull-up' AND name = 'Pull-up'`); got != 1 {
		t.Fatal("revert not applied")
	}
}

func TestSeedDryRunWritesNothing(t *testing.T) {
	pool := pgtest.New(t)
	res, err := store.SeedContent(context.Background(), pool, load(t, contentDir(t)), store.SeedOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged {
		t.Fatal("dry run against an empty database reported unchanged")
	}
	if got := count(t, pool, `SELECT count(*) FROM skills`); got != 0 {
		t.Fatalf("dry run wrote %d skills", got)
	}
}
