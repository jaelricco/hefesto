-- Media assets and their attachment to set elements.

-- name: InsertMediaAsset :one
INSERT INTO media_assets (id, owner_user_id, kind, storage_key, mime, bytes, width, height)
VALUES (@id, @owner_user_id, @kind, @storage_key, @mime, @bytes, @width, @height)
ON CONFLICT (id) DO NOTHING
RETURNING *;

-- name: GetMediaAsset :one
SELECT * FROM media_assets WHERE id = @id AND owner_user_id = @owner_user_id AND deleted_at IS NULL;

-- name: MarkMediaReady :one
UPDATE media_assets SET status = 'ready', ready_at = COALESCE(ready_at, now())
WHERE id = @id AND owner_user_id = @owner_user_id AND deleted_at IS NULL AND status IN ('pending', 'ready')
RETURNING *;

-- name: MarkMediaFailed :one
UPDATE media_assets SET status = 'failed'
WHERE id = @id AND owner_user_id = @owner_user_id AND deleted_at IS NULL AND status = 'pending'
RETURNING *;

-- Touches the elements an asset is attached to, so the detachment syncs.
-- name: TouchElementsWithMedia :exec
UPDATE set_elements el SET server_updated_at = now()
FROM set_element_media sem
WHERE sem.media_id = @media_id AND sem.user_id = @user_id AND el.id = sem.set_element_id;

-- name: DetachMedia :exec
DELETE FROM set_element_media WHERE media_id = @media_id AND user_id = @user_id;

-- name: SoftDeleteMediaAsset :one
UPDATE media_assets SET deleted_at = now()
WHERE id = @id AND owner_user_id = @owner_user_id AND deleted_at IS NULL
RETURNING *;

-- name: ReadyMediaIDs :many
SELECT id FROM media_assets
WHERE owner_user_id = @owner_user_id AND id = ANY(@ids::uuid[]) AND status = 'ready' AND deleted_at IS NULL;

-- name: ListElementMedia :many
SELECT sem.set_element_id, sem.media_id
FROM set_element_media sem
JOIN set_elements el ON el.id = sem.set_element_id
JOIN media_assets m ON m.id = sem.media_id
WHERE sem.user_id = @user_id AND el.session_id = @session_id AND m.deleted_at IS NULL
ORDER BY sem.set_element_id, sem.order_index, sem.media_id;

-- name: DetachOtherElementMedia :exec
DELETE FROM set_element_media
WHERE set_element_id = @set_element_id AND user_id = @user_id AND NOT (media_id = ANY(@keep::uuid[]));

-- name: AttachElementMedia :exec
INSERT INTO set_element_media (set_element_id, user_id, media_id, order_index)
SELECT @set_element_id, @user_id, m.id, (m.ord - 1)::smallint
FROM unnest(@media_ids::uuid[]) WITH ORDINALITY AS m(id, ord)
ON CONFLICT (set_element_id, media_id) DO UPDATE SET order_index = EXCLUDED.order_index;

-- Uploads never completed: their objects, if any, are removed and the asset
-- marked failed.
-- name: ExpirePendingMedia :many
UPDATE media_assets SET status = 'failed'
WHERE status = 'pending' AND created_at < @before AND deleted_at IS NULL
RETURNING storage_key;
