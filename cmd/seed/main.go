// Command seed imports the authored content tree into the database.
//
// Contract:
//
//   - runs contentlint's checks first and refuses to touch the database if
//     any error is found (warnings are reported, not fatal);
//   - one transaction for the whole import, so a partial content tree can
//     never be observed by a client;
//   - idempotent upsert keyed on slug — re-running with unchanged content is
//     a no-op and does not bump updated_at;
//   - computes a checksum over the normalised content tree and writes a
//     content_versions row; the /v1/skills ETag is derived from it;
//   - removal is soft: content rows that disappear from the tree are marked
//     status='retired', never deleted, because user data references them.
//     Removing a level from a skill is refused outright.
//
// Content is never inserted by a migration. Migrations are structure, seed is
// data.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/store"
)

func main() {
	dir := flag.String("dir", "./content", "path to the content directory")
	dryRun := flag.Bool("dry-run", false, "import inside a transaction, report, and roll back; without DATABASE_URL, validate only")
	gitSHA := flag.String("git-sha", os.Getenv("HEFESTO_GIT_SHA"), "commit the content came from, recorded on the content version")
	appliedBy := flag.String("applied-by", defaultAppliedBy(), "recorded on the content version")
	flag.Parse()

	cfg := config{
		dir: *dir, dryRun: *dryRun, databaseURL: os.Getenv("DATABASE_URL"),
		opts: store.SeedOptions{GitSHA: *gitSHA, AppliedBy: *appliedBy, DryRun: *dryRun},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Stdout, cfg)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

type config struct {
	dir         string
	dryRun      bool
	databaseURL string
	opts        store.SeedOptions
}

var errInvalidContent = errors.New("content has errors; run make content-validate")

func run(ctx context.Context, w io.Writer, cfg config) error {
	tree, issues, err := content.Load(cfg.dir)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	if !content.HasErrors(issues, false) {
		issues = append(issues, content.Validate(tree)...)
	}
	content.SortIssues(issues)
	for _, i := range issues {
		_, _ = fmt.Fprintln(w, i)
	}
	if content.HasErrors(issues, false) {
		return errInvalidContent
	}

	if cfg.databaseURL == "" {
		if !cfg.dryRun {
			return errors.New("DATABASE_URL is not set")
		}
		sum, err := content.Checksum(tree)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(w, "seed: dry run without a database — content is valid, checksum %s\n", hex.EncodeToString(sum))
		return nil
	}

	pool, err := store.Open(ctx, cfg.databaseURL, 2, 0)
	if err != nil {
		return err
	}
	defer pool.Close()

	res, err := store.SeedContent(ctx, pool, tree, cfg.opts)
	if err != nil {
		return err
	}

	sum := hex.EncodeToString(res.Checksum)
	switch {
	case res.Unchanged:
		_, _ = fmt.Fprintf(w, "seed: content %s is already current — nothing to do\n", sum[:12])
	case cfg.dryRun:
		_, _ = fmt.Fprintf(w, "seed: dry run — would apply content %s (%v); rolled back\n", sum[:12], tree.Counts())
	default:
		_, _ = fmt.Fprintf(w, "seed: applied content %s as version %s (%v)\n", sum[:12], res.ContentVersionID, tree.Counts())
	}
	if res.RetiredExercises+res.RetiredSkills+res.RemovedBands > 0 {
		_, _ = fmt.Fprintf(w, "seed: retired %d exercises, %d skills; removed %d bands\n",
			res.RetiredExercises, res.RetiredSkills, res.RemovedBands)
	}
	return nil
}

func defaultAppliedBy() string {
	host, err := os.Hostname()
	if err != nil {
		return "seed"
	}
	return "seed@" + host
}
