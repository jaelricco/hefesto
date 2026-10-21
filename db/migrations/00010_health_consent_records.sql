-- A record of each decision on the consent to keep health data: what was
-- decided, when, and to which text (ADR 0031). The decision itself is
-- user_training_profiles.health_data_consent; this is its history.
--
-- It is not health data: a withdrawal keeps it, as the proof of what the
-- athlete agreed to and when that ended. It goes with the account. A client
-- that does not say which text it showed leaves text_version and locale
-- empty.

-- +goose Up
CREATE TABLE health_consent_records (
    id           uuid        PRIMARY KEY,
    user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted      boolean     NOT NULL,
    source       text        NOT NULL
        CONSTRAINT health_consent_records_source_ck CHECK (source IN ('onboarding','change')),
    -- The client's id of a change (POST /v1/me/health-consent).
    change_id    text            NULL
        CONSTRAINT health_consent_records_change_id_ck CHECK (change_id <> ''),
    text_version text            NULL
        CONSTRAINT health_consent_records_text_version_ck CHECK (text_version ~ '^[A-Za-z0-9._-]{1,32}$'),
    locale       text            NULL
        CONSTRAINT health_consent_records_locale_ck CHECK (locale ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$'),
    recorded_at  timestamptz NOT NULL,
    CONSTRAINT health_consent_records_text_ck CHECK ((text_version IS NULL) = (locale IS NULL)),
    CONSTRAINT health_consent_records_change_ck CHECK ((source = 'change') = (change_id IS NOT NULL))
);
CREATE UNIQUE INDEX health_consent_records_change_uk ON health_consent_records (user_id, change_id)
    WHERE change_id IS NOT NULL;
CREATE UNIQUE INDEX health_consent_records_onboarding_uk ON health_consent_records (user_id)
    WHERE source = 'onboarding';
CREATE INDEX health_consent_records_user_time_idx ON health_consent_records (user_id, recorded_at DESC, id DESC);

-- Append-only, like plan_decisions; a cascade from deleting the user
-- reaches this trigger nested inside the FK's own trigger.
-- +goose StatementBegin
CREATE FUNCTION health_consent_records_append_only() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'health_consent_records is append-only (% refused)', TG_OP
        USING ERRCODE = 'check_violation', CONSTRAINT = 'health_consent_records_append_only';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER health_consent_records_append_only BEFORE UPDATE OR DELETE ON health_consent_records
    FOR EACH ROW EXECUTE FUNCTION health_consent_records_append_only();

-- +goose Down
DROP TABLE health_consent_records;
DROP FUNCTION health_consent_records_append_only();
