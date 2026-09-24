// Command contentlint validates the authored content tree in content/.
//
// Checks (all must pass before seeding; cmd/seed runs the same ones):
//
//  1. every YAML file validates against its JSON Schema in content/schema/
//  2. slugs are unique within their kind (skill, skill level, exercise, band)
//  3. every prerequisite edge names a skill + level that exists
//  4. every exercise referenced by a skill level exists
//  5. the prerequisite graph over skill levels is acyclic (topological sort)
//  6. no orphan exercises (defined but referenced by nothing) — warning
//  7. every skill has at least one level and every level exactly one
//     primary_test exercise
//  8. map coordinates are present on every milestone skill and do not collide
//  9. injury entries carry disclaimer: educational_only
//
// The checks live in internal/content; this command only reports them.
// Exit status is 1 on any error, or on any warning with -strict.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jaelricco/hefesto/internal/content"
)

func main() {
	dir := flag.String("dir", "./content", "path to the content directory")
	strict := flag.Bool("strict", false, "treat warnings as errors")
	flag.Parse()

	ok, err := run(os.Stdout, *dir, *strict)
	if err != nil {
		fmt.Fprintln(os.Stderr, "contentlint:", err)
		os.Exit(2)
	}
	if !ok {
		os.Exit(1)
	}
}

func run(w io.Writer, dir string, strict bool) (bool, error) {
	tree, issues, err := content.Load(dir)
	if err != nil {
		return false, fmt.Errorf("loading %s: %w", dir, err)
	}
	// Cross-file checks on a tree with schema errors only produce noise
	// about the files that failed to load.
	if !content.HasErrors(issues, false) {
		issues = append(issues, content.Validate(tree)...)
	}
	content.SortIssues(issues)

	errs, warns := 0, 0
	for _, i := range issues {
		_, _ = fmt.Fprintln(w, i)
		if i.Severity == content.SeverityError {
			errs++
		} else {
			warns++
		}
	}

	counts := tree.Counts()
	_, _ = fmt.Fprintf(w, "contentlint: %d skills, %d levels, %d edges, %d exercises, %d bands — %d errors, %d warnings\n",
		counts["skills"], counts["levels"], counts["edges"], counts["exercises"], counts["bands"], errs, warns)
	return !content.HasErrors(issues, strict), nil
}
