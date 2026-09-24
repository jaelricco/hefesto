-- Exercises and bands as the API reads them.

-- name: ListExercises :many
SELECT e.*, f.slug AS family_slug
FROM exercises e
JOIN families f ON f.id = e.family_id
WHERE e.status <> 'retired'
  AND (sqlc.narg('family')::text IS NULL OR f.slug = sqlc.narg('family'))
  AND (sqlc.narg('equipment')::text IS NULL OR sqlc.narg('equipment')::text = ANY(e.equipment))
  AND (sqlc.narg('q')::text IS NULL
       OR e.name ILIKE '%' || sqlc.narg('q') || '%'
       OR e.name % sqlc.narg('q')
       OR EXISTS (SELECT 1 FROM unnest(e.aka) a WHERE a ILIKE '%' || sqlc.narg('q') || '%'))
ORDER BY
  CASE WHEN sqlc.narg('q')::text IS NULL THEN 0 ELSE similarity(e.name, sqlc.narg('q')) END DESC,
  e.name, e.slug;

-- name: GetExerciseBySlug :one
SELECT e.*, f.slug AS family_slug
FROM exercises e
JOIN families f ON f.id = e.family_id
WHERE e.slug = $1 AND e.status <> 'retired';

-- name: ExerciseStatuses :many
SELECT id, status FROM exercises WHERE id = ANY(@ids::uuid[]);

-- name: ListBandsForUser :many
SELECT * FROM bands
WHERE deleted_at IS NULL AND (owner_user_id IS NULL OR owner_user_id = $1)
ORDER BY brand, colour_label, id;

-- Bands a user may log with: the catalogue and their own, deleted or not,
-- so repeating an old set keeps working after a band is retired.
-- name: VisibleBandIDs :many
SELECT id FROM bands
WHERE id = ANY(@ids::uuid[]) AND (owner_user_id IS NULL OR owner_user_id = @user_id);

-- name: InsertUserBand :one
INSERT INTO bands (id, owner_user_id, brand, colour_label, resistance_min_kg, resistance_max_kg, length_cm, thickness_mm)
VALUES (@id, @owner_user_id, @brand, @colour_label, @resistance_min_kg, @resistance_max_kg, @length_cm, @thickness_mm)
RETURNING *;

-- name: SoftDeleteUserBand :execrows
UPDATE bands SET deleted_at = now(), updated_at = now()
WHERE id = @id AND owner_user_id = @owner_user_id AND deleted_at IS NULL;
