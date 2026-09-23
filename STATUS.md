# Status

The project is built in the numbered phases of `docs/PROJECT_BRIEF.md` §12.
Each phase ends with a summary and a stop for review. This file records where
that stands.

## Current phase: 0 — scaffold (complete, awaiting review)

Built:

- Repository layout from brief §2, with `internal/domain` I/O-free and enforced
  by `internal/domain/arch_test.go`.
- Docker Compose dev stack (postgres, migrate, api with air, minio + init,
  mailpit, pgweb), Makefile, `.env.example`.
- CI (`.github/workflows/ci.yml`) and the tag-triggered deploy pipeline to
  Hetzner (`deploy.yml`, `scripts/deploy.sh`, ADR 0004, `docs/DEPLOYMENT.md`).
- `cmd/api` serving `/healthz` and `/readyz` (503 until a DB pool exists);
  `cmd/seed` and `cmd/contentlint` are entrypoints with their contracts
  documented and no implementation.
- ADRs 0001 (stack), 0002 (v1 scope), 0003 (gamification guardrails),
  0004 (deployment).
- The full schema proposal: `docs/DDL_PROPOSAL.md`. **No migrations exist.**

## Blocking Phase 1

`docs/DDL_PROPOSAL.md` needs a decision on:

- deviations D-1 … D-8 (§12), each a yes or no;
- open questions 1 … 7 (§13).

Once answered, the decisions are recorded (in the proposal and, where
non-obvious, an ADR) and Phase 1 turns the schema into goose migrations.

## Next: Phase 1

Migrations, sqlc setup, content JSON Schemas, `contentlint`, `seed`, 3–4
`draft_placeholder` skill files, `docs/CONTENT_AUTHORING.md`.
Done when `make up && make seed` works end to end.
