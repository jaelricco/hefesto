package content_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaelricco/hefesto/internal/content"
)

// base is a minimal valid tree. Each test case overrides or removes files.
var base = map[string]string{
	"families.yaml": `families:
  - { slug: pull, name: Pull }
  - { slug: push, name: Push }
`,
	"exercises/pull-up.yaml": `slug: pull-up
name: Pull-up
family: pull
default_measure: reps
`,
	"exercises/fl-tuck.yaml": `slug: fl-tuck
name: Tuck FL
family: pull
default_measure: hold_seconds
`,
	"exercises/curl.yaml": `slug: curl
name: Curl
family: pull
default_measure: reps
`,
	"skills/pull-up.yaml": `slug: pull-up
name: Pull-up
family: pull
difficulty_tier: 2
levels:
  - slug: strict-5
    name: Five
    order: 1
    exercises: [ { slug: pull-up, role: primary_test } ]
    unlock_criteria:
      all: [ { exercise: pull-up, measure: reps, op: ">=", value: 5, occurrences: 2 } ]
`,
	"skills/front-lever.yaml": `slug: front-lever
name: Front Lever
family: pull
difficulty_tier: 7
is_milestone: true
map: { constellation: pull_north, x: 1, y: 2 }
levels:
  - slug: tuck
    name: Tuck
    order: 1
    exercises: [ { slug: fl-tuck, role: primary_test } ]
    prerequisites: [ { skill: pull-up, level: strict-5 } ]
  - slug: adv
    name: Advanced
    order: 2
    exercises: [ { slug: fl-tuck, role: primary_test } ]
injuries:
  - region: elbow
    name: Elbow thing
    prehab_exercises: [ { slug: curl } ]
    disclaimer: educational_only
`,
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	schemas, err := filepath.Glob("../../content/schema/*.json")
	if err != nil || len(schemas) == 0 {
		t.Fatalf("finding schemas: %v", err)
	}
	for _, s := range schemas {
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, "schema", filepath.Base(s)), string(b))
	}
	for name, body := range files {
		if body == "" {
			continue
		}
		write(t, filepath.Join(dir, name), body)
	}
	return dir
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func with(overrides map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v // "" deletes
	}
	return out
}

// lint mirrors cmd/contentlint: schema issues first, cross-file checks only
// on a schema-clean tree.
func lint(t *testing.T, files map[string]string) (content.Tree, []content.Issue) {
	t.Helper()
	tree, issues, err := content.Load(writeTree(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !content.HasErrors(issues, false) {
		issues = append(issues, content.Validate(tree)...)
	}
	return tree, issues
}

func TestValidTreeIsClean(t *testing.T) {
	tree, issues := lint(t, base)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if got := len(tree.Skills); got != 2 {
		t.Fatalf("skills: got %d, want 2", got)
	}
	// Defaults are explicit after loading.
	c := tree.Skills[1].Levels[0].UnlockCriteria.All[0] // pull-up sorts after front-lever
	if c.Assistance != "none" || c.Occurrences != 2 {
		t.Fatalf("criteria defaults not applied: %+v", c)
	}
	if tree.Exercises[0].Status != content.StatusActive || tree.Exercises[0].IsBodyweight == nil || !*tree.Exercises[0].IsBodyweight {
		t.Fatalf("exercise defaults not applied: %+v", tree.Exercises[0])
	}
}

func TestChecks(t *testing.T) {
	cases := []struct {
		name     string
		files    map[string]string
		severity content.Severity
		want     string
	}{
		{
			name:     "schema: unknown field",
			files:    with(map[string]string{"exercises/curl.yaml": base["exercises/curl.yaml"] + "colour: red\n"}),
			severity: content.SeverityError,
			want:     "additional properties 'colour' not allowed",
		},
		{
			name:     "schema: bad slug",
			files:    with(map[string]string{"exercises/curl.yaml": strings.Replace(base["exercises/curl.yaml"], "slug: curl", "slug: Curl_X", 1)}),
			severity: content.SeverityError,
			want:     "does not match pattern",
		},
		{
			name:     "schema: milestone without map",
			files:    with(map[string]string{"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"], "map: { constellation: pull_north, x: 1, y: 2 }\n", "", 1)}),
			severity: content.SeverityError,
			want:     "missing property 'map'",
		},
		{
			name:     "schema: injury without disclaimer",
			files:    with(map[string]string{"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"], "    disclaimer: educational_only\n", "", 1)}),
			severity: content.SeverityError,
			want:     "missing property 'disclaimer'",
		},
		{
			name:     "schema: wrong disclaimer",
			files:    with(map[string]string{"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"], "disclaimer: educational_only", "disclaimer: medical_advice", 1)}),
			severity: content.SeverityError,
			want:     "disclaimer",
		},
		{
			name:     "schema: invalid yaml",
			files:    with(map[string]string{"exercises/curl.yaml": "slug: [unclosed\n"}),
			severity: content.SeverityError,
			want:     "invalid YAML",
		},
		{
			name:     "schema: .yml extension",
			files:    with(map[string]string{"exercises/x.yml": "slug: x\n"}),
			severity: content.SeverityError,
			want:     "use the .yaml extension",
		},
		{
			name:     "families file missing",
			files:    with(map[string]string{"families.yaml": ""}),
			severity: content.SeverityError,
			want:     "file is missing",
		},
		{
			name: "duplicate exercise slug",
			files: with(map[string]string{"exercises/curl-2.yaml": `slug: curl
name: Curl again
family: pull
default_measure: reps
`}),
			severity: content.SeverityError,
			want:     `exercise slug "curl" is already defined`,
		},
		{
			name:     "file not named after slug",
			files:    with(map[string]string{"exercises/curl.yaml": "", "exercises/curling.yaml": base["exercises/curl.yaml"]}),
			severity: content.SeverityError,
			want:     "expected curl.yaml",
		},
		{
			name:     "unknown family",
			files:    with(map[string]string{"exercises/curl.yaml": strings.Replace(base["exercises/curl.yaml"], "family: pull", "family: legs", 1)}),
			severity: content.SeverityError,
			want:     `unknown family "legs"`,
		},
		{
			name:     "prerequisite to a missing level",
			files:    with(map[string]string{"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"], "level: strict-5", "level: strict-10", 1)}),
			severity: content.SeverityError,
			want:     "pull-up/strict-10, which does not exist",
		},
		{
			name:     "level references an unknown exercise",
			files:    with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"], "{ slug: pull-up, role: primary_test }", "{ slug: chin-up, role: primary_test }", 1)}),
			severity: content.SeverityError,
			want:     `unknown exercise "chin-up"`,
		},
		{
			name: "prerequisite cycle across skills",
			files: with(map[string]string{"skills/pull-up.yaml": base["skills/pull-up.yaml"] +
				"    prerequisites: [ { skill: front-lever, level: adv } ]\n"}),
			severity: content.SeverityError,
			want:     "prerequisite cycle",
		},
		{
			name:     "self prerequisite",
			files:    with(map[string]string{"skills/pull-up.yaml": base["skills/pull-up.yaml"] + "    prerequisites: [ { skill: pull-up, level: strict-5 } ]\n"}),
			severity: content.SeverityError,
			want:     "lists itself as a prerequisite",
		},
		{
			name:     "orphan exercise",
			files:    with(map[string]string{"exercises/dip.yaml": "slug: dip\nname: Dip\nfamily: push\ndefault_measure: reps\n"}),
			severity: content.SeverityWarning,
			want:     `exercise "dip" is not used`,
		},
		{
			name:     "no primary test",
			files:    with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"], "role: primary_test", "role: progression", 1)}),
			severity: content.SeverityError,
			want:     "exactly one primary_test exercise, has 0",
		},
		{
			name: "two primary tests",
			files: with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"],
				"exercises: [ { slug: pull-up, role: primary_test } ]",
				"exercises: [ { slug: pull-up, role: primary_test }, { slug: curl, role: primary_test } ]", 1)}),
			severity: content.SeverityError,
			want:     "has 2",
		},
		{
			name:     "gap in level order",
			files:    with(map[string]string{"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"], "order: 2", "order: 3", 1)}),
			severity: content.SeverityError,
			want:     "orders must run 1..2",
		},
		{
			name: "map collision",
			files: with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"],
				"difficulty_tier: 2\n", "difficulty_tier: 2\nmap: { constellation: pull_north, x: 1, y: 2 }\n", 1)}),
			severity: content.SeverityError,
			want:     "is already taken by front-lever",
		},
		{
			name:     "criteria test an unattached exercise",
			files:    with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"], "all: [ { exercise: pull-up, measure: reps", "all: [ { exercise: curl, measure: reps", 1)}),
			severity: content.SeverityWarning,
			want:     "not attached to the level",
		},
		{
			name:     "criteria measure disagrees with exercise",
			files:    with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"], "measure: reps", "measure: hold_seconds", 1)}),
			severity: content.SeverityWarning,
			want:     `whose default measure is "reps"`,
		},
		{
			name: "criteria min load above max load",
			files: with(map[string]string{"skills/pull-up.yaml": strings.Replace(base["skills/pull-up.yaml"],
				"occurrences: 2 }", "occurrences: 2, min_load_kg: 20, max_load_kg: 10 }", 1)}),
			severity: content.SeverityError,
			want:     "min_load_kg is greater than max_load_kg",
		},
		{
			name: "duplicate band",
			files: with(map[string]string{"bands/acme.yaml": `bands:
  - { brand: Acme, colour_label: Red, resistance_min_kg: 5, resistance_max_kg: 15 }
  - { brand: acme, colour_label: red, resistance_min_kg: 5, resistance_max_kg: 15 }
`}),
			severity: content.SeverityError,
			want:     "is already defined",
		},
		{
			name: "band range inverted",
			files: with(map[string]string{"bands/acme.yaml": `bands:
  - { brand: Acme, colour_label: Red, resistance_min_kg: 20, resistance_max_kg: 15 }
`}),
			severity: content.SeverityError,
			want:     "resistance_min_kg is greater",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issues := lint(t, tc.files)
			for _, i := range issues {
				if i.Severity == tc.severity && strings.Contains(i.Msg, tc.want) {
					return
				}
			}
			t.Fatalf("no %s containing %q; got:\n%v", tc.severity, tc.want, issues)
		})
	}
}

func TestEdgesIncludeImplicitLadder(t *testing.T) {
	tree, issues := lint(t, base)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	got := map[string]bool{}
	for _, e := range tree.Edges() {
		got[e.From+">"+e.To+":"+e.Relation] = true
	}
	for _, want := range []string{
		"pull-up/strict-5>front-lever/tuck:prerequisite",
		"front-lever/tuck>front-lever/adv:prerequisite",
	} {
		if !got[want] {
			t.Errorf("missing edge %s in %v", want, got)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 edges, got %v", got)
	}
}

func TestChecksumIgnoresFormatting(t *testing.T) {
	a, _, err := content.Load(writeTree(t, base))
	if err != nil {
		t.Fatal(err)
	}
	// Same meaning: flow vs block style, reordered keys, explicit defaults.
	b, _, err := content.Load(writeTree(t, with(map[string]string{
		"exercises/curl.yaml": "family: pull\ndefault_measure: reps\nname: Curl\nslug: curl\nstatus: active\nis_bodyweight: true\n",
		"families.yaml":       "families:\n  - slug: pull\n    name: Pull\n  - name: Push\n    slug: push\n",
	})))
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := content.Checksum(a)
	cb, _ := content.Checksum(b)
	if !bytes.Equal(ca, cb) {
		t.Fatal("checksums differ for semantically identical trees")
	}

	c, _, err := content.Load(writeTree(t, with(map[string]string{
		"exercises/curl.yaml": strings.Replace(base["exercises/curl.yaml"], "name: Curl", "name: Curl!", 1),
	})))
	if err != nil {
		t.Fatal(err)
	}
	cc, _ := content.Checksum(c)
	if bytes.Equal(ca, cc) {
		t.Fatal("checksum did not change when content did")
	}
}

func TestRepositoryContentIsClean(t *testing.T) {
	tree, issues, err := content.Load("../../content")
	if err != nil {
		t.Fatal(err)
	}
	issues = append(issues, content.Validate(tree)...)
	if content.HasErrors(issues, true) {
		t.Fatalf("content/ has issues:\n%v", issues)
	}
}

func TestLoadErrors(t *testing.T) {
	if _, _, err := content.Load(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
	if _, _, err := content.Load(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no schemas") {
		t.Fatalf("expected a missing-schema error, got %v", err)
	}
}

func TestPlaceholdersAndRelease(t *testing.T) {
	placeholder := `slug: fl-tuck
status: draft_placeholder
name: Tuck FL
family: pull
default_measure: hold_seconds
`
	// A researched skill leaning on a placeholder exercise is a warning.
	tree, issues := lint(t, with(map[string]string{"exercises/fl-tuck.yaml": placeholder}))
	if content.HasErrors(issues, false) || !content.HasErrors(issues, true) {
		t.Fatalf("want only warnings, got %v", issues)
	}
	if !strings.Contains(issues[0].Msg, `"fl-tuck", which is still a draft placeholder`) {
		t.Fatalf("issues %v", issues)
	}
	r := content.ReadinessOf(tree)
	if r.Ready() || r.PlaceholderExercises != 1 || r.Exercises != 3 || r.PlaceholderSkills != 0 || r.Skills != 2 {
		t.Fatalf("readiness %+v", r)
	}
	rel := content.ReleaseIssues(tree)
	if len(rel) != 1 || rel[0].Severity != content.SeverityError || rel[0].File != "exercises/fl-tuck.yaml" {
		t.Fatalf("release issues %v", rel)
	}

	// A placeholder skill may use placeholder exercises without a warning,
	// and researched content has nothing to report for a release.
	tree, issues = lint(t, with(map[string]string{
		"exercises/fl-tuck.yaml": placeholder,
		"skills/front-lever.yaml": strings.Replace(base["skills/front-lever.yaml"],
			"name: Front Lever\n", "status: draft_placeholder\nname: Front Lever\n", 1),
	}))
	if len(issues) != 0 {
		t.Fatalf("placeholder skill: %v", issues)
	}
	if got := len(content.ReleaseIssues(tree)); got != 2 {
		t.Fatalf("release issues: %d, want 2", got)
	}
	tree, _ = lint(t, base)
	if rel := content.ReleaseIssues(tree); len(rel) != 0 || !content.ReadinessOf(tree).Ready() {
		t.Fatalf("researched tree: %v", rel)
	}
}
