-- Workout templates. They mirror the logging shape so that starting a session
-- from a template is a structural copy. Private-only in v1 (ADR 0002 §5).
-- Every table here is syncable: see docs/DDL_PROPOSAL.md §1.4.

-- +goose Up
CREATE TABLE workout_templates (
    id                 uuid        PRIMARY KEY,
    user_id            uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               text        NOT NULL,
    description        text        NOT NULL DEFAULT '',
    visibility         text        NOT NULL DEFAULT 'private'
        CONSTRAINT workout_templates_visibility_ck
        CHECK (visibility IN ('private','unlisted','public')),   -- only 'private' reachable in v1
    source_template_id uuid            NULL REFERENCES workout_templates(id) ON DELETE SET NULL,
    est_duration_min   smallint        NULL
        CONSTRAINT workout_templates_duration_ck CHECK (est_duration_min IS NULL OR est_duration_min > 0),

    client_id          uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at         timestamptz NOT NULL,
    server_updated_at  timestamptz NOT NULL DEFAULT now(),
    server_seq         bigint      NOT NULL DEFAULT 0,
    deleted_at         timestamptz     NULL,

    CONSTRAINT workout_templates_id_user_uk UNIQUE (id, user_id)
);
CREATE INDEX workout_templates_user_idx ON workout_templates (user_id) WHERE deleted_at IS NULL;
CREATE INDEX workout_templates_sync_idx ON workout_templates (user_id, server_seq);

CREATE TABLE template_blocks (
    id                uuid        PRIMARY KEY,
    user_id           uuid        NOT NULL,
    template_id       uuid        NOT NULL,
    order_index       int         NOT NULL
        CONSTRAINT template_blocks_order_index_ck CHECK (order_index >= 0),
    kind              text        NOT NULL DEFAULT 'straight'
        CONSTRAINT template_blocks_kind_ck
        CHECK (kind IN ('straight','superset','circuit','emom','amrap')),
    rounds_planned    smallint        NULL
        CONSTRAINT template_blocks_rounds_ck CHECK (rounds_planned IS NULL OR rounds_planned > 0),
    interval_s        int             NULL,
    notes             text        NOT NULL DEFAULT '',

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL DEFAULT 0,
    deleted_at        timestamptz     NULL,

    CONSTRAINT template_blocks_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT template_blocks_order_uk UNIQUE (template_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_id, user_id)
        REFERENCES workout_templates (id, user_id) ON DELETE CASCADE,
    CONSTRAINT template_blocks_interval_ck
        CHECK (kind IN ('emom','amrap') OR interval_s IS NULL)
);
CREATE INDEX template_blocks_sync_idx ON template_blocks (user_id, server_seq);

CREATE TABLE template_set_entries (
    id                   uuid         PRIMARY KEY,
    user_id              uuid         NOT NULL,
    template_block_id    uuid         NOT NULL,
    order_index          int          NOT NULL
        CONSTRAINT template_set_entries_order_index_ck CHECK (order_index >= 0),
    kind                 text         NOT NULL DEFAULT 'working'
        CONSTRAINT template_set_entries_kind_ck
        CHECK (kind IN ('working','warmup','backoff','drop','cluster','test')),
    rest_after_planned_s int              NULL
        CONSTRAINT template_set_entries_rest_ck CHECK (rest_after_planned_s IS NULL OR rest_after_planned_s >= 0),
    target_rpe           numeric(3,1)     NULL
        CONSTRAINT template_set_entries_rpe_ck CHECK (target_rpe IS NULL OR target_rpe BETWEEN 1.0 AND 10.0),
    notes                text         NOT NULL DEFAULT '',

    client_id            uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at           timestamptz  NOT NULL,
    server_updated_at    timestamptz  NOT NULL DEFAULT now(),
    server_seq           bigint       NOT NULL DEFAULT 0,
    deleted_at           timestamptz      NULL,

    CONSTRAINT template_set_entries_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT template_set_entries_order_uk
        UNIQUE (template_block_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_block_id, user_id)
        REFERENCES template_blocks (id, user_id) ON DELETE CASCADE
);
CREATE INDEX template_set_entries_sync_idx ON template_set_entries (user_id, server_seq);

CREATE TABLE template_set_elements (
    id                    uuid         PRIMARY KEY,
    user_id               uuid         NOT NULL,
    template_set_entry_id uuid         NOT NULL,
    order_index           int          NOT NULL
        CONSTRAINT template_set_elements_order_index_ck CHECK (order_index >= 0),
    exercise_id           uuid         NOT NULL REFERENCES exercises(id),
    measure               text         NOT NULL
        CONSTRAINT template_set_elements_measure_ck
        CHECK (measure IN ('reps','hold_seconds','distance_m','none')),
    target_reps           int              NULL
        CONSTRAINT template_set_elements_reps_ck CHECK (target_reps IS NULL OR target_reps >= 0),
    target_hold_seconds   numeric(7,2)     NULL
        CONSTRAINT template_set_elements_hold_ck CHECK (target_hold_seconds IS NULL OR target_hold_seconds >= 0),
    target_distance_m     numeric(7,2)     NULL
        CONSTRAINT template_set_elements_distance_ck CHECK (target_distance_m IS NULL OR target_distance_m >= 0),
    -- Added external load only; assistance is never a negative load.
    target_load_kg        numeric(6,2) NOT NULL DEFAULT 0
        CONSTRAINT template_set_elements_load_ck CHECK (target_load_kg >= 0),
    tempo                 text             NULL
        CONSTRAINT template_set_elements_tempo_ck CHECK (tempo IS NULL OR tempo ~ '^[0-9X]{4}$'),

    client_id             uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at            timestamptz  NOT NULL,
    server_updated_at     timestamptz  NOT NULL DEFAULT now(),
    server_seq            bigint       NOT NULL DEFAULT 0,
    deleted_at            timestamptz      NULL,

    CONSTRAINT template_set_elements_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT template_set_elements_order_uk
        UNIQUE (template_set_entry_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_set_entry_id, user_id)
        REFERENCES template_set_entries (id, user_id) ON DELETE CASCADE,
    CONSTRAINT template_set_elements_single_value_ck
        CHECK (num_nonnulls(target_reps, target_hold_seconds, target_distance_m) <= 1),
    CONSTRAINT template_set_elements_value_matches_measure_ck
        CHECK ((target_reps         IS NULL OR measure = 'reps')
           AND (target_hold_seconds IS NULL OR measure = 'hold_seconds')
           AND (target_distance_m   IS NULL OR measure = 'distance_m'))
);
CREATE INDEX template_set_elements_sync_idx ON template_set_elements (user_id, server_seq);

CREATE TRIGGER workout_templates_touch_sync BEFORE INSERT OR UPDATE ON workout_templates
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER template_blocks_touch_sync BEFORE INSERT OR UPDATE ON template_blocks
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER template_set_entries_touch_sync BEFORE INSERT OR UPDATE ON template_set_entries
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER template_set_elements_touch_sync BEFORE INSERT OR UPDATE ON template_set_elements
    FOR EACH ROW EXECUTE FUNCTION touch_sync();

-- +goose Down
DROP TABLE template_set_elements;
DROP TABLE template_set_entries;
DROP TABLE template_blocks;
DROP TABLE workout_templates;
