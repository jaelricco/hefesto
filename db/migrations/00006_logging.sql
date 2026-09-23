-- Logging: sessions -> blocks -> set entries -> set elements (+ assistance).
--
-- The invariant: a plain set of 8 pull-ups is one set_entry with one
-- set_element; a combo is one set_entry with N ordered set_elements. There is
-- no second code path for the simple case. See docs/DDL_PROPOSAL.md §7.

-- +goose Up
CREATE TABLE workout_sessions (
    id                uuid         PRIMARY KEY,
    user_id           uuid         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at        timestamptz  NOT NULL,
    ended_at          timestamptz      NULL,
    timezone          text         NOT NULL,   -- IANA, e.g. Europe/Zurich
    local_date        date         NOT NULL,   -- computed by the application; DDL §1.2
    title             text         NOT NULL DEFAULT '',
    notes             text         NOT NULL DEFAULT '',
    perceived_fatigue smallint         NULL
        CONSTRAINT workout_sessions_fatigue_ck CHECK (perceived_fatigue BETWEEN 1 AND 10),
    bodyweight_kg     numeric(5,2)     NULL
        CONSTRAINT workout_sessions_bodyweight_ck CHECK (bodyweight_kg IS NULL OR bodyweight_kg > 0),
    status            text         NOT NULL DEFAULT 'draft'
        CONSTRAINT workout_sessions_status_ck CHECK (status IN ('draft','completed','abandoned')),
    is_rest_day       boolean      NOT NULL DEFAULT false,   -- deviation D-4
    template_id       uuid             NULL REFERENCES workout_templates(id) ON DELETE SET NULL,
    completed_at      timestamptz      NULL,

    client_id         uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz  NOT NULL,
    server_updated_at timestamptz  NOT NULL DEFAULT now(),
    server_seq        bigint       NOT NULL DEFAULT 0,
    deleted_at        timestamptz      NULL,

    CONSTRAINT workout_sessions_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT workout_sessions_interval_ck CHECK (ended_at IS NULL OR ended_at >= started_at),
    CONSTRAINT workout_sessions_completed_ck
        CHECK ((status = 'completed') = (completed_at IS NOT NULL))
);
CREATE INDEX workout_sessions_user_date_idx
    ON workout_sessions (user_id, local_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX workout_sessions_sync_idx ON workout_sessions (user_id, server_seq);

CREATE TABLE session_blocks (
    id                uuid        PRIMARY KEY,
    user_id           uuid        NOT NULL,
    session_id        uuid        NOT NULL,
    order_index       int         NOT NULL
        CONSTRAINT session_blocks_order_index_ck CHECK (order_index >= 0),
    kind              text        NOT NULL DEFAULT 'straight'
        CONSTRAINT session_blocks_kind_ck
        CHECK (kind IN ('straight','superset','circuit','emom','amrap')),
    rounds_planned    smallint        NULL
        CONSTRAINT session_blocks_rounds_planned_ck CHECK (rounds_planned IS NULL OR rounds_planned > 0),
    rounds_done       smallint        NULL
        CONSTRAINT session_blocks_rounds_done_ck CHECK (rounds_done IS NULL OR rounds_done >= 0),
    interval_s        int             NULL,   -- emom window / amrap cap
    notes             text        NOT NULL DEFAULT '',

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL DEFAULT 0,
    deleted_at        timestamptz     NULL,

    CONSTRAINT session_blocks_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT session_blocks_order_uk UNIQUE (session_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (session_id, user_id)
        REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE,
    CONSTRAINT session_blocks_interval_ck
        CHECK (kind IN ('emom','amrap') OR interval_s IS NULL)
);
CREATE INDEX session_blocks_sync_idx ON session_blocks (user_id, server_seq);

CREATE TABLE set_entries (
    id                   uuid         PRIMARY KEY,
    user_id              uuid         NOT NULL,
    session_id           uuid         NOT NULL,   -- denormalised one level; deviation D-2
    block_id             uuid         NOT NULL,
    order_index          int          NOT NULL
        CONSTRAINT set_entries_order_index_ck CHECK (order_index >= 0),
    -- Each round of a circuit is its own set_entry; this says which round.
    round_index          smallint         NULL
        CONSTRAINT set_entries_round_index_ck CHECK (round_index IS NULL OR round_index >= 0),
    kind                 text         NOT NULL DEFAULT 'working'
        CONSTRAINT set_entries_kind_ck
        CHECK (kind IN ('working','warmup','backoff','drop','cluster','test')),
    is_planned           boolean      NOT NULL DEFAULT false,
    -- Rest lives on the entry: inside a combo there is by definition none.
    rest_after_planned_s int              NULL
        CONSTRAINT set_entries_rest_planned_ck CHECK (rest_after_planned_s IS NULL OR rest_after_planned_s >= 0),
    rest_after_actual_s  int              NULL
        CONSTRAINT set_entries_rest_actual_ck CHECK (rest_after_actual_s IS NULL OR rest_after_actual_s >= 0),
    rpe                  numeric(3,1)     NULL
        CONSTRAINT set_entries_rpe_ck CHECK (rpe IS NULL OR rpe BETWEEN 1.0 AND 10.0),
    rir                  smallint         NULL
        CONSTRAINT set_entries_rir_ck CHECK (rir IS NULL OR rir BETWEEN 0 AND 10),
    completed_at         timestamptz      NULL,
    notes                text         NOT NULL DEFAULT '',

    client_id            uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at           timestamptz  NOT NULL,
    server_updated_at    timestamptz  NOT NULL DEFAULT now(),
    server_seq           bigint       NOT NULL DEFAULT 0,
    deleted_at           timestamptz      NULL,

    CONSTRAINT set_entries_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT set_entries_order_uk UNIQUE (block_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (block_id, user_id)   REFERENCES session_blocks   (id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (session_id, user_id) REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE,
    -- A planned set has not happened yet; a performed set has a timestamp.
    CONSTRAINT set_entries_planned_ck CHECK (NOT is_planned OR completed_at IS NULL)
);
CREATE INDEX set_entries_session_idx ON set_entries (session_id) WHERE deleted_at IS NULL;
CREATE INDEX set_entries_block_idx   ON set_entries (block_id);
CREATE INDEX set_entries_sync_idx    ON set_entries (user_id, server_seq);

CREATE TABLE set_elements (
    id                uuid         PRIMARY KEY,
    user_id           uuid         NOT NULL,
    session_id        uuid         NOT NULL,   -- denormalised two levels; deviation D-2
    set_entry_id      uuid         NOT NULL,
    order_index       int          NOT NULL
        CONSTRAINT set_elements_order_index_ck CHECK (order_index >= 0),
    exercise_id       uuid         NOT NULL REFERENCES exercises(id),

    measure           text         NOT NULL
        CONSTRAINT set_elements_measure_ck
        CHECK (measure IN ('reps','hold_seconds','distance_m','none')),
    reps              int              NULL
        CONSTRAINT set_elements_reps_ck CHECK (reps IS NULL OR reps >= 0),
    hold_seconds      numeric(7,2)     NULL
        CONSTRAINT set_elements_hold_ck CHECK (hold_seconds IS NULL OR hold_seconds >= 0),
    distance_m        numeric(7,2)     NULL
        CONSTRAINT set_elements_distance_ck CHECK (distance_m IS NULL OR distance_m >= 0),

    tempo             text             NULL
        CONSTRAINT set_elements_tempo_ck CHECK (tempo IS NULL OR tempo ~ '^[0-9X]{4}$'),  -- "30X1"
    -- Added external load only, never negative. All assistance, including a
    -- counterweight, lives in set_element_assistance (docs/adr/0005).
    load_kg           numeric(6,2) NOT NULL DEFAULT 0
        CONSTRAINT set_elements_load_ck CHECK (load_kg >= 0),

    is_eccentric_only boolean      NOT NULL DEFAULT false,
    is_partial_rom    boolean      NOT NULL DEFAULT false,
    rom_note          text             NULL,
    form_quality      smallint         NULL
        CONSTRAINT set_elements_form_quality_ck CHECK (form_quality BETWEEN 1 AND 5),
    failed            boolean      NOT NULL DEFAULT false,
    -- Derived from set_element_assistance and load_kg for the evaluator's hot
    -- path (deviation D-3); written by the same code that writes assistance.
    assistance_class  text         NOT NULL DEFAULT 'unassisted'
        CONSTRAINT set_elements_assistance_class_ck
        CHECK (assistance_class IN ('unassisted','assisted','loaded')),

    client_id         uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz  NOT NULL,
    server_updated_at timestamptz  NOT NULL DEFAULT now(),
    server_seq        bigint       NOT NULL DEFAULT 0,
    deleted_at        timestamptz      NULL,

    CONSTRAINT set_elements_id_user_uk UNIQUE (id, user_id),
    CONSTRAINT set_elements_order_uk UNIQUE (set_entry_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (set_entry_id, user_id) REFERENCES set_entries      (id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (session_id, user_id)   REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE,
    -- At most one measured value, and it must be the one `measure` names.
    -- Permissive about NULL so a planned-but-unperformed element is
    -- representable without a second table.
    CONSTRAINT set_elements_single_value_ck
        CHECK (num_nonnulls(reps, hold_seconds, distance_m) <= 1),
    CONSTRAINT set_elements_value_matches_measure_ck
        CHECK ((reps         IS NULL OR measure = 'reps')
           AND (hold_seconds IS NULL OR measure = 'hold_seconds')
           AND (distance_m   IS NULL OR measure = 'distance_m')),
    CONSTRAINT set_elements_rom_note_ck
        CHECK (rom_note IS NULL OR is_partial_rom)
);
-- The evaluator's hot path: this user's elements for exercise X.
CREATE INDEX set_elements_user_exercise_idx
    ON set_elements (user_id, exercise_id, session_id)
    WHERE deleted_at IS NULL;
CREATE INDEX set_elements_entry_idx ON set_elements (set_entry_id) WHERE deleted_at IS NULL;
CREATE INDEX set_elements_sync_idx  ON set_elements (user_id, server_seq);

CREATE TABLE set_element_assistance (
    id                  uuid         PRIMARY KEY,
    user_id             uuid         NOT NULL,
    set_element_id      uuid         NOT NULL,
    type                text         NOT NULL
        CONSTRAINT set_element_assistance_type_ck
        CHECK (type IN ('none','band','partner','machine','incline','counterweight','foot_support')),
    band_id             uuid             NULL REFERENCES bands(id) ON DELETE RESTRICT,
    band_count          smallint     NOT NULL DEFAULT 0
        CONSTRAINT set_element_assistance_band_count_ck CHECK (band_count >= 0),
    anchor              text             NULL
        CONSTRAINT set_element_assistance_anchor_ck
        CHECK (anchor IS NULL OR anchor IN ('overhead','under_foot','under_knee','hip','other')),
    estimated_assist_kg numeric(6,2)     NULL
        CONSTRAINT set_element_assistance_kg_ck CHECK (estimated_assist_kg IS NULL OR estimated_assist_kg >= 0),
    note                text         NOT NULL DEFAULT '',

    client_id           uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at          timestamptz  NOT NULL,
    server_updated_at   timestamptz  NOT NULL DEFAULT now(),
    server_seq          bigint       NOT NULL DEFAULT 0,
    deleted_at          timestamptz      NULL,

    CONSTRAINT set_element_assistance_id_user_uk UNIQUE (id, user_id),
    FOREIGN KEY (set_element_id, user_id) REFERENCES set_elements (id, user_id) ON DELETE CASCADE,
    CONSTRAINT set_element_assistance_band_ck
        CHECK ((type = 'band') = (band_id IS NOT NULL AND band_count > 0))
);
CREATE INDEX set_element_assistance_element_idx ON set_element_assistance (set_element_id);
CREATE INDEX set_element_assistance_sync_idx    ON set_element_assistance (user_id, server_seq);

CREATE TABLE set_element_media (
    set_element_id uuid        NOT NULL,
    user_id        uuid        NOT NULL,
    media_id       uuid        NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    order_index    smallint    NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (set_element_id, media_id),
    FOREIGN KEY (set_element_id, user_id) REFERENCES set_elements (id, user_id) ON DELETE CASCADE
);

-- Bodyweight on days without training, so weighted-vs-bodyweight PR maths has
-- a value to use (DDL open question 2).
CREATE TABLE user_bodyweight_log (
    id                uuid         PRIMARY KEY,
    user_id           uuid         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    measured_at       timestamptz  NOT NULL,
    local_date        date         NOT NULL,
    bodyweight_kg     numeric(5,2) NOT NULL
        CONSTRAINT user_bodyweight_log_kg_ck CHECK (bodyweight_kg > 0),
    note              text         NOT NULL DEFAULT '',

    client_id         uuid             NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz  NOT NULL,
    server_updated_at timestamptz  NOT NULL DEFAULT now(),
    server_seq        bigint       NOT NULL DEFAULT 0,
    deleted_at        timestamptz      NULL
);
CREATE INDEX user_bodyweight_log_user_time_idx
    ON user_bodyweight_log (user_id, measured_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX user_bodyweight_log_sync_idx ON user_bodyweight_log (user_id, server_seq);

CREATE TRIGGER workout_sessions_touch_sync BEFORE INSERT OR UPDATE ON workout_sessions
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER session_blocks_touch_sync BEFORE INSERT OR UPDATE ON session_blocks
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER set_entries_touch_sync BEFORE INSERT OR UPDATE ON set_entries
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER set_elements_touch_sync BEFORE INSERT OR UPDATE ON set_elements
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER set_element_assistance_touch_sync BEFORE INSERT OR UPDATE ON set_element_assistance
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
CREATE TRIGGER user_bodyweight_log_touch_sync BEFORE INSERT OR UPDATE ON user_bodyweight_log
    FOR EACH ROW EXECUTE FUNCTION touch_sync();

-- +goose Down
DROP TABLE user_bodyweight_log;
DROP TABLE set_element_media;
DROP TABLE set_element_assistance;
DROP TABLE set_elements;
DROP TABLE set_entries;
DROP TABLE session_blocks;
DROP TABLE workout_sessions;
