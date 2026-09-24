-- The skill graph, user progress, XP and streaks.

-- ----------------------------------------------------------------- graph

-- name: ListGraphSkills :many
SELECT sk.id, sk.slug, sk.name, sk.aka, sk.difficulty_tier, sk.is_milestone, sk.summary, sk.primary_muscles, sk.common_faults, sk.map_constellation, sk.map_x, sk.map_y, sk.status, f.slug AS family_slug
FROM skills sk JOIN families f ON f.id = sk.family_id
WHERE sk.status <> 'retired'
ORDER BY sk.slug;

-- name: GetGraphSkill :one
SELECT sk.id, sk.slug, sk.name, sk.aka, sk.difficulty_tier, sk.is_milestone, sk.summary, sk.primary_muscles, sk.common_faults, sk.map_constellation, sk.map_x, sk.map_y, sk.status, f.slug AS family_slug
FROM skills sk JOIN families f ON f.id = sk.family_id
WHERE sk.slug = $1 AND sk.status <> 'retired';

-- Every level of every skill, retired or not: prerequisites may point at a
-- retired skill's level, and unlocks on them still stand.
-- name: ListGraphLevels :many
SELECT l.id, l.skill_id, l.order_index, l.slug, l.name, l.description, l.unlock_criteria,
       l.est_weeks_from_prev, s.slug AS skill_slug, s.name AS skill_name,
       s.difficulty_tier, s.status AS skill_status
FROM skill_levels l JOIN skills s ON s.id = l.skill_id
ORDER BY s.slug, l.order_index;

-- name: ListGraphEdges :many
SELECT from_skill_level_id, to_skill_level_id, relation, weight
FROM skill_edges
ORDER BY to_skill_level_id, from_skill_level_id, relation;

-- name: ListGraphLevelExercises :many
SELECT le.skill_level_id, le.exercise_id, e.slug, le.role
FROM skill_level_exercises le JOIN exercises e ON e.id = le.exercise_id
ORDER BY le.skill_level_id, le.order_index, e.slug;

-- name: ListSkillInjuries :many
SELECT * FROM skill_injury_risks WHERE skill_id = $1 ORDER BY order_index, name;

-- name: ListInjuryPrehab :many
SELECT p.risk_id, p.exercise_id, e.slug
FROM injury_prehab_exercises p
JOIN exercises e ON e.id = p.exercise_id
JOIN skill_injury_risks r ON r.id = p.risk_id
WHERE r.skill_id = $1
ORDER BY p.risk_id, p.order_index;

-- ----------------------------------------------------------- user state

-- name: ListUserSkillStates :many
SELECT * FROM user_skill_states WHERE user_id = $1;

-- name: GetUserSkillState :one
SELECT * FROM user_skill_states WHERE user_id = $1 AND skill_level_id = $2;

-- first_achieved_at, evidence and verification are only ever set once: the
-- COALESCEs keep them, and the monotonic trigger refuses anything else.
-- name: UpsertUserSkillState :exec
INSERT INTO user_skill_states AS u (
    user_id, skill_level_id, state, best_value, best_unit, best_at, stale_since,
    first_achieved_at, evidence_set_entry_id, verification, attempts_count, updated_at
) VALUES (
    @user_id, @skill_level_id, @state, @best_value, @best_unit, @best_at, @stale_since,
    @first_achieved_at, @evidence_set_entry_id, @verification, @attempts_count, now()
)
ON CONFLICT (user_id, skill_level_id) DO UPDATE SET
    state = EXCLUDED.state,
    best_value = EXCLUDED.best_value, best_unit = EXCLUDED.best_unit, best_at = EXCLUDED.best_at,
    stale_since = EXCLUDED.stale_since, attempts_count = EXCLUDED.attempts_count,
    first_achieved_at = COALESCE(u.first_achieved_at, EXCLUDED.first_achieved_at),
    evidence_set_entry_id = COALESCE(u.evidence_set_entry_id, EXCLUDED.evidence_set_entry_id),
    verification = CASE WHEN u.first_achieved_at IS NULL THEN EXCLUDED.verification ELSE u.verification END,
    updated_at = now()
WHERE (u.state, u.best_value, u.best_unit, u.stale_since, u.attempts_count)
      IS DISTINCT FROM
      (EXCLUDED.state, EXCLUDED.best_value, EXCLUDED.best_unit, EXCLUDED.stale_since, EXCLUDED.attempts_count);

-- name: InsertUnlockEvent :exec
INSERT INTO skill_unlock_events (
    id, user_id, skill_level_id, occurred_at, session_id, evidence_set_entry_id,
    verification, criteria_snapshot
) VALUES (
    @id, @user_id, @skill_level_id, @occurred_at, @session_id, @evidence_set_entry_id,
    @verification, @criteria_snapshot
)
ON CONFLICT (user_id, skill_level_id) DO NOTHING;

-- name: ListUnlockEventsForSession :many
SELECT * FROM skill_unlock_events
WHERE user_id = @user_id AND session_id = @session_id
ORDER BY occurred_at, id;

-- name: GetUnlockEvent :one
SELECT * FROM skill_unlock_events WHERE user_id = $1 AND skill_level_id = $2;

-- ----------------------------------------------------------------- history

-- Every performed element of the named exercises, from completed sessions.
-- Planned sets, deleted rows and abandoned or draft sessions are not evidence.
-- name: ObservationsForExercises :many
SELECT el.set_entry_id, el.session_id, e.slug AS exercise, el.measure,
       el.reps, el.hold_seconds, el.distance_m, el.load_kg, el.assistance_class,
       el.form_quality, el.failed, el.is_partial_rom, el.is_eccentric_only,
       COALESCE(se.completed_at, ws.started_at)::timestamptz AS performed_at
FROM set_elements el
JOIN exercises e ON e.id = el.exercise_id
JOIN set_entries se ON se.id = el.set_entry_id
JOIN workout_sessions ws ON ws.id = el.session_id
WHERE el.user_id = @user_id
  AND e.slug = ANY(@slugs::text[])
  AND el.deleted_at IS NULL AND se.deleted_at IS NULL AND ws.deleted_at IS NULL
  AND NOT se.is_planned
  AND ws.status = 'completed';

-- name: SessionPerformance :one
SELECT
    count(*) FILTER (WHERE el.reps IS NOT NULL OR el.hold_seconds IS NOT NULL
                        OR el.distance_m IS NOT NULL OR el.measure = 'none')::int AS performed_elements,
    COALESCE(array_agg(el.form_quality) FILTER (WHERE el.form_quality IS NOT NULL), '{}')::smallint[] AS form_ratings
FROM set_elements el
JOIN set_entries se ON se.id = el.set_entry_id
WHERE el.session_id = @session_id AND el.user_id = @user_id
  AND el.deleted_at IS NULL AND se.deleted_at IS NULL AND NOT se.is_planned;

-- name: CompletedSessionsOnDay :one
SELECT count(*)::int FROM workout_sessions
WHERE user_id = @user_id AND local_date = @local_date AND status = 'completed'
  AND deleted_at IS NULL AND id <> @except_id;

-- ------------------------------------------------------------------- xp

-- name: InsertXPEvent :one
INSERT INTO user_xp_events (id, user_id, source, amount, ref_type, ref_id, occurred_at)
VALUES (@id, @user_id, @source, @amount, @ref_type, @ref_id, @occurred_at)
ON CONFLICT (user_id, source, ref_type, ref_id) DO NOTHING
RETURNING amount;

-- name: TotalXP :one
SELECT COALESCE(sum(amount), 0)::int FROM user_xp_events WHERE user_id = $1;

-- --------------------------------------------------------------- streaks

-- name: MarkTrainingDay :exec
INSERT INTO user_training_days AS d (user_id, local_date, session_count, had_session, planned_rest)
VALUES (@user_id, @local_date, 1, @had_session, @planned_rest)
ON CONFLICT (user_id, local_date) DO UPDATE SET
    session_count = d.session_count + 1,
    had_session = d.had_session OR EXCLUDED.had_session,
    planned_rest = d.planned_rest OR EXCLUDED.planned_rest;

-- Days that count on their own merit. Freeze days are not included: which
-- days a freeze bridges is recomputed from these every time.
-- name: CountedDays :many
SELECT local_date FROM user_training_days
WHERE user_id = $1 AND (had_session OR planned_rest OR deload)
ORDER BY local_date;

-- Freezes are recomputed from scratch: clear the old ones, then mark the
-- days the current walk bridged.
-- name: ClearFreezes :exec
UPDATE user_training_days SET freeze_used = false WHERE user_id = $1 AND freeze_used;

-- name: DeleteEmptyFreezeDays :exec
DELETE FROM user_training_days
WHERE user_id = $1 AND NOT (had_session OR planned_rest OR deload OR freeze_used);

-- name: MarkFreezeDays :exec
INSERT INTO user_training_days AS d (user_id, local_date, freeze_used)
SELECT @user_id, unnest(@days::date[]), true
ON CONFLICT (user_id, local_date) DO UPDATE SET freeze_used = true;

-- name: UpsertStreak :exec
INSERT INTO user_streaks AS s (user_id, current_days, longest_days, last_counted_date, freeze_credits, freeze_credits_updated_at, updated_at)
VALUES (@user_id, @current_days, @longest_days, @last_counted_date, @freeze_credits, now(), now())
ON CONFLICT (user_id) DO UPDATE SET
    current_days = EXCLUDED.current_days, longest_days = EXCLUDED.longest_days,
    last_counted_date = EXCLUDED.last_counted_date,
    freeze_credits = EXCLUDED.freeze_credits,
    freeze_credits_updated_at = CASE WHEN s.freeze_credits <> EXCLUDED.freeze_credits THEN now() ELSE s.freeze_credits_updated_at END,
    updated_at = now();

-- name: GetStreak :one
SELECT * FROM user_streaks WHERE user_id = $1;

-- name: LockUserProgress :exec
SELECT pg_advisory_xact_lock(hashtextextended(@user_key::text, 0));
