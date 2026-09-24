# Working in this repository

Hefesto is a calisthenics training logger coupled to a skill map. The logger
writes what an athlete actually did; the map reads that and unlocks skills from
it. Those two halves are the product — never build one without the other in
mind.

Read `docs/PROJECT_BRIEF.md` for the full spec and `docs/adr/` for decisions
already taken. The schema is `db/migrations/`; `docs/DDL_PROPOSAL.md` explains
the reasoning behind it and ADR 0005 records where it changed on review.
Content is authored per `docs/CONTENT_AUTHORING.md`.

## Phase discipline

The project is built in numbered phases (brief §12). **Finish the current
phase, summarise, and stop for review.** Do not start the next phase without
being asked. Current state is in the project's `STATUS.md`.

## Commands

```sh
make up               # full stack, seeded
make down / make reset
make test             # unit, no database
make test-integration # testcontainers against real Postgres
                      # (no Docker? HEFESTO_TEST_DATABASE_URL=<server url>)
make lint
make check            # what CI runs
make content-validate
make sqlc             # regenerate after any schema change
make migrate-new name=add_foo
make deploy-check     # validate prod compose, scripts, workflows
```

Production, in full in `docs/DEPLOYMENT.md`:

```sh
make release version=v0.2.0   # tag + push -> builds and deploys
make deploy tag=v0.1.9        # redeploy an existing tag; this is also rollback
make prod-version / prod-logs / prod-ps
```

## Rules that are not negotiable

- **`internal/domain` imports no I/O.** Not `internal/store`, not
  `internal/http`, not `database/sql`, not `os`.
  `internal/domain/arch_test.go` fails the build otherwise.
- **One code path for sets.** A set of 8 pull-ups is one `set_entry` with one
  `set_element`. A combo is one `set_entry` with N ordered `set_elements`.
  Never add a "simple set" shortcut.
- **Migrations are structure, seed is data.** Content is never inserted by a
  migration. Migrations are never edited once merged.
- **The OpenAPI spec is the source of truth.** Swift DTOs are generated from
  it, never hand-written. Change `api/openapi.yaml` first: request bodies are
  validated against it at runtime, and tests fail when routes or responses
  drift from it (ADR 0007).
- **Errors**: wrapped with `%w`, never swallowed; `application/problem+json`
  (RFC 9457) at the HTTP edge; `slog` with the request ID on context.
- **No `panic` outside `main`.**
- **Integration tests hit real Postgres** via testcontainers. Never mock SQL.
- **Unlocks are never revoked.** The map records achievement, not current form.
- **No mechanic that punishes rest.** See `docs/adr/0003`. Streaks count
  planned rest and deloads as maintained; no leaderboard; no notification that
  implies the user is falling behind.
- **No medical claims.** Injury content is educational and carries its
  disclaimer in the API payload, not only in the UI.

Do not add: GraphQL, Redis (before a measured need), a message queue,
microservices, analytics SDKs, or invented skill/exercise content.

## Schema conventions

- UUIDv7 primary keys generated in Go, never a database `DEFAULT`.
- Closed vocabularies are `text` + a **named** `CHECK`, never Postgres `ENUM`.
- Every user-owned row carries `user_id`, and children reference parents by the
  composite key `(id, user_id)` so a cross-user row is structurally impossible.
- Syncable rows carry `client_id`, `updated_at`, `server_updated_at`,
  `server_seq`, `deleted_at`. `server_seq` is assigned by the `touch_sync`
  trigger — never by a client.
- Sibling order is `order_index` with a `DEFERRABLE` unique constraint.

## Migration safety

Deploys run migrations **before** the new API starts, so every migration must
be backward compatible with the currently running version for the length of one
deploy. Expand, deploy, contract — never rename or drop a column in the same
release that stops using it.

## Commits

Conventional Commits, small and reviewable. Every non-obvious decision gets an
ADR in `docs/adr/NNNN-title.md` (context / decision / consequences).

## Secrets

`.env.prod` exists only on the production host. It is not in the repo, not in
CI, and not in any log. Never write a real secret into a file in this
repository, and never add a default credential to `.env.example`.
