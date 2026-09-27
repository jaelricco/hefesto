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
- **Phase 5** — the iOS app's foundation and the logger (ADR 0010). Merged in
  #8. Its open questions (rest-day logging, the bodyweight time zone, wiping
  the database on sign-out) still stand.
- **Phase 6** — skill map, skill detail, celebration, history and stats
  (ADR 0011). Merged in #9. Its open questions (showing `stale_since`,
  self-attestation for levels with criteria, stats on the device) still stand.

## Current phase: 7 — polish (complete, awaiting review)

The decisions are in ADR 0012.

- **Localization**
  - `ios/scripts/check-strings.py` checks:
    - every user-facing literal is in the string catalog;
    - every key has a translated German entry with matching placeholders;
    - no key is unused;
    - counts before nouns have plural forms in both languages.
  - It runs in CI on every push.
  - The chart axes and the new accessibility strings were the only gaps. There
    are now 168 keys, all translated.
- **Accessibility**
  - The skill map:
    - respects Reduce Motion;
    - reads top to bottom with VoiceOver;
    - keeps locked labels readable (over 4.5:1);
    - has a **list view** grouped by standing, remembered per device.
  - Level rows and the rest banner no longer swallow their buttons into one
    VoiceOver element.
  - The rest clock scales with Dynamic Type. The effort sheet drops to three
    columns at accessibility sizes and scrolls.
  - Chart bars read as "date, value".
  - Logging a set, a finished rest and sign-in errors are announced.
  - A UI test target runs Apple's accessibility audit on every main screen, at
    the default size, the largest accessibility size and in German. It uses a
    Debug-only offline fixture (`-uiTestFixture`).
- **TestFlight**
  - `testflight.yml` runs on a GitHub-hosted Mac: automatic signing with an
    App Store Connect API key, build number = run number, upload with
    `xcodebuild -exportArchive`.
  - The app now has an icon (a generated placeholder), an export-compliance
    declaration, a privacy manifest, and version 0.7.0.
  - Setup is in `docs/DEPLOYMENT.md`.
- **Content import**
  - `contentlint` reports how much content is still placeholder (today all 3
    skills and 7 exercises).
  - It warns when a researched skill uses a placeholder exercise.
  - `make content-release-check` fails on any placeholder.
  - `CONTENT_AUTHORING.md` describes the import and the slugs production may
    already hold.
- **Load smoke test**
  - `cmd/loadsmoke`: virtual athletes log sessions through sync batches and
    REST, pull, and read the map. It fails on any error or a p95 over budget
    (300 ms reads, 400 ms writes, 1.5 s completion and sync batches).
  - CI runs 25 athletes for 60 s against the release build on a seeded
    Postgres.
  - `HEFESTO_AUTH_PER_MINUTE` makes the auth rate limit configurable. The
    default is unchanged at 10.

### Verification

- **Load smoke, locally.** Release build, Postgres 16, 25 athletes for 30 s:
  17,322 requests (577/s) with 0 errors.
  - p95 was 6–44 ms for reads and single writes.
  - It was 62 ms for completing a session.
  - It was 228 ms for a whole-session sync batch.
- **Go unit tests** cover the load smoke's percentiles and pass/fail rules,
  and the content readiness and release checks. `golangci-lint` is clean.
- **String check**: 168 keys, 0 problems. It caught every string added in this
  phase before its translation existed.
- **Swift** — on the self-hosted Mac runner: VERIFICATION_PENDING

### Deliberately not in this phase

- **The research itself.** It is not in the repository, and nothing may be
  invented (CLAUDE.md). The import path and the release gate are ready for it.
- **A TestFlight upload.** The workflow needs the App Store Connect app record
  and four repository secrets, which only the owner can create. Until then it
  stops at its first step and names what is missing.
- **Running the accessibility audit in CI by default.** It boots a simulator,
  which writes outside the self-hosted runner's workspace, so it runs only
  when the ios workflow is dispatched with `ui_tests`.
- **A designed app icon.** The generated one is a placeholder.

### Open questions for review

1. **The research.** Where is it, and in what form? The import expects YAML per
   `CONTENT_AUTHORING.md`, keeping today's skill, level and exercise slugs.
2. **The accessibility audit on the Mac runner.** May the ios workflow boot a
   simulator on every run? That writes simulator state under
   `~/Library/Developer/CoreSimulator`. If yes, the audit becomes a regular
   check instead of an opt-in one.
3. **TestFlight on a hosted runner.** GitHub-hosted macOS minutes count ten
   times against the plan's allowance, about 15 minutes per build. Is that
   acceptable, or should releases use the self-hosted Mac despite the signing
   state it would leave there?
4. **Load budgets.** Are 300 ms, 400 ms and 1.5 s the right p95 budgets for
   reads, writes and completion, and should the smoke test also run against
   production-sized data?

## Next

Phase 7 is the last phase in the brief. After review: the open questions
above and from earlier phases, the research import, and the first TestFlight
build.
