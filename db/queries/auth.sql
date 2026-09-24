-- Users, devices, refresh tokens and Apple identities.

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, display_name, locale, timezone, email_verified_at)
VALUES (@id, @email, @password_hash, @display_name, @locale, @timezone, @email_verified_at)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserProfile :one
UPDATE users SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    locale       = COALESCE(sqlc.narg('locale'), locale),
    unit_system  = COALESCE(sqlc.narg('unit_system'), unit_system),
    week_start   = COALESCE(sqlc.narg('week_start'), week_start),
    timezone     = COALESCE(sqlc.narg('timezone'), timezone),
    updated_at   = now()
WHERE id = @id
RETURNING *;

-- name: SetUserEmail :exec
UPDATE users SET email = @email, email_verified_at = @email_verified_at, updated_at = now()
WHERE id = @id AND email IS NULL;

-- name: RequestUserDeletion :one
UPDATE users SET
    status = 'deletion_pending',
    deletion_requested_at = COALESCE(deletion_requested_at, now()),
    updated_at = now()
WHERE id = $1 AND status IN ('active', 'deletion_pending')
RETURNING *;

-- A sign-in during the grace period cancels the deletion.
-- name: CancelUserDeletion :execrows
UPDATE users SET status = 'active', deletion_requested_at = NULL, updated_at = now()
WHERE id = $1 AND status = 'deletion_pending';

-- Hard deletion after the grace period. Everything the user owns goes with
-- the row through ON DELETE CASCADE.
-- name: UsersDueForReaping :many
SELECT id FROM users
WHERE status = 'deletion_pending' AND deletion_requested_at < @cutoff
ORDER BY deletion_requested_at;

-- name: ReapUser :execrows
DELETE FROM users
WHERE id = @id AND status = 'deletion_pending' AND deletion_requested_at < @cutoff;

-- A device id is claimed by the first account that signs in with it and is
-- never re-owned: the row returns nothing if another account holds the id.
-- name: UpsertDevice :one
INSERT INTO devices (id, user_id, platform, model, os_version, app_version, last_seen_at)
VALUES (@id, @user_id, @platform, @model, @os_version, @app_version, now())
ON CONFLICT (id) DO UPDATE SET
    platform = EXCLUDED.platform, model = EXCLUDED.model,
    os_version = EXCLUDED.os_version, app_version = EXCLUDED.app_version,
    last_seen_at = now()
WHERE devices.user_id = EXCLUDED.user_id
RETURNING id;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, family_id, token_hash, device_id, expires_at, user_agent, ip)
VALUES (@id, @user_id, @family_id, @token_hash, @device_id, @expires_at, @user_agent, @ip);

-- Locks the row: two concurrent refreshes with the same token must not both
-- succeed.
-- name: GetRefreshTokenForUpdate :one
SELECT * FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE;

-- name: MarkRefreshTokenRotated :exec
UPDATE refresh_tokens
SET used_at = now(), revoked_at = now(), revoked_reason = 'rotated', replaced_by = @replaced_by
WHERE id = @id;

-- name: RevokeRefreshFamily :execrows
UPDATE refresh_tokens SET revoked_at = now(), revoked_reason = @reason
WHERE family_id = @family_id AND revoked_at IS NULL;

-- name: RevokeUserRefreshTokens :execrows
UPDATE refresh_tokens SET revoked_at = now(), revoked_reason = @reason
WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: GetAppleIdentity :one
SELECT * FROM apple_identities WHERE apple_sub = $1;

-- name: InsertAppleIdentity :exec
INSERT INTO apple_identities (user_id, apple_sub, is_private_email)
VALUES (@user_id, @apple_sub, @is_private_email);
