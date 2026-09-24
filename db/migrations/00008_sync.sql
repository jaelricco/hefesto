-- Sync bookkeeping. See docs/DDL_PROPOSAL.md §10.

-- +goose Up
CREATE TABLE idempotency_keys (
    user_id             uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key                 text        NOT NULL,
    request_fingerprint bytea       NOT NULL,   -- sha256 of method + path + body
    state               text        NOT NULL DEFAULT 'in_progress'
        CONSTRAINT idempotency_keys_state_ck CHECK (state IN ('in_progress','completed','failed')),
    response_status     int             NULL,
    response_body       jsonb           NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    completed_at        timestamptz     NULL,
    expires_at          timestamptz NOT NULL DEFAULT now() + interval '24 hours',
    PRIMARY KEY (user_id, key)
);
CREATE INDEX idempotency_keys_expiry_idx ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE idempotency_keys;
