# ADR 0006 — Content pipeline shape

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

Brief §7 asks for YAML content validated by JSON Schema, a linter, and an
idempotent seeder, built before the real content exists. Several details were
left open, and each one decides what an author can and cannot express.

## Decisions

1. **One file per skill and per exercise, named after its slug.** Families
   live in `content/families.yaml` (file order is display order); bands in
   `content/bands/<brand>.yaml`. The linter rejects a file whose name and slug
   disagree, so a slug can be found without grep.
2. **Injury notes are written inside the skill file**, under `injuries:`, as in
   the brief's reference file. The separate `content/injuries/` directory from
   the brief's layout is dropped: an injury note is always about one skill,
   and a second file would only add a cross-reference to get wrong.
3. **A skill's levels are an implicit prerequisite ladder.** Level *n* requires
   level *n − 1* of the same skill. Authors write only cross-skill
   prerequisites. Without this, every level with no written prerequisite would
   be a root, available from account creation — the opposite of intent.
   `Tree.Edges()` is the single definition, used by both the cycle check and
   the seed.
4. **Non-prerequisite edges are `links:`** on a level, with `relation`
   `recommended`, `alternative` or `antagonist`.
5. **Stored criteria are explicit.** Defaults (`assistance: none`,
   `occurrences: 1`) are applied at load time, so the database, the checksum
   and the Phase 3 evaluator never interpret an absent field. Empty criteria
   mean the level can only be self-attested.
6. **The checksum is over the loaded tree, not the bytes.** Re-formatting YAML
   or re-ordering keys does not create a new content version; changing any
   value does.
7. **Removal is soft, except levels, which cannot be removed.** A missing
   skill or exercise is retired; a missing global band is soft-deleted. A
   missing level aborts the seed, because user progress points at levels and
   there is no retired state for one. Retire the whole skill instead.
8. **Production seeds on every deploy**, after migrations and before the API
   restarts. The content is inside the image, the import is one transaction,
   and unchanged content is a no-op.
9. **No invented data.** The three skill files and seven exercises are
   `status: draft_placeholder`. The band catalogue is empty until real
   manufacturer figures are researched.

## Consequences

- An author cannot express a level that skips its predecessor. If that is
  ever needed, it becomes an explicit opt-out field, not a silent default.
- Renaming a level slug is a removal plus an addition, so the seed refuses it.
  Level renames need a deliberate migration of user progress when they come up.
- The seeder trusts a validated tree; `cmd/seed` always runs the linter's
  checks first, and the seed re-checks acyclicity in SQL as a second line.
