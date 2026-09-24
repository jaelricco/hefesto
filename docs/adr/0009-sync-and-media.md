# ADR 0009 — Delta sync, offline push, idempotency and media

- Status: accepted
- Date: 2026-09-24
- Deciders: Jaelricco

## Context

The logger has to work with no network (brief §5), so the iOS client (Phase 5)
keeps a local database and syncs it. The schema was built for this from the
start: every syncable row carries `server_seq`, `updated_at` (the client's
clock), `server_updated_at` and `deleted_at`, and ids are client-generated
UUIDv7s. The brief (§4) asks for three things:

- a delta pull and a batched push that requires `Idempotency-Key`;
- conflicts resolved last-write-wins on the server's receive time, except
  that for `set_elements` the client copy wins;
- media uploaded through presigned URLs.

This ADR records how each is done.

## Decisions

### Pull: one cursor over a per-user total order

`GET /v1/sync?cursor=N` returns every unit that changed with
`server_seq > N`, deletions included, and the cursor to send next.

- **No gaps.** `server_seq` comes from `next_sync_seq(user_id)`, which
  increments the user's own row. That row lock serialises a user's writing
  transactions, so they commit in `server_seq` order. The page is read from
  one repeatable-read snapshot, so a cursor never skips a change that commits
  later with a lower number.
- **A set is one unit.** It comes with its current elements, their assistance
  and their media. Its `server_seq` is the latest change to any of those rows.
  This matches how sets are written (always whole), and the client already has
  the `SetEntry` shape from REST.
- **A unit appears once per page, at its latest change,** carrying its current
  state. Because of that, a set can appear before its block when the block
  changed later. The page then also carries the block (and likewise sessions
  for blocks and sets), so every page applies on its own in the order
  sessions → blocks → sets. The parent comes again at its own position later;
  re-applying it is a harmless upsert.
- Each device's acknowledged cursor is recorded in `devices.last_sync_seq`,
  for support and for any future pruning.

Tombstones are never pruned; they are how deletions reach other devices.

### Push: the REST code paths, in order, one result per op

`POST /v1/sync` takes up to 200 ops, each of the form
`{op, entity, id, session_id?, data?}`:

| Entity | Ops |
|---|---|
| `session` | `put`, `complete`, `delete` |
| `block` | `put`, `delete` |
| `set` | `put`, `delete` |
| `bodyweight` | `put`, `delete` |

- **Same code as REST.** Each op runs through exactly the function its REST
  endpoint uses, now shared as `applyCreateSession`, `applyPutSet` and so on.
  A set is still written whole, with its ordered elements. There is no second
  path, and so no "simple set" shortcut (CLAUDE.md).
- **`session.put` carries the session's full client-side state.** It creates
  the session or updates the fields it carries. `complete` is its own op
  because completion runs the unlock engine and returns the celebration
  (ADR 0008).
- **One result per op.** Each op runs in its own transaction and gets one of
  three results:
  - `applied`;
  - `superseded`: the server's copy wins, so the client drops the op and takes
    the server's copy at the next pull;
  - `rejected`: the op is invalid, and its problem has pointers into the op.

  One bad op does not block the queue behind it. A server error aborts the
  whole request, and a retry re-runs every op. That is safe because every op
  is idempotent: a put is an upsert, deleting a tombstone applies, and
  `complete` repeats its first result.
- **Bodyweight** has no REST endpoints. Sync is its API.
- **User bands and templates** are not in sync yet:
  - Bands have no sync columns. Clients create them online and refresh them
    with `GET /v1/bands`.
  - Templates have no API at all yet.

### Conflicts

- **Sessions, blocks and bodyweight: last write received wins.** The server
  applies a put when it arrives, whatever is stored.
- **Sets: the athlete's clock decides.** A set write whose `updated_at` is
  older than the stored copy's is not applied. Sync reports it as
  `superseded`; REST answers `409 stale-write`. This is how "the client copy
  wins for set elements" is read here. The copy the athlete edited last is the
  truth about what they did, even when a phone that was offline in the gym
  delivers an older edit hours later. The server never changes an element's
  values, apart from deriving `assistance_class`. A REST client that sends no
  `updated_at` is stamped with the current time and always wins. This check
  lives in the store's one set path, so REST and sync cannot diverge.
- **Deletions are final.** A write to a tombstone is `superseded` (404 over
  REST). This holds for children of a deleted session too.
- **Completion is final.** `session.put` never changes a completed session's
  status. Unlocks are never revoked (CLAUDE.md).

### Idempotency

`Idempotency-Key` is required on `POST /v1/sync` and is kept for 24 hours per
user in `idempotency_keys`.

- **Fingerprint.** Each request is fingerprinted with SHA-256 over its method,
  path and body.
- **Same key and body:** the stored response is replayed, with
  `Idempotent-Replayed: true`.
- **Same key, different body:** `422 idempotency-key-reuse`, a client bug.
- **Same key while the first request is still running:**
  `409 idempotency-in-progress`. A claim stalled for five minutes (a crashed
  process) can be taken over by the same request.
- **Failed requests.** A request that fails as a whole, for a malformed
  envelope or a server error, gives its key back so a retry runs again.
- Expired keys are purged hourly.

### Media: presigned URLs, private bucket, images only

Clients never send bytes through the API:

1. `POST /v1/media/uploads` reserves a `pending` asset and returns a presigned
   `PUT`. Its signature covers `Content-Type` and `Content-Length`, so the
   bucket refuses any other file.
2. The client uploads the file straight to the bucket.
3. `POST /v1/media/{id}/complete` checks the stored object's size and type
   against the reservation:
   - if they match, the asset becomes `ready`;
   - if not, the object is removed and the asset is marked `failed`.
4. `GET /v1/media/{id}` returns a short-lived download URL. The bucket stays
   private.

Further rules:

- **Images only.** JPEG, PNG, HEIC and WebP, up to 15 MB (ADR 0002 §4). The
  schema is ready for video.
- **Attaching.** Assets attach to set elements through `media_ids` on the one
  set write path. Only the athlete's own ready images can be attached.
- **Deleting.** Deleting an asset detaches it and touches the elements, so the
  detachment syncs.
- **Object keys** are `u/<user id>/<asset id>`. The account reaper removes a
  user's prefix before hard-deleting the row, and keeps the account for the
  next run if that fails. Uploads never completed are marked failed and
  removed after 24 hours.
- **Two endpoints.** The API talks to storage at `HEFESTO_S3_ENDPOINT`, but
  signs URLs for `HEFESTO_S3_PUBLIC_ENDPOINT`. In Compose the phone cannot
  reach `minio:9000`.
- **Storage is optional in dev only.** Without it the media endpoints answer
  `503`. Outside dev the API refuses to start without it.

## Consequences

- The iOS sync engine is simple:
  - pull pages until `has_more` is false, applying each page in one local
    transaction;
  - push the local queue in batches under one key per batch;
  - drop `superseded` and `rejected` ops.
- A full resync from cursor 0 costs one read per unit. The indexes on
  `(user_id, server_seq)` serve it.
- Clock skew on a phone can make its set edits lose to (or beat) another
  device's. That is the accepted cost of trusting the athlete's clock for
  sets. Everything else follows the server's order.
- The integration tests run against MinIO in CI (testcontainers). Locally,
  `HEFESTO_TEST_S3_ENDPOINT` can point at any S3-compatible server.
