-- Content: the authored catalogue. Structure only — rows arrive via cmd/seed,
-- never via a migration. See docs/DDL_PROPOSAL.md §4–§5.

-- +goose Up
CREATE TABLE content_versions (
    id          uuid        PRIMARY KEY,
    -- sha256 over the normalised content tree. Not unique: reverting content
    -- (A -> B -> A) legitimately records A again as the newest version.
    checksum    bytea       NOT NULL,
    git_sha     text            NULL,
    item_counts jsonb       NOT NULL DEFAULT '{}'::jsonb,
    applied_at  timestamptz NOT NULL DEFAULT now(),
    applied_by  text            NULL
);
CREATE INDEX content_versions_applied_idx ON content_versions (applied_at DESC, id DESC);

-- One family vocabulary shared by skills and exercises (deviation D-1).
CREATE TABLE families (
    id          uuid     PRIMARY KEY,
    slug        text     NOT NULL UNIQUE,
    name        text     NOT NULL,
    order_index smallint NOT NULL DEFAULT 0
);

CREATE TABLE exercises (
    id               uuid     PRIMARY KEY,
    slug             text     NOT NULL UNIQUE,
    name             text     NOT NULL,
    aka              text[]   NOT NULL DEFAULT '{}',
    family_id        uuid     NOT NULL REFERENCES families(id),
    default_measure  text     NOT NULL
        CONSTRAINT exercises_default_measure_ck
        CHECK (default_measure IN ('reps','hold_seconds','distance_m','none')),
    -- How load_kg on a set element reads for this exercise:
    --   added    : bodyweight movement, load_kg is extra weight
    --   external : the implement is the load
    --   none     : load is meaningless (mobility drills)
    load_semantics   text     NOT NULL DEFAULT 'added'
        CONSTRAINT exercises_load_semantics_ck
        CHECK (load_semantics IN ('added','external','none')),
    is_bodyweight    boolean  NOT NULL DEFAULT true,
    unilateral       boolean  NOT NULL DEFAULT false,
    tempo_applicable boolean  NOT NULL DEFAULT true,
    equipment        text[]   NOT NULL DEFAULT '{}',
    summary          text     NOT NULL DEFAULT '',
    cues             text[]   NOT NULL DEFAULT '{}',
    common_faults    text[]   NOT NULL DEFAULT '{}',
    status           text     NOT NULL DEFAULT 'active'
        CONSTRAINT exercises_status_ck
        CHECK (status IN ('active','draft_placeholder','retired')),
    content_version_id uuid   NOT NULL REFERENCES content_versions(id),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX exercises_family_idx ON exercises (family_id) WHERE status <> 'retired';
CREATE INDEX exercises_name_trgm  ON exercises USING gin (name gin_trgm_ops);
CREATE INDEX exercises_aka_idx    ON exercises USING gin (aka);

CREATE TABLE bands (
    id                uuid         PRIMARY KEY,
    owner_user_id     uuid             NULL REFERENCES users(id) ON DELETE CASCADE,  -- NULL = global catalogue
    brand             text         NOT NULL,
    colour_label      text         NOT NULL,
    resistance_min_kg numeric(5,2) NOT NULL,
    resistance_max_kg numeric(5,2) NOT NULL,
    length_cm         numeric(6,1)     NULL,
    thickness_mm      numeric(5,1)     NULL,
    created_at        timestamptz  NOT NULL DEFAULT now(),
    updated_at        timestamptz  NOT NULL DEFAULT now(),
    deleted_at        timestamptz      NULL,

    CONSTRAINT bands_resistance_ck         CHECK (resistance_min_kg <= resistance_max_kg),
    CONSTRAINT bands_resistance_nonneg_ck  CHECK (resistance_min_kg >= 0)
);
-- NULLS NOT DISTINCT so two global entries cannot collide on the same pair.
CREATE UNIQUE INDEX bands_identity_uk
    ON bands (owner_user_id, brand, colour_label) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;

CREATE TABLE skills (
    id                uuid         PRIMARY KEY,
    slug              text         NOT NULL UNIQUE,
    family_id         uuid         NOT NULL REFERENCES families(id),
    name              text         NOT NULL,
    aka               text[]       NOT NULL DEFAULT '{}',
    difficulty_tier   smallint     NOT NULL
        CONSTRAINT skills_difficulty_tier_ck CHECK (difficulty_tier BETWEEN 1 AND 10),
    is_milestone      boolean      NOT NULL DEFAULT false,
    summary           text         NOT NULL DEFAULT '',
    primary_muscles   text[]       NOT NULL DEFAULT '{}',
    common_faults     text[]       NOT NULL DEFAULT '{}',
    -- Map layout is authored in content, never force-directed at runtime.
    map_constellation text             NULL,
    map_x             numeric(8,2)     NULL,
    map_y             numeric(8,2)     NULL,
    status            text         NOT NULL DEFAULT 'active'
        CONSTRAINT skills_status_ck CHECK (status IN ('active','draft_placeholder','retired')),
    content_version_id uuid        NOT NULL REFERENCES content_versions(id),
    created_at        timestamptz  NOT NULL DEFAULT now(),
    updated_at        timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT skills_milestone_placed_ck
        CHECK (NOT is_milestone OR (map_constellation IS NOT NULL AND map_x IS NOT NULL AND map_y IS NOT NULL))
);
CREATE INDEX skills_family_idx ON skills (family_id);
CREATE INDEX skills_name_trgm  ON skills USING gin (name gin_trgm_ops);

CREATE TABLE skill_levels (
    id                  uuid     PRIMARY KEY,
    skill_id            uuid     NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    order_index         smallint NOT NULL
        CONSTRAINT skill_levels_order_index_ck CHECK (order_index >= 0),
    slug                text     NOT NULL,   -- unique within the skill
    name                text     NOT NULL,
    description         text     NOT NULL DEFAULT '',
    unlock_criteria     jsonb    NOT NULL DEFAULT '{}'::jsonb
        CONSTRAINT skill_levels_criteria_object_ck CHECK (jsonb_typeof(unlock_criteria) = 'object'),
    -- A display hint shared by all users; never authoritative in the client.
    est_weeks_from_prev smallint     NULL
        CONSTRAINT skill_levels_est_weeks_ck CHECK (est_weeks_from_prev IS NULL OR est_weeks_from_prev > 0),
    updated_at          timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT skill_levels_slug_uk  UNIQUE (skill_id, slug),
    CONSTRAINT skill_levels_order_uk UNIQUE (skill_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT skill_levels_id_skill_uk UNIQUE (id, skill_id)
);

CREATE TABLE skill_edges (
    from_skill_level_id uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    to_skill_level_id   uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    relation            text NOT NULL
        CONSTRAINT skill_edges_relation_ck
        CHECK (relation IN ('prerequisite','recommended','alternative','antagonist')),
    weight              numeric(4,2) NOT NULL DEFAULT 1.0
        CONSTRAINT skill_edges_weight_ck CHECK (weight > 0),

    PRIMARY KEY (from_skill_level_id, to_skill_level_id, relation),
    CONSTRAINT skill_edges_no_self_ck CHECK (from_skill_level_id <> to_skill_level_id)
);
CREATE INDEX skill_edges_to_idx ON skill_edges (to_skill_level_id, relation);
-- Acyclicity over relation='prerequisite' is checked by contentlint and again
-- inside the seed transaction (deviation D-7), not by the database.

CREATE TABLE skill_level_exercises (
    skill_level_id uuid     NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    exercise_id    uuid     NOT NULL REFERENCES exercises(id)    ON DELETE CASCADE,
    role           text     NOT NULL
        CONSTRAINT skill_level_exercises_role_ck
        CHECK (role IN ('primary_test','progression','accessory','prehab')),
    order_index    smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (skill_level_id, exercise_id, role)
);
-- Exactly one primary_test per level; contentlint also asserts it exists.
CREATE UNIQUE INDEX skill_level_primary_test_uk
    ON skill_level_exercises (skill_level_id)
    WHERE role = 'primary_test';
CREATE INDEX skill_level_exercises_exercise_idx ON skill_level_exercises (exercise_id);

CREATE TABLE skill_injury_risks (
    id            uuid     PRIMARY KEY,
    skill_id      uuid     NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    region        text     NOT NULL,
    name          text     NOT NULL,
    description   text     NOT NULL DEFAULT '',
    risk_factors  text[]   NOT NULL DEFAULT '{}',
    early_signs   text[]   NOT NULL DEFAULT '{}',
    -- Constrained to one value on purpose: injury content cannot exist
    -- without its disclaimer (ADR 0003).
    disclaimer    text     NOT NULL DEFAULT 'educational_only'
        CONSTRAINT skill_injury_risks_disclaimer_ck CHECK (disclaimer = 'educational_only'),
    order_index   smallint NOT NULL DEFAULT 0,
    content_version_id uuid NOT NULL REFERENCES content_versions(id),

    CONSTRAINT skill_injury_risks_identity_uk UNIQUE (skill_id, region, name)
);

CREATE TABLE injury_prehab_exercises (
    risk_id     uuid     NOT NULL REFERENCES skill_injury_risks(id) ON DELETE CASCADE,
    exercise_id uuid     NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    order_index smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (risk_id, exercise_id)
);

-- +goose Down
DROP TABLE injury_prehab_exercises;
DROP TABLE skill_injury_risks;
DROP TABLE skill_level_exercises;
DROP TABLE skill_edges;
DROP TABLE skill_levels;
DROP TABLE skills;
DROP TABLE bands;
DROP TABLE exercises;
DROP TABLE families;
DROP TABLE content_versions;
