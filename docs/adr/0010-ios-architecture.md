# ADR 0010 — iOS app architecture

- Status: accepted
- Date: 2026-09-25
- Deciders: Jaelricco

## Context

Phase 5 starts the iOS client. The brief (§10) fixes the platform (SwiftUI,
iOS 17+, `@Observable`, structured concurrency, no third-party architecture
framework), the generated API client, a GRDB database that mirrors the synced
tables, and offline as the default path. ADR 0009 fixes the sync contract the
client has to speak. What remains open is how the code is laid out, how the
local database and outbox work, and how the logger holds its state.

## Decisions

### Layout: a thin app target over one local package

The repository holds the app and its logic in `ios/`:

```
ios/
  project.yml                  XcodeGen; the .xcodeproj is generated, not committed
  Hefesto/                     the app target: SwiftUI views, string catalog, assets
  Packages/HefestoKit/         everything that is not a view, in testable targets
    HefestoAPI                 client generated from api/openapi.yaml
    HefestoStore               GRDB database, migrations, records, outbox
    HefestoAuth                sign-in, token storage and refresh
    HefestoSync                push then pull, over the outbox and the store
    HefestoLogger              the session logger's model and the rest timer
```

- **The app target holds only views.** Everything else is in the package's
  targets, which build and test with `swift test` on macOS and on Linux. The
  views are thin enough that CI building them on a simulator is enough.
- **XcodeGen** generates the project from `project.yml`. It is a build tool,
  not an architecture framework. A generated project cannot have merge
  conflicts, and nobody has to hand-edit a `project.pbxproj` file.

### The API client is generated, and nothing else is

`HefestoAPI` runs the `swift-openapi-generator` build plugin on
`api/openapi.yaml`. The file is symlinked, not copied, so it cannot drift. DTOs
are never written by hand. A request middleware adds the access token.

### The UI reads only the local database

GRDB holds the athlete's sessions, blocks, sets, bodyweight and the exercise
catalogue. Views observe the database through GRDB's value observation, and
never wait for the network.

- **Tables mirror the sync feed** (ADR 0009). A set's elements are rows of
  their own. Assistance is a column group on its element, because it is
  one-to-one. An element's `media_ids` are stored as JSON.
- **Ids are UUIDv7, generated on the device.** They are the same ids the server
  stores.

### Every local write goes through the outbox

A write changes the local rows and queues an op in the same database
transaction, so the two cannot disagree.

- **Op shapes.** Ops take exactly the shapes of the sync push (ADR 0009). A set
  is always queued whole, with its elements.
- **Coalescing.** A newer op on the same row replaces the queued one but keeps
  its position, so parents stay ahead of children. A delete replaces a queued
  put.
- **Resending.** A batch is sent under one `Idempotency-Key`, and its exact body
  is stored with the key. A retry after a lost response resends the same bytes,
  so the server replays its answer instead of applying anything twice.

### Sync: push, then pull

`SyncEngine.sync()` runs one push-then-pull cycle. The app triggers it on
launch, on foregrounding, after a session is completed, and when the network
comes back.

- **Push.** Each result settles its op:
  - `applied` removes it;
  - `superseded` removes it, and the next pull overwrites the local copy;
  - `rejected` removes it and records the problem for the UI.
- **Completion.** An applied `complete` op carries the unlocks, which the logger
  shows as the celebration.
- **Pull.** Pages are applied in one transaction each, parents first. A row
  with a queued op is left alone: the local edit is newer and still on its
  way. The cursor is stored only after its page is applied.

### Auth

`AuthService` signs in with email and password, or with Sign in with Apple.

- **Token storage.** Tokens live in the Keychain on Apple platforms, behind a
  protocol so tests and Linux use memory.
- **Refresh.** The access token is refreshed once, shortly before it expires or
  after a 401, and concurrent requests wait for the same refresh.

### The logger's model

`LoggerModel` is an `@Observable` class over one session. It is the only
place views change a session, and every change goes through the store and the
outbox.

- **One code path for sets** (CLAUDE.md). A plain set is a set with one
  element; a combo adds elements to the same set.
- **Repeat last set** copies the athlete's most recent set of the same
  exercise from the local database, in one tap.
- **The rest timer** is a value: when it started and the planned rest. Time
  left is computed from the wall clock, so it is right after the app was in
  the background. A local notification marks the end of the rest. The actual
  rest is recorded on the set when the next one starts.

## Consequences

- The logic that matters (outbox, sync, logger, timer) is tested without a
  simulator, on Linux in this environment and on macOS in CI.
- Adding a synced table means a GRDB migration, a record, a pull mapping and
  an op mapping. That is deliberate friction that mirrors the server.
- The Live Activity for the rest timer needs a widget extension target. It
  comes with the polish phase; until then a local notification carries the
  timer.
