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
- **Phase 4** — sync, idempotency and media (ADR 0009). Merged in #7. Its open
  questions (the set conflict rule, per-op results, offline bands, media
  retention) still stand.

## Current phase: 5 — the iOS app's foundation and the logger (complete, awaiting review)

The app lives in `ios/` (ADR 0010). There is a thin SwiftUI target, generated
by XcodeGen from `project.yml`, over a local package, `HefestoKit`, which holds
everything that is not a view:

- **HefestoAPI**
  - The client is generated from `api/openapi.yaml` at build time; no DTO is
    written by hand.
  - Timestamps are read with or without fractional seconds, as Go writes them.
- **HefestoStore**
  - GRDB tables mirror the sync feed. Every local write queues its outbox op
    in the same transaction.
  - A newer put replaces a queued one in place, so parents stay ahead of
    children.
  - A batch freezes its payloads under one `Idempotency-Key` until it is
    settled.
  - Pull pages skip rows that still have a queued op, and the cursor never
    moves back.
  - Views observe streams from the store and never import GRDB.
- **HefestoAuth**
  - Email and password, or Sign in with Apple; tokens are kept in the
    Keychain.
  - The access token is refreshed once, before it expires or after a 401.
    Concurrent callers share the refresh.
  - A middleware adds the token and retries a refused request once.
- **HefestoSync**
  - Push, then pull.
  - A retry after a lost response resends the same bytes under the same key.
  - `superseded` fetches the server's copy of the session; `rejected` is kept
    as a problem.
  - An offline completion reports its unlocks.
  - The exercise catalogue is refreshed by ETag.
- **HefestoLogger**
  - `LoggerModel` is the only way a view changes a session.
  - A plain set is one element and a combo is several, on the same set path.
  - Repeat last set takes one tap.
  - The rest timer is computed from the wall clock, so it is right after the
    app was backgrounded. The actual rest is recorded on the previous set.

Screens:
- sign in and register, including Sign in with Apple;
- **Today**: start a session, log a planned rest day, see the sessions so far
  and a count of changes waiting to sync;
- **the logger**:
  - dark, high contrast, 56 pt targets;
  - blocks and sets;
  - a set composer where adding an exercise makes a combo;
  - repeat;
  - a rest timer with haptics and a local notification;
  - finish, with an optional perceived effort;
- **the unlock celebration** when the completion syncs.

Strings are in a catalog, in English and German. The app syncs on launch, on
foregrounding, after completing a session, and when the network returns.

One spec change came out of this phase. The generator silently dropped
`assistance` (and a skill's `map`), because they were written as
`oneOf: [$ref, null]`. The nullability now sits on the component. The JSON
Schema is the same, so the server's validation and responses are unchanged,
and the Go unit and HTTP integration suites are green on it.

### Verification

CI runs on a self-hosted Mac runner (`.github/workflows/ios.yml`). It runs
`swift test` on HefestoKit and builds the app for the iOS Simulator; it uses
whatever Xcode the machine has selected and changes nothing outside its
workspace.

- On the runner (run 7), `swift test` passes: 36 tests in 10 suites, covering
  the outbox, pull, sets, UUIDv7, the logger, the rest timer, auth, the auth
  middleware, push and pull. The app builds for the iOS Simulator with
  `** BUILD SUCCEEDED **`, under Swift 6 strict concurrency.
- The Go unit tests, `golangci-lint` and the HTTP integration tests (real
  PostgreSQL and MinIO) are green on the changed spec.

This container has no Swift toolchain (`download.swift.org` is not reachable),
so all Swift was compiled and tested on the runner.

### Deliberately not in this phase

- **History, the Skill Map and Skill detail**: Phase 6 in the brief, as is the
  full unlock celebration. This phase shows a plain sheet listing what
  unlocked. The brief places **Profile** in no phase yet.
- **The Live Activity** for the rest timer. It needs a widget extension; until
  it exists, the local notification carries the timer.
- **Band assistance in the logger.** A band names one of the athlete's bands,
  and there is no bands screen yet. The other kinds of assistance are there.
- **Editing a logged set's values in place.** Sets can be deleted and logged
  again; the model already supports editing (`editSet`).
- **Media attachments** from the app.

### Open questions for review

1. **Rest-day logging.** "Log a rest day" creates and completes a rest-day
   session at once, so it counts for the streak. Is that the planned-rest
   mechanic you want, or should rest days come only from a plan?
2. **Bodyweight time zone.** The feed does not carry a
   bodyweight entry's time zone. An entry edited on a device other than the
   one that created it takes that device's zone. Should `SyncBodyweight` carry
   `timezone`? That would be an additive spec change.
3. **Signing out.** Signing out keeps the local database, so the next account
   on the device would see it. Should signing out wipe it? It would wipe with
   a warning when changes are still unsynced.

## Next: Phase 6

iOS: the skill map constellation, skill detail, the unlock celebration, and
history and stats.
