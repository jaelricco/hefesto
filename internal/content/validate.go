package content

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Severity grades an issue. Errors block seeding; warnings do not, unless the
// linter runs with -strict (which CI does).
type Severity string

// Severities.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one problem found in the content tree.
type Issue struct {
	Severity Severity
	File     string
	Msg      string
}

func (i Issue) String() string {
	if i.File == "" {
		return fmt.Sprintf("%s: %s", i.Severity, i.Msg)
	}
	return fmt.Sprintf("%s: %s: %s", i.Severity, i.File, i.Msg)
}

func errorf(file, format string, args ...any) Issue {
	return Issue{Severity: SeverityError, File: file, Msg: fmt.Sprintf(format, args...)}
}

func warnf(file, format string, args ...any) Issue {
	return Issue{Severity: SeverityWarning, File: file, Msg: fmt.Sprintf(format, args...)}
}

// HasErrors reports whether any issue is an error — or, when strict, any issue
// at all.
func HasErrors(issues []Issue, strict bool) bool {
	return slices.ContainsFunc(issues, func(i Issue) bool {
		return strict || i.Severity == SeverityError
	})
}

// SortIssues orders issues by file, then errors before warnings, then message.
func SortIssues(issues []Issue) {
	slices.SortStableFunc(issues, func(a, b Issue) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Severity, b.Severity),
			cmp.Compare(a.Msg, b.Msg),
		)
	})
}

// Validate runs the cross-file checks on a tree that has already passed its
// schemas (check 1, done by Load):
//
//  2. slugs are unique within their kind (family, exercise, skill, level
//     within a skill, band brand+colour), and a file is named after its slug
//  3. every prerequisite and link names a skill and level that exist
//  4. every exercise and family referenced anywhere exists
//  5. the prerequisite graph over levels, including each skill's implicit
//     level ladder, is acyclic
//  6. no orphan exercises — warning
//  7. every level has exactly one primary_test, and level orders are 1..n
//  8. milestone skills are placed on the map, and no two skills share a spot
//  9. injury entries carry disclaimer: educational_only
//
// plus consistency checks on unlock criteria.
func Validate(t Tree) []Issue {
	v := validator{t: t}
	v.index()
	v.checkExercises()
	v.checkBands()
	v.checkSkills()
	v.checkMap()
	v.checkCycles()
	v.checkOrphans()
	SortIssues(v.issues)
	return v.issues
}

type validator struct {
	t         Tree
	issues    []Issue
	families  map[string]bool
	exercises map[string]Exercise
	levels    map[string]bool // "skill/level"
	used      map[string]bool // exercise slugs referenced by anything
}

func (v *validator) add(i Issue) { v.issues = append(v.issues, i) }

func (v *validator) index() {
	v.families = map[string]bool{}
	for _, f := range v.t.Families {
		if v.families[f.Slug] {
			v.add(errorf(FamiliesFile, "family %q is defined twice", f.Slug))
		}
		v.families[f.Slug] = true
	}

	v.exercises = map[string]Exercise{}
	for _, e := range v.t.Exercises {
		if prev, dup := v.exercises[e.Slug]; dup {
			v.add(errorf(e.File, "exercise slug %q is already defined in %s", e.Slug, prev.File))
			continue
		}
		v.exercises[e.Slug] = e
	}

	v.levels = map[string]bool{}
	skills := map[string]string{}
	for _, s := range v.t.Skills {
		if prev, dup := skills[s.Slug]; dup {
			v.add(errorf(s.File, "skill slug %q is already defined in %s", s.Slug, prev))
			continue
		}
		skills[s.Slug] = s.File
		for _, l := range s.Levels {
			k := LevelKey(s.Slug, l.Slug)
			if v.levels[k] {
				v.add(errorf(s.File, "level slug %q appears twice", l.Slug))
			}
			v.levels[k] = true
		}
	}
	v.used = map[string]bool{}
}

func (v *validator) checkExercises() {
	for _, e := range v.t.Exercises {
		if want := fileSlug(e.File); want != e.Slug {
			v.add(errorf(e.File, "file must be named after its slug: expected %s.yaml", e.Slug))
		}
		if !v.families[e.Family] {
			v.add(errorf(e.File, "unknown family %q", e.Family))
		}
	}
}

func (v *validator) checkBands() {
	seen := map[string]string{}
	for _, b := range v.t.Bands {
		k := strings.ToLower(b.Brand) + "\x00" + strings.ToLower(b.ColourLabel)
		if prev, dup := seen[k]; dup {
			v.add(errorf(b.File, "band %s %s is already defined in %s", b.Brand, b.ColourLabel, prev))
		}
		seen[k] = b.File
		if b.ResistanceMinKg > b.ResistanceMaxKg {
			v.add(errorf(b.File, "band %s %s: resistance_min_kg is greater than resistance_max_kg", b.Brand, b.ColourLabel))
		}
	}
}

func (v *validator) checkSkills() {
	for _, s := range v.t.Skills {
		if want := fileSlug(s.File); want != s.Slug {
			v.add(errorf(s.File, "file must be named after its slug: expected %s.yaml", s.Slug))
		}
		if !v.families[s.Family] {
			v.add(errorf(s.File, "unknown family %q", s.Family))
		}

		for i, l := range s.Levels { // sorted by order in Load
			if l.Order != i+1 {
				v.add(errorf(s.File, "level %q has order %d; orders must run 1..%d without gaps or repeats", l.Slug, l.Order, len(s.Levels)))
			}
			v.checkLevel(s, l)
		}

		for _, in := range s.Injuries {
			if in.Disclaimer != "educational_only" {
				v.add(errorf(s.File, "injury %q must carry disclaimer: educational_only", in.Name))
			}
			for _, p := range in.PrehabExercises {
				v.useExercise(s.File, fmt.Sprintf("injury %q prehab", in.Name), p.Slug)
			}
		}
	}
}

func (v *validator) checkLevel(s Skill, l Level) {
	where := fmt.Sprintf("level %q", l.Slug)

	primary := 0
	attached := map[string]bool{}
	seen := map[string]bool{}
	for _, e := range l.Exercises {
		if seen[e.Slug+"\x00"+e.Role] {
			v.add(errorf(s.File, "%s lists exercise %q as %s twice", where, e.Slug, e.Role))
		}
		seen[e.Slug+"\x00"+e.Role] = true
		attached[e.Slug] = true
		if e.Role == RolePrimaryTest {
			primary++
		}
		v.useExercise(s.File, where, e.Slug)
	}
	if primary != 1 {
		v.add(errorf(s.File, "%s must have exactly one primary_test exercise, has %d", where, primary))
	}

	for _, p := range l.Prerequisites {
		v.checkRef(s.File, where, "prerequisite", p.Skill, p.Level)
	}
	for _, k := range l.Links {
		v.checkRef(s.File, where, k.Relation, k.Skill, k.Level)
	}

	for _, c := range l.UnlockCriteria.Conditions() {
		ex, ok := v.useExercise(s.File, where+" unlock criteria", c.Exercise)
		if !ok {
			continue
		}
		if !attached[c.Exercise] {
			v.add(warnf(s.File, "%s unlock criteria test %q, which is not attached to the level", where, c.Exercise))
		}
		if ex.DefaultMeasure != c.Measure {
			v.add(warnf(s.File, "%s unlock criteria measure %q for %q, whose default measure is %q", where, c.Measure, c.Exercise, ex.DefaultMeasure))
		}
		if c.MinLoadKg != nil && c.MaxLoadKg != nil && *c.MinLoadKg > *c.MaxLoadKg {
			v.add(errorf(s.File, "%s unlock criteria for %q: min_load_kg is greater than max_load_kg", where, c.Exercise))
		}
	}
}

func (v *validator) checkRef(file, where, relation, skill, level string) {
	if !v.levels[LevelKey(skill, level)] {
		v.add(errorf(file, "%s %s names %s, which does not exist", where, relation, LevelKey(skill, level)))
	}
}

func (v *validator) useExercise(file, where, slug string) (Exercise, bool) {
	v.used[slug] = true
	e, ok := v.exercises[slug]
	if !ok {
		v.add(errorf(file, "%s references unknown exercise %q", where, slug))
	}
	return e, ok
}

func (v *validator) checkMap() {
	type spot struct {
		c    string
		x, y float64
	}
	seen := map[spot]string{}
	for _, s := range v.t.Skills {
		if s.IsMilestone && s.Map == nil {
			v.add(errorf(s.File, "milestone skill must have map coordinates"))
		}
		if s.Map == nil {
			continue
		}
		k := spot{s.Map.Constellation, s.Map.X, s.Map.Y}
		if prev, dup := seen[k]; dup {
			v.add(errorf(s.File, "map position %s (%g, %g) is already taken by %s", k.c, k.x, k.y, prev))
		}
		seen[k] = s.Slug
	}
}

// checkCycles runs Kahn's algorithm over prerequisite edges between levels
// that exist; edges to missing levels are reported by checkRef.
func (v *validator) checkCycles() {
	indeg := map[string]int{}
	out := map[string][]string{}
	for k := range v.levels {
		indeg[k] = 0
	}
	for _, e := range v.t.Edges() {
		if e.Relation != RelationPrerequisite || !v.levels[e.From] || !v.levels[e.To] {
			continue
		}
		if e.From == e.To {
			v.add(errorf("", "level %s lists itself as a prerequisite", e.To))
			continue
		}
		out[e.From] = append(out[e.From], e.To)
		indeg[e.To]++
	}

	var queue []string
	for k, d := range indeg {
		if d == 0 {
			queue = append(queue, k)
		}
	}
	visited := 0
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		visited++
		for _, m := range out[n] {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
	}
	if visited == len(indeg) {
		return
	}
	var cyclic []string
	for k, d := range indeg {
		if d > 0 {
			cyclic = append(cyclic, k)
		}
	}
	slices.Sort(cyclic)
	v.add(errorf("", "prerequisite cycle: these levels are on or behind a cycle: %s", strings.Join(cyclic, ", ")))
}

func (v *validator) checkOrphans() {
	for _, e := range v.t.Exercises {
		if !v.used[e.Slug] && e.Status != StatusRetired {
			v.add(warnf(e.File, "exercise %q is not used by any skill level, criteria or injury", e.Slug))
		}
	}
}

func fileSlug(rel string) string {
	base := rel[strings.LastIndexAny(rel, `/\`)+1:]
	return strings.TrimSuffix(base, ".yaml")
}
