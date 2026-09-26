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

## Current phase: 6 — skill map, skill detail, celebration, history and stats (complete, awaiting review)

The decisions are in ADR 0011. Everything below reads the local database, like
the logger, so it works offline.

- **The skill map (Skills tab)**
  - A constellation drawn with SwiftUI `Canvas` at the positions authored in
    content, with pan and zoom.
  - Locked skills are dim, skills open to work on are outlined, and unlocked
    ones glow. A skill in progress shows an arc for how many of its levels are
    unlocked.
  - Lines join skills that build on one another and light up from an unlocked
    level. The first time the map shows a new unlock, its line draws itself;
    then it counts as seen (a local `seenAt`). A new device does not replay old
    unlocks.
  - Every node is a real button, 56 pt, with a VoiceOver label (the skill),
    value (its standing) and hint (the next level). Motion respects Reduce
    Motion in the celebration.
  - XP and the streak sit in the toolbar and show only what was kept.
- **Skill detail**
  - Each level with its state, what unlocks it (the criteria DSL in words),
    the athlete's best, and when it was unlocked (self-attested levels say so).
  - "I can already do this" self-attests an open level, after a confirmation
    that explains it earns no XP. Missing prerequisites get a clear message.
  - Injury notes are fetched and kept for offline reading, and always shown
    with the disclaimer the API sends. Nothing is presented as medical advice.
- **The celebration**
  - An animated star and burst for an unlock, the levels unlocked, XP, newly
    open levels, and "See it on the map", which switches to the map where the
    line lights up.
- **History**
  - Sessions grouped by ISO week, Monday first. A session opens read-only, and
    a draft can be continued in the logger.
  - Each logged exercise shows its bests (most reps, longest hold, longest
    distance, most added load, with dates) and a Swift Charts bar chart of
    each day's work. Bests count only full, unassisted repetitions.
- **Store and sync**
  - A new local migration, `v2-skill-map`: skills, levels, edges, level states,
    progress, injury notes.
  - The app refreshes the map after every sync.
  - A refresh never moves an unlocked level back.

### Verification

On the self-hosted Mac runner (run 13):
- `swift test` passes 47 tests in 13 suites. That is 11 new, covering:
  - the skill map store: the first refresh counts unlocks as seen, a new
    unlock stays unseen until shown, unlocks are never revoked, content is
    replaced only on a new version, and a skill's standing;
  - ISO weeks across a year boundary;
  - bests that ignore assisted and partial reps;
  - stats from logged sets;
  - map, progress and injury-note sync, and a refused attestation.
- The app builds for the iOS Simulator (`** BUILD SUCCEEDED **`) under Swift 6
  strict concurrency.

No server code or API spec changed in this phase.

### Deliberately not in this phase

- **`stale_since`** is stored but not shown. See the open questions.
- **`GET /v1/me/stats/exercises/{id}`**, from the brief's API list, is not
  built. Stats are computed on the device (ADR 0011).
- **Profile and settings screens.** The brief places them in no phase. Sign out
  stays in Today's menu.
- **The Live Activity**, band assistance and media in the logger are still
  open from Phase 5.

### Open questions for review

1. **`stale_since`.** Should the app show that an unlocked skill has not been
   practised lately, and if so, how, without implying the athlete is falling
   behind (ADR 0003)? Today it is not shown.
2. **Self-attestation.** Any open level can be self-attested, not only levels
   whose criteria the log cannot show. The server allows both. Should the app
   offer it only for criteria-less levels?
3. **Stats on the device.** Is computing stats locally acceptable for v1, with
   the endpoint left for a future web or coach client?

## Next: Phase 7

Polish: localization, an accessibility pass, a TestFlight build, importing
the researched seed content, and a load smoke test.
