-- Identity, devices, auth tokens, and the per-user sync counter.
-- See docs/DDL_PROPOSAL.md §2–§3.

-- +goose Up
CREATE TABLE users (
    id                    uuid        PRIMARY KEY,
    email                 citext          NULL UNIQUE,   -- NULL for Apple-only with hidden relay
    email_verified_at     timestamptz     NULL,
    password_hash         text            NULL,          -- argon2id encoded; NULL = no password login
    display_name          text        NOT NULL DEFAULT '',
    locale                text        NOT NULL DEFAULT 'de-CH',
    unit_system           text        NOT NULL DEFAULT 'metric'
        CONSTRAINT users_unit_system_ck CHECK (unit_system IN ('metric','imperial')),
    week_start            smallint    NOT NULL DEFAULT 1  -- ISO: Monday
        CONSTRAINT users_week_start_ck CHECK (week_start BETWEEN 1 AND 7),
    timezone              text        NOT NULL DEFAULT 'Europe/Zurich',
    status                text        NOT NULL DEFAULT 'active'
        CONSTRAINT users_status_ck CHECK (status IN ('active','suspended','deletion_pending')),
    -- Soft-then-hard deletion: the reaper hard-deletes 30 days after this.
    deletion_requested_at timestamptz     NULL,
    sync_seq              bigint      NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    deleted_at            timestamptz     NULL,

    CONSTRAINT users_login_method_ck
        CHECK (password_hash IS NULL OR email IS NOT NULL),
    CONSTRAINT users_deletion_pending_ck
        CHECK ((status = 'deletion_pending') = (deletion_requested_at IS NOT NULL))
);
CREATE INDEX users_deletion_idx ON users (deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;

-- Per-user monotonic sync counter. Serialised by the row lock on users, which
-- is what makes a client cursor safe against holes (DDL §10).
-- +goose StatementBegin
CREATE FUNCTION next_sync_seq(p_user_id uuid) RETURNS bigint
LANGUAGE sql AS $$
    UPDATE users SET sync_seq = sync_seq + 1
    WHERE id = p_user_id
    RETURNING sync_seq;
$$;
-- +goose StatementEnd

-- Applied BEFORE INSERT OR UPDATE on every syncable table. server_seq is
-- assigned unconditionally: whatever a client sends for it is overwritten, so
-- it is never the client's to set.
-- +goose StatementBegin
CREATE FUNCTION touch_sync() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.server_updated_at := now();
    NEW.server_seq := next_sync_seq(NEW.user_id);
    IF NEW.server_seq IS NULL THEN
        RAISE EXCEPTION 'touch_sync: no user % for %', NEW.user_id, TG_TABLE_NAME
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TABLE apple_identities (
    user_id          uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    apple_sub        text        NOT NULL PRIMARY KEY,   -- stable Apple subject
    is_private_email boolean     NOT NULL DEFAULT false,
    linked_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);

CREATE TABLE devices (
    id            uuid        PRIMARY KEY,   -- this is the client_id on syncable rows
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform      text        NOT NULL
        CONSTRAINT devices_platform_ck CHECK (platform IN ('ios','android','web')),
    model         text            NULL,
    os_version    text            NULL,
    app_version   text            NULL,
    push_token    text            NULL,
    last_sync_seq bigint      NOT NULL DEFAULT 0,  -- last cursor this device acknowledged
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX devices_user_idx ON devices (user_id);

-- Rotating refresh tokens with reuse detection. A reused token revokes its
-- whole family.
CREATE TABLE refresh_tokens (
    id             uuid        PRIMARY KEY,
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id      uuid        NOT NULL,
    token_hash     bytea       NOT NULL UNIQUE,   -- sha256 of the opaque token, never the token
    device_id      uuid            NULL REFERENCES devices(id) ON DELETE SET NULL,
    issued_at      timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL,
    used_at        timestamptz     NULL,
    revoked_at     timestamptz     NULL,
    revoked_reason text            NULL
        CONSTRAINT refresh_tokens_revoked_reason_ck
        CHECK (revoked_reason IS NULL OR revoked_reason IN ('rotated','logout','reuse_detected','admin','expired')),
    replaced_by    uuid            NULL REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    user_agent     text            NULL,
    ip             inet            NULL
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
CREATE INDEX email_verifications_user_idx ON email_verifications (user_id);

-- +goose Down
DROP TABLE email_verifications;
DROP TABLE refresh_tokens;
DROP TABLE devices;
DROP TABLE apple_identities;
DROP FUNCTION touch_sync();
DROP FUNCTION next_sync_seq(uuid);
DROP TABLE users;
