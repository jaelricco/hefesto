//go:build integration

package store_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

// These tests pin the schema's invariants (docs/DDL_PROPOSAL.md §14) so that
// a later migration cannot quietly weaken one.

func id() uuid.UUID { return uuid.Must(uuid.NewV7()) }

type fixture struct {
	pool     *pgxpool.Pool
	user     uuid.UUID
	other    uuid.UUID
	exercise uuid.UUID
	session  uuid.UUID
	block    uuid.UUID
	entry    uuid.UUID
}

func exec(t *testing.T, db *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// violates asserts err is a Postgres error naming the given constraint.
func violates(t *testing.T, err error, constraint string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		t.Fatalf("expected a violation of %s, got %v", constraint, err)
	}
	if pg.ConstraintName != constraint {
		t.Fatalf("expected a violation of %s, got %s (%s: %s)", constraint, pg.ConstraintName, pg.Code, pg.Message)
	}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	f := fixture{pool: pgtest.New(t), user: id(), other: id(), exercise: id(), session: id(), block: id(), entry: id()}
	db := f.pool
	exec(t, db, `INSERT INTO users (id, email) VALUES ($1, 'a@example.test'), ($2, 'b@example.test')`, f.user, f.other)
	cv, fam := id(), id()
	exec(t, db, `INSERT INTO content_versions (id, checksum) VALUES ($1, '\x00')`, cv)
	exec(t, db, `INSERT INTO families (id, slug, name) VALUES ($1, 'pull', 'Pull')`, fam)
	exec(t, db, `INSERT INTO exercises (id, slug, name, family_id, default_measure, content_version_id)
	             VALUES ($1, 'pull-up', 'Pull-up', $2, 'reps', $3)`, f.exercise, fam, cv)
	exec(t, db, `INSERT INTO workout_sessions (id, user_id, started_at, timezone, local_date, updated_at)
	             VALUES ($1, $2, now(), 'Europe/Zurich', current_date, now())`, f.session, f.user)
	exec(t, db, `INSERT INTO session_blocks (id, user_id, session_id, order_index, updated_at)
	             VALUES ($1, $2, $3, 0, now())`, f.block, f.user, f.session)
	exec(t, db, `INSERT INTO set_entries (id, user_id, session_id, block_id, order_index, updated_at)
	             VALUES ($1, $2, $3, $4, 0, now())`, f.entry, f.user, f.session, f.block)
	return f
}

func (f fixture) element(ctx context.Context, order int, extra string, args ...any) error {
	sql := `INSERT INTO set_elements (id, user_id, session_id, set_entry_id, order_index, exercise_id, measure, updated_at` +
		colsOf(extra) + `) VALUES ($1, $2, $3, $4, $5, $6, 'reps', now()` + valsOf(extra, 7) + `)`
	all := append([]any{id(), f.user, f.session, f.entry, order, f.exercise}, args...)
	_, err := f.pool.Exec(ctx, sql, all...)
	return err
}

// colsOf/valsOf let a test add columns to the element insert: extra is a
// comma-separated column list whose values follow in args.
func colsOf(extra string) string {
	if extra == "" {
		return ""
	}
	return ", " + extra
}

func valsOf(extra string, from int) string {
	if extra == "" {
		return ""
	}
	var b strings.Builder
	for i := 0; i <= strings.Count(extra, ","); i++ {
		b.WriteString(", $" + strconv.Itoa(from+i))
	}
	return b.String()
}

func TestComboIsOneEntryWithOrderedElements(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := f.element(ctx, i, "reps", 1); err != nil {
			t.Fatal(err)
		}
	}
	var entries, elements int
	if err := f.pool.QueryRow(ctx, `SELECT count(DISTINCT set_entry_id), count(*) FROM set_elements`).Scan(&entries, &elements); err != nil {
		t.Fatal(err)
	}
	if entries != 1 || elements != 3 {
		t.Fatalf("got %d entries, %d elements; want 1, 3", entries, elements)
	}
	violates(t, f.element(ctx, 1, "reps", 1), "set_elements_order_uk")
}

func TestCrossUserChildIsRejected(t *testing.T) {
	f := newFixture(t)
	_, err := f.pool.Exec(context.Background(),
		`INSERT INTO set_entries (id, user_id, session_id, block_id, order_index, updated_at)
		 VALUES ($1, $2, $3, $4, 1, now())`, id(), f.other, f.session, f.block)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23503" {
		t.Fatalf("expected a foreign-key violation, got %v", err)
	}
}

func TestSetElementConstraints(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	cases := []struct {
		name, extra, constraint string
		args                    []any
	}{
		{"value must match measure", "hold_seconds", "set_elements_value_matches_measure_ck", []any{10}},
		{"one value at most", "reps, hold_seconds", "set_elements_single_value_ck", []any{5, 10}},
		{"malformed tempo", "tempo", "set_elements_tempo_ck", []any{"3-0-X-1"}},
		{"load is never negative", "load_kg", "set_elements_load_ck", []any{-10}},
		{"form quality 1..5", "form_quality", "set_elements_form_quality_ck", []any{6}},
		{"rom note needs partial rom", "rom_note", "set_elements_rom_note_ck", []any{"half"}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			violates(t, f.element(ctx, i, tc.extra, tc.args...), tc.constraint)
		})
	}
	if err := f.element(ctx, 10, "tempo, load_kg", "30X1", 20); err != nil {
		t.Fatalf("valid tempo and added load rejected: %v", err)
	}
}

func TestOrderSwapInsideOneTransaction(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := f.element(ctx, i, ""); err != nil {
			t.Fatal(err)
		}
	}
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE set_elements SET order_index = 1 - order_index`)
		return err
	})
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
}

func TestTouchSyncOwnsServerSeq(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	bw := id()
	// A client-supplied server_seq is overwritten on insert…
	exec(t, f.pool, `INSERT INTO user_bodyweight_log (id, user_id, measured_at, local_date, bodyweight_kg, updated_at, server_seq)
	                 VALUES ($1, $2, now(), current_date, 70, now(), 999999)`, bw, f.user)
	var seq, counter int64
	if err := f.pool.QueryRow(ctx, `SELECT l.server_seq, u.sync_seq FROM user_bodyweight_log l JOIN users u ON u.id = l.user_id WHERE l.id = $1`, bw).Scan(&seq, &counter); err != nil {
		t.Fatal(err)
	}
	if seq != counter || seq == 999999 {
		t.Fatalf("server_seq %d, user counter %d: trigger did not assign it", seq, counter)
	}
	// …and on update, even when the client tries to set it.
	exec(t, f.pool, `UPDATE user_bodyweight_log SET bodyweight_kg = 71, server_seq = 5 WHERE id = $1`, bw)
	var after int64
	if err := f.pool.QueryRow(ctx, `SELECT server_seq FROM user_bodyweight_log WHERE id = $1`, bw).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != seq+1 {
		t.Fatalf("server_seq after update: got %d, want %d", after, seq+1)
	}
	// The counter is per user.
	var otherSeq int64
	if err := f.pool.QueryRow(ctx, `SELECT sync_seq FROM users WHERE id = $1`, f.other).Scan(&otherSeq); err != nil {
		t.Fatal(err)
	}
	if otherSeq != 0 {
		t.Fatalf("another user's counter moved to %d", otherSeq)
	}
}

func TestAssistanceBandNeedsABand(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	el := id()
	exec(t, f.pool, `INSERT INTO set_elements (id, user_id, session_id, set_entry_id, order_index, exercise_id, measure, updated_at)
	                 VALUES ($1, $2, $3, $4, 0, $5, 'reps', now())`, el, f.user, f.session, f.entry, f.exercise)
	_, err := f.pool.Exec(ctx, `INSERT INTO set_element_assistance (id, user_id, set_element_id, type, updated_at)
	                            VALUES ($1, $2, $3, 'band', now())`, id(), f.user, el)
	violates(t, err, "set_element_assistance_band_ck")
}

func TestSessionAndUserChecks(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, err := f.pool.Exec(ctx, `UPDATE workout_sessions SET status = 'completed' WHERE id = $1`, f.session)
	violates(t, err, "workout_sessions_completed_ck")

	_, err = f.pool.Exec(ctx, `UPDATE users SET status = 'deletion_pending' WHERE id = $1`, f.user)
	violates(t, err, "users_deletion_pending_ck")
	exec(t, f.pool, `UPDATE users SET status = 'deletion_pending', deletion_requested_at = now() WHERE id = $1`, f.user)
}

func TestPlannedRestCountsForStreak(t *testing.T) {
	f := newFixture(t)
	exec(t, f.pool, `INSERT INTO user_training_days (user_id, local_date, planned_rest) VALUES ($1, current_date, true)`, f.user)
	var counts bool
	if err := f.pool.QueryRow(context.Background(), `SELECT counts_for_streak FROM user_training_days WHERE user_id = $1`, f.user).Scan(&counts); err != nil {
		t.Fatal(err)
	}
	if !counts {
		t.Fatal("a planned rest day must count for the streak (ADR 0003)")
	}
}

func contentLevel(t *testing.T, db *pgxpool.Pool) (level uuid.UUID) {
	t.Helper()
	var cv, fam uuid.UUID
	if err := db.QueryRow(context.Background(), `SELECT content_version_id, family_id FROM exercises LIMIT 1`).Scan(&cv, &fam); err != nil {
		t.Fatal(err)
	}
	skill := id()
	level = id()
	exec(t, db, `INSERT INTO skills (id, slug, family_id, name, difficulty_tier, content_version_id)
	             VALUES ($1, 'fl', $2, 'FL', 7, $3)`, skill, fam, cv)
	exec(t, db, `INSERT INTO skill_levels (id, skill_id, order_index, slug, name) VALUES ($1, $2, 1, 'tuck', 'Tuck')`, level, skill)
	return level
}

func TestUnlocksAreNeverRevoked(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	level := contentLevel(t, f.pool)

	_, err := f.pool.Exec(ctx, `INSERT INTO user_skill_states (user_id, skill_level_id, state) VALUES ($1, $2, 'unlocked')`, f.user, level)
	violates(t, err, "user_skill_states_unlocked_ck")

	exec(t, f.pool, `INSERT INTO user_skill_states (user_id, skill_level_id, state, first_achieved_at, evidence_set_entry_id)
	                 VALUES ($1, $2, 'unlocked', now(), $3)`, f.user, level, f.entry)
	_, err = f.pool.Exec(ctx, `UPDATE user_skill_states SET state = 'in_progress', first_achieved_at = NULL WHERE user_id = $1`, f.user)
	violates(t, err, "user_skill_states_monotonic")
	_, err = f.pool.Exec(ctx, `UPDATE user_skill_states SET first_achieved_at = now() - interval '1 day' WHERE user_id = $1`, f.user)
	violates(t, err, "user_skill_states_monotonic")
	// Current-form fields stay writable.
	exec(t, f.pool, `UPDATE user_skill_states SET best_value = 12, best_unit = 'hold_seconds', stale_since = now() WHERE user_id = $1`, f.user)

	// Hard-deleting the evidence nulls the evidence column only.
	exec(t, f.pool, `DELETE FROM set_entries WHERE id = $1`, f.entry)
	var evidence *uuid.UUID
	var state string
	if err := f.pool.QueryRow(ctx, `SELECT state, evidence_set_entry_id FROM user_skill_states WHERE user_id = $1`, f.user).Scan(&state, &evidence); err != nil {
		t.Fatal(err)
	}
	if state != "unlocked" || evidence != nil {
		t.Fatalf("after evidence deletion: state %q, evidence %v", state, evidence)
	}
}

func TestUnlockEventsAreAppendOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	level := contentLevel(t, f.pool)
	ev := id()
	exec(t, f.pool, `INSERT INTO skill_unlock_events (id, user_id, skill_level_id, verification, criteria_snapshot)
	                 VALUES ($1, $2, $3, 'auto', '{}')`, ev, f.user, level)

	_, err := f.pool.Exec(ctx, `UPDATE skill_unlock_events SET verification = 'coach' WHERE id = $1`, ev)
	violates(t, err, "skill_unlock_events_append_only")
	_, err = f.pool.Exec(ctx, `DELETE FROM skill_unlock_events WHERE id = $1`, ev)
	violates(t, err, "skill_unlock_events_append_only")
	_, err = f.pool.Exec(ctx, `INSERT INTO skill_unlock_events (id, user_id, skill_level_id, verification, criteria_snapshot)
	                           VALUES ($1, $2, $3, 'auto', '{}')`, id(), f.user, level)
	violates(t, err, "skill_unlock_events_once_uk")

	// Account deletion still works: the cascade is allowed through.
	exec(t, f.pool, `DELETE FROM users WHERE id = $1`, f.user)
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM skill_unlock_events`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d unlock events survived account deletion", n)
	}
}

func TestContentConstraints(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	level := contentLevel(t, f.pool)

	_, err := f.pool.Exec(ctx, `INSERT INTO skill_edges (from_skill_level_id, to_skill_level_id, relation) VALUES ($1, $1, 'prerequisite')`, level)
	violates(t, err, "skill_edges_no_self_ck")

	exec(t, f.pool, `INSERT INTO skill_level_exercises (skill_level_id, exercise_id, role) VALUES ($1, $2, 'primary_test')`, level, f.exercise)
	other := id()
	exec(t, f.pool, `INSERT INTO exercises (id, slug, name, family_id, default_measure, content_version_id)
	                 SELECT $1, 'other', 'Other', family_id, 'reps', content_version_id FROM exercises WHERE id = $2`, other, f.exercise)
	_, err = f.pool.Exec(ctx, `INSERT INTO skill_level_exercises (skill_level_id, exercise_id, role) VALUES ($1, $2, 'primary_test')`, level, other)
	violates(t, err, "skill_level_primary_test_uk")

	_, err = f.pool.Exec(ctx, `UPDATE skills SET is_milestone = true`)
	violates(t, err, "skills_milestone_placed_ck")

	var cv uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT content_version_id FROM skills LIMIT 1`).Scan(&cv); err != nil {
		t.Fatal(err)
	}
	var skill uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT skill_id FROM skill_levels WHERE id = $1`, level).Scan(&skill); err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(ctx, `INSERT INTO skill_injury_risks (id, skill_id, region, name, disclaimer, content_version_id)
	                           VALUES ($1, $2, 'elbow', 'x', 'medical_advice', $3)`, id(), skill, cv)
	violates(t, err, "skill_injury_risks_disclaimer_ck")

	exec(t, f.pool, `INSERT INTO bands (id, brand, colour_label, resistance_min_kg, resistance_max_kg) VALUES ($1, 'Acme', 'Red', 5, 15)`, id())
	_, err = f.pool.Exec(ctx, `INSERT INTO bands (id, brand, colour_label, resistance_min_kg, resistance_max_kg) VALUES ($1, 'Acme', 'Red', 5, 15)`, id())
	violates(t, err, "bands_identity_uk")
}
