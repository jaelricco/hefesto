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
INSERT INTO planned_sessions (id, user_id, plan_id, order_index, scheduled_date, kind, est_minutes)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListPlannedSessions :many
SELECT * FROM planned_sessions WHERE plan_id = $1 AND user_id = $2 ORDER BY order_index;

-- --------------------------------------------------------------- decisions

-- name: DecisionSeen :one
SELECT EXISTS (
    SELECT 1 FROM plan_decisions WHERE user_id = $1 AND trigger = $2 AND source_id = $3
) AS seen;

-- name: InsertDecision :exec
INSERT INTO plan_decisions (id, user_id, trigger, source_id, occurred_at, changes)
VALUES ($1, $2, $3, $4, now(), $5);

-- name: ListDecisions :many
SELECT * FROM plan_decisions WHERE user_id = $1 ORDER BY occurred_at, id;
