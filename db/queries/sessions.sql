-- Sessions -> blocks -> set entries -> set elements (+ assistance).
--
-- Every query filters on user_id. Soft-deleted rows are "parked": their
-- order_index moves above 999999999, so a tombstone never blocks a live
-- sibling from taking its old position (the unique constraint cannot be
-- partial and deferrable at the same time).

-- ------------------------------------------------------------- sessions

-- name: InsertSession :one
INSERT INTO workout_sessions (
    id, user_id, started_at, timezone, local_date, title, notes, bodyweight_kg,
    is_rest_day, template_id, client_id, updated_at
) VALUES (
    @id, @user_id, @started_at, @timezone, @local_date, @title, @notes, @bodyweight_kg,
    @is_rest_day, @template_id, @client_id, @updated_at
)
ON CONFLICT (id) DO NOTHING
RETURNING *;

-- name: GetSession :one
SELECT * FROM workout_sessions
WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL;

-- name: GetSessionForUpdate :one
SELECT * FROM workout_sessions
WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdateSession :one
UPDATE workout_sessions SET
    started_at = @started_at, ended_at = @ended_at, timezone = @timezone,
    local_date = @local_date, title = @title, notes = @notes,
    perceived_fatigue = @perceived_fatigue, bodyweight_kg = @bodyweight_kg,
    is_rest_day = @is_rest_day, status = @status,
    client_id = @client_id, updated_at = @updated_at
WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteSession :execrows
UPDATE workout_sessions SET deleted_at = now(), client_id = @client_id, updated_at = @updated_at
WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL;

-- Newest first; the cursor is the (started_at, id) of the last row seen.
-- name: ListSessions :many
SELECT s.id, s.started_at, s.ended_at, s.timezone, s.local_date, s.title, s.status, s.is_rest_day,
    (SELECT count(*) FROM session_blocks b WHERE b.session_id = s.id AND b.deleted_at IS NULL)::int AS block_count,
    (SELECT count(*) FROM set_entries e WHERE e.session_id = s.id AND e.deleted_at IS NULL)::int AS set_count
FROM workout_sessions s
WHERE s.user_id = @user_id AND s.deleted_at IS NULL
  AND (sqlc.narg('from_date')::date IS NULL OR s.local_date >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::date IS NULL OR s.local_date <= sqlc.narg('to_date'))
  AND (sqlc.narg('cursor_started_at')::timestamptz IS NULL
       OR (s.started_at, s.id) < (sqlc.narg('cursor_started_at'), sqlc.narg('cursor_id')::uuid))
ORDER BY s.started_at DESC, s.id DESC
LIMIT @page_limit;

-- --------------------------------------------------------------- blocks

-- name: ListBlocks :many
SELECT * FROM session_blocks
WHERE session_id = @session_id AND user_id = @user_id AND deleted_at IS NULL
ORDER BY order_index;

-- name: GetBlock :one
SELECT * FROM session_blocks WHERE id = @id AND user_id = @user_id;

-- Inserts, or replaces a live block of the same session. Returns no row when
-- the id belongs to another session or user, or is a tombstone.
-- name: UpsertBlock :one
INSERT INTO session_blocks AS b (
    id, user_id, session_id, order_index, kind, rounds_planned, rounds_done,
    interval_s, notes, client_id, updated_at
) VALUES (
    @id, @user_id, @session_id, @order_index, @kind, @rounds_planned, @rounds_done,
    @interval_s, @notes, @client_id, @updated_at
)
ON CONFLICT (id) DO UPDATE SET
    order_index = EXCLUDED.order_index, kind = EXCLUDED.kind,
    rounds_planned = EXCLUDED.rounds_planned, rounds_done = EXCLUDED.rounds_done,
    interval_s = EXCLUDED.interval_s, notes = EXCLUDED.notes,
    client_id = EXCLUDED.client_id, updated_at = EXCLUDED.updated_at
WHERE b.user_id = EXCLUDED.user_id AND b.session_id = EXCLUDED.session_id AND b.deleted_at IS NULL
RETURNING *, (xmax = 0) AS inserted;

-- name: SoftDeleteBlocksOfSession :exec
WITH base AS (
    SELECT GREATEST(COALESCE(MAX(sb.order_index), 0), 999999999) AS m
    FROM session_blocks sb WHERE sb.session_id = @session_id
), doomed AS (
    SELECT d.id, row_number() OVER (ORDER BY d.order_index) AS n
    FROM session_blocks d
    WHERE d.session_id = @session_id AND d.user_id = @user_id AND d.deleted_at IS NULL
      AND (sqlc.narg('only_id')::uuid IS NULL OR d.id = sqlc.narg('only_id'))
)
UPDATE session_blocks b
SET deleted_at = now(), order_index = base.m + doomed.n, client_id = @client_id, updated_at = @updated_at
FROM doomed, base
WHERE b.id = doomed.id;

-- name: SetBlockOrder :exec
UPDATE session_blocks b
SET order_index = o.ord - 1, client_id = @client_id, updated_at = @updated_at
FROM unnest(@ids::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE b.id = o.id AND b.user_id = @user_id AND b.session_id = @session_id AND b.deleted_at IS NULL;

-- ----------------------------------------------------------------- sets

-- name: ListSetEntries :many
SELECT * FROM set_entries
WHERE session_id = @session_id AND user_id = @user_id AND deleted_at IS NULL
ORDER BY block_id, order_index;

-- name: GetSetEntry :one
SELECT * FROM set_entries WHERE id = @id AND user_id = @user_id;

-- name: UpsertSetEntry :one
INSERT INTO set_entries AS s (
    id, user_id, session_id, block_id, order_index, round_index, kind, is_planned,
    rest_after_planned_s, rest_after_actual_s, rpe, rir, completed_at, notes,
    client_id, updated_at
) VALUES (
    @id, @user_id, @session_id, @block_id, @order_index, @round_index, @kind, @is_planned,
    @rest_after_planned_s, @rest_after_actual_s, @rpe, @rir, @completed_at, @notes,
    @client_id, @updated_at
)
ON CONFLICT (id) DO UPDATE SET
    block_id = EXCLUDED.block_id, order_index = EXCLUDED.order_index,
    round_index = EXCLUDED.round_index, kind = EXCLUDED.kind, is_planned = EXCLUDED.is_planned,
    rest_after_planned_s = EXCLUDED.rest_after_planned_s,
    rest_after_actual_s = EXCLUDED.rest_after_actual_s,
    rpe = EXCLUDED.rpe, rir = EXCLUDED.rir, completed_at = EXCLUDED.completed_at,
    notes = EXCLUDED.notes, client_id = EXCLUDED.client_id, updated_at = EXCLUDED.updated_at
WHERE s.user_id = EXCLUDED.user_id AND s.session_id = EXCLUDED.session_id AND s.deleted_at IS NULL
RETURNING *, (xmax = 0) AS inserted;

-- Parks and tombstones live sets of a block, or of the whole session, or one set.
-- name: SoftDeleteSetEntries :exec
WITH doomed AS (
    SELECT d.id, d.block_id, row_number() OVER (PARTITION BY d.block_id ORDER BY d.order_index) AS n
    FROM set_entries d
    WHERE d.session_id = @session_id AND d.user_id = @user_id AND d.deleted_at IS NULL
      AND (sqlc.narg('block_id')::uuid IS NULL OR d.block_id = sqlc.narg('block_id'))
      AND (sqlc.narg('only_id')::uuid IS NULL OR d.id = sqlc.narg('only_id'))
), base AS (
    SELECT x.block_id, GREATEST(COALESCE(MAX(x.order_index), 0), 999999999) AS m
    FROM set_entries x WHERE x.block_id IN (SELECT block_id FROM doomed)
    GROUP BY x.block_id
)
UPDATE set_entries s
SET deleted_at = now(), order_index = base.m + doomed.n, client_id = @client_id, updated_at = @updated_at
FROM doomed JOIN base ON base.block_id = doomed.block_id
WHERE s.id = doomed.id;

-- name: SetSetOrder :exec
UPDATE set_entries s
SET order_index = o.ord - 1, client_id = @client_id, updated_at = @updated_at
FROM unnest(@ids::uuid[]) WITH ORDINALITY AS o(id, ord)
WHERE s.id = o.id AND s.user_id = @user_id AND s.block_id = @block_id AND s.deleted_at IS NULL;

-- ------------------------------------------------------------- elements

-- name: ListSetElements :many
SELECT * FROM set_elements
WHERE session_id = @session_id AND user_id = @user_id AND deleted_at IS NULL
ORDER BY set_entry_id, order_index;

-- name: ListElementsOfSet :many
SELECT * FROM set_elements
WHERE set_entry_id = @set_entry_id AND user_id = @user_id AND deleted_at IS NULL
ORDER BY order_index;

-- name: UpsertSetElement :one
INSERT INTO set_elements AS e (
    id, user_id, session_id, set_entry_id, order_index, exercise_id, measure,
    reps, hold_seconds, distance_m, tempo, load_kg, is_eccentric_only,
    is_partial_rom, rom_note, form_quality, failed, assistance_class,
    client_id, updated_at
) VALUES (
    @id, @user_id, @session_id, @set_entry_id, @order_index, @exercise_id, @measure,
    @reps, @hold_seconds, @distance_m, @tempo, @load_kg, @is_eccentric_only,
    @is_partial_rom, @rom_note, @form_quality, @failed, @assistance_class,
    @client_id, @updated_at
)
ON CONFLICT (id) DO UPDATE SET
    order_index = EXCLUDED.order_index, exercise_id = EXCLUDED.exercise_id,
    measure = EXCLUDED.measure, reps = EXCLUDED.reps, hold_seconds = EXCLUDED.hold_seconds,
    distance_m = EXCLUDED.distance_m, tempo = EXCLUDED.tempo, load_kg = EXCLUDED.load_kg,
    is_eccentric_only = EXCLUDED.is_eccentric_only, is_partial_rom = EXCLUDED.is_partial_rom,
    rom_note = EXCLUDED.rom_note, form_quality = EXCLUDED.form_quality, failed = EXCLUDED.failed,
    assistance_class = EXCLUDED.assistance_class,
    client_id = EXCLUDED.client_id, updated_at = EXCLUDED.updated_at
WHERE e.user_id = EXCLUDED.user_id AND e.set_entry_id = EXCLUDED.set_entry_id AND e.deleted_at IS NULL
RETURNING id;

-- Tombstones the live elements of sets (a whole session, one set, or those
-- of one set not in keep) along with their assistance rows.
-- name: SoftDeleteSetElements :exec
WITH doomed AS (
    SELECT d.id, d.set_entry_id, row_number() OVER (PARTITION BY d.set_entry_id ORDER BY d.order_index) AS n
    FROM set_elements d
    WHERE d.session_id = @session_id AND d.user_id = @user_id AND d.deleted_at IS NULL
      AND (sqlc.narg('set_entry_id')::uuid IS NULL OR d.set_entry_id = sqlc.narg('set_entry_id'))
      AND (sqlc.narg('block_id')::uuid IS NULL
           OR d.set_entry_id IN (SELECT se.id FROM set_entries se WHERE se.block_id = sqlc.narg('block_id')))
      AND NOT (d.id = ANY(@keep::uuid[]))
), base AS (
    SELECT x.set_entry_id, GREATEST(COALESCE(MAX(x.order_index), 0), 999999999) AS m
    FROM set_elements x WHERE x.set_entry_id IN (SELECT set_entry_id FROM doomed)
    GROUP BY x.set_entry_id
), gone_assistance AS (
    UPDATE set_element_assistance a
    SET deleted_at = now(), client_id = @client_id, updated_at = @updated_at
    WHERE a.set_element_id IN (SELECT id FROM doomed) AND a.deleted_at IS NULL
)
UPDATE set_elements e
SET deleted_at = now(), order_index = base.m + doomed.n, client_id = @client_id, updated_at = @updated_at
FROM doomed JOIN base ON base.set_entry_id = doomed.set_entry_id
WHERE e.id = doomed.id;

-- ----------------------------------------------------------- assistance

-- name: ListAssistance :many
SELECT a.* FROM set_element_assistance a
JOIN set_elements e ON e.id = a.set_element_id
WHERE e.session_id = @session_id AND a.user_id = @user_id AND a.deleted_at IS NULL;

-- name: UpsertAssistance :one
INSERT INTO set_element_assistance AS a (
    id, user_id, set_element_id, type, band_id, band_count, anchor,
    estimated_assist_kg, note, client_id, updated_at
) VALUES (
    @id, @user_id, @set_element_id, @type, @band_id, @band_count, @anchor,
    @estimated_assist_kg, @note, @client_id, @updated_at
)
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type, band_id = EXCLUDED.band_id, band_count = EXCLUDED.band_count,
    anchor = EXCLUDED.anchor, estimated_assist_kg = EXCLUDED.estimated_assist_kg,
    note = EXCLUDED.note, client_id = EXCLUDED.client_id, updated_at = EXCLUDED.updated_at
WHERE a.user_id = EXCLUDED.user_id AND a.set_element_id = EXCLUDED.set_element_id AND a.deleted_at IS NULL
RETURNING id;

-- An element has at most one live assistance row: the one being kept, if any.
-- name: SoftDeleteOtherAssistance :exec
UPDATE set_element_assistance
SET deleted_at = now(), client_id = @client_id, updated_at = @updated_at
WHERE set_element_id = @set_element_id AND user_id = @user_id AND deleted_at IS NULL
  AND (sqlc.narg('keep_id')::uuid IS NULL OR id <> sqlc.narg('keep_id'));

-- ------------------------------------------------------------- last set

-- name: LastSetWithExercise :one
SELECT se.id AS set_entry_id, se.session_id, ws.local_date
FROM set_elements el
JOIN set_entries se ON se.id = el.set_entry_id
JOIN workout_sessions ws ON ws.id = se.session_id
WHERE el.user_id = @user_id AND el.exercise_id = @exercise_id
  AND el.deleted_at IS NULL AND se.deleted_at IS NULL AND ws.deleted_at IS NULL
  AND NOT se.is_planned
ORDER BY COALESCE(se.completed_at, ws.started_at) DESC, ws.started_at DESC, se.order_index DESC
LIMIT 1;

-- name: TemplateBelongsToUser :one
SELECT EXISTS (
    SELECT 1 FROM workout_templates
    WHERE id = @id AND user_id = @user_id AND deleted_at IS NULL
);
