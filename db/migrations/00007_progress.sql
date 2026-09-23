-- Progress: skill states, the unlock log, XP, streaks, badges, bests.
-- See docs/DDL_PROPOSAL.md §9 and ADR 0003.

-- +goose Up
CREATE TABLE user_skill_states (
    user_id               uuid         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_level_id        uuid         NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    state                 text         NOT NULL DEFAULT 'locked'
        CONSTRAINT user_skill_states_state_ck
        CHECK (state IN ('locked','available','in_progress','unlocked')),
    best_value            numeric(9,2)     NULL,
    best_unit             text             NULL
        CONSTRAINT user_skill_states_best_unit_ck
        CHECK (best_unit IS NULL OR best_unit IN ('reps','hold_seconds','distance_m')),
    best_at               timestamptz      NULL,
    -- Informational only; never changes `state` (ADR 0003 §6).
    stale_since           timestamptz      NULL,
    first_achieved_at     timestamptz      NULL,
    evidence_set_entry_id uuid             NULL,
    verification          text         NOT NULL DEFAULT 'auto'
        CONSTRAINT user_skill_states_verification_ck
        CHECK (verification IN ('auto','self_attested','coach')),
    attempts_count        int          NOT NULL DEFAULT 0
        CONSTRAINT user_skill_states_attempts_ck CHECK (attempts_count >= 0),
    updated_at            timestamptz  NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, skill_level_id),
    -- SET NULL on the evidence column only: nulling user_id as well would
    -- violate NOT NULL.
    FOREIGN KEY (evidence_set_entry_id, user_id)
        REFERENCES set_entries (id, user_id) ON DELETE SET NULL (evidence_set_entry_id),
    CONSTRAINT user_skill_states_unlocked_ck
        CHECK ((state = 'unlocked') = (first_achieved_at IS NOT NULL))
);
CREATE INDEX user_skill_states_user_idx ON user_skill_states (user_id, state);

-- Unlocks are never revoked: once unlocked, state and first_achieved_at are
-- frozen. The map records achievement, not current form.
-- +goose StatementBegin
CREATE FUNCTION user_skill_states_monotonic() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.state = 'unlocked' THEN
        IF NEW.state <> 'unlocked' THEN
            RAISE EXCEPTION 'unlocks are never revoked (user %, level %)', OLD.user_id, OLD.skill_level_id
                USING ERRCODE = 'check_violation', CONSTRAINT = 'user_skill_states_monotonic';
        END IF;
        IF NEW.first_achieved_at IS DISTINCT FROM OLD.first_achieved_at THEN
            RAISE EXCEPTION 'first_achieved_at is immutable once set (user %, level %)', OLD.user_id, OLD.skill_level_id
                USING ERRCODE = 'check_violation', CONSTRAINT = 'user_skill_states_monotonic';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER user_skill_states_monotonic BEFORE UPDATE ON user_skill_states
    FOR EACH ROW EXECUTE FUNCTION user_skill_states_monotonic();

CREATE TABLE skill_unlock_events (
    id                    uuid        PRIMARY KEY,
    user_id               uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_level_id        uuid        NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    occurred_at           timestamptz NOT NULL DEFAULT now(),
    -- No foreign keys: an immutable log cannot take ON DELETE SET NULL.
    session_id            uuid            NULL,
    evidence_set_entry_id uuid            NULL,
    verification          text        NOT NULL
        CONSTRAINT skill_unlock_events_verification_ck
        CHECK (verification IN ('auto','self_attested','coach')),
    criteria_snapshot     jsonb       NOT NULL,   -- the criteria as they read at unlock time
    CONSTRAINT skill_unlock_events_once_uk UNIQUE (user_id, skill_level_id)
);
CREATE INDEX skill_unlock_events_user_time_idx ON skill_unlock_events (user_id, occurred_at DESC);

-- Append-only. The one exception is a cascade from deleting the user (or the
-- content row), which reaches this trigger nested inside the FK's own trigger.
-- +goose StatementBegin
CREATE FUNCTION skill_unlock_events_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'skill_unlock_events is append-only (% refused)', TG_OP
        USING ERRCODE = 'check_violation', CONSTRAINT = 'skill_unlock_events_append_only';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER skill_unlock_events_append_only BEFORE UPDATE OR DELETE ON skill_unlock_events
    FOR EACH ROW EXECUTE FUNCTION skill_unlock_events_append_only();

CREATE TABLE user_xp_events (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source      text        NOT NULL
        CONSTRAINT user_xp_events_source_ck
        CHECK (source IN ('session_completed','skill_unlocked','badge_earned',
                          'streak_milestone','plan_adherence','deload_completed')),
    amount      int         NOT NULL
        CONSTRAINT user_xp_events_amount_ck CHECK (amount > 0),
    ref_type    text            NULL,
    ref_id      uuid            NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    -- Awarding the same thing twice is the bug this prevents.
    CONSTRAINT user_xp_events_once_uk UNIQUE (user_id, source, ref_type, ref_id)
);
CREATE INDEX user_xp_events_user_time_idx ON user_xp_events (user_id, occurred_at DESC);

-- One row per user per local day. Records the day's intent, not just activity,
-- which is what lets a planned rest day maintain a streak (ADR 0003 §1).
CREATE TABLE user_training_days (
    user_id           uuid     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    local_date        date     NOT NULL,
    session_count     smallint NOT NULL DEFAULT 0
        CONSTRAINT user_training_days_count_ck CHECK (session_count >= 0),
    had_session       boolean  NOT NULL DEFAULT false,
    planned_rest      boolean  NOT NULL DEFAULT false,
    deload            boolean  NOT NULL DEFAULT false,
    freeze_used       boolean  NOT NULL DEFAULT false,
    counts_for_streak boolean  NOT NULL
        GENERATED ALWAYS AS (had_session OR planned_rest OR deload OR freeze_used) STORED,
    PRIMARY KEY (user_id, local_date)
);

CREATE TABLE user_streaks (
    user_id                   uuid        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_days              int         NOT NULL DEFAULT 0
        CONSTRAINT user_streaks_current_ck CHECK (current_days >= 0),
    longest_days              int         NOT NULL DEFAULT 0
        CONSTRAINT user_streaks_longest_ck CHECK (longest_days >= 0),
    last_counted_date         date            NULL,
    freeze_credits            smallint    NOT NULL DEFAULT 2
        CONSTRAINT user_streaks_freeze_ck CHECK (freeze_credits >= 0),
    freeze_credits_updated_at timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE badges (
    id                 uuid     PRIMARY KEY,
    slug               text     NOT NULL UNIQUE,
    name               text     NOT NULL,
    description        text     NOT NULL DEFAULT '',
    tier               smallint NOT NULL DEFAULT 1,
    criteria           jsonb    NOT NULL DEFAULT '{}'::jsonb,
    content_version_id uuid     NOT NULL REFERENCES content_versions(id)
);

CREATE TABLE user_badges (
    user_id   uuid        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    badge_id  uuid        NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at timestamptz NOT NULL DEFAULT now(),
    ref_id    uuid            NULL,
    PRIMARY KEY (user_id, badge_id)
);

-- A cache with a single writer (session completion). Must stay rebuildable
-- from set_elements alone (deviation D-8).
CREATE TABLE user_exercise_bests (
    user_id                 uuid         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id             uuid         NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    measure                 text         NOT NULL
        CONSTRAINT user_exercise_bests_measure_ck
        CHECK (measure IN ('reps','hold_seconds','distance_m')),
    assistance_class        text         NOT NULL
        CONSTRAINT user_exercise_bests_assistance_class_ck
        CHECK (assistance_class IN ('unassisted','assisted','loaded')),
    best_value              numeric(9,2) NOT NULL,
    best_load_kg            numeric(6,2) NOT NULL DEFAULT 0,
    achieved_at             timestamptz  NOT NULL,
    evidence_set_element_id uuid             NULL,
    occurrences_30d         int          NOT NULL DEFAULT 0,
    updated_at              timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, exercise_id, measure, assistance_class)
);

-- +goose Down
DROP TABLE user_exercise_bests;
DROP TABLE user_badges;
DROP TABLE badges;
DROP TABLE user_streaks;
DROP TABLE user_training_days;
DROP TABLE user_xp_events;
DROP TABLE skill_unlock_events;
DROP FUNCTION skill_unlock_events_append_only();
DROP TABLE user_skill_states;
DROP FUNCTION user_skill_states_monotonic();
