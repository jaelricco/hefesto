-- Media assets. Generic schema, images-only UI in v1 (ADR 0002 §4).

-- +goose Up
CREATE TABLE media_assets (
    id              uuid         PRIMARY KEY,
    owner_user_id   uuid             NULL REFERENCES users(id) ON DELETE CASCADE,  -- NULL = content media
    kind            text         NOT NULL
        CONSTRAINT media_assets_kind_ck CHECK (kind IN ('image','video')),
    storage_key     text         NOT NULL UNIQUE,
    mime            text         NOT NULL,
    bytes           bigint       NOT NULL
        CONSTRAINT media_assets_bytes_ck CHECK (bytes > 0),
    width           int              NULL,
    height          int              NULL,
    duration_s      numeric(7,2)     NULL,   -- video only
    poster_key      text             NULL,   -- video only
    checksum_sha256 bytea            NULL,
    status          text         NOT NULL DEFAULT 'pending'
        CONSTRAINT media_assets_status_ck CHECK (status IN ('pending','ready','failed','quarantined')),
    created_at      timestamptz  NOT NULL DEFAULT now(),
    ready_at        timestamptz      NULL,
    deleted_at      timestamptz      NULL,

    CONSTRAINT media_assets_video_fields_ck
        CHECK (kind = 'video' OR (duration_s IS NULL AND poster_key IS NULL))
);
CREATE INDEX media_assets_owner_idx ON media_assets (owner_user_id) WHERE deleted_at IS NULL;

CREATE TABLE skill_media (
    skill_id    uuid     NOT NULL REFERENCES skills(id)       ON DELETE CASCADE,
    media_id    uuid     NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    role        text     NOT NULL
        CONSTRAINT skill_media_role_ck CHECK (role IN ('hero','demo','fault')),
    order_index smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (skill_id, media_id)
);

CREATE TABLE exercise_media (
    exercise_id uuid     NOT NULL REFERENCES exercises(id)    ON DELETE CASCADE,
    media_id    uuid     NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    role        text     NOT NULL
        CONSTRAINT exercise_media_role_ck CHECK (role IN ('hero','demo','fault')),
    order_index smallint NOT NULL DEFAULT 0,
    PRIMARY KEY (exercise_id, media_id)
);

-- +goose Down
DROP TABLE exercise_media;
DROP TABLE skill_media;
DROP TABLE media_assets;
