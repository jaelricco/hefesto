-- Content import (cmd/seed). Every upsert is keyed on slug and leaves a row
-- untouched — updated_at and content_version_id included — when nothing in it
-- changed. The CTE returns the id whether the row was inserted, updated or
-- left alone: the INSERT returns nothing when its WHERE suppresses the update,
-- and the plain SELECT sees the pre-statement snapshot, so exactly one side
-- yields the id.

-- The newest version is what the database currently holds; its checksum is
-- the /v1/skills ETag.
-- name: GetLatestContentVersion :one
SELECT id, checksum, git_sha, item_counts, applied_at, applied_by
FROM content_versions
ORDER BY applied_at DESC, id DESC
LIMIT 1;

-- name: InsertContentVersion :exec
INSERT INTO content_versions (id, checksum, git_sha, item_counts, applied_by)
VALUES ($1, $2, $3, $4, $5);

-- name: UpsertFamily :one
WITH up AS (
    INSERT INTO families AS f (id, slug, name, order_index)
    VALUES (@id, @slug, @name, @order_index)
    ON CONFLICT (slug) DO UPDATE
        SET name = EXCLUDED.name, order_index = EXCLUDED.order_index
        WHERE (f.name, f.order_index) IS DISTINCT FROM (EXCLUDED.name, EXCLUDED.order_index)
    RETURNING f.id
)
SELECT up.id FROM up
UNION ALL
SELECT families.id FROM families WHERE families.slug = @slug
LIMIT 1;

-- name: UpsertExercise :one
WITH up AS (
    INSERT INTO exercises AS e (
        id, slug, name, aka, family_id, default_measure, load_semantics,
        is_bodyweight, unilateral, tempo_applicable, equipment, summary, cues,
        common_faults, status, content_version_id
    ) VALUES (
        @id, @slug, @name, @aka, @family_id, @default_measure, @load_semantics,
        @is_bodyweight, @unilateral, @tempo_applicable, @equipment, @summary, @cues,
        @common_faults, @status, @content_version_id
    )
    ON CONFLICT (slug) DO UPDATE SET
        name = EXCLUDED.name, aka = EXCLUDED.aka, family_id = EXCLUDED.family_id,
        default_measure = EXCLUDED.default_measure, load_semantics = EXCLUDED.load_semantics,
        is_bodyweight = EXCLUDED.is_bodyweight, unilateral = EXCLUDED.unilateral,
        tempo_applicable = EXCLUDED.tempo_applicable, equipment = EXCLUDED.equipment,
        summary = EXCLUDED.summary, cues = EXCLUDED.cues, common_faults = EXCLUDED.common_faults,
        status = EXCLUDED.status, content_version_id = EXCLUDED.content_version_id,
        updated_at = now()
    WHERE (e.name, e.aka, e.family_id, e.default_measure, e.load_semantics, e.is_bodyweight,
           e.unilateral, e.tempo_applicable, e.equipment, e.summary, e.cues, e.common_faults, e.status)
      IS DISTINCT FROM
          (EXCLUDED.name, EXCLUDED.aka, EXCLUDED.family_id, EXCLUDED.default_measure,
           EXCLUDED.load_semantics, EXCLUDED.is_bodyweight, EXCLUDED.unilateral,
           EXCLUDED.tempo_applicable, EXCLUDED.equipment, EXCLUDED.summary, EXCLUDED.cues,
           EXCLUDED.common_faults, EXCLUDED.status)
    RETURNING e.id
)
SELECT up.id FROM up
UNION ALL
SELECT exercises.id FROM exercises WHERE exercises.slug = @slug
LIMIT 1;

-- Removal is soft: rows gone from content are retired, never deleted,
-- because logged set elements reference them.
-- name: RetireExercisesNotIn :execrows
UPDATE exercises
SET status = 'retired', content_version_id = @content_version_id, updated_at = now()
WHERE status <> 'retired' AND NOT (slug = ANY(@slugs::text[]));

-- name: UpsertGlobalBand :exec
INSERT INTO bands (id, owner_user_id, brand, colour_label, resistance_min_kg,
                   resistance_max_kg, length_cm, thickness_mm)
VALUES (@id, NULL, @brand, @colour_label, @resistance_min_kg, @resistance_max_kg,
        @length_cm, @thickness_mm)
ON CONFLICT (owner_user_id, brand, colour_label) WHERE deleted_at IS NULL DO UPDATE SET
    resistance_min_kg = EXCLUDED.resistance_min_kg, resistance_max_kg = EXCLUDED.resistance_max_kg,
    length_cm = EXCLUDED.length_cm, thickness_mm = EXCLUDED.thickness_mm, updated_at = now()
WHERE (bands.resistance_min_kg, bands.resistance_max_kg, bands.length_cm, bands.thickness_mm)
  IS DISTINCT FROM
      (EXCLUDED.resistance_min_kg, EXCLUDED.resistance_max_kg, EXCLUDED.length_cm, EXCLUDED.thickness_mm);

-- Global bands gone from content are soft-deleted; logged assistance keeps
-- pointing at them.
-- name: SoftDeleteGlobalBandsNotIn :execrows
UPDATE bands
SET deleted_at = now(), updated_at = now()
WHERE owner_user_id IS NULL AND deleted_at IS NULL
  AND NOT ((brand || E'\x1f' || colour_label) = ANY(@keys::text[]));

-- name: UpsertSkill :one
WITH up AS (
    INSERT INTO skills AS s (
        id, slug, family_id, name, aka, difficulty_tier, is_milestone, summary,
        primary_muscles, common_faults, map_constellation, map_x, map_y, status,
        content_version_id
    ) VALUES (
        @id, @slug, @family_id, @name, @aka, @difficulty_tier, @is_milestone, @summary,
        @primary_muscles, @common_faults, @map_constellation, @map_x, @map_y, @status,
        @content_version_id
    )
    ON CONFLICT (slug) DO UPDATE SET
        family_id = EXCLUDED.family_id, name = EXCLUDED.name, aka = EXCLUDED.aka,
        difficulty_tier = EXCLUDED.difficulty_tier, is_milestone = EXCLUDED.is_milestone,
        summary = EXCLUDED.summary, primary_muscles = EXCLUDED.primary_muscles,
        common_faults = EXCLUDED.common_faults, map_constellation = EXCLUDED.map_constellation,
        map_x = EXCLUDED.map_x, map_y = EXCLUDED.map_y, status = EXCLUDED.status,
        content_version_id = EXCLUDED.content_version_id, updated_at = now()
    WHERE (s.family_id, s.name, s.aka, s.difficulty_tier, s.is_milestone, s.summary,
           s.primary_muscles, s.common_faults, s.map_constellation, s.map_x, s.map_y, s.status)
      IS DISTINCT FROM
          (EXCLUDED.family_id, EXCLUDED.name, EXCLUDED.aka, EXCLUDED.difficulty_tier,
           EXCLUDED.is_milestone, EXCLUDED.summary, EXCLUDED.primary_muscles,
           EXCLUDED.common_faults, EXCLUDED.map_constellation, EXCLUDED.map_x, EXCLUDED.map_y,
           EXCLUDED.status)
    RETURNING s.id
)
SELECT up.id FROM up
UNION ALL
SELECT skills.id FROM skills WHERE skills.slug = @slug
LIMIT 1;

-- name: RetireSkillsNotIn :execrows
UPDATE skills
SET status = 'retired', content_version_id = @content_version_id, updated_at = now()
WHERE status <> 'retired' AND NOT (slug = ANY(@slugs::text[]));

-- name: ListSkillLevelSlugs :many
SELECT slug FROM skill_levels WHERE skill_id = $1 ORDER BY order_index;

-- name: UpsertSkillLevel :one
WITH up AS (
    INSERT INTO skill_levels AS l (
        id, skill_id, order_index, slug, name, description, unlock_criteria, est_weeks_from_prev
    ) VALUES (
        @id, @skill_id, @order_index, @slug, @name, @description, @unlock_criteria, @est_weeks_from_prev
    )
    ON CONFLICT (skill_id, slug) DO UPDATE SET
        order_index = EXCLUDED.order_index, name = EXCLUDED.name,
        description = EXCLUDED.description, unlock_criteria = EXCLUDED.unlock_criteria,
        est_weeks_from_prev = EXCLUDED.est_weeks_from_prev, updated_at = now()
    WHERE (l.order_index, l.name, l.description, l.unlock_criteria, l.est_weeks_from_prev)
      IS DISTINCT FROM
          (EXCLUDED.order_index, EXCLUDED.name, EXCLUDED.description,
           EXCLUDED.unlock_criteria, EXCLUDED.est_weeks_from_prev)
    RETURNING l.id
)
SELECT up.id FROM up
UNION ALL
SELECT skill_levels.id FROM skill_levels
WHERE skill_levels.skill_id = @skill_id AND skill_levels.slug = @slug
LIMIT 1;

-- Associations carry no user data, so they are replaced wholesale.

-- name: DeleteSkillLevelExercises :exec
DELETE FROM skill_level_exercises WHERE skill_level_id = $1;

-- name: InsertSkillLevelExercise :exec
INSERT INTO skill_level_exercises (skill_level_id, exercise_id, role, order_index)
VALUES ($1, $2, $3, $4);

-- name: DeleteAllSkillEdges :exec
DELETE FROM skill_edges;

-- name: InsertSkillEdge :exec
INSERT INTO skill_edges (from_skill_level_id, to_skill_level_id, relation, weight)
VALUES ($1, $2, $3, $4);

-- Belt and braces for deviation D-7: contentlint already rejected cycles, but
-- the seed re-checks the graph as the database now holds it. Returns the ids
-- of levels that can reach themselves through prerequisite edges.
-- name: FindPrerequisiteCycles :many
WITH RECURSIVE walk (origin, node, path, cyclic) AS (
    SELECT e.from_skill_level_id, e.to_skill_level_id,
           ARRAY[e.from_skill_level_id, e.to_skill_level_id],
           e.from_skill_level_id = e.to_skill_level_id
    FROM skill_edges e
    WHERE e.relation = 'prerequisite'
  UNION ALL
    SELECT w.origin, e.to_skill_level_id,
           w.path || e.to_skill_level_id,
           e.to_skill_level_id = ANY(w.path)
    FROM walk w
    JOIN skill_edges e ON e.from_skill_level_id = w.node AND e.relation = 'prerequisite'
    WHERE NOT w.cyclic
)
SELECT DISTINCT origin::uuid AS skill_level_id FROM walk WHERE cyclic AND node = origin;

-- name: UpsertInjuryRisk :one
WITH up AS (
    INSERT INTO skill_injury_risks AS r (
        id, skill_id, region, name, description, risk_factors, early_signs,
        disclaimer, order_index, content_version_id
    ) VALUES (
        @id, @skill_id, @region, @name, @description, @risk_factors, @early_signs,
        'educational_only', @order_index, @content_version_id
    )
    ON CONFLICT (skill_id, region, name) DO UPDATE SET
        description = EXCLUDED.description, risk_factors = EXCLUDED.risk_factors,
        early_signs = EXCLUDED.early_signs, order_index = EXCLUDED.order_index,
        content_version_id = EXCLUDED.content_version_id
    WHERE (r.description, r.risk_factors, r.early_signs, r.order_index)
      IS DISTINCT FROM
          (EXCLUDED.description, EXCLUDED.risk_factors, EXCLUDED.early_signs, EXCLUDED.order_index)
    RETURNING r.id
)
SELECT up.id FROM up
UNION ALL
SELECT skill_injury_risks.id FROM skill_injury_risks
WHERE skill_injury_risks.skill_id = @skill_id
  AND skill_injury_risks.region = @region AND skill_injury_risks.name = @name
LIMIT 1;

-- Injury content references no user rows, so entries gone from a skill file
-- are deleted outright.
-- name: DeleteInjuryRisksNotIn :exec
DELETE FROM skill_injury_risks
WHERE skill_id = @skill_id AND NOT (id = ANY(@keep::uuid[]));

-- name: DeleteInjuryPrehab :exec
DELETE FROM injury_prehab_exercises WHERE risk_id = $1;

-- name: InsertInjuryPrehab :exec
INSERT INTO injury_prehab_exercises (risk_id, exercise_id, order_index)
VALUES ($1, $2, $3);
