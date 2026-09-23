// Command seed imports the authored content tree into the database.
//
// Contract (Phase 1 implements it; Phase 0 fixes the shape):
//
//   - runs contentlint's checks first and refuses to touch the database if
//     any fail;
//   - one transaction for the whole import, so a partial content tree can
//     never be observed by a client;
//   - idempotent upsert keyed on slug — re-running with unchanged content is
//     a no-op and must not bump updated_at;
//   - computes a checksum over the normalised content tree and writes a
//     content_versions row; the /v1/skills ETag is derived from it;
//   - removal is soft: content rows that disappear from the tree are marked
//     status='retired', never deleted, because user_skill_states reference them.
//
// Content is never inserted by a migration. Migrations are structure, seed is
// data.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	dir := flag.String("dir", "./content", "path to the content directory")
	dryRun := flag.Bool("dry-run", false, "report what would change without writing")
	flag.Parse()

	if err := run(*dir, *dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run(dir string, dryRun bool) error {
	if os.Getenv("DATABASE_URL") == "" && !dryRun {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("content directory: %w", err)
	}
	// TODO(phase-1): parse, validate, upsert in one transaction.
	fmt.Printf("seed: %s — importer not implemented yet (Phase 1)\n", dir)
	return nil
}
