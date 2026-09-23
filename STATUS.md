# Status

The project is built in the numbered phases of `docs/PROJECT_BRIEF.md` §12.
Each phase ends with a summary and a stop for review. This file records where
that stands.

## Current phase: 1 — schema and content pipeline (complete, awaiting review)

Built:

- **Migrations** `db/migrations/00001`–`00008`: the approved schema, with the
  review decisions of ADR 0005 applied (`load_kg ≥ 0`, `user_bodyweight_log`,
  30-day soft deletion). Up, down and up again are tested.
- **sqlc** wired: `db/queries/content.sql` → `internal/store/dbgen`. CI fails
  if the generated code is stale (`make sqlc-check`).
- **Content JSON Schemas** in `content/schema/`, including the unlock-criteria
  DSL; the Go types for criteria live in `internal/domain/progress`, ready for
  the Phase 3 evaluator.
- **`internal/content`**: load, schema-validate, cross-file checks, checksum.
  `cmd/contentlint` reports; `cmd/seed` refuses to seed a tree with errors.
- **`cmd/seed`**: one transaction, advisory-locked, slug-keyed upserts that do
  not touch unchanged rows, soft retirement, refusal to drop a level, SQL
  re-check for prerequisite cycles, a `content_versions` row per change.
  Production now seeds on every deploy, after migrations.
- **Placeholder content**: `pull-up`, `front-lever` (the brief's reference
  file) and `handstand`, with seven exercises, all `draft_placeholder`.
- **`docs/CONTENT_AUTHORING.md`**, ADR 0005 (schema review), ADR 0006
  (content pipeline).
- `/readyz` now checks a real pgx pool.
- **Tests**: 25 content-check cases, schema-invariant integration tests
  (the 15 assertions of DDL §14, plus the unlock and append-only triggers,
  sync ownership, deletion), 8 seed integration tests.

Fixed on the way — pre-existing CI breakage:

- `.env.prod.example` was never committed (`.gitignore`'s `.env.*` swallowed
  it), so the CI `scripts` job and `make deploy-check` could not pass. It is
  now tracked, with every secret blank.
- `.golangci.yml` was a v1 config while CI installed `latest` (v2), which
  refuses it. Migrated to v2, and CI pins `v2.5.0`.

### Verification

Run in the development container against PostgreSQL 16.13:
`golangci-lint` clean; `go test -race -short ./...` and
`go test -race -tags integration ./...` green; `contentlint -strict` clean;
seed applied, re-run as a no-op, and dry-run; `/readyz` 200 against the
database.

**Not run here:** Docker Hub image pulls are blocked in the development
container, so neither `make up && make seed` through Compose nor the
testcontainers path was exercised; the integration tests ran through
`HEFESTO_TEST_DATABASE_URL` against a local Postgres 16 instead. CI runs the
testcontainers path.

### Open questions for review

1. **Implicit level ladder** (ADR 0006 §3): level *n* requires level *n − 1*
   of the same skill without being written. Right default?
2. **Levels cannot be removed or renamed** once seeded (ADR 0006 §7). A rename
   will need a progress-migration story when the real content lands.
3. **Criteria `assistance`** is `none | any` for now. Should a criterion be
   able to demand a specific assistance type (e.g. "band-assisted only"), or
   is that never an unlock standard?
4. **User bands are not syncable yet** (ADR 0005). Planned as an expand-only
   migration in Phase 4 — confirm.

## Next: Phase 2

Auth (Sign in with Apple, email/password, JWT + rotating refresh, the 30-day
deletion reaper) and the core API: exercises, sessions, blocks, sets, set
elements, assistance. OpenAPI spec complete for these; integration tests.
