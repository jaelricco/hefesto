-- The training planner's state, plans and decision log.
-- See docs/algorithm/spec.md §4.9 and docs/adr/0013-planner-persistence.md.
--
-- The core works on one snapshot per user (internal/domain/planning); these
-- tables hold it split by concern, so health data can be deleted on its own
-- and the API can read single parts. Skills, levels, exercises and regions
-- are knowledge-base slugs (content/training/), not foreign keys: the
-- planner's knowledge base is not in the content tables yet (ADR 0013 §2).
--
-- Calendar days are kept as timestamptz at midnight UTC, as the core keeps
-- them, and computed numbers as double precision, so a snapshot reads back
-- exactly as it was written (ADR 0013 §4). Nothing here is synced yet; the
-- tables clients will write (profile, goals, pain reports) get their sync
-- columns with the endpoints.

-- +goose Up
CREATE TABLE user_training_profiles (
    user_id               uuid             PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    birth_year            smallint         NOT NULL
        CONSTRAINT user_training_profiles_birth_year_ck CHECK (birth_year BETWEEN 1900 AND 2100),
    sessions_per_week     smallint         NOT NULL
        CONSTRAINT user_training_profiles_sessions_ck CHECK (sessions_per_week BETWEEN 1 AND 7),
    session_minutes       smallint         NOT NULL
        CONSTRAINT user_training_profiles_minutes_ck CHECK (session_minutes IN (20, 30, 45, 60, 75, 90)),
    -- Expanded equipment keys (onboarding.md §3.4), in the order the core
    -- keeps them.
    equipment             text[]           NOT NULL
        CONSTRAINT user_training_profiles_equipment_ck
        CHECK (array_to_string(equipment, ',') ~ '^([a-z][a-z0-9_]*(,[a-z][a-z0-9_]*)*)?$'),
    bodyweight_kg         double precision NOT NULL
        CONSTRAINT user_training_profiles_bodyweight_ck CHECK (bodyweight_kg > 0),
    training_level        text             NOT NULL
        CONSTRAINT user_training_profiles_level_ck
        CHECK (training_level IN ('sedentary','recreational','trained','highly_trained')),
    -- Lower bound of the training age class, advanced by the logs (PAR-S-20).
    training_months       double precision NOT NULL
        CONSTRAINT user_training_profiles_months_ck CHECK (training_months >= 0),
    last_regular_training text                 NULL
        CONSTRAINT user_training_profiles_last_regular_ck
        CHECK (last_regular_training IN ('current_or_lt_3_weeks','3_to_6_weeks','7_to_16_weeks',
                                         '17_to_26_weeks','gt_26_weeks','never')),
    health_data_consent   boolean          NOT NULL,
    disclaimer_ack        boolean          NOT NULL,
    onboarded_at          timestamptz      NOT NULL,
    -- ISO weekdays, 1 = Monday.
    preferred_days        smallint[]       NOT NULL DEFAULT '{}'
        CONSTRAINT user_training_profiles_days_ck
        CHECK (preferred_days <@ ARRAY[1,2,3,4,5,6,7]::smallint[]),
    mobility              jsonb                NULL
        CONSTRAINT user_training_profiles_mobility_ck CHECK (mobility IS NULL OR jsonb_typeof(mobility) = 'object'),
    max_added_load_kg     double precision NOT NULL DEFAULT 0
        CONSTRAINT user_training_profiles_max_load_ck CHECK (max_added_load_kg >= 0),
    smallest_plate_kg     double precision NOT NULL DEFAULT 0
        CONSTRAINT user_training_profiles_plate_ck CHECK (smallest_plate_kg >= 0),
    created_at            timestamptz      NOT NULL DEFAULT now(),
    updated_at            timestamptz      NOT NULL DEFAULT now()
);

CREATE TABLE user_goals (
    id           uuid        PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill        text        NOT NULL
        CONSTRAINT user_goals_skill_ck CHECK (skill ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    target_level text        NOT NULL
        CONSTRAINT user_goals_level_ck CHECK (target_level ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    -- Only active goals have a priority, and no two share one.
    priority     smallint        NULL
        CONSTRAINT user_goals_priority_ck CHECK (priority BETWEEN 1 AND 3),
    target_date  timestamptz     NULL,
    status       text        NOT NULL DEFAULT 'active'
        CONSTRAINT user_goals_status_ck CHECK (status IN ('active','achieved','dropped')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_goals_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT user_goals_active_priority_ck CHECK ((status = 'active') = (priority IS NOT NULL)),
    CONSTRAINT user_goals_priority_uk UNIQUE (user_id, priority) DEFERRABLE INITIALLY IMMEDIATE
);
CREATE INDEX user_goals_user_idx ON user_goals (user_id) WHERE status = 'active';

-- Health data (onboarding.md §3.7): deleted with the consent.
CREATE TABLE user_screening (
    user_id           uuid        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    exertion_symptoms boolean     NOT NULL,
    any_yes           boolean     NOT NULL,
    cleared           boolean     NOT NULL,
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- Safety constraints without answers or values (ENT-S-7). They are not
-- health data and survive a withdrawn consent; a lifted one keeps its row
-- with cleared_at.
CREATE TABLE planning_constraints (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind        text        NOT NULL
        CONSTRAINT planning_constraints_kind_ck
        CHECK (kind IN ('plan_stopped','region_locked','region_excluded')),
    region      text            NULL
        CONSTRAINT planning_constraints_region_ck CHECK (region ~ '^[a-z][a-z0-9_]*$'),
    created_at  timestamptz NOT NULL,
    position    int         NOT NULL
        CONSTRAINT planning_constraints_position_ck CHECK (position >= 0),
    cleared_at  timestamptz     NULL,
    -- A stop may name the region whose red flag caused it; a lock or an
    -- exclusion always does.
    CONSTRAINT planning_constraints_region_kind_ck CHECK (kind = 'plan_stopped' OR region IS NOT NULL)
);
CREATE INDEX planning_constraints_user_idx ON planning_constraints (user_id, position) WHERE cleared_at IS NULL;

-- Health data: the tolerance state of each body-map region (spec §4.6).
CREATE TABLE user_region_status (
    user_id        uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    region         text             NOT NULL
        CONSTRAINT user_region_status_region_ck CHECK (region ~ '^[a-z][a-z0-9_]*$'),
    state          text             NOT NULL
        CONSTRAINT user_region_status_state_ck
        CHECK (state IN ('normal','locked','rtt_0','rtt_1','rtt_2','rtt_3','rtt_4','rtt_5')),
    entered_via    text                 NULL
        CONSTRAINT user_region_status_via_ck
        CHECK (entered_via IN ('onboarding','pain_report','red_flag','red_flags_negative','break','clearance','consent')),
    since          timestamptz          NULL,
    start_fraction double precision NOT NULL DEFAULT 0
        CONSTRAINT user_region_status_fraction_ck CHECK (start_fraction BETWEEN 0 AND 1),
    step           smallint         NOT NULL DEFAULT 0
        CONSTRAINT user_region_status_step_ck CHECK (step BETWEEN 0 AND 3),
    step_since     timestamptz          NULL,
    step_sessions  int              NOT NULL DEFAULT 0
        CONSTRAINT user_region_status_step_sessions_ck CHECK (step_sessions >= 0),
    -- Logged pre-complaint weekly load per load account (spec §7.2).
    reference      jsonb                NULL
        CONSTRAINT user_region_status_reference_ck CHECK (reference IS NULL OR jsonb_typeof(reference) = 'object'),
    prior_injury   boolean          NOT NULL DEFAULT false,
    complaint      boolean          NOT NULL DEFAULT false,
    complaint_at   timestamptz          NULL,
    restrictions   text[]           NOT NULL DEFAULT '{}',
    breaches       timestamptz[]    NOT NULL DEFAULT '{}',
    pain_deload_to timestamptz          NULL,
    rest_until     timestamptz          NULL,
    hold_at_ref    boolean          NOT NULL DEFAULT false,
    referral       text                 NULL
        CONSTRAINT user_region_status_referral_ck CHECK (referral IN ('soft','advise')),
    updated_at     timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, region)
);

-- Health data: pain reports (spec §8.6). The ID is the client's or the
-- planner's; session_id has no foreign key because a report written offline
-- may arrive before its session.
CREATE TABLE user_pain_reports (
    id                   uuid             PRIMARY KEY,
    user_id              uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    region               text             NOT NULL
        CONSTRAINT user_pain_reports_region_ck CHECK (region ~ '^[a-z][a-z0-9_]*$'),
    timepoint            text             NOT NULL
        CONSTRAINT user_pain_reports_timepoint_ck
        CHECK (timepoint IN ('before_session','warmup','during','after','next_morning','daily')),
    nrs                  double precision NOT NULL
        CONSTRAINT user_pain_reports_nrs_ck CHECK (nrs BETWEEN 0 AND 10),
    lasted_over_1h       boolean          NOT NULL DEFAULT false,
    persisted_over_15min boolean          NOT NULL DEFAULT false,
    sudden_sharp         boolean          NOT NULL DEFAULT false,
    session_id           uuid                 NULL,
    reported_at          timestamptz      NOT NULL,
    -- Order in which the planner received the reports.
    position             int              NOT NULL
        CONSTRAINT user_pain_reports_position_ck CHECK (position >= 0),
    created_at           timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT user_pain_reports_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT user_pain_reports_position_uk UNIQUE (user_id, position)
);

CREATE TABLE user_capacity_estimates (
    user_id     uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise    text             NOT NULL
        CONSTRAINT user_capacity_estimates_exercise_ck CHECK (exercise ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    assistance  text             NOT NULL
        CONSTRAINT user_capacity_estimates_assistance_ck CHECK (assistance IN ('none','band')),
    mu          double precision NOT NULL,
    sigma       double precision NOT NULL
        CONSTRAINT user_capacity_estimates_sigma_ck CHECK (sigma >= 0),
    origin      text             NOT NULL
        CONSTRAINT user_capacity_estimates_origin_ck
        CHECK (origin IN ('self_report','test','log','derived')),
    observed_at timestamptz          NULL,
    -- Last lower bound that confirmed the estimate (spec §4.3).
    seen_at     timestamptz          NULL,
    n_obs       int              NOT NULL
        CONSTRAINT user_capacity_estimates_n_ck CHECK (n_obs >= 0),
    -- Last contradicting observation (ADAPT-03).
    pending     double precision     NULL,
    updated_at  timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, exercise, assistance)
);

CREATE TABLE user_ladder_states (
    user_id     uuid             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill       text             NOT NULL
        CONSTRAINT user_ladder_states_skill_ck CHECK (skill ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    rung        text                 NULL
        CONSTRAINT user_ladder_states_rung_ck CHECK (rung ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    status      text                 NULL
        CONSTRAINT user_ladder_states_status_ck CHECK (status IN ('claimed','calibrated')),
    since       timestamptz          NULL,
    exposures   int              NOT NULL DEFAULT 0
        CONSTRAINT user_ladder_states_exposures_ck CHECK (exposures >= 0),
    claimed     text                 NULL
        CONSTRAINT user_ladder_states_claimed_ck CHECK (claimed ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    -- Highest rung during a break ramp (pre_break_level).
    cap_rung    text                 NULL
        CONSTRAINT user_ladder_states_cap_rung_ck CHECK (cap_rung ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    probe_offer boolean          NOT NULL DEFAULT false,
    rep_target  double precision NOT NULL DEFAULT 0,
    load_kg     double precision NOT NULL DEFAULT 0
        CONSTRAINT user_ladder_states_load_ck CHECK (load_kg >= 0),
    ecc_s       double precision NOT NULL DEFAULT 0
        CONSTRAINT user_ladder_states_ecc_ck CHECK (ecc_s >= 0),
    last_up     timestamptz          NULL,
    -- The working rung is capped at cap_to until cap_until (pain deload).
    cap_to      text                 NULL
        CONSTRAINT user_ladder_states_cap_to_ck CHECK (cap_to ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    cap_until   timestamptz          NULL,
    updated_at  timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, skill)
);

-- The rest of the core's bookkeeping: mesocycle and deloads (spec §4.7),
-- the carried headroom (PAR-S-35), the entry-ramp accounts (LOAD-04b) and
-- the reached levels. Maps are jsonb; NULL is a map the core left nil.
CREATE TABLE user_planner_states (
    user_id     uuid        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    meso_start  timestamptz     NULL,
    last_deload timestamptz     NULL,
    deload_week timestamptz     NULL,
    deload_kind text            NULL
        CONSTRAINT user_planner_states_deload_kind_ck
        CHECK (deload_kind IN ('planned','stagnation','fatigue','max_build','pain')),
    deload_next text            NULL
        CONSTRAINT user_planner_states_deload_next_ck
        CHECK (deload_next IN ('planned','stagnation','fatigue','max_build','pain')),
    calibrate   text[]      NOT NULL DEFAULT '{}',
    headroom    jsonb           NULL
        CONSTRAINT user_planner_states_headroom_ck CHECK (headroom IS NULL OR jsonb_typeof(headroom) = 'object'),
    entry       jsonb           NULL
        CONSTRAINT user_planner_states_entry_ck CHECK (entry IS NULL OR jsonb_typeof(entry) = 'object'),
    unlocked    jsonb           NULL
        CONSTRAINT user_planner_states_unlocked_ck CHECK (unlocked IS NULL OR jsonb_typeof(unlocked) = 'object'),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- A return after a pause (spec §6.11); at most one per user.
CREATE TABLE user_training_breaks (
    user_id       uuid             PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    days          double precision NOT NULL
        CONSTRAINT user_training_breaks_days_ck CHECK (days >= 0),
    straight_days double precision NOT NULL
        CONSTRAINT user_training_breaks_straight_ck CHECK (straight_days >= 0),
    since         timestamptz          NULL,
    step          smallint         NOT NULL
        CONSTRAINT user_training_breaks_step_ck CHECK (step BETWEEN 0 AND 3),
    step_since    timestamptz          NULL,
    step_sessions int              NOT NULL
        CONSTRAINT user_training_breaks_sessions_ck CHECK (step_sessions >= 0),
    logged        boolean          NOT NULL,
    reference     jsonb                NULL
        CONSTRAINT user_training_breaks_reference_ck CHECK (reference IS NULL OR jsonb_typeof(reference) = 'object'),
    -- Week-1 target frozen at the onboarding (ENT-R-1).
    base          jsonb                NULL
        CONSTRAINT user_training_breaks_base_ck CHECK (base IS NULL OR jsonb_typeof(base) = 'object'),
    updated_at    timestamptz      NOT NULL DEFAULT now()
);

-- The completed sessions as the planner adapted to them, with exercises as
-- knowledge-base slugs. The session is the log's; until the planner's
-- exercises are content rows this is the planner's own copy (ADR 0013 §3).
CREATE TABLE planner_sessions (
    id          uuid             PRIMARY KEY,
    user_id     uuid             NOT NULL,
    local_date  timestamptz      NOT NULL,
    deload      boolean          NOT NULL DEFAULT false,
    fatigue     double precision     NULL
        CONSTRAINT planner_sessions_fatigue_ck CHECK (fatigue BETWEEN 1 AND 10),
    sets        jsonb            NOT NULL
        CONSTRAINT planner_sessions_sets_ck CHECK (jsonb_typeof(sets) = 'array'),
    -- Order of arrival; the history is sorted by local_date, then this.
    position    int              NOT NULL
        CONSTRAINT planner_sessions_position_ck CHECK (position >= 0),
    received_at timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT planner_sessions_position_uk UNIQUE (user_id, position),
    FOREIGN KEY (id, user_id) REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE
);
CREATE INDEX planner_sessions_user_idx ON planner_sessions (user_id, local_date, position);

-- Generated week plans. Every regeneration adds a plan and supersedes the
-- active one of its week (spec §11.3).
CREATE TABLE training_plans (
    id              uuid        PRIMARY KEY,
    user_id         uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start      date        NOT NULL
        CONSTRAINT training_plans_week_start_ck CHECK (extract(isodow FROM week_start) = 1),
    ruleset_version text        NOT NULL,
    input_hash      text        NOT NULL
        CONSTRAINT training_plans_input_hash_ck CHECK (input_hash ~ '^sha256:[0-9a-f]{64}$'),
    status          text        NOT NULL DEFAULT 'active'
        CONSTRAINT training_plans_status_ck CHECK (status IN ('active','superseded','expired')),
    -- The whole plan as the core returned it (spec §5.10).
    payload         jsonb       NOT NULL
        CONSTRAINT training_plans_payload_ck CHECK (jsonb_typeof(payload) = 'object'),
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT training_plans_id_user_uk UNIQUE (id, user_id)
);
CREATE UNIQUE INDEX training_plans_active_uk ON training_plans (user_id, week_start) WHERE status = 'active';

-- The addressable sessions of a plan: what the start endpoint and the log
-- refer to (spec §10.2). Their content is the plan payload's session at
-- order_index.
CREATE TABLE planned_sessions (
    id                 uuid             PRIMARY KEY,
    user_id            uuid             NOT NULL,
    plan_id            uuid             NOT NULL,
    order_index        int              NOT NULL
        CONSTRAINT planned_sessions_order_index_ck CHECK (order_index >= 0),
    scheduled_date     date             NOT NULL,
    kind               text             NOT NULL
        CONSTRAINT planned_sessions_kind_ck CHECK (kind IN ('full','light','deload')),
    est_minutes        double precision NOT NULL
        CONSTRAINT planned_sessions_minutes_ck CHECK (est_minutes >= 0),
    status             text             NOT NULL DEFAULT 'planned'
        CONSTRAINT planned_sessions_status_ck CHECK (status IN ('planned','started','completed','expired')),
    workout_session_id uuid                 NULL,
    CONSTRAINT planned_sessions_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT planned_sessions_order_uk UNIQUE (plan_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (plan_id, user_id) REFERENCES training_plans (id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (workout_session_id, user_id)
        REFERENCES workout_sessions (id, user_id) ON DELETE SET NULL (workout_session_id)
);
CREATE INDEX planned_sessions_user_date_idx ON planned_sessions (user_id, scheduled_date);

-- Every event the planner applied, with the changes the user sees (spec
-- §6.14, §9.4). One row per event; the trigger and source make events
-- idempotent.
CREATE TABLE plan_decisions (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger     text        NOT NULL
        CONSTRAINT plan_decisions_trigger_ck
        CHECK (trigger IN ('onboarding','session_completed','pain_report','red_flags',
                           'clearance','symptoms','week_start','consent')),
    source_id   text        NOT NULL
        CONSTRAINT plan_decisions_source_ck CHECK (source_id <> ''),
    occurred_at timestamptz NOT NULL,
    changes     jsonb       NOT NULL
        CONSTRAINT plan_decisions_changes_ck CHECK (jsonb_typeof(changes) = 'array'),
    CONSTRAINT plan_decisions_event_uk UNIQUE (user_id, trigger, source_id)
);
CREATE INDEX plan_decisions_user_time_idx ON plan_decisions (user_id, occurred_at DESC, id DESC);

-- Append-only, like skill_unlock_events; a cascade from deleting the user
-- reaches this trigger nested inside the FK's own trigger.
-- +goose StatementBegin
CREATE FUNCTION plan_decisions_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'plan_decisions is append-only (% refused)', TG_OP
        USING ERRCODE = 'check_violation', CONSTRAINT = 'plan_decisions_append_only';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER plan_decisions_append_only BEFORE UPDATE OR DELETE ON plan_decisions
    FOR EACH ROW EXECUTE FUNCTION plan_decisions_append_only();

-- +goose Down
DROP TABLE plan_decisions;
DROP FUNCTION plan_decisions_append_only();
DROP TABLE planned_sessions;
DROP TABLE training_plans;
DROP TABLE planner_sessions;
DROP TABLE user_training_breaks;
DROP TABLE user_planner_states;
DROP TABLE user_ladder_states;
DROP TABLE user_capacity_estimates;
DROP TABLE user_pain_reports;
DROP TABLE user_region_status;
DROP TABLE planning_constraints;
DROP TABLE user_screening;
DROP TABLE user_goals;
DROP TABLE user_training_profiles;
