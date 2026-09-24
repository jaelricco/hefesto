# Status

The project is built in the numbered phases of `docs/PROJECT_BRIEF.md` §12.
Each phase ends with a summary and a stop for review. This file records where
that stands.

## Done

- **Phase 0** — scaffold, Compose, CI, deploy pipeline, ADRs 0001–0004, DDL proposal.
- **Phase 1** — migrations, content pipeline, seed (ADRs 0005–0006). Merged in #4.

- **Phase 2** — auth, catalogue and the logging API (ADR 0007). Merged in #5.

## Current phase: 3 — unlock engine, XP and streaks (complete, awaiting review)

Built:

- **Unlock engine** (`internal/domain/progress`). All of it is pure and has
  100 % statement coverage.
  - `Evaluate` checks criteria against the athlete's completed sets.
  - `Advance` moves levels through `locked → available → in_progress →
    unlocked`, in graph order and gated on prerequisites.
  - `SessionXP`, `UnlockXP` and `StreakXP` award XP.
  - `ComputeStreak` computes streaks with freezes.
  - The rules are in ADR 0008.
- **Golden tests**: 26 cases in `testdata/criteria/`, one per rule. They
  cover the brief's example, combos, windows, assistance and load, form
  quality, excluded elements, defaults, evidence choice, and seven
  invalid-criteria cases.
- **API** (`api/openapi.yaml` v0.3.0):
  - `POST /v1/sessions/{id}/complete` returns the levels unlocked, the levels
    newly available, the XP awarded and total, and the streak. Completing
    twice is idempotent.
  - `GET /v1/skills` returns the graph, with an `ETag` on the content version.
  - `GET /v1/skills/{slug}` returns a skill with its injury notes. The
    educational-only disclaimer is in the payload.
  - `GET /v1/me/skill-map` returns the graph and the athlete's state on every
    level in one payload.
  - `POST /v1/me/skills/{levelId}/attest` records a self-attested unlock.
  - `GET /v1/me/progress` returns XP and the streak.
- **Completion** runs in one transaction under a per-user advisory lock. It
  marks the session completed, records the training day, evaluates, writes
  the states and append-only unlock events, awards XP (unique per source and
  reference, so it cannot be double-awarded), and recomputes the streak.
- **Tests**: 8 new HTTP integration tests against real Postgres.
  - A partial unlock, then an unlock.
  - Assisted and draft sets do not count.
  - Evidence, newly available levels and the XP amounts.
  - An idempotent repeat.
  - The first session of the day.
  - Unlocks survive deleting their evidence.
  - A prerequisite chain unlocking in one completion.
  - Self-attest, and the 409.
  - Rest days and freezes keep a streak.
  - The 7-day milestone.
  - Completion errors.
  - The graph with its `ETag`/304, edges, and the injury disclaimer.

### Verification

Run in the development container against PostgreSQL 16:
- `golangci-lint` is clean.
- `go test -race -short ./...` and `go test -race -tags integration ./...` are
  green.
- `sqlc` output is current.
- `redocly lint` is clean apart from the four warnings from Phase 0.
- `oasdiff` reports no breaking change against `main`.

Writing the integration tests caught a real bug, now fixed. Postgres checks a
`CHECK` constraint against the proposed `INSERT` row before `ON CONFLICT`
applies. Re-saving an unlocked level therefore has to send its
`first_achieved_at`.

### Deliberately not in this phase

- **`user_exercise_bests` and `make rebuild-bests`.** Nothing reads them yet.
  Evaluation reads the history directly, which is fast enough for years of
  logs. They arrive when a screen needs personal bests.
- **Badges.** The tables exist, but there is no badge content to award.
- **Plan adherence and deload XP.** Both need plans, which have no endpoints
  yet. Deload days already count for streaks once something records them.

### Open questions for review

1. **Prerequisite gating.** Should evidence for an advanced level unlock the
   levels before it too? At the moment a met level stays locked until its
   prerequisites unlock, although one completion can unlock a whole chain.
2. **Self-attested unlocks earn no XP.** Is that too strict for athletes who
   arrive already strong?
3. **Daily streaks.** A three-times-a-week athlete needs to log rest days or
   spend freezes. Would a weekly streak ("trained N times this week") suit the
   brief better?
4. **Occurrences count set entries**, so two qualifying sets in one session
   satisfy `occurrences: 2`. Should it be two separate sessions instead?
5. Phase 2's questions still stand (registration revealing a taken email,
   in-memory rate limits, per-account device ids).

## Next: Phase 4

Sync endpoints, idempotency, and media uploads through MinIO presigned URLs.
