# Lodestar

A calisthenics training app in two tightly coupled halves:

- **A precision logger** — sets, reps, isometric holds, added weight, band
  assistance, tempo, rest intervals, RPE/RIR, and **combos**
  (`full planche → press to handstand → full planche` as one set with three
  ordered elements).
- **A skill map** — a DAG of calisthenics skills rendered as a star
  constellation, where unlocks are detected from what you actually logged.

The logger feeds the map. That coupling is the product.

## Status

**Phase 0 — scaffold.** The repository, Compose stack, Makefile and CI exist;
the schema is proposed in [`docs/DDL_PROPOSAL.md`](docs/DDL_PROPOSAL.md) and
awaits review. There are no migrations yet, so `make up` starts a stack against
an empty database.

See [`docs/PROJECT_BRIEF.md`](docs/PROJECT_BRIEF.md) for the full plan and
[`docs/adr/`](docs/adr/) for decisions taken so far.

## Getting started

```sh
cp .env.example .env     # then fill in the CHANGE_ME values
make up                  # postgres, api, minio, mailpit, pgweb — and seed
```

| Service | URL |
|---|---|
| API | http://localhost:8080/healthz |
| pgweb | http://localhost:8081 |
| Mailpit | http://localhost:8025 |
| MinIO console | http://localhost:9001 |

`make help` lists every target.

## Layout

```
cmd/          api server, content seeder, content linter
internal/
  domain/     pure types and rules — no I/O, enforced by arch_test.go
  store/      sqlc-generated queries and repositories
  http/       handlers, middleware, problem+json errors
db/           goose migrations and the .sql files sqlc consumes
content/      authored skills, exercises, bands and injury notes (YAML + JSON Schema)
api/          openapi.yaml — the source of truth for the generated Swift client
ios/          Xcode project (Phase 5)
docs/adr/     one ADR per significant decision
```

## Stack

Go 1.24 · chi · pgx/v5 · sqlc · goose · PostgreSQL 16 · OpenAPI 3.1 ·
SwiftUI (iOS 17+) with GRDB · Docker Compose · GitHub Actions.

Rationale in [ADR 0001](docs/adr/0001-stack.md).

## A note on the gamification

Overuse injury is the dominant risk in calisthenics, and the standard mechanics
of gamified fitness apps push directly toward it. Streaks here count planned
rest and deloads as maintained, XP is weighted by quality rather than volume,
there is no leaderboard, and no notification implies you are falling behind.
[ADR 0003](docs/adr/0003-gamification-guardrails.md) makes those binding.

Injury content in the app is educational and carries a disclaimer in the API
payload itself. Lodestar makes no medical claims.

## Development notes

- Conventional Commits. Small, reviewable commits.
- Every non-obvious decision gets an ADR.
- `internal/domain` imports nothing from `internal/store` or `internal/http`.
- Integration tests hit a real Postgres via testcontainers, never a mocked SQL layer.
- `make check` is what CI runs.
