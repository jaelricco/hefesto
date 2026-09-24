# Hefesto — schema (approved Phase 0 proposal)

**Status: approved 2026-09-23 and implemented in Phase 1** as
`db/migrations/00001`–`00008`. The decisions on §12 and §13, and the
corrections made while implementing, are recorded in
[ADR 0005](adr/0005-schema-review-decisions.md). The migrations are now the
source of truth; this document explains the reasoning behind them and is kept
as written, except where a note below says otherwise.

Review outcome, in short:

- D-1 … D-8 accepted.
- Open questions: one `set_entry` per circuit round; no rest inside a combo;
  edges between levels; `est_weeks_from_prev` is a display hint.
- **Changed from this proposal:** `load_kg` is never negative (all assistance,
  counterweight included, lives in `set_element_assistance`); a
  `user_bodyweight_log` table is added; deletion is soft for 30 days
  (`users.deletion_requested_at`); `touch_sync` always assigns `server_seq`;
  `content_versions.checksum` is not unique. See ADR 0005.

---

## 1. Conventions applied to every table

### 1.1 Primary keys

Every primary key is `uuid`, holding a **UUIDv7**, generated in application code
— on the client for syncable rows, in the Go server for everything else. No
table carries a `DEFAULT` for its id.

Postgres 16 has no native `uuidv7()` (that arrived in 18), and `pg_uuidv7`
would force a custom Postgres image into Compose, CI and production. Since the
sync contract already requires client-generated ids, generating them in Go
everywhere is the consistent choice rather than a compromise. UUIDv7 keeps the
time-ordered insert locality that matters for the large tables
(`set_elements` above all).

### 1.2 Time

- All instants are `timestamptz`. The database runs in UTC; nothing stores local
  time in a timestamp column.
- `workout_sessions` additionally stores the IANA `timezone` the session was
  logged in **and** a plain `local_date date`, computed by the application.

  The `local_date` column is not redundant. `started_at AT TIME ZONE timezone`
  is `STABLE`, not `IMMUTABLE`, so Postgres will not allow it as a generated
  column or index it usefully. Streaks, calendars and "sessions this week" all
  need the athlete's local day, and they need it to survive the athlete moving
  timezones: a session logged at 23:30 in Zurich stays on that date forever.

### 1.3 Closed vocabularies

`text` with a **named** `CHECK` constraint, never a Postgres `ENUM`.
`ALTER TYPE ... ADD VALUE` cannot be used in the same transaction that then
references the new value, which fights goose's transactional migrations. Named
constraints also produce readable violation errors we can map to
`problem+json` field errors. Go constants provide the type safety at the
boundary.

### 1.4 Soft delete and sync columns

Every **syncable** table (sessions, blocks, set entries, set elements,
assistance, templates and their children, user bands, media links) carries:

```sql
    user_id           uuid        NOT NULL,       -- denormalised; see §1.5
    client_id         uuid            NULL,       -- device that authored this revision
    updated_at        timestamptz NOT NULL,       -- the client's clock; the client's claim
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL DEFAULT 0,  -- per-user monotonic cursor; see §10
    deleted_at        timestamptz     NULL
```

`server_seq` is assigned by the `touch_sync` trigger on both INSERT and UPDATE
(§2). The `DEFAULT 0` exists only so an INSERT need not mention the column; the
value never survives the trigger. A client cannot set it.

Rows are never hard-deleted by user action. `deleted_at` is set and the row
still syncs, so other devices learn about the deletion. Hard deletion happens
only on account deletion, via `ON DELETE CASCADE` from `users`.

Content tables (skills, exercises, bands catalogue) are **not** syncable in this
sense. They version as a whole through `content_versions` and are fetched with
an ETag.

### 1.5 `user_id` on every owned row, enforced by composite foreign keys

`set_elements` could reach its owner through
`set_entries → session_blocks → workout_sessions`. It carries `user_id` anyway,
for three reasons: sync partitions by user and must not join four levels up to
do it; every authorisation check becomes a single predicate; and the hot
evaluator query ("this user's set elements for exercise X in the last N days")
becomes a two-table read.

The denormalisation is kept honest by composite foreign keys rather than by
trust:

```sql
-- parent
CREATE TABLE session_blocks (
    id      uuid NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL,
    ...
    UNIQUE (id, user_id)
);

-- child references BOTH columns, so a child can never name a parent
-- belonging to a different user
CREATE TABLE set_entries (
    id       uuid NOT NULL PRIMARY KEY,
    user_id  uuid NOT NULL,
    block_id uuid NOT NULL,
    ...
    UNIQUE (id, user_id),
    FOREIGN KEY (block_id, user_id)
        REFERENCES session_blocks (id, user_id) ON DELETE CASCADE
);
```

This pattern repeats down the whole logging chain. It costs one redundant unique
index per table and makes a whole class of cross-tenant bug structurally
impossible.

### 1.6 Ordering

Sibling ordering is `order_index int NOT NULL CHECK (order_index >= 0)` with
`UNIQUE (parent_id, order_index) DEFERRABLE INITIALLY IMMEDIATE`. Deferrable
because reordering three elements inside one transaction otherwise needs a
temporary negative-index dance. Gaps are allowed; the API renumbers on write.

### 1.7 Authorisation

No row-level security in v1. The API connects as a single role and every query
filters on `user_id` supplied by the `actor_id`/`subject_user_id` helper
(ADR 0002). RLS was considered and rejected: it would add a per-request
`SET LOCAL` round trip, interact badly with pgx's connection pooling, and
duplicate a check that sqlc-generated queries already make explicit and
reviewable.

---

## 2. Extensions and shared helpers

```sql
CREATE EXTENSION IF NOT EXISTS citext;    -- case-insensitive email
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- exercise / skill name search

-- Per-user monotonic sync counter. Serialised by the row lock on users,
-- which is what makes a client cursor safe against holes (§10).
CREATE FUNCTION next_sync_seq(p_user_id uuid) RETURNS bigint
LANGUAGE sql AS $$
    UPDATE users SET sync_seq = sync_seq + 1
    WHERE id = p_user_id
    RETURNING sync_seq;
$$;

-- Applied by a BEFORE INSERT OR UPDATE trigger on every syncable table, so
-- server_seq is never the client's to set. TG_OP is required: OLD does not
-- exist on INSERT.
CREATE FUNCTION touch_sync() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.server_updated_at := now();
    IF TG_OP = 'INSERT' OR NEW.server_seq IS NOT DISTINCT FROM OLD.server_seq THEN
        NEW.server_seq := next_sync_seq(NEW.user_id);
    END IF;
    RETURN NEW;
END;
$$;

-- …and per syncable table:
CREATE TRIGGER <table>_touch_sync BEFORE INSERT OR UPDATE ON <table>
    FOR EACH ROW EXECUTE FUNCTION touch_sync();
```

---

## 3. Identity and auth

```sql
CREATE TABLE users (
    id                  uuid        PRIMARY KEY,
    email               citext          NULL UNIQUE,   -- NULL for Apple-only with hidden relay
    email_verified_at   timestamptz     NULL,
    password_hash       text            NULL,          -- argon2id encoded string; NULL = no password login
    display_name        text        NOT NULL DEFAULT '',
    locale              text        NOT NULL DEFAULT 'de-CH',
    unit_system         text        NOT NULL DEFAULT 'metric'
        CONSTRAINT users_unit_system_ck CHECK (unit_system IN ('metric','imperial')),
    week_start          smallint    NOT NULL DEFAULT 1  -- ISO: Monday
        CONSTRAINT users_week_start_ck CHECK (week_start BETWEEN 1 AND 7),
    timezone            text        NOT NULL DEFAULT 'Europe/Zurich',
    status              text        NOT NULL DEFAULT 'active'
        CONSTRAINT users_status_ck CHECK (status IN ('active','suspended','deletion_pending')),
    sync_seq            bigint      NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    deleted_at          timestamptz     NULL,

    CONSTRAINT users_login_method_ck
        CHECK (password_hash IS NULL OR email IS NOT NULL)
);

CREATE TABLE apple_identities (
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    apple_sub     text        NOT NULL PRIMARY KEY,      -- stable Apple subject
    is_private_email boolean  NOT NULL DEFAULT false,
    linked_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);

-- Rotating refresh tokens with reuse detection. A reused token revokes its
-- whole family, which is the standard response to a stolen refresh token.
CREATE TABLE refresh_tokens (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id   uuid        NOT NULL,
    token_hash  bytea       NOT NULL UNIQUE,   -- sha256 of the opaque token; never the token itself
    device_id   uuid            NULL,          -- FK added after devices; see §3.1
    issued_at   timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz     NULL,
    revoked_at  timestamptz     NULL,
    revoked_reason text          NULL
        CONSTRAINT refresh_tokens_revoked_reason_ck
        CHECK (revoked_reason IS NULL OR revoked_reason IN ('rotated','logout','reuse_detected','admin','expired')),
    replaced_by uuid            NULL REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent  text            NULL,
    ip          inet            NULL
);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id) WHERE revoked_at IS NULL;
CREATE INDEX refresh_tokens_user_idx   ON refresh_tokens (user_id, expires_at);

CREATE TABLE email_verifications (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose     text        NOT NULL
        CONSTRAINT email_verifications_purpose_ck CHECK (purpose IN ('verify_email','reset_password')),
    token_hash  bytea       NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz     NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
```

### 3.1 Devices

```sql
CREATE TABLE devices (
    id             uuid        PRIMARY KEY,        -- this is the client_id on syncable rows
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform       text        NOT NULL
        CONSTRAINT devices_platform_ck CHECK (platform IN ('ios','android','web')),
    model          text            NULL,
    os_version     text            NULL,
    app_version    text            NULL,
    push_token     text            NULL,
    last_sync_seq  bigint      NOT NULL DEFAULT 0, -- last cursor this device acknowledged
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX devices_user_idx ON devices (user_id);

ALTER TABLE refresh_tokens
    ADD CONSTRAINT refresh_tokens_device_fk
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL;
```

---

## 4. Content: shared vocabulary, exercises, bands

```sql
-- One family vocabulary, shared by skills and exercises. See deviation D-1.
CREATE TABLE families (
    id          uuid        PRIMARY KEY,
    slug        text        NOT NULL UNIQUE,
    name        text        NOT NULL,
    order_index smallint    NOT NULL DEFAULT 0
    -- push, pull, core, legs, handstand, dynamic, mobility
);

CREATE TABLE exercises (
    id              uuid     PRIMARY KEY,
    slug            text     NOT NULL UNIQUE,
    name            text     NOT NULL,
    aka             text[]   NOT NULL DEFAULT '{}',
    family_id       uuid     NOT NULL REFERENCES families(id),

    default_measure text     NOT NULL
        CONSTRAINT exercises_default_measure_ck
        CHECK (default_measure IN ('reps','hold_seconds','distance_m','none')),

    -- How load_kg on a set element is to be read for this exercise.
    load_semantics  text     NOT NULL DEFAULT 'added'
        CONSTRAINT exercises_load_semantics_ck
        CHECK (load_semantics IN ('added','external','none')),
        -- added    : bodyweight movement, load_kg is extra weight (negative = counterweight)
        -- external : the implement is the load (e.g. a weighted dip belt-free accessory)
        -- none     : load is meaningless (mobility drills)

    is_bodyweight   boolean  NOT NULL DEFAULT true,
    unilateral      boolean  NOT NULL DEFAULT false,
    tempo_applicable boolean NOT NULL DEFAULT true,
    equipment       text[]   NOT NULL DEFAULT '{}',   -- bar, rings, parallettes, floor, wall
    summary         text     NOT NULL DEFAULT '',
    cues            text[]   NOT NULL DEFAULT '{}',
    common_faults   text[]   NOT NULL DEFAULT '{}',

    status          text     NOT NULL DEFAULT 'active'
        CONSTRAINT exercises_status_ck
        CHECK (status IN ('active','draft_placeholder','retired')),
    content_version_id uuid  NOT NULL REFERENCES content_versions(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX exercises_family_idx ON exercises (family_id) WHERE status = 'active';
CREATE INDEX exercises_name_trgm  ON exercises USING gin (name gin_trgm_ops);
CREATE INDEX exercises_aka_idx    ON exercises USING gin (aka);

CREATE TABLE bands (
    id                uuid        PRIMARY KEY,
    owner_user_id     uuid            NULL REFERENCES users(id) ON DELETE CASCADE,  -- NULL = global catalogue
    brand             text        NOT NULL,
    colour_label      text        NOT NULL,
    resistance_min_kg numeric(5,2) NOT NULL,
    resistance_max_kg numeric(5,2) NOT NULL,
    length_cm         numeric(6,1)    NULL,
    thickness_mm      numeric(5,1)    NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz     NULL,

    CONSTRAINT bands_resistance_ck CHECK (resistance_min_kg <= resistance_max_kg),
    CONSTRAINT bands_resistance_nonneg_ck CHECK (resistance_min_kg >= 0)
);
-- NULLS NOT DISTINCT so two global entries cannot collide on the same triple.
CREATE UNIQUE INDEX bands_identity_uk
    ON bands (owner_user_id, brand, colour_label) NULLS NOT DISTINCT
    WHERE deleted_at IS NULL;
```

---

## 5. Content: the skill graph

```sql
CREATE TABLE skills (
    id              uuid     PRIMARY KEY,
    slug            text     NOT NULL UNIQUE,
    family_id       uuid     NOT NULL REFERENCES families(id),
    name            text     NOT NULL,
    aka             text[]   NOT NULL DEFAULT '{}',
    difficulty_tier smallint NOT NULL
        CONSTRAINT skills_difficulty_tier_ck CHECK (difficulty_tier BETWEEN 1 AND 10),
    is_milestone    boolean  NOT NULL DEFAULT false,
    summary         text     NOT NULL DEFAULT '',
    primary_muscles text[]   NOT NULL DEFAULT '{}',
    common_faults   text[]   NOT NULL DEFAULT '{}',

    -- Map layout, authored in content, not force-directed at runtime (§3.5)
    map_constellation text       NULL,
    map_x             numeric(8,2) NULL,
    map_y             numeric(8,2) NULL,

    status          text     NOT NULL DEFAULT 'active'
        CONSTRAINT skills_status_ck CHECK (status IN ('active','draft_placeholder','retired')),
    content_version_id uuid  NOT NULL REFERENCES content_versions(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    -- A milestone must be placed on the map; a non-milestone may be
    CONSTRAINT skills_milestone_placed_ck
        CHECK (NOT is_milestone OR (map_constellation IS NOT NULL AND map_x IS NOT NULL AND map_y IS NOT NULL))
);
CREATE INDEX skills_family_idx ON skills (family_id);
CREATE INDEX skills_name_trgm  ON skills USING gin (name gin_trgm_ops);

CREATE TABLE skill_levels (
    id              uuid     PRIMARY KEY,
    skill_id        uuid     NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    order_index     smallint NOT NULL CHECK (order_index >= 0),
    slug            text     NOT NULL,           -- unique within the skill
    name            text     NOT NULL,
    description     text     NOT NULL DEFAULT '',
    unlock_criteria jsonb    NOT NULL DEFAULT '{}'::jsonb
        CONSTRAINT skill_levels_criteria_object_ck CHECK (jsonb_typeof(unlock_criteria) = 'object'),
    est_weeks_from_prev smallint NULL CHECK (est_weeks_from_prev IS NULL OR est_weeks_from_prev > 0),

    UNIQUE (skill_id, slug),
    UNIQUE (skill_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    UNIQUE (id, skill_id)                      -- lets children pin both
);
-- The API addresses levels as skill-slug/level-slug; this is the lookup index.
CREATE INDEX skill_levels_skill_idx ON skill_levels (skill_id, order_index);

CREATE TABLE skill_edges (
    from_skill_level_id uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    to_skill_level_id   uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    relation            text NOT NULL
        CONSTRAINT skill_edges_relation_ck
        CHECK (relation IN ('prerequisite','recommended','alternative','antagonist')),
    weight              numeric(4,2) NOT NULL DEFAULT 1.0 CHECK (weight > 0),

    PRIMARY KEY (from_skill_level_id, to_skill_level_id, relation),
    CONSTRAINT skill_edges_no_self_ck CHECK (from_skill_level_id <> to_skill_level_id)
);
CREATE INDEX skill_edges_to_idx ON skill_edges (to_skill_level_id, relation);

-- Acyclicity over relation='prerequisite' is NOT a database constraint.
-- It is checked by cmd/contentlint (topological sort) before seeding, and
-- re-checked inside the seed transaction with a recursive CTE, which aborts
-- the transaction on a cycle. A database-level check would need a trigger with
-- a full graph walk on every row.

CREATE TABLE skill_level_exercises (
    skill_level_id uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    exercise_id    uuid NOT NULL REFERENCES exercises(id)    ON DELETE CASCADE,
    role           text NOT NULL
        CONSTRAINT skill_level_exercises_role_ck
        CHECK (role IN ('primary_test','progression','accessory','prehab')),
    order_index    smallint NOT NULL DEFAULT 0,

    PRIMARY KEY (skill_level_id, exercise_id, role)
);
-- Exactly one primary_test per level; contentlint also asserts it exists.
CREATE UNIQUE INDEX skill_level_primary_test_uk
    ON skill_level_exercises (skill_level_id)
    WHERE role = 'primary_test';
```

### 5.1 Injury content

```sql
CREATE TABLE skill_injury_risks (
    id            uuid     PRIMARY KEY,
    skill_id      uuid     NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    region        text     NOT NULL,      -- elbow, shoulder, wrist, lumbar, ...
    name          text     NOT NULL,
    description   text     NOT NULL DEFAULT '',
    risk_factors  text[]   NOT NULL DEFAULT '{}',
    early_signs   text[]   NOT NULL DEFAULT '{}',
    disclaimer    text     NOT NULL DEFAULT 'educational_only'
        CONSTRAINT skill_injury_risks_disclaimer_ck CHECK (disclaimer = 'educational_only'),
    content_version_id uuid NOT NULL REFERENCES content_versions(id),

    UNIQUE (skill_id, region, name)
);

CREATE TABLE injury_prehab_exercises (
    risk_id     uuid NOT NULL REFERENCES skill_injury_risks(id) ON DELETE CASCADE,
    exercise_id uuid NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    order_index smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (risk_id, exercise_id)
);
```

The `disclaimer` column is constrained to a single value on purpose. It cannot
be omitted, so no API payload can carry injury content without it (ADR 0003).

### 5.2 Content versioning

```sql
CREATE TABLE content_versions (
    id          uuid        PRIMARY KEY,
    checksum    bytea       NOT NULL UNIQUE,   -- sha256 over the normalised content tree
    git_sha     text            NULL,
    item_counts jsonb       NOT NULL DEFAULT '{}'::jsonb,
    applied_at  timestamptz NOT NULL DEFAULT now(),
    applied_by  text            NULL
);
```

`GET /v1/skills` sets `ETag: "<checksum hex>"`. A client with a matching
`If-None-Match` gets 304 and never re-downloads an 80-node graph.

---

## 6. Media

```sql
CREATE TABLE media_assets (
    id            uuid        PRIMARY KEY,
    owner_user_id uuid            NULL REFERENCES users(id) ON DELETE CASCADE,  -- NULL = content media
    kind          text        NOT NULL
        CONSTRAINT media_assets_kind_ck CHECK (kind IN ('image','video')),
    storage_key   text        NOT NULL UNIQUE,
    mime          text        NOT NULL,
    bytes         bigint      NOT NULL CHECK (bytes > 0),
    width         int             NULL,
    height        int             NULL,
    duration_s    numeric(7,2)    NULL,     -- video only; see ADR 0002 §4
    poster_key    text            NULL,     -- video only
    checksum_sha256 bytea         NULL,
    status        text        NOT NULL DEFAULT 'pending'
        CONSTRAINT media_assets_status_ck CHECK (status IN ('pending','ready','failed','quarantined')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    ready_at      timestamptz     NULL,
    deleted_at    timestamptz     NULL,

    CONSTRAINT media_assets_video_fields_ck
        CHECK (kind = 'video' OR (duration_s IS NULL AND poster_key IS NULL))
);
CREATE INDEX media_assets_owner_idx ON media_assets (owner_user_id) WHERE deleted_at IS NULL;

CREATE TABLE skill_media    (skill_id    uuid NOT NULL REFERENCES skills(id)    ON DELETE CASCADE,
                             media_id    uuid NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
                             role text NOT NULL CHECK (role IN ('hero','demo','fault')),
                             order_index smallint NOT NULL DEFAULT 0,
                             PRIMARY KEY (skill_id, media_id));

CREATE TABLE exercise_media (exercise_id uuid NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
                             media_id    uuid NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
                             role text NOT NULL CHECK (role IN ('hero','demo','fault')),
                             order_index smallint NOT NULL DEFAULT 0,
                             PRIMARY KEY (exercise_id, media_id));
```

Form-check media attaches to a set element — see §7.5.

---

## 7. Logging — the core

### 7.1 Sessions

```sql
CREATE TABLE workout_sessions (
    id                uuid        PRIMARY KEY,
    user_id           uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at        timestamptz NOT NULL,
    ended_at          timestamptz     NULL,
    timezone          text        NOT NULL,          -- IANA, e.g. Europe/Zurich
    local_date        date        NOT NULL,          -- see §1.2
    title             text        NOT NULL DEFAULT '',
    notes             text        NOT NULL DEFAULT '',
    perceived_fatigue smallint        NULL
        CONSTRAINT workout_sessions_fatigue_ck CHECK (perceived_fatigue BETWEEN 1 AND 10),
    bodyweight_kg     numeric(5,2)    NULL CHECK (bodyweight_kg IS NULL OR bodyweight_kg > 0),
    status            text        NOT NULL DEFAULT 'draft'
        CONSTRAINT workout_sessions_status_ck CHECK (status IN ('draft','completed','abandoned')),
    is_rest_day       boolean     NOT NULL DEFAULT false,   -- see D-4
    template_id       uuid            NULL,
    completed_at      timestamptz     NULL,

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,

    UNIQUE (id, user_id),
    CONSTRAINT workout_sessions_interval_ck CHECK (ended_at IS NULL OR ended_at >= started_at),
    CONSTRAINT workout_sessions_completed_ck
        CHECK ((status = 'completed') = (completed_at IS NOT NULL))
);
CREATE INDEX workout_sessions_user_date_idx
    ON workout_sessions (user_id, local_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX workout_sessions_sync_idx
    ON workout_sessions (user_id, server_seq);
```

### 7.2 Blocks

```sql
CREATE TABLE session_blocks (
    id             uuid     PRIMARY KEY,
    user_id        uuid     NOT NULL,
    session_id     uuid     NOT NULL,
    order_index    int      NOT NULL CHECK (order_index >= 0),
    kind           text     NOT NULL DEFAULT 'straight'
        CONSTRAINT session_blocks_kind_ck
        CHECK (kind IN ('straight','superset','circuit','emom','amrap')),
    rounds_planned smallint     NULL CHECK (rounds_planned IS NULL OR rounds_planned > 0),
    rounds_done    smallint     NULL CHECK (rounds_done    IS NULL OR rounds_done    >= 0),
    interval_s     int          NULL,   -- emom window / amrap cap
    notes          text     NOT NULL DEFAULT '',

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,

    UNIQUE (id, user_id),
    UNIQUE (session_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (session_id, user_id)
        REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE,
    CONSTRAINT session_blocks_interval_ck
        CHECK (kind IN ('emom','amrap') OR interval_s IS NULL)
);
CREATE INDEX session_blocks_sync_idx ON session_blocks (user_id, server_seq);
```

### 7.3 Set entries

```sql
CREATE TABLE set_entries (
    id          uuid     PRIMARY KEY,
    user_id     uuid     NOT NULL,
    session_id  uuid     NOT NULL,          -- denormalised one level; see D-2
    block_id    uuid     NOT NULL,
    order_index int      NOT NULL CHECK (order_index >= 0),
    round_index smallint     NULL,          -- which circuit round this belongs to

    kind        text     NOT NULL DEFAULT 'working'
        CONSTRAINT set_entries_kind_ck
        CHECK (kind IN ('working','warmup','backoff','drop','cluster','test')),
    is_planned  boolean  NOT NULL DEFAULT false,

    rest_after_planned_s int NULL CHECK (rest_after_planned_s IS NULL OR rest_after_planned_s >= 0),
    rest_after_actual_s  int NULL CHECK (rest_after_actual_s  IS NULL OR rest_after_actual_s  >= 0),

    rpe         numeric(3,1) NULL CHECK (rpe IS NULL OR rpe BETWEEN 1.0 AND 10.0),
    rir         smallint     NULL CHECK (rir IS NULL OR rir BETWEEN 0 AND 10),
    completed_at timestamptz NULL,
    notes       text     NOT NULL DEFAULT '',

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,

    UNIQUE (id, user_id),
    UNIQUE (block_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (block_id, user_id)   REFERENCES session_blocks   (id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (session_id, user_id) REFERENCES workout_sessions (id, user_id) ON DELETE CASCADE,
    -- A planned set has not happened yet; a performed set has a timestamp.
    CONSTRAINT set_entries_planned_ck CHECK (NOT is_planned OR completed_at IS NULL)
);
CREATE INDEX set_entries_session_idx ON set_entries (session_id) WHERE deleted_at IS NULL;
CREATE INDEX set_entries_sync_idx    ON set_entries (user_id, server_seq);
```

### 7.4 Set elements — the combo mechanism

```sql
CREATE TABLE set_elements (
    id            uuid     PRIMARY KEY,
    user_id       uuid     NOT NULL,
    session_id    uuid     NOT NULL,        -- denormalised two levels; see D-2
    set_entry_id  uuid     NOT NULL,
    order_index   int      NOT NULL CHECK (order_index >= 0),
    exercise_id   uuid     NOT NULL REFERENCES exercises(id),

    measure       text     NOT NULL
        CONSTRAINT set_elements_measure_ck
        CHECK (measure IN ('reps','hold_seconds','distance_m','none')),
    reps          int          NULL CHECK (reps         IS NULL OR reps         >= 0),
    hold_seconds  numeric(7,2) NULL CHECK (hold_seconds IS NULL OR hold_seconds >= 0),
    distance_m    numeric(7,2) NULL CHECK (distance_m   IS NULL OR distance_m   >= 0),

    tempo         text         NULL
        CONSTRAINT set_elements_tempo_ck CHECK (tempo IS NULL OR tempo ~ '^[0-9X]{4}$'),  -- "30X1"
    load_kg       numeric(6,2) NOT NULL DEFAULT 0,   -- negative = counterweight / assistance

    is_eccentric_only boolean NOT NULL DEFAULT false,
    is_partial_rom    boolean NOT NULL DEFAULT false,
    rom_note      text         NULL,
    form_quality  smallint     NULL
        CONSTRAINT set_elements_form_quality_ck CHECK (form_quality BETWEEN 1 AND 5),
    failed        boolean  NOT NULL DEFAULT false,
    -- Denormalised from set_element_assistance for the evaluator's hot path; D-3
    assistance_class text  NOT NULL DEFAULT 'unassisted'
        CONSTRAINT set_elements_assistance_class_ck
        CHECK (assistance_class IN ('unassisted','assisted','loaded')),

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,

    UNIQUE (id, user_id),
    UNIQUE (set_entry_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (set_entry_id, user_id) REFERENCES set_entries       (id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (session_id, user_id)   REFERENCES workout_sessions  (id, user_id) ON DELETE CASCADE,

    -- At most one measured value, and it must be the one `measure` names.
    -- Deliberately permissive about NULL so a planned-but-unperformed element
    -- is representable without a second table.
    CONSTRAINT set_elements_single_value_ck
        CHECK (num_nonnulls(reps, hold_seconds, distance_m) <= 1),
    CONSTRAINT set_elements_value_matches_measure_ck
        CHECK ((reps         IS NULL OR measure = 'reps')
           AND (hold_seconds IS NULL OR measure = 'hold_seconds')
           AND (distance_m   IS NULL OR measure = 'distance_m')),
    CONSTRAINT set_elements_rom_note_ck
        CHECK (rom_note IS NULL OR is_partial_rom)
);

-- The evaluator's hot path: "this user's elements for exercise X, recent first".
CREATE INDEX set_elements_user_exercise_idx
    ON set_elements (user_id, exercise_id, session_id)
    WHERE deleted_at IS NULL;
CREATE INDEX set_elements_entry_idx ON set_elements (set_entry_id) WHERE deleted_at IS NULL;
CREATE INDEX set_elements_sync_idx  ON set_elements (user_id, server_seq);
```

**The invariant this table exists to protect:** a plain set of 8 pull-ups is one
`set_entry` with one `set_element`. A planche combo is one `set_entry` with
three ordered `set_elements`. Nothing in the API, the store, or the iOS client
is allowed a second code path for the simple case.

### 7.5 Assistance and form-check media

```sql
CREATE TABLE set_element_assistance (
    id             uuid     PRIMARY KEY,
    user_id        uuid     NOT NULL,
    set_element_id uuid     NOT NULL,
    type           text     NOT NULL
        CONSTRAINT set_element_assistance_type_ck
        CHECK (type IN ('none','band','partner','machine','incline','counterweight','foot_support')),
    band_id        uuid         NULL REFERENCES bands(id) ON DELETE RESTRICT,
    band_count     smallint NOT NULL DEFAULT 0 CHECK (band_count >= 0),
    anchor         text         NULL
        CONSTRAINT set_element_assistance_anchor_ck
        CHECK (anchor IS NULL OR anchor IN ('overhead','under_foot','under_knee','hip','other')),
    estimated_assist_kg numeric(6,2) NULL CHECK (estimated_assist_kg IS NULL OR estimated_assist_kg >= 0),
    note           text     NOT NULL DEFAULT '',

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,

    UNIQUE (id, user_id),
    FOREIGN KEY (set_element_id, user_id) REFERENCES set_elements (id, user_id) ON DELETE CASCADE,
    CONSTRAINT set_element_assistance_band_ck
        CHECK ((type = 'band') = (band_id IS NOT NULL AND band_count > 0))
);
CREATE INDEX set_element_assistance_element_idx ON set_element_assistance (set_element_id);

CREATE TABLE set_element_media (
    set_element_id uuid NOT NULL,
    user_id        uuid NOT NULL,
    media_id       uuid NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    order_index    smallint NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (set_element_id, media_id),
    FOREIGN KEY (set_element_id, user_id) REFERENCES set_elements (id, user_id) ON DELETE CASCADE
);
```

Note `set_element_assistance` is one-to-one in practice but modelled as a child
table, exactly as the brief specifies. Keeping it separate means an element with
no assistance stores no row at all, which is the overwhelmingly common case.

---

## 8. Templates

Templates mirror the logging shape so that "start a session from this template"
is a structural copy, not a translation.

```sql
CREATE TABLE workout_templates (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          text NOT NULL,
    description   text NOT NULL DEFAULT '',
    visibility    text NOT NULL DEFAULT 'private'
        CONSTRAINT workout_templates_visibility_ck
        CHECK (visibility IN ('private','unlisted','public')),   -- only 'private' reachable in v1
    source_template_id uuid NULL REFERENCES workout_templates(id) ON DELETE SET NULL,
    est_duration_min smallint NULL,

    client_id         uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    updated_at        timestamptz NOT NULL,
    server_updated_at timestamptz NOT NULL DEFAULT now(),
    server_seq        bigint      NOT NULL,
    deleted_at        timestamptz     NULL,
    UNIQUE (id, user_id)
);

CREATE TABLE template_blocks (
    id uuid PRIMARY KEY, user_id uuid NOT NULL, template_id uuid NOT NULL,
    order_index int NOT NULL, kind text NOT NULL, rounds_planned smallint NULL,
    interval_s int NULL, notes text NOT NULL DEFAULT '',
    -- + the same sync columns
    UNIQUE (id, user_id),
    UNIQUE (template_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_id, user_id) REFERENCES workout_templates (id, user_id) ON DELETE CASCADE
);

CREATE TABLE template_set_entries (
    id uuid PRIMARY KEY, user_id uuid NOT NULL, template_block_id uuid NOT NULL,
    order_index int NOT NULL, kind text NOT NULL DEFAULT 'working',
    rest_after_planned_s int NULL, target_rpe numeric(3,1) NULL, notes text NOT NULL DEFAULT '',
    UNIQUE (id, user_id),
    UNIQUE (template_block_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_block_id, user_id) REFERENCES template_blocks (id, user_id) ON DELETE CASCADE
);

CREATE TABLE template_set_elements (
    id uuid PRIMARY KEY, user_id uuid NOT NULL, template_set_entry_id uuid NOT NULL,
    order_index int NOT NULL, exercise_id uuid NOT NULL REFERENCES exercises(id),
    measure text NOT NULL,
    target_reps int NULL, target_hold_seconds numeric(7,2) NULL, target_distance_m numeric(7,2) NULL,
    target_load_kg numeric(6,2) NOT NULL DEFAULT 0, tempo text NULL,
    UNIQUE (id, user_id),
    UNIQUE (template_set_entry_id, order_index) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (template_set_entry_id, user_id) REFERENCES template_set_entries (id, user_id) ON DELETE CASCADE
);

ALTER TABLE workout_sessions
    ADD CONSTRAINT workout_sessions_template_fk
    FOREIGN KEY (template_id) REFERENCES workout_templates(id) ON DELETE SET NULL;
```

---

## 9. Progress

### 9.1 Skill state

```sql
CREATE TABLE user_skill_states (
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_level_id uuid NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    state          text NOT NULL DEFAULT 'locked'
        CONSTRAINT user_skill_states_state_ck
        CHECK (state IN ('locked','available','in_progress','unlocked')),

    best_value     numeric(9,2) NULL,
    best_unit      text         NULL
        CONSTRAINT user_skill_states_best_unit_ck
        CHECK (best_unit IS NULL OR best_unit IN ('reps','hold_seconds','distance_m')),
    best_at        timestamptz  NULL,
    -- Informational only. Set when the primary_test has not been performed for
    -- a while. It never changes `state` — see ADR 0003 §6.
    stale_since    timestamptz  NULL,

    first_achieved_at     timestamptz NULL,
    evidence_set_entry_id uuid        NULL,
    verification   text NOT NULL DEFAULT 'auto'
        CONSTRAINT user_skill_states_verification_ck
        CHECK (verification IN ('auto','self_attested','coach')),
    attempts_count int NOT NULL DEFAULT 0 CHECK (attempts_count >= 0),
    updated_at     timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, skill_level_id),
    FOREIGN KEY (evidence_set_entry_id, user_id)
        REFERENCES set_entries (id, user_id) ON DELETE SET NULL,
    CONSTRAINT user_skill_states_unlocked_ck
        CHECK ((state = 'unlocked') = (first_achieved_at IS NOT NULL))
);
CREATE INDEX user_skill_states_user_idx ON user_skill_states (user_id, state);
```

A `BEFORE UPDATE` trigger enforces monotonicity: once `state = 'unlocked'`, it
cannot move to any other value, and `first_achieved_at` cannot be cleared. The
map is a record of achievement, not of current form.

```sql
CREATE TABLE skill_unlock_events (
    id             uuid        PRIMARY KEY,
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_level_id uuid        NOT NULL REFERENCES skill_levels(id) ON DELETE CASCADE,
    occurred_at    timestamptz NOT NULL DEFAULT now(),
    session_id     uuid            NULL,
    evidence_set_entry_id uuid     NULL,
    verification   text        NOT NULL
        CONSTRAINT skill_unlock_events_verification_ck
        CHECK (verification IN ('auto','self_attested','coach')),
    criteria_snapshot jsonb    NOT NULL,   -- the criteria as they read at unlock time
    UNIQUE (user_id, skill_level_id)       -- an unlock happens once
);
```

`criteria_snapshot` matters: content evolves, and without it a year-old unlock
becomes unexplainable. The table is append-only, enforced by a rule that raises
on `UPDATE` and `DELETE`.

### 9.2 XP, streaks, badges

```sql
CREATE TABLE user_xp_events (
    id          uuid        PRIMARY KEY,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source      text        NOT NULL
        CONSTRAINT user_xp_events_source_ck
        CHECK (source IN ('session_completed','skill_unlocked','badge_earned',
                          'streak_milestone','plan_adherence','deload_completed')),
    amount      int         NOT NULL CHECK (amount > 0),
    ref_type    text            NULL,
    ref_id      uuid            NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    -- Awarding the same thing twice is the bug this prevents.
    UNIQUE (user_id, source, ref_type, ref_id)
);
CREATE INDEX user_xp_events_user_time_idx ON user_xp_events (user_id, occurred_at DESC);

-- One row per user per local day. Records the day's INTENT, not just activity,
-- which is what lets a planned rest day maintain a streak (ADR 0003 §1).
CREATE TABLE user_training_days (
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    local_date    date NOT NULL,
    session_count smallint NOT NULL DEFAULT 0,
    had_session   boolean  NOT NULL DEFAULT false,
    planned_rest  boolean  NOT NULL DEFAULT false,
    deload        boolean  NOT NULL DEFAULT false,
    freeze_used   boolean  NOT NULL DEFAULT false,
    counts_for_streak boolean NOT NULL
        GENERATED ALWAYS AS (had_session OR planned_rest OR deload OR freeze_used) STORED,
    PRIMARY KEY (user_id, local_date)
);

CREATE TABLE user_streaks (
    user_id           uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_days      int  NOT NULL DEFAULT 0 CHECK (current_days >= 0),
    longest_days      int  NOT NULL DEFAULT 0 CHECK (longest_days >= 0),
    last_counted_date date NULL,
    freeze_credits    smallint NOT NULL DEFAULT 2 CHECK (freeze_credits >= 0),
    freeze_credits_updated_at timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE badges (
    id          uuid PRIMARY KEY,
    slug        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    tier        smallint NOT NULL DEFAULT 1,
    criteria    jsonb NOT NULL DEFAULT '{}'::jsonb,
    content_version_id uuid NOT NULL REFERENCES content_versions(id)
);

CREATE TABLE user_badges (
    user_id  uuid NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    badge_id uuid NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at timestamptz NOT NULL DEFAULT now(),
    ref_id   uuid NULL,
    PRIMARY KEY (user_id, badge_id)
);
```

### 9.3 Personal bests (derived, maintained transactionally)

```sql
-- Maintained inside the same transaction as POST /sessions/{id}/complete.
-- Both the unlock evaluator and the stats screen read it; recomputing either
-- from raw set_elements on every request is the query we are avoiding.
CREATE TABLE user_exercise_bests (
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id      uuid NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    measure          text NOT NULL,
    assistance_class text NOT NULL,        -- unassisted | assisted | loaded
    best_value       numeric(9,2) NOT NULL,
    best_load_kg     numeric(6,2) NOT NULL DEFAULT 0,
    achieved_at      timestamptz  NOT NULL,
    evidence_set_element_id uuid NULL,
    occurrences_30d  int NOT NULL DEFAULT 0,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, exercise_id, measure, assistance_class)
);
```

This table is a cache with a single writer. It must be rebuildable from
`set_elements` alone, and Phase 3 ships a `make rebuild-bests` path plus an
integration test that asserts the cache equals the recomputation.

---

## 10. Sync bookkeeping

```sql
CREATE TABLE idempotency_keys (
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key           text        NOT NULL,
    request_fingerprint bytea NOT NULL,     -- sha256 of method + path + body
    state         text        NOT NULL DEFAULT 'in_progress'
        CONSTRAINT idempotency_keys_state_ck CHECK (state IN ('in_progress','completed','failed')),
    response_status int           NULL,
    response_body jsonb           NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    completed_at  timestamptz     NULL,
    expires_at    timestamptz NOT NULL DEFAULT now() + interval '24 hours',
    PRIMARY KEY (user_id, key)
);
CREATE INDEX idempotency_keys_expiry_idx ON idempotency_keys (expires_at);
```

A replay with the same key but a different `request_fingerprint` is a client
bug and gets `422`, not a silent success.

**On the sync cursor.** A single global sequence would produce holes: if
transaction A takes seq 5 and commits after transaction B took seq 6, a client
whose cursor is already past 6 never sees 5. `next_sync_seq(user_id)` avoids
this by taking the counter from the user's own row, which serialises all of that
user's concurrent write transactions on a row lock. Since a user's devices
number in the single digits, the contention is irrelevant and the guarantee is
exact: the cursor is a total order over that user's changes.

`GET /v1/sync?cursor=N` returns rows with `server_seq > N` per table, ordered,
plus the new cursor. `deleted_at IS NOT NULL` rows are included — tombstones are
how a deletion propagates.

---

## 11. Query shapes the schema is designed for

| Query | Path |
|---|---|
| Session detail (logger opening a workout) | `workout_sessions` → `session_blocks` → `set_entries` → `set_elements` (+ assistance), 5 indexed reads |
| "Repeat last set" pre-fill | `set_elements (user_id, exercise_id, session_id)` index, order by session date desc, limit 1 |
| Unlock evaluation on completion | for each candidate level: `set_elements` by `(user_id, exercise_id)` filtered on the window in `within_days`, joined to `set_entries` for RPE |
| `/v1/me/skill-map` | full `skills`+`skill_levels`+`skill_edges` (ETag-cached, content-versioned) LEFT JOIN `user_skill_states` by user |
| Delta sync pull | per table: `WHERE user_id = $1 AND server_seq > $2 ORDER BY server_seq LIMIT $3` |
| Streak | `user_training_days` by `(user_id, local_date)` descending walk |

---

## 12. Deviations from the brief — each needs your yes or no

**D-1 — `skill_families` renamed to `families`, shared with exercises.**
The brief names the table `skill_families`. Exercises need the same vocabulary
(push/pull/core/…), and two parallel family tables would drift. Proposal: one
`families` table referenced by both. Say the word and I will keep the original
name with exercises pointing at it.

**D-2 — `session_id` denormalised onto `set_entries` and `set_elements`.**
Not in the brief's column lists. It removes one and two join levels from the
hottest read paths, and the composite FK makes it impossible for it to disagree
with the block's session. Cost: one extra uuid per row.

**D-3 — `assistance_class` denormalised onto `set_elements`.**
Unlock criteria filter on `assistance: none` constantly. Without this column,
every criteria evaluation LEFT JOINs `set_element_assistance` for rows that
mostly do not exist. It is derived and maintained by the same code that writes
the assistance row.

**D-4 — `workout_sessions.is_rest_day` and the `user_training_days` table.**
Neither is in §3. Both exist to satisfy ADR 0003's requirement that a planned
rest day maintains a streak. Without a place to record intent, "rest day" is
indistinguishable from "did not train".

**D-5 — `server_seq` and `server_updated_at` in addition to `updated_at`.**
The brief specifies `updated_at` for conflict resolution. I have kept it (it is
the client's claim and part of the LWW rule) but added a server-assigned
monotonic sequence, because `updated_at` cannot safely drive a sync cursor —
clock skew across devices would silently drop rows.

**D-6 — `devices` table, with `client_id` as an FK to it.**
The brief has `client_id` on `workout_sessions` without saying what it points
at. Making it a registered device gives us per-device sync cursors, push tokens
and a "sign out this device" story for free.

**D-7 — Criteria acyclicity is not a database constraint.**
Enforced in `contentlint` and re-verified by a recursive CTE inside the seed
transaction. A trigger-based check would walk the whole graph on every row.

**D-8 — `user_exercise_bests` as a maintained cache.**
Not in the brief. Justified in §9.3; it must stay rebuildable and be tested
against a recomputation.

---

## 13. Open questions — I need answers before writing migrations

1. **`set_entries.round_index`.** For a circuit performed for 3 rounds, is each
   round a separate `set_entry` (my proposal: yes, with `round_index` marking
   which round), or one entry with a rounds count? The former logs reality; the
   latter is fewer rows.

2. **Bodyweight.** It sits on `workout_sessions` per the brief. Do you also want
   a standalone `user_bodyweight_log` so weight can be tracked on non-training
   days? It matters for weighted-vs-bodyweight PR maths. I would add it, but it
   is scope.

3. **`load_kg` sign convention.** The brief says negative means counterweight.
   That collides with `set_element_assistance.estimated_assist_kg`, which is
   positive. Proposal: `load_kg` is strictly *added* external load and never
   negative; all assistance lives in the assistance row. Cleaner, but it is a
   change to your stated model — your call.

4. **Combos and rest.** Rest lives on `set_entry`. Inside a three-element combo
   there is by definition no rest. Confirm that is right, or elements need their
   own transition/rest field.

5. **`skill_edges` between levels vs between skills.** §3.3 puts edges between
   *levels*, which is more precise and what I have modelled. It also means the
   map must render skill nodes by aggregating their levels' edges. Confirm.

6. **Account deletion.** `ON DELETE CASCADE` everywhere gives a hard delete.
   Swiss/EU expectation is usually a grace period — hence
   `users.status = 'deletion_pending'`. Confirm you want soft-then-hard with a
   window (my proposal: 30 days), and I will add the reaper job to Phase 2.

7. **`est_weeks_from_prev`.** It is content, and content is shared by all users.
   Is it meant as a display hint only, or should the map personalise it later
   from the user's actual pace? If the latter, it wants a companion table, not
   now but the column should not be treated as authoritative in the client.

---

## 14. Verification already done

This is not a schema sketched on paper. The whole of §2–§10 was materialised in
dependency order and applied to **PostgreSQL 16.13**; it creates cleanly, every
constraint and index is legal, and the following fifteen behaviours were
asserted against live data:

| # | Assertion | Result |
|---|---|---|
| 1 | A three-element planche combo lives under **one** `set_entry` | 3 elements, one entry |
| 2 | A child row naming another user's parent is **rejected** by the composite FK | rejected |
| 3 | `measure='reps'` with `hold_seconds` set is **rejected** | rejected |
| 4 | Malformed tempo `3-0-X-1` is **rejected** | rejected |
| 5 | Valid tempo `30X1` is accepted | accepted |
| 6 | Swapping two `order_index` values inside one transaction works under `SET CONSTRAINTS ALL DEFERRED` | swapped |
| 7 | `touch_sync` bumps `server_seq` from the per-user counter | seq and counter agree |
| 8 | `type='band'` without a `band_id` is **rejected** | rejected |
| 9 | Two global bands with the same brand+colour collide (`NULLS NOT DISTINCT`) | rejected |
| 10 | `status='completed'` without `completed_at` is **rejected** | rejected |
| 11 | `counts_for_streak` is true for a planned rest day with no session | true |
| 12 | `state='unlocked'` without `first_achieved_at` is **rejected** | rejected |
| 13 | A self-referencing skill edge is **rejected** | rejected |
| 14 | A second `primary_test` on one level is **rejected** | rejected |
| 15 | A milestone skill with no map coordinates is **rejected** | rejected |

One defect was found and fixed in the process: the first version of
`touch_sync` fired only `BEFORE UPDATE`, which left `server_seq` at whatever the
client sent on INSERT. It now fires on INSERT as well and branches on `TG_OP`,
since `OLD` does not exist during an INSERT.

## 15. What Phase 1 does with this, once approved

1. Migrations `0001_extensions` … `0009_progress`, in dependency order, each
   with a working `-- +goose Down`.
2. `db/queries/*.sql` for every read the API surface in §4 implies, and
   `make sqlc` wired into CI so a stale checkout fails.
3. `content/schema/*.schema.json` mirroring §4–§5, plus `docs/CONTENT_AUTHORING.md`.
4. `cmd/contentlint` with the nine checks listed in its source header.
5. `cmd/seed` with checksum-keyed idempotent upsert in one transaction.
6. Three or four `status: draft_placeholder` skill files, including the
   `front-lever` example from the brief.
7. `make up && make seed` green end to end.
