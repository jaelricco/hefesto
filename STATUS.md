# Status

The project is built in the numbered phases of `docs/PROJECT_BRIEF.md` §12.
Each phase ends with a summary and a stop for review. This file records where
that stands.

## Done

- **Phase 0** — scaffold, Compose, CI, deploy pipeline, ADRs 0001–0004, DDL proposal.
- **Phase 1** — migrations, content pipeline, seed (ADRs 0005–0006). Merged in #4.
- **Phase 2** — auth, catalogue and the logging API (ADR 0007). Merged in #5.
- **Phase 3** — unlock engine, XP and streaks (ADR 0008). Merged in #6. Its open
  questions (prerequisite gating, self-attest XP, daily vs weekly streaks,
  occurrences per set) still stand.

## Current phase: 4 — sync, idempotency and media (complete, awaiting review)

Built (`api/openapi.yaml` v0.4.0; the rules are in ADR 0009):

- **Delta pull**, `GET /v1/sync?cursor=`.
  - Every change after the cursor, deletions included. It pages over the
    per-user `server_seq` from one consistent snapshot, so no change is skipped.
  - A set comes whole, with its elements, their assistance and their media.
  - A page also carries any parent the client cannot have yet, so each page
    applies on its own.
  - The device's acknowledged cursor is recorded.
- **Batched push**, `POST /v1/sync`.
  - Up to 200 ops on sessions, blocks, sets and the bodyweight log: `put`,
    `delete`, and `complete` for a session.
  - Every op runs through the same function as its REST endpoint (now shared),
    so a set is still written whole.
  - Each op gets its own result:
    - `applied`;
    - `superseded`: the server's copy wins;
    - `rejected`: with its problem located in the op.
  - One bad op never blocks the queue.
  - An offline `complete` returns the unlock celebration.
- **Conflicts.** The last write received wins, except for sets. There the
  athlete's clock decides: an older set write is `superseded` over sync and
  `409 stale-write` over REST. Deletions and completion are final.
- **Idempotency.** `Idempotency-Key` is required on push and kept for 24 hours.
  - The same request is replayed from the stored response.
  - The same key with another body is a 422.
  - A concurrent duplicate is a 409.
  - A failed request gives its key back.
- **Media**, through presigned URLs to a private bucket (MinIO in dev, Hetzner
  Object Storage in prod):
  - `POST /v1/media/uploads` reserves an asset and signs a `PUT` bound to its
    type and size.
  - `POST /v1/media/{id}/complete` checks what arrived.
  - `GET /v1/media/{id}` signs a download; `DELETE` removes it.
  - Images only (ADR 0002 §4).
  - Attached to set elements with `media_ids` on the one set path.
- **Housekeeping**, hourly:
  - deleting an account also removes its objects;
  - uploads never completed expire after 24 hours;
  - expired idempotency keys are purged.
- **Tests.**
  - The S3 client runs against real storage (MinIO via testcontainers in CI).
  - 10 new HTTP integration tests cover:
    - paging and tombstones in the feed;
    - parents brought forward;
    - an offline session pushed whole and completed, with its replay and key
      reuse;
    - stale set writes over sync and REST;
    - writes to tombstones;
    - per-op rejections;
    - bodyweight via sync;
    - the media upload, attach, sync and delete flow;
    - a mismatched upload;
    - media without storage;
    - the reaper removing objects.
  - Deliberately breaking three behaviours showed the tests catch each:
    - pulling parents forward;
    - the stale-set rule;
    - syncing a detachment.

### Verification

Run in the development container against PostgreSQL 16:
- `golangci-lint` is clean.
- `go test -race -short ./...` and `go test -race -tags integration ./...` are
  green.
- `sqlc` output is current.
- `redocly lint` is clean apart from the four warnings from Phase 0.
- `oasdiff` reports no breaking change against `main`.

This container cannot pull Docker images or download MinIO. The storage tests
therefore ran against an S3-compatible stand-in (`gofakes3`, run from outside
the repository) through `HEFESTO_TEST_S3_ENDPOINT`. That stand-in does not
check signatures, so the assertions that a presigned upload refuses another
type or size run only against real MinIO, which is what CI uses.

### Deliberately not in this phase

- **Templates** in sync: there is no templates API yet.
- **User bands** in sync: bands have no sync columns. They are created online
  and refreshed with `GET /v1/bands`. Adding them is an expand migration when
  the app needs offline band creation.
- **Video**: the schema and endpoints are ready, but only images are accepted.
- **Upload checksums**: the size and type are signed and checked; content
  hashes are not.

### Open questions for review

1. **The set conflict rule.** I read "the client copy wins for set_elements"
   as "the athlete's clock decides between two copies of a set". Is that the
   intent? The alternative is plain last-received-wins, which lets a phone
   that was offline overwrite a later correction.
2. **Per-op results rather than all-or-nothing.** A rejected op is dropped by
   the client. Is it acceptable that invalid data never reaches the server,
   given it is only possible through a client bug?
3. **Offline band creation.** Should bands get sync columns now, or wait for
   the app to need them?
4. **Media retention.** Should an asset whose element is deleted also be
   deleted? Today it stays until the athlete deletes it or the account goes.

## Next: Phase 5

iOS: project setup, generated client, GRDB store, sync engine, auth, and the
session logger including combos and the rest timer.
