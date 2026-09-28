-- The training planner: snapshot parts, plans and the decision log.
-- internal/store/planning.go assembles them; see ADR 0013.

-- Serialises the planner calls of one user for the rest of the
-- transaction. The first key names the planner, so other advisory locks
-- cannot collide with it.
-- name: LockPlanner :exec
SELECT pg_advisory_xact_lock(1885433198, hashtext(sqlc.arg(user_id)::uuid::text));

-- ----------------------------------------------------------------- profile

-- name: GetTrainingProfile :one
SELECT * FROM user_training_profiles WHERE user_id = $1;

-- name: UpsertTrainingProfile :exec
INSERT INTO user_training_profiles (
    user_id, birth_year, sessions_per_week, session_minutes, equipment, bodyweight_kg,
    training_level, training_months, last_regular_training, health_data_consent,
    disclaimer_ack, onboarded_at, preferred_days, mobility, max_added_load_kg, smallest_plate_kg
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (user_id) DO UPDATE SET
    birth_year = EXCLUDED.birth_year,
    sessions_per_week = EXCLUDED.sessions_per_week,
    session_minutes = EXCLUDED.session_minutes,
    equipment = EXCLUDED.equipment,
    bodyweight_kg = EXCLUDED.bodyweight_kg,
    training_level = EXCLUDED.training_level,
    training_months = EXCLUDED.training_months,
    last_regular_training = EXCLUDED.last_regular_training,
    health_data_consent = EXCLUDED.health_data_consent,
    disclaimer_ack = EXCLUDED.disclaimer_ack,
    onboarded_at = EXCLUDED.onboarded_at,
    preferred_days = EXCLUDED.preferred_days,
    mobility = EXCLUDED.mobility,
    max_added_load_kg = EXCLUDED.max_added_load_kg,
    smallest_plate_kg = EXCLUDED.smallest_plate_kg,
    updated_at = now();

-- ------------------------------------------------------------------- goals

-- name: ListActiveGoals :many
SELECT * FROM user_goals WHERE user_id = $1 AND status = 'active' ORDER BY priority;

-- name: InsertGoal :exec
INSERT INTO user_goals (id, user_id, skill, target_level, priority, target_date)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateGoal :exec
UPDATE user_goals SET priority = $3, target_date = $4, updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: DropGoal :exec
UPDATE user_goals SET status = 'dropped', priority = NULL, updated_at = now()
WHERE id = $1 AND user_id = $2;

-- --------------------------------------------------------------- screening

-- name: GetScreening :one
SELECT * FROM user_screening WHERE user_id = $1;

-- name: UpsertScreening :exec
INSERT INTO user_screening (user_id, exertion_symptoms, any_yes, cleared)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    exertion_symptoms = EXCLUDED.exertion_symptoms,
    any_yes = EXCLUDED.any_yes,
    cleared = EXCLUDED.cleared,
    updated_at = now();

-- name: DeleteScreening :exec
DELETE FROM user_screening WHERE user_id = $1;

-- ------------------------------------------------------------- constraints

-- name: ListActiveConstraints :many
SELECT * FROM planning_constraints WHERE user_id = $1 AND cleared_at IS NULL ORDER BY position;

-- name: NextConstraintPosition :one
SELECT coalesce(max(position) + 1, 0)::int AS next FROM planning_constraints WHERE user_id = $1;

-- name: InsertConstraint :exec
INSERT INTO planning_constraints (id, user_id, kind, region, created_at, position)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ClearConstraint :exec
UPDATE planning_constraints SET cleared_at = now() WHERE id = $1 AND user_id = $2;

-- ----------------------------------------------------------------- regions

-- name: ListRegionStatus :many
SELECT * FROM user_region_status WHERE user_id = $1 ORDER BY region;

-- name: UpsertRegionStatus :exec
INSERT INTO user_region_status (
    user_id, region, state, entered_via, since, start_fraction, step, step_since, step_sessions,
    reference, prior_injury, complaint, complaint_at, restrictions, breaches, pain_deload_to,
    rest_until, hold_at_ref, referral
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
ON CONFLICT (user_id, region) DO UPDATE SET
    state = EXCLUDED.state,
    entered_via = EXCLUDED.entered_via,
    since = EXCLUDED.since,
    start_fraction = EXCLUDED.start_fraction,
    step = EXCLUDED.step,
    step_since = EXCLUDED.step_since,
    step_sessions = EXCLUDED.step_sessions,
    reference = EXCLUDED.reference,
    prior_injury = EXCLUDED.prior_injury,
    complaint = EXCLUDED.complaint,
    complaint_at = EXCLUDED.complaint_at,
    restrictions = EXCLUDED.restrictions,
    breaches = EXCLUDED.breaches,
    pain_deload_to = EXCLUDED.pain_deload_to,
    rest_until = EXCLUDED.rest_until,
    hold_at_ref = EXCLUDED.hold_at_ref,
    referral = EXCLUDED.referral,
    updated_at = now();

-- name: DeleteRegionStatus :exec
DELETE FROM user_region_status WHERE user_id = $1 AND region = $2;

-- ------------------------------------------------------------ pain reports

-- name: ListPainReports :many
SELECT * FROM user_pain_reports WHERE user_id = $1 ORDER BY position;

-- A withdrawn consent deletes the reports (spec §4.9); keep holds the ones
-- the snapshot still has.
-- name: DeletePainReportsExcept :exec
DELETE FROM user_pain_reports WHERE user_id = @user_id AND NOT (id = ANY(@keep::uuid[]));

-- name: NextPainPosition :one
SELECT coalesce(max(position) + 1, 0)::int AS next FROM user_pain_reports WHERE user_id = $1;

-- A report the client already wrote keeps its row.
-- name: InsertPainReport :execrows
INSERT INTO user_pain_reports (
    id, user_id, region, timepoint, nrs, lasted_over_1h, persisted_over_15min, sudden_sharp,
    session_id, reported_at, position
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO NOTHING;

-- -------------------------------------------------------------- capacities

-- name: ListCapacityEstimates :many
SELECT * FROM user_capacity_estimates WHERE user_id = $1 ORDER BY exercise, assistance;

-- name: UpsertCapacityEstimate :exec
INSERT INTO user_capacity_estimates (
    user_id, exercise, assistance, mu, sigma, origin, observed_at, seen_at, n_obs, pending
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id, exercise, assistance) DO UPDATE SET
    mu = EXCLUDED.mu,
    sigma = EXCLUDED.sigma,
    origin = EXCLUDED.origin,
    observed_at = EXCLUDED.observed_at,
    seen_at = EXCLUDED.seen_at,
    n_obs = EXCLUDED.n_obs,
    pending = EXCLUDED.pending,
    updated_at = now();

-- name: DeleteCapacityEstimate :exec
DELETE FROM user_capacity_estimates WHERE user_id = $1 AND exercise = $2 AND assistance = $3;

-- ----------------------------------------------------------------- ladders

-- name: ListLadderStates :many
SELECT * FROM user_ladder_states WHERE user_id = $1 ORDER BY skill;

-- name: UpsertLadderState :exec
INSERT INTO user_ladder_states (
    user_id, skill, rung, status, since, exposures, claimed, cap_rung, probe_offer, rep_target,
    load_kg, ecc_s, last_up, cap_to, cap_until
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (user_id, skill) DO UPDATE SET
    rung = EXCLUDED.rung,
    status = EXCLUDED.status,
    since = EXCLUDED.since,
    exposures = EXCLUDED.exposures,
    claimed = EXCLUDED.claimed,
    cap_rung = EXCLUDED.cap_rung,
    probe_offer = EXCLUDED.probe_offer,
    rep_target = EXCLUDED.rep_target,
    load_kg = EXCLUDED.load_kg,
    ecc_s = EXCLUDED.ecc_s,
    last_up = EXCLUDED.last_up,
    cap_to = EXCLUDED.cap_to,
    cap_until = EXCLUDED.cap_until,
    updated_at = now();

-- name: DeleteLadderState :exec
DELETE FROM user_ladder_states WHERE user_id = $1 AND skill = $2;

-- ----------------------------------------------------- phase and bookkeeping

-- name: GetPlannerState :one
SELECT * FROM user_planner_states WHERE user_id = $1;

-- name: UpsertPlannerState :exec
INSERT INTO user_planner_states (
    user_id, meso_start, last_deload, deload_week, deload_kind, deload_next, calibrate,
    headroom, entry, unlocked
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id) DO UPDATE SET
    meso_start = EXCLUDED.meso_start,
    last_deload = EXCLUDED.last_deload,
    deload_week = EXCLUDED.deload_week,
    deload_kind = EXCLUDED.deload_kind,
    deload_next = EXCLUDED.deload_next,
    calibrate = EXCLUDED.calibrate,
    headroom = EXCLUDED.headroom,
    entry = EXCLUDED.entry,
    unlocked = EXCLUDED.unlocked,
    updated_at = now();

-- ------------------------------------------------------------------ breaks

-- name: GetTrainingBreak :one
SELECT * FROM user_training_breaks WHERE user_id = $1;

-- name: UpsertTrainingBreak :exec
INSERT INTO user_training_breaks (
    user_id, days, straight_days, since, step, step_since, step_sessions, logged, reference, base
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id) DO UPDATE SET
    days = EXCLUDED.days,
    straight_days = EXCLUDED.straight_days,
    since = EXCLUDED.since,
    step = EXCLUDED.step,
    step_since = EXCLUDED.step_since,
    step_sessions = EXCLUDED.step_sessions,
    logged = EXCLUDED.logged,
    reference = EXCLUDED.reference,
    base = EXCLUDED.base,
    updated_at = now();

-- name: DeleteTrainingBreak :exec
DELETE FROM user_training_breaks WHERE user_id = $1;

-- ----------------------------------------------------------------- history

-- name: ListPlannerSessions :many
SELECT * FROM planner_sessions WHERE user_id = $1 ORDER BY local_date, position;

-- name: NextPlannerSessionPosition :one
SELECT coalesce(max(position) + 1, 0)::int AS next FROM planner_sessions WHERE user_id = $1;

-- name: InsertPlannerSession :exec
INSERT INTO planner_sessions (id, user_id, local_date, deload, fatigue, sets, position)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- ------------------------------------------------------------------- plans

-- name: GetActivePlan :one
SELECT * FROM training_plans WHERE user_id = $1 AND week_start = $2 AND status = 'active';

-- name: SupersedeActivePlan :exec
UPDATE training_plans SET status = 'superseded'
WHERE user_id = $1 AND week_start = $2 AND status = 'active';

-- name: InsertPlan :exec
INSERT INTO training_plans (id, user_id, week_start, ruleset_version, input_hash, payload)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertPlannedSession :exec
INSERT INTO planned_sessions (
    id, user_id, plan_id, order_index, scheduled_date, kind, est_minutes, status, workout_session_id
) VALUES (
    @id, @user_id, @plan_id, @order_index, @scheduled_date, @kind, @est_minutes, @status, @workout_session_id
);

-- name: ListPlannedSessions :many
SELECT * FROM planned_sessions WHERE plan_id = $1 AND user_id = $2 ORDER BY order_index;

-- Whether each session of a plan was started, and its log session. A
-- session whose log session was deleted counts as planned again.
-- name: ListPlannedSessionStates :many
SELECT s.order_index, s.status, s.workout_session_id,
       COALESCE(ws.deleted_at IS NULL, false)::boolean AS session_live
FROM planned_sessions s
LEFT JOIN workout_sessions ws ON ws.id = s.workout_session_id AND ws.user_id = s.user_id
WHERE s.plan_id = @plan_id AND s.user_id = @user_id
ORDER BY s.order_index;

-- The planner's exercises in the log's catalogue.
-- name: ExerciseIDsBySlug :many
SELECT slug, id FROM exercises WHERE slug = ANY(@slugs::text[]) AND status <> 'retired';

-- The planned session to start, locked for the start (spec §10.2).
-- name: LockActivePlannedSession :one
SELECT s.* FROM planned_sessions s
JOIN training_plans p ON p.id = s.plan_id AND p.user_id = s.user_id
WHERE s.id = $1 AND s.user_id = $2 AND p.status = 'active'
FOR UPDATE OF s;

-- name: MarkPlannedSessionStarted :exec
UPDATE planned_sessions SET status = 'started', workout_session_id = @workout_session_id
WHERE id = @id AND user_id = @user_id;

-- name: InsertPlannedWorkoutSession :one
INSERT INTO workout_sessions (
    id, user_id, started_at, timezone, local_date, planned_session_id, client_id, updated_at
) VALUES (
    @id, @user_id, @started_at, @timezone, @local_date, @planned_session_id, @client_id, @updated_at
)
ON CONFLICT (id) DO NOTHING
RETURNING *;

-- name: InsertPlannedSetEntry :exec
INSERT INTO set_entries (
    id, user_id, session_id, block_id, order_index, round_index, kind, is_planned,
    rest_after_planned_s, rir, planned_item_id, client_id, updated_at
) VALUES (
    @id, @user_id, @session_id, @block_id, @order_index, @round_index, @kind, true,
    @rest_after_planned_s, @rir, @planned_item_id, @client_id, @updated_at
);

-- The live sets of a started draft in log order, for the reconciliation
-- with a new plan (ADR 0017). Open is a planned set not yet performed.
-- name: ListDraftSets :many
SELECT se.id, se.block_id, se.planned_item_id,
       (se.is_planned AND se.completed_at IS NULL)::boolean AS open
FROM set_entries se
JOIN session_blocks b ON b.id = se.block_id AND b.user_id = se.user_id
WHERE se.session_id = @session_id AND se.user_id = @user_id
  AND se.deleted_at IS NULL AND b.deleted_at IS NULL
ORDER BY b.order_index, se.order_index;

-- An open planned set follows its item into a new plan. updated_at stays:
-- it is the athlete's clock, and the set's values did not change.
-- name: RelinkPlannedSet :exec
UPDATE set_entries SET planned_item_id = @planned_item_id
WHERE id = @id AND user_id = @user_id AND session_id = @session_id
  AND deleted_at IS NULL AND is_planned AND completed_at IS NULL;

-- name: NextSetOrder :one
SELECT COALESCE(MAX(order_index) + 1, 0)::int FROM set_entries
WHERE block_id = @block_id AND user_id = @user_id AND deleted_at IS NULL;

-- name: NextBlockOrder :one
SELECT COALESCE(MAX(order_index) + 1, 0)::int FROM session_blocks
WHERE session_id = @session_id AND user_id = @user_id AND deleted_at IS NULL;

-- name: CountLiveSetsOfBlock :one
SELECT count(*)::int FROM set_entries
WHERE block_id = @block_id AND user_id = @user_id AND deleted_at IS NULL;

-- A completed session as the planner reads it (ADR 0018): deload when it
-- was started from a deload session of the plan.
-- name: GetCompletedSessionForPlanner :one
SELECT ws.id, ws.local_date, ws.perceived_fatigue, ws.is_rest_day,
       COALESCE(ps.kind = 'deload', false)::boolean AS deload
FROM workout_sessions ws
LEFT JOIN planned_sessions ps ON ps.id = ws.planned_session_id AND ps.user_id = ws.user_id
WHERE ws.id = @id AND ws.user_id = @user_id AND ws.deleted_at IS NULL AND ws.status = 'completed';

-- The performed elements of a session in the order performed.
-- name: ListPerformedElements :many
SELECT el.id, e.slug AS exercise, se.kind, el.measure, el.reps, el.hold_seconds, el.load_kg,
       se.rir, el.form_quality, el.failed, el.is_partial_rom, el.is_eccentric_only,
       COALESCE(a.type, 'none')::text AS assistance
FROM set_elements el
JOIN set_entries se ON se.id = el.set_entry_id AND se.user_id = el.user_id
JOIN session_blocks b ON b.id = se.block_id AND b.user_id = se.user_id
JOIN exercises e ON e.id = el.exercise_id
LEFT JOIN set_element_assistance a ON a.set_element_id = el.id AND a.user_id = el.user_id AND a.deleted_at IS NULL
WHERE el.session_id = @session_id AND el.user_id = @user_id AND NOT se.is_planned
  AND el.deleted_at IS NULL AND se.deleted_at IS NULL AND b.deleted_at IS NULL
ORDER BY se.completed_at NULLS LAST, b.order_index, se.order_index, el.order_index;

-- name: MarkPlannedSessionsCompleted :exec
UPDATE planned_sessions SET status = 'completed'
WHERE workout_session_id = @workout_session_id AND user_id = @user_id;

-- A session completed in a deload marks its day (spec §10.2, ADR 0003).
-- name: MarkDeloadDay :exec
UPDATE user_training_days d SET deload = true
FROM workout_sessions ws
WHERE ws.id = @session_id AND ws.user_id = @user_id
  AND d.user_id = ws.user_id AND d.local_date = ws.local_date;

-- Completed sessions the planner has not applied yet, oldest first: the
-- catch-up when the adaptation after a completion failed (spec §6.1). Only
-- sessions from the day of the onboarding on; before it, none.
-- name: ListPendingCompletions :many
SELECT ws.id FROM workout_sessions ws
JOIN user_training_profiles p ON p.user_id = ws.user_id
WHERE ws.user_id = @user_id AND ws.status = 'completed' AND ws.deleted_at IS NULL AND NOT ws.is_rest_day
  AND ws.completed_at >= p.onboarded_at
  AND NOT EXISTS (SELECT 1 FROM plan_decisions d
                  WHERE d.user_id = ws.user_id AND d.trigger = 'session_completed' AND d.source_id = ws.id::text)
ORDER BY ws.completed_at, ws.id
LIMIT @max_rows;

-- name: GetActivePlannedSession :one
SELECT p.id AS plan_id, p.payload, s.order_index
FROM planned_sessions s
JOIN training_plans p ON p.id = s.plan_id AND p.user_id = s.user_id
WHERE s.id = $1 AND s.user_id = $2 AND p.status = 'active';

-- --------------------------------------------------------------- decisions

-- name: GetDecision :one
SELECT * FROM plan_decisions WHERE user_id = $1 AND trigger = $2 AND source_id = $3;

-- name: InsertDecision :exec
INSERT INTO plan_decisions (id, user_id, trigger, source_id, occurred_at, changes)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListDecisions :many
SELECT * FROM plan_decisions
WHERE user_id = @user_id
  AND (sqlc.narg('cursor_at')::timestamptz IS NULL
       OR (occurred_at, id) < (sqlc.narg('cursor_at'), sqlc.narg('cursor_id')::uuid))
ORDER BY occurred_at DESC, id DESC
LIMIT @page_limit;
