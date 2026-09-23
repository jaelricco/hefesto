# ADR 0001 — Backend stack: Go, Postgres, sqlc, goose

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

Hefesto is a training logger plus a skill-graph engine, served to an offline-first
iOS client. The workload is:

- almost entirely I/O-bound CRUD with a sync protocol on top;
- one genuinely interesting piece of computation — the unlock rules evaluator —
  which is pure, small, and has no performance requirement worth naming
  (it runs once per completed session over a bounded slice of set history);
- a content pipeline that turns authored YAML into database rows.

Concurrency demand is trivial. There is no streaming, no media transcoding in v1,
no hot loop.

## Decision

Go 1.24 with:

| Concern | Choice |
|---|---|
| Router | `chi/v5` — `net/http` semantics, no framework lock-in, middleware is just `http.Handler` |
| Driver | `pgx/v5` native interface (not `database/sql`) — real Postgres types, `COPY`, proper `numeric` handling |
| Queries | `sqlc` — hand-written SQL, generated typed Go. No ORM. |
| Migrations | `goose` — plain numbered SQL, up and down, never edited after merge |
| DB | PostgreSQL 16 with `citext` and `pg_trgm` |
| Errors | wrapped with `%w`, `application/problem+json` at the edge (RFC 9457) |
| Logging | `log/slog`, JSON outside dev, request ID propagated on context |
| Tests | table-driven; integration tests against real Postgres via testcontainers, never a mocked SQL layer |

The domain layer (`internal/domain`) holds pure types and rules and imports
neither persistence nor transport. `internal/domain/arch_test.go` enforces this
as a test, so the rule fails CI rather than eroding quietly.

## The Rust question

The brief asks for one honest statement on Rust, so here it is: **Rust is not a
better fit for this project, and I would not switch even if the choice were
reopened.**

Where Rust would win — memory footprint, CPU-bound throughput, compile-time
elimination of data races — this project has no exposure. What it would cost is
concrete:

- The equivalent stack (`axum` + `sqlx` or `diesel` + `refinery`) is workable,
  but `sqlx`'s compile-time query checking requires a live database or a
  committed offline cache, which makes CI and migration workflows measurably
  more awkward than `sqlc`'s generate-and-commit model.
- `testcontainers-rs` is markedly less mature than the Go equivalent, and
  integration tests against real Postgres are the backbone of the test strategy
  here.
- Async Rust adds lifetime and `Send` friction to exactly the layer that should
  be boring (handlers, repositories) while offering nothing to the layer that is
  actually interesting (a pure evaluator that is equally easy in either language).

The one part of the codebase where a rewrite would ever be tempting is the
criteria evaluator, and it is deliberately pure: `Evaluate(criteria, history, now)`
with no I/O and golden-file tests. If it ever needed to move — to a shared core
compiled for the device, say — it ports in an afternoon in either direction.
That optionality is a property of the architecture, not of the language.

## Consequences

- `internal/store` and `internal/http` stay thin; business rules live in
  `internal/domain` and are testable without a database or an HTTP server.
- `sqlc` means the SQL is visible and reviewable. It also means schema changes
  require regenerating code — `make sqlc` runs in CI and a stale checkout fails.
- No ORM means no lazy loading and no accidental N+1 from a relation traversal;
  every query is written and read deliberately.
- Choosing `pgx`'s native interface over `database/sql` costs compatibility with
  generic SQL tooling; that is acceptable because Postgres is not negotiable here
  (arrays, JSONB, `numeric`, `citext`, `tstzrange` are all load-bearing).

## Notes recorded in this phase, to become their own ADRs once the schema is approved

- **UUIDv7 generation happens in application code, not Postgres.** Postgres 16 has
  no native `uuidv7()`; the `pg_uuidv7` extension would require a custom image.
  Since the sync contract already has clients generating ids, the server does the
  same via `github.com/google/uuid`. No table carries a `DEFAULT` for its primary key.
- **Closed vocabularies are `text` + `CHECK`, not Postgres `ENUM` types.**
  `ALTER TYPE ... ADD VALUE` cannot be used in the same transaction that then
  references the new value, which fights goose's transactional migrations. A named
  `CHECK` constraint is cheaper to evolve, and Go constants give the same type
  safety at the boundary. The cost is that sqlc emits `string` rather than a
  generated enum type.
