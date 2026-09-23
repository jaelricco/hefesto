# ADR 0002 — v1 scope decisions (naming, hosting, multi-user, media, sharing)

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

Five decisions from the project brief §13 had to be settled before Phase 0,
because each one either fixes an identifier that is painful to change later or
determines whether a column exists in the first migration.

## Decisions

### 1. Name and identifiers

**Lodestar.** A lodestar is the star you navigate by — it matches the
constellation map, which is the product's signature screen.

| | |
|---|---|
| Go module | `github.com/jaelricco/lodestar` |
| iOS bundle id | `ch.riske.lodestar` |
| Database / role | `lodestar` |
| S3 bucket | `lodestar-media` |
| Problem type URIs | `https://lodestar.app/problems/{slug}` |

The App Store display name is not fixed by this ADR and can change without
touching the bundle id.

### 2. Production hosting: Hetzner Cloud

A single CAX-series ARM instance in Falkenstein or Nuremberg, running the same
Docker Compose topology as development behind Caddy for TLS. Postgres runs on
the box with `pgBackRest` shipping WAL to Hetzner Object Storage, which is
S3-compatible and therefore doubles as the production replacement for MinIO.
Deployment is a GitHub Actions job over SSH.

Rejected: Fly.io (more expensive for a persistent Postgres, and its managed
Postgres is explicitly not a managed service), AWS (operational surface far
beyond what a single-developer project needs).

Consequences: backups and Postgres upgrades are our responsibility. The restore
procedure must be documented and rehearsed before the first real user data
exists — not after. This is a Phase 7 checklist item.

### 3. Single-user v1, with no schema debt

No `coach_*` tables in v1. But two rules apply from the first migration:

- every user-owned row carries `user_id` — including deep rows such as
  `set_elements`, which could have reached it through joins;
- no handler queries "the current user's" data implicitly. Authorisation goes
  through one helper with an explicit `actor_id` and `subject_user_id`, which in
  v1 always match.

Adding coach↔athlete later is then a new link table plus a change to that one
helper, rather than an audit of every query in the codebase.

### 4. Media: generic schema, images-only UI

`media_assets` carries `kind ('image'|'video')`, `mime`, `bytes`, `width`,
`height`, `duration_s` and `poster_key` from the first migration, and the
presigned upload endpoint accepts both kinds. The iOS v1 picker offers images
only, gated by a server-advertised capability flag rather than a client
constant.

No transcoding pipeline in v1. When video is enabled, clips are stored as
uploaded with a client-side size and duration cap, and a transcoding decision
gets its own ADR.

### 5. Templates are private in v1

`workout_templates` gets `visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','unlisted','public'))`
and `source_template_id uuid NULL` for copy-on-import lineage. Only `'private'`
is reachable through the v1 API. Sharing becomes a new endpoint plus a
validation change, not a migration.

## Consequences

- Three columns exist in v1 that nothing reads yet (`visibility`,
  `source_template_id`, the video fields on `media_assets`). That is deliberate
  and cheap; each is documented in the schema so a future reader does not mistake
  them for dead weight.
- The `actor_id` / `subject_user_id` discipline costs a little ceremony in every
  handler from day one. It is the difference between adding coaching in a week
  and adding it in a month.
