-- Delta sync, offline push bookkeeping and the bodyweight log.
--
-- server_seq is a per-user total order over writes (next_sync_seq takes it
-- from the user's row, so concurrent writers of one user serialise). The feed
-- pages over it exactly: every unit appears at its latest change.

-- ------------------------------------------------------------------ pull

-- A set is one unit with its elements and their assistance: it changes when
-- any of them does, and appears at the latest of those changes.
-- name: SyncChanges :many
WITH set_units AS (
    SELECT u.set_entry_id AS id, max(u.seq)::bigint AS seq
    FROM (
        SELECT se.id AS set_entry_id, se.server_seq AS seq
        FROM set_entries se WHERE se.user_id = @user_id AND se.server_seq > @cursor
        UNION ALL
        SELECT el.set_entry_id, el.server_seq
        FROM set_elements el WHERE el.user_id = @user_id AND el.server_seq > @cursor
        UNION ALL
        SELECT el.set_entry_id, a.server_seq
        FROM set_element_assistance a JOIN set_elements el ON el.id = a.set_element_id
        WHERE a.user_id = @user_id AND a.server_seq > @cursor
    ) u
    GROUP BY u.set_entry_id
)
SELECT c.entity::text AS entity, c.id::uuid AS id, c.seq::bigint AS seq
FROM (
    SELECT 'session' AS entity, s.id, s.server_seq AS seq
    FROM workout_sessions s WHERE s.user_id = @user_id AND s.server_seq > @cursor
    UNION ALL
    SELECT 'block', b.id, b.server_seq
    FROM session_blocks b WHERE b.user_id = @user_id AND b.server_seq > @cursor
    UNION ALL
    SELECT 'set', su.id, su.seq FROM set_units su
    UNION ALL
    SELECT 'bodyweight', w.id, w.server_seq
    FROM user_bodyweight_log w WHERE w.user_id = @user_id AND w.server_seq > @cursor
) c
ORDER BY c.seq
LIMIT @page_limit;

-- name: SyncSessions :many
SELECT * FROM workout_sessions WHERE user_id = @user_id AND id = ANY(@ids::uuid[]);

-- name: SyncBlocks :many
SELECT * FROM session_blocks WHERE user_id = @user_id AND id = ANY(@ids::uuid[]);

-- name: SyncSetEntries :many
SELECT * FROM set_entries WHERE user_id = @user_id AND id = ANY(@ids::uuid[]);

-- The latest change to each set unit, for sets pulled in as a whole.
-- name: SyncSetSeqs :many
SELECT se.id,
       GREATEST(se.server_seq,
                COALESCE((SELECT max(el.server_seq) FROM set_elements el WHERE el.set_entry_id = se.id), 0),
                COALESCE((SELECT max(a.server_seq) FROM set_element_assistance a
                          JOIN set_elements el ON el.id = a.set_element_id
                          WHERE el.set_entry_id = se.id), 0))::bigint AS seq
FROM set_entries se
WHERE se.user_id = @user_id AND se.id = ANY(@ids::uuid[]);

-- name: SyncElementsOfSets :many
SELECT * FROM set_elements
WHERE user_id = @user_id AND set_entry_id = ANY(@set_ids::uuid[]) AND deleted_at IS NULL
ORDER BY set_entry_id, order_index;

-- name: SyncAssistanceOfSets :many
SELECT a.* FROM set_element_assistance a
JOIN set_elements el ON el.id = a.set_element_id
WHERE a.user_id = @user_id AND el.set_entry_id = ANY(@set_ids::uuid[])
  AND a.deleted_at IS NULL AND el.deleted_at IS NULL;

-- name: SyncElementMediaOfSets :many
SELECT sem.set_element_id, sem.media_id
FROM set_element_media sem
JOIN set_elements el ON el.id = sem.set_element_id
JOIN media_assets m ON m.id = sem.media_id
WHERE sem.user_id = @user_id AND el.set_entry_id = ANY(@set_ids::uuid[]) AND m.deleted_at IS NULL
ORDER BY sem.set_element_id, sem.order_index, sem.media_id;

-- name: SyncBodyweight :many
SELECT * FROM user_bodyweight_log WHERE user_id = @user_id AND id = ANY(@ids::uuid[]);

-- The device has everything up to this cursor.
-- name: AckSyncCursor :exec
UPDATE devices SET last_sync_seq = GREATEST(last_sync_seq, @cursor::bigint), last_seen_at = now()
WHERE id = @device_id AND user_id = @user_id;

-- ----------------------------------------------------------- idempotency

-- Claims a key: a new key, an expired one, or one whose earlier attempt
-- stalled (in progress for five minutes, same request) is taken. Returns no
-- row when the key is live and held.
-- name: ClaimIdempotencyKey :one
INSERT INTO idempotency_keys AS k (user_id, key, request_fingerprint)
VALUES (@user_id, @key, @request_fingerprint)
ON CONFLICT (user_id, key) DO UPDATE SET
    request_fingerprint = EXCLUDED.request_fingerprint, state = 'in_progress',
    response_status = NULL, response_body = NULL, created_at = now(), completed_at = NULL,
    expires_at = now() + interval '24 hours'
WHERE k.expires_at < now()
   OR (k.state = 'in_progress' AND k.created_at < now() - interval '5 minutes'
       AND k.request_fingerprint = EXCLUDED.request_fingerprint)
RETURNING k.key;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys WHERE user_id = @user_id AND key = @key;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys SET state = 'completed', response_status = @response_status,
    response_body = @response_body, completed_at = now()
WHERE user_id = @user_id AND key = @key;

-- A failed attempt gives the key back, so the client can retry with it.
-- name: ReleaseIdempotencyKey :exec
DELETE FROM idempotency_keys WHERE user_id = @user_id AND key = @key AND state = 'in_progress';

-- name: PurgeExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_keys WHERE expires_at < now();

-- ------------------------------------------------------------ bodyweight

-- Inserts, or replaces a live entry of the same user. No row for a tombstone
-- or another user's id.
-- name: UpsertBodyweight :one
INSERT INTO user_bodyweight_log AS w (
    id, user_id, measured_at, local_date, bodyweight_kg, note, client_id, updated_at
) VALUES (
    @id, @user_id, @measured_at, @local_date, @bodyweight_kg, @note, @client_id, @updated_at
)
ON CONFLICT (id) DO UPDATE SET
    measured_at = EXCLUDED.measured_at, local_date = EXCLUDED.local_date,
    bodyweight_kg = EXCLUDED.bodyweight_kg, note = EXCLUDED.note,
    client_id = EXCLUDED.client_id, updated_at = EXCLUDED.updated_at
WHERE w.user_id = EXCLUDED.user_id AND w.deleted_at IS NULL
RETURNING *;

-- name: GetBodyweight :one
SELECT * FROM user_bodyweight_log WHERE id = @id AND user_id = @user_id;

-- name: SoftDeleteBodyweight :execrows
UPDATE user_bodyweight_log SET deleted_at = now(), client_id = @client_id, updated_at = @updated_at
WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL;
