// Command contentlint validates the authored content tree in content/.
//
// Phase 1 implements the checks below. Phase 0 ships the entrypoint and the
// check list so that `make content-validate` exists from the first commit and
// CI has something to call.
//
// Checks (all must pass before seeding):
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
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	dir := flag.String("dir", "./content", "path to the content directory")
	strict := flag.Bool("strict", false, "treat warnings as errors")
	flag.Parse()

	if err := run(*dir, *strict); err != nil {
		fmt.Fprintln(os.Stderr, "contentlint:", err)
		os.Exit(1)
	}
}

func run(dir string, _ bool) error {
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("content directory: %w", err)
	}
	// TODO(phase-1): implement the nine checks documented above.
	fmt.Printf("contentlint: %s — no checks implemented yet (Phase 1)\n", dir)
	return nil
}
