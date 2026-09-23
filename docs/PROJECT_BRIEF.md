# Project Brief — Lodestar (Calisthenics Trainer & Skill Tree)

> Authored by Jaelricco. This is the canonical spec; ADRs record where
> implementation has since diverged, and `docs/DDL_PROPOSAL.md` is the
> schema proposal that §3 asks for.

---

## 0. Mission

Build a calisthenics training app with two tightly coupled halves:

1. **A precision logger.** An athlete must be able to reproduce their real
   session in the app almost exactly: sets, reps, isometric holds, added
   weight, band assistance, tempo, rest intervals, RPE/RIR, and **combos**
   (e.g. `full planche → press to handstand → full planche` logged as a single
   set with three ordered elements).
2. **A skill map.** A DAG of every major calisthenics skill (front lever,
   planche, maltese, muscle-up, handstand, human flag, …) rendered as a star
   constellation. Nodes are locked / available / in-progress / unlocked.
   Unlocks are **detected automatically from logged training** where possible,
   with a manual self-attest fallback.

The logger feeds the map. That coupling is the product. Do not build them as
two disconnected features.

---

## 1. Stack — fixed

| Layer | Choice | Notes |
|---|---|---|
| Backend | **Go 1.23+** | `chi` router, `pgx/v5`, **`sqlc`** for typed queries, `goose` for migrations. No ORM. |
| DB | **PostgreSQL 16** | UUIDv7 primary keys, `citext`, `pg_trgm`, JSONB for criteria only. |
| API | REST + **OpenAPI 3.1** | Spec is source of truth; Swift client generated from it. |
| Client | **iOS, SwiftUI**, iOS 17+ | Swift Concurrency, `swift-openapi-generator`, **GRDB** local store. |
| Local dev | **Docker Compose** | api, postgres, migrate, minio, mailpit, pgweb. |
| CI | GitHub Actions | lint, test, migrate-check, OpenAPI diff, build. |
| Auth | Sign in with Apple + email/password | JWT access (15 min) + rotating refresh token. argon2id. |
| Object storage | MinIO local / S3-compatible prod | Skill demo media + user form-check clips. |

If Rust looks like a better fit, say so once in an ADR and then proceed with Go
anyway unless overruled. Structure the domain layer so the persistence and HTTP
layers are thin enough to port. (Answered in ADR 0001.)

---

## 2. Repo layout

```
.
├── cmd/
│   ├── api/                  # HTTP server entrypoint
│   ├── seed/                 # content importer (YAML -> DB)
│   └── contentlint/          # validates content/ against JSON Schema
├── internal/
│   ├── domain/               # pure types + business rules, zero I/O
│   │   ├── training/         # sessions, sets, set elements, assistance
│   │   ├── skills/           # skill graph, levels, states
│   │   └── progress/         # unlock rules engine, XP, streaks
│   ├── store/                # sqlc-generated code + repositories
│   ├── http/                 # handlers, middleware, problem+json errors
│   ├── auth/
│   ├── sync/                 # delta sync endpoints
│   └── media/
├── db/
│   ├── migrations/           # goose, numbered, never edited once merged
│   └── queries/              # .sql files consumed by sqlc
├── content/                  # THE RESEARCH LIVES HERE (see §7)
│   ├── schema/*.schema.json
│   ├── skills/*.yaml
│   ├── exercises/*.yaml
│   ├── bands/*.yaml
│   └── injuries/*.yaml
├── api/openapi.yaml
├── ios/                      # Xcode project (later phase)
├── docker/
├── docs/adr/                 # one ADR per significant decision
├── docker-compose.yml
├── Makefile
└── .env.example
```

---

## 3. Domain model — the important part

Most of the project's value is in getting these tables right. Propose the full
DDL and **stop for review before writing migrations.**

### 3.1 Exercises vs skills

- **Exercise** — something you perform and log. `front lever hold (straddle)`,
  `weighted pull-up`, `band-assisted muscle-up`.
- **Skill** — a node on the map. `Front Lever`. It has ordered **skill levels**
  (`tuck → advanced tuck → straddle → half lay → full`).
- A skill level maps to one or more exercises via `skill_level_exercises` with
  a `role`: `primary_test`, `progression`, `accessory`, `prehab`.

Never conflate them. The logger writes exercises; the map reads skills.

### 3.2 Logging tables

```
workout_sessions
  id, user_id, started_at, ended_at, timezone, title, notes,
  perceived_fatigue, bodyweight_kg, status(draft|completed|abandoned),
  template_id?, client_id, updated_at, deleted_at

session_blocks           -- superset / circuit / straight grouping
  id, session_id, order_index, kind(straight|superset|circuit|emom|amrap),
  rounds_planned, rounds_done, notes

set_entries
  id, block_id, order_index,
  kind(working|warmup|backoff|drop|cluster|test),
  is_planned bool,                 -- planned vs actually performed
  rest_after_planned_s, rest_after_actual_s,
  rpe numeric(3,1), rir int,
  completed_at, notes

set_elements             -- >>> THE COMBO MECHANISM <<<
  id, set_entry_id, order_index,
  exercise_id,
  measure(reps|hold_seconds|distance_m|none),
  reps int?, hold_seconds numeric?, distance_m numeric?,
  tempo text?,                     -- "30X1"
  load_kg numeric,                 -- external load; negative = counterweight
  is_eccentric_only bool, is_partial_rom bool,
  rom_note text?,
  form_quality smallint?,          -- 1..5, drives unlock confidence
  failed bool

set_element_assistance
  id, set_element_id,
  type(none|band|partner|machine|incline|counterweight|foot_support),
  band_id?, band_count int,
  anchor(overhead|under_foot|under_knee|hip|other)?,
  estimated_assist_kg numeric?,
  note text

bands
  id, owner_user_id?,              -- NULL = global catalog entry
  brand, colour_label, resistance_min_kg, resistance_max_kg,
  length_cm, thickness_mm
```

A normal set of 8 pull-ups is **one** `set_entry` with **one** `set_element`.
A planche combo is one `set_entry` with three ordered `set_elements`. There is
no second code path — enforce this.

### 3.3 Skill graph

```
skill_families            -- push, pull, core, legs, handstand, dynamic, mobility
skills
  id, slug UNIQUE, family_id, name, aka text[],
  difficulty_tier smallint,        -- 1..10
  is_milestone bool,               -- shown large on the map
  summary, primary_muscles text[], common_faults text[]

skill_levels
  id, skill_id, order_index, slug, name,
  description, unlock_criteria jsonb,   -- see §6
  est_weeks_from_prev smallint?

skill_edges                            -- the DAG
  from_skill_level_id, to_skill_level_id,
  relation(prerequisite|recommended|alternative|antagonist),
  weight numeric

skill_level_exercises
  skill_level_id, exercise_id, role
```

Constraints: reject cycles at seed time (topological sort in `contentlint`).
A skill level with no inbound `prerequisite` edge is a root and is available
from account creation.

### 3.4 User progress

```
user_skill_states
  user_id, skill_level_id,
  state(locked|available|in_progress|unlocked),
  best_value numeric?, best_unit text?,
  first_achieved_at?, evidence_set_entry_id?,
  verification(auto|self_attested|coach), attempts_count,
  updated_at

skill_unlock_events        -- immutable log, drives the celebration UI
user_xp_events             -- {source, amount, ref_id, occurred_at}
user_streaks               -- current, longest, freeze_credits
badges / user_badges
```

### 3.5 Map/constellation layout

Store node positions in content (`x`, `y`, `constellation` group) so the map is
stable and designed, not force-directed at runtime. Force-directed graphs of
80+ nodes look like spilled spaghetti.

---

## 4. API surface

Generate `api/openapi.yaml` first, then implement. Errors use
`application/problem+json` (RFC 9457).

```
POST   /v1/auth/apple            POST /v1/auth/register  /login  /refresh  /logout
GET    /v1/me                    PATCH /v1/me
GET    /v1/exercises             ?q=&family=&equipment=
GET    /v1/skills                # full graph, ETag-cached, content-version keyed
GET    /v1/skills/{slug}
GET    /v1/me/skill-map          # graph + this user's states in one payload
POST   /v1/me/skills/{levelId}/attest
GET    /v1/templates             POST /v1/templates  ...
POST   /v1/sessions              # create draft
PATCH  /v1/sessions/{id}
POST   /v1/sessions/{id}/complete   -> returns unlocked[] for the celebration
GET    /v1/sessions              ?from=&to=&cursor=
GET    /v1/me/stats/exercises/{id}   # PR history, volume, hold-time curve
POST   /v1/media/uploads         # presigned PUT
GET    /v1/sync?cursor=          # delta pull
POST   /v1/sync                  # batch push, Idempotency-Key required
```

Sync contract: client generates UUIDv7 ids; every syncable row has
`updated_at` and nullable `deleted_at`; conflicts resolve last-write-wins on
server-received timestamp **except** `set_elements`, where the client copy wins
(the athlete was there, the server wasn't). Document this in an ADR.

---

## 5. Rest timer & logger UX requirements (drives API design)

- Timer starts automatically when a set is marked complete; planned rest comes
  from the template, actual rest is measured and stored.
- "Repeat last set" must be one tap. Pre-fill from the same exercise's most
  recent set.
- Everything works with **zero network**. Queue and sync later.
- Hold-based exercises need a running stopwatch with haptic marks at the
  target, not a manual number entry.

---

## 6. Unlock rules engine

`skill_levels.unlock_criteria` holds a small declarative DSL. Implement the
evaluator in `internal/domain/progress` as a pure function:

```go
func Evaluate(c Criteria, h SetHistory, now time.Time) (Result, error)
```

Example criteria:

```json
{
  "all": [
    {
      "exercise": "front-lever-full",
      "measure": "hold_seconds",
      "op": ">=", "value": 5,
      "assistance": "none",
      "max_load_kg": 0,
      "min_form_quality": 4,
      "occurrences": 2,
      "within_days": 30
    }
  ],
  "any": [
    { "exercise": "front-lever-half-lay", "measure": "hold_seconds",
      "op": ">=", "value": 12, "occurrences": 1 }
  ]
}
```

Rules:
- Evaluate on session completion; return newly unlocked levels in the response.
- Also recompute `available` transitions when prerequisites are satisfied.
- Never auto-revoke an unlock. Skills regress in real life but the map is a
  record of achievement, not current form. Track current form separately via
  `best_value` and a `stale_since` hint.
- Golden-file test suite: `testdata/criteria/*.json` + expected results.
  This is the one part of the codebase that must have near-total coverage.

---

## 7. Content pipeline

Deep research on skills, progressions, injuries and coaching cues is authored
separately and lands in `content/`. Build the pipeline **before** the content
exists, with 3–4 hand-written example files so the shape is proven.

- One YAML file per skill in `content/skills/`, validated by JSON Schema in
  `content/schema/`.
- `make content-validate` → `cmd/contentlint`: schema check, slug uniqueness,
  edge target existence, DAG acyclicity, orphan detection.
- `make seed` → `cmd/seed`: **idempotent upsert keyed on slug**, wrapped in one
  transaction, writes a `content_versions` row with a content checksum.
- Content is never inserted by a migration. Migrations = structure, seed = data.
- Include a `docs/CONTENT_AUTHORING.md` describing every field so content can be
  filled in without reading Go code.

Reference skill file:

```yaml
slug: front-lever
name: Front Lever
family: pull
difficulty_tier: 7
is_milestone: true
aka: [FL]
summary: >
  A horizontal body hold in supination-neutral grip with the body facing up...
primary_muscles: [latissimus_dorsi, lower_trapezius, rectus_abdominis, ...]
common_faults:
  - Hips sagging below shoulder line
  - Elbows bending to shorten the lever
map: { constellation: pull_north, x: 340, y: 120 }
levels:
  - slug: tuck
    name: Tuck Front Lever
    order: 1
    est_weeks_from_prev: 6
    exercises:
      - { slug: front-lever-tuck, role: primary_test }
      - { slug: scapular-pull-up, role: progression }
      - { slug: band-assisted-tuck-fl, role: progression }
    unlock_criteria: { all: [ { exercise: front-lever-tuck, measure: hold_seconds, op: ">=", value: 15, assistance: none, occurrences: 2, within_days: 30 } ] }
    prerequisites: [ { skill: pull-up, level: strict-5 } ]
injuries:
  - region: elbow
    name: Medial epicondylalgia
    risk_factors: [Rapid volume jumps, Full-lay attempts before straddle is solid]
    prehab_exercises: [ { slug: pronated-curl-eccentric } ]
    disclaimer: educational_only
```

**Do not invent skill data.** Generate the schema, the tooling, and 3–4
placeholder files clearly marked `status: draft_placeholder`.

Add a global disclaimer surface: injury content is educational, not medical
advice. Put it in the API payload, not just the app's footer.

---

## 8. Gamification — with guardrails

Ship: skill map states, unlock events with a celebration payload, XP per
logged session weighted by quality not volume, badges, training streaks.

Explicitly **do not** build mechanics that punish rest:
- Streaks count *planned* rest days and deloads as maintained.
- Provide streak freezes.
- No push notifications implying the user is falling behind.
- No leaderboard in v1.

Overuse injury is the dominant risk in calisthenics. A gamified trainer that
rewards daily maximal attempts is actively harmful. Noted in ADR 0003.

---

## 9. Infrastructure

`docker-compose.yml` with: `postgres:16` (named volume, healthcheck), `api`
(air/hot reload in dev), `migrate` (one-shot goose), `minio` + `minio-init`,
`mailpit`, `pgweb`. `docker compose up` must yield a seeded, working API with
no further steps.

`Makefile` targets: `up down logs psql migrate-up migrate-down migrate-new
sqlc gen-openapi gen-ios-client seed content-validate lint test test-integration
cover reset`.

CI: `golangci-lint`, `go test ./...`, integration tests via **testcontainers**,
`goose status` check against a fresh DB, OpenAPI breaking-change diff,
`docker build`.

`.env.example` with every variable documented. No secrets in the repo, no
default admin password.

---

## 10. iOS app (Phase 5+)

- SwiftUI, iOS 17+, `@Observable`, structured concurrency. No third-party
  architecture framework.
- `swift-openapi-generator` client from `api/openapi.yaml` — never hand-write
  DTOs.
- GRDB local database mirroring the sync-relevant tables; the UI reads local
  only, the sync engine writes it. Offline is the default path, not a fallback.
- Screens for v1: Today/Log, Session player (the logger), History, Skill Map,
  Skill detail, Profile.
- Skill map: SwiftUI `Canvas` (or SpriteKit if pan/zoom perf demands it), with
  positions from content. Dim locked nodes, glow unlocked ones, animate edges
  on unlock.
- Logger UI: dark, high-contrast, large tap targets, one-handed reachable,
  haptics on set completion, timer survives backgrounding (Live Activity).
- Localization from day one: `en` and `de` (de-CH). Metric units, kg, ISO
  weeks, Monday-first calendars. Keep all user-facing strings in catalogs.
- Accessibility: Dynamic Type, VoiceOver labels on map nodes.

---

## 11. Conventions

- Conventional Commits. Small, reviewable commits.
- Every non-obvious decision → `docs/adr/NNNN-title.md` (context / decision /
  consequences).
- `internal/domain` imports nothing from `internal/store` or `internal/http`.
- Errors wrapped with `%w`, never swallowed. Structured logging via `slog`,
  request IDs propagated.
- No `panic` outside `main`.
- Table-driven tests. Integration tests hit real Postgres, never mocks of SQL.

**Do not:** add GraphQL, add Redis before there is a measured need, add a
message queue, split into microservices, generate fake skill/exercise content,
add analytics SDKs, or make medical claims.

---

## 12. Phase plan — stop at every checkpoint

At the end of each phase, summarise what was built, list open questions, and
**wait for go-ahead.**

| Phase | Deliverable |
|---|---|
| **0** | Repo scaffold, Docker Compose, Makefile, CI skeleton, ADR 0001 (stack), and a written proposal of the full DDL from §3 for review. **Stop.** |
| **1** | Migrations, sqlc setup, content JSON Schemas, `contentlint`, `seed`, 3–4 placeholder skill files. `make up && make seed` works end to end. |
| **2** | Auth + core API: exercises, sessions, blocks, sets, set elements, assistance. OpenAPI spec complete for these. Integration tests. |
| **3** | Unlock rules engine + golden tests, `/v1/me/skill-map`, `/complete` returning unlocks, XP/streaks. |
| **4** | Sync endpoints, idempotency, media uploads via MinIO presigned URLs. |
| **5** | iOS: project setup, generated client, GRDB store, sync engine, auth, and the session logger including combos and the rest timer. |
| **6** | iOS: skill map constellation, skill detail, unlock celebration, history/stats. |
| **7** | Polish: localization, accessibility pass, TestFlight build, seed-content import of the research, load smoke test. |

---

## 13. Open decisions — settled in ADR 0002

1. App name and bundle identifier → **Lodestar**, `ch.riske.lodestar`.
2. Hosting target for prod → **Hetzner Cloud**.
3. Single-user v1, or coach↔athlete in the schema → **single-user, no schema debt**.
4. User-uploaded form-check videos in v1 → **generic media schema, images-only UI**.
5. Shareable templates in v1 → **no; private-only with the columns in place**.
