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

## Parallel track: training-plan algorithm (stage 5 done; persistence, API and the log link, awaiting review)

A separate track, with its own five-stage plan, researches, specifies and
implements a planner that turns an onboarding and the logs into individual
training plans. It is developed on `claude/busy-babbage-fqio1j` (PR #10). Its
documents are in German.

- **Stages 1–4:** research (572 sources in `docs/research/00_sources.md`),
  synthesis, onboarding design, `docs/algorithm/spec.md` and ADR 0012. All
  checkpoint proposals were accepted.
- **Stage 5 (this checkpoint):**
  - `internal/domain/planning`: the pure, deterministic core (`Build`,
    `Start`, `Generate`, `Adapt`). Every decision carries a reason with rule,
    parameter and source IDs.
  - `content/training/`: the knowledge base as YAML with JSON schemas. It is
    validated by `contentlint` in CI and at start-up. A test checks every
    research parameter against its row in `docs/research/`.
  - `internal/planning`: the application service with ports and in-memory
    adapters. With an invalid knowledge base only planning answers
    "unavailable".
  - Tests: rule tables, the knowledge-base checks KB-01 to KB-12, six personas
    as golden files, twelve simulated weeks per persona, the multi-week
    scenarios from spec §12.5, and plan invariants I-1 to I-11 on every
    generated plan.
  - `docs/algorithm/personas.md`: the persona plans and their plausibility.
    Spec §15 records the implementation decisions (U-1 to U-35) and findings.
- **Independent review (stage 5 closing):** three reviewers without prior
  knowledge checked safety, numbers and the persona plans against the
  research (`docs/algorithm/review.md`). They found 6 critical, 18 major
  and 21 minor issues. All critical ones are fixed. U-13 is withdrawn.
- **Review decisions (ENT-R-1 to ENT-R-5, all proposals accepted):**
  - ENT-R-1: a pause from the onboarding ramps on the week-1 target frozen
    at the onboarding, with f(a) on the steps.
  - ENT-R-2: a small straight-arm or wrist account gets one set more every
    three weeks (LOAD-12). A planned deload week counts as held.
  - ENT-R-3: the entry and break ramps use the normal session cap.
  - ENT-R-4: the heuristics `PAR-S-47`, `PAR-S-48` and `PAR-S-38` are
    confirmed.
  - ENT-R-5: a session with fewer than two working sets becomes a rest day,
    and its sets move to other sessions (WEEK-09). Sessions a ramp counts
    stay.
- **Open for review:** the implementation of the decisions and the findings
  in spec §15.4. The session cap (`PAR-S-48` with the smallest set) now
  limits straight-arm volume most; some personas train on fewer days than
  they chose.
- **Postgres adapter and migration (ADR 0013):**
  - Migration `00009_planning.sql` adds the planner's tables: profile,
    goals, health data, constraints, capacities, ladders, phase, pause,
    history, plans and the decision log.
  - `internal/store/planning.go` stores and reads the snapshot, the plans
    and the decisions.
  - Every call of the planning service runs in one transaction with a lock
    per user, so events of one user apply one at a time.
  - Integration tests against real Postgres: a full snapshot reads back
    byte-identical, and the service produces the same snapshots, plans and
    changes on Postgres as in memory.
- **HTTP endpoints (this checkpoint, ADR 0014):**
  - `api/openapi.yaml` gains the `planning` tag: onboarding, training
    profile and goals, the week plan with its sessions and decision log,
    pain reports, exertion symptoms, regions with red flags and clearances,
    capacities, and a public catalogue (rules, sources, parameters, and the
    planner's skills, exercises, regions and answer classes). The change is
    additive.
  - The onboarding runs once; later changes go through the profile and the
    goals. Events are idempotent by a client ID, and a repeat answers with
    the changes recorded the first time.
  - Answers are checked per field against the knowledge base, with JSON
    pointers. A current complaint needs an answer to every red-flag question
    of its region.
  - Reasons tied to a region carry no sources (EXPL-07). Pain reports need
    the health-data consent.
  - `cmd/api` wires the planner on Postgres. With an invalid knowledge base,
    or with draft content in production (ENT-10), only the planning
    endpoints answer 503.
  - Every endpoint has an integration test against real Postgres that checks
    each response against its schema.
- **Health-data consent (this checkpoint, ADR 0015):**
  - `POST /v1/me/health-consent` grants or withdraws the consent.
  - A withdrawal deletes region states, screening and pain reports. Locks,
    stops and exclusions of regions with a complaint stay as constraints.
  - A grant asks the screening and past injuries again and tracks every
    excluded region in stage 0 of the return ramp.
  - Without consent a cleared lock and a stage-0 red flag now leave the
    region excluded; before, the region was left unprotected.
- **Starting a planned session (this checkpoint, ADR 0016):**
  - The 49 planner exercises missing from the catalogue are added to
    `content/exercises/` as `draft_placeholder`. Name, measure and equipment
    come from the knowledge base. Validation fails when a planner exercise is
    missing from the catalogue.
  - `POST /v1/me/plan/sessions/{id}/start` writes the planned session into
    the log as a draft: one planned set entry per planned set. Each set names
    its plan item (`planned_item_id`); the session names the planned session
    (`planned_session_id`). Sets are then performed through the log API or
    sync as before.
  - A planned session starts once; a second start returns the same session.
    Deleting the draft frees the planned session.
  - A stop answers `409 training-stopped` and names the rule.
  - Plans show each session's status, and their `ETag` follows the starts.
    A new plan of the week keeps a start on the same day.
  - Offers, and blocks without sets, are not written. A band target stays in
    the plan item, because the log needs the actual band.
- **A hold's reserve in the log (this checkpoint, U-65, U-66):**
  - `set_entries.sir_s` (0–60 s) records the seconds a hold could have gone
    on; `rir` stays repetitions (ENT-S-4). It runs through REST, sync and
    the responses.
  - A planned hold carries its target reserve there, and a completed hold's
    `sir_s` becomes its reserve in the planner's history. Holds count as
    full observations again where the reserve allows.
  - The iOS app asks for it when a set holds exactly one hold (U-67): an
    optional "Reserve" picker, 0–60 s. It is stored locally and synced both
    ways. Repeating the last set does not copy it.
  - It asks for RIR the same way when a set holds exactly one rep element
    (U-68), 0–10. An element taken to failure is not asked about.
- **The plan in the iOS app, the foundation (this checkpoint, ADR 0021):**
  - Screens waited for the design pass (ADR 0020, now merged), so this step
    touches only the `HefestoKit` package.
  - The app keeps each week's plan as the document the server sent
    (`cachedPlan`, with its `ETag`), readable offline through the generated
    `TrainingPlan`. `refreshPlan` asks with `If-None-Match`; an unchanged
    plan costs a 304, and `onboarding-required` forgets every kept plan.
  - `startPlannedSession` starts a planned session with an optional
    check-in and keeps the returned draft. A repeated start leaves a session
    already kept here alone. Starting needs the network.
  - `LoggerModel.perform(plannedSet:)` turns a planned set into the
    performed one in place, with the athlete's values and reserve.
  - Sessions and sets mirror `plannedSessionId` and `plannedItemId`, never
    sent back.
  - Fixed on the way: the generated client percent-encoded `If-None-Match`,
    so no `ETag` ever matched, for the exercise catalogue either. Every
    client of the app now sends the header as HTTP defines it.
  - Not built: the screens, the onboarding in the app, pain reports from
    the app, choosing a band for a planned band set, the offline start.
- **A started draft follows the plan (this checkpoint, ADR 0017):**
  - Every new plan of the week reconciles the drafts of started sessions in
    the same transaction. Their open planned sets follow the day's session
    in the new plan, less what is done.
  - Unchanged items keep their sets and the athlete's own edits.
  - A changed target replaces the open sets, and dropped items lose theirs.
    New items come in a new block.
  - Without a session on the day, as after a stop, every open planned set
    goes. Performed sets never change.
  - The event answers `session_adjusted` with the session (rule ADAPT-19).
- **Completing a session (this checkpoint, ADR 0018):**
  - Every completed session since the onboarding reaches the planner after
    the log's commit, not only planned ones. Rest days are the exception.
  - Performed sets become the planner's history in the order performed:
    reps or seconds, without assistance or with a band, and the RIR of rep
    sets. Other assistance and distances are left out.
  - The planned session becomes `completed`, and a deload session marks its
    day. The session is applied once per session.
  - `POST /v1/sessions/{id}/complete` and sync answer with `plan_changes`.
    If the planner fails, the completion stays valid and the next
    `GET /v1/me/plan` catches it up.
- **Check-in at the start (this checkpoint, ADR 0019):**
  - The start takes an optional check-in: hours slept and fatigue 1–10.
    Six hours or less, or fatigue 8 or more, makes the session lighter
    (ADAPT-17).
  - In the max block, offers go and every hold becomes submaximal technique
    (DOSE-09). Rep work (strength, rep skills, eccentrics) and the other
    blocks stay.
  - The answers are never stored. The planned session keeps only
    `check_in_applied`, so plans, new plans and the reconciliation keep the
    session lighter. Streak, XP and progress are not affected.
- **Open for review:** the API decisions in ADR 0014 and spec §15.2
  (U-39 to U-46), the consent decisions in ADR 0015 (U-47 to U-49), the
  start decisions in ADR 0016 (U-50 to U-54), the reconciliation in
  ADR 0017 (U-55 to U-57), the completion in ADR 0018 (U-58 to U-61), the
  check-in in ADR 0019 (U-62 to U-64), the reserve (U-65 to U-68), the
  plan's foundation in the iOS app in ADR 0021 (U-69 to U-71),
  and these findings in spec §15.4:
  - sets with partner or machine assistance reach neither the capacities
    nor the load history;
  - a set performed offline after the server replaced or removed it is
    refused by sync, because deletions are final (ADR 0009). The app should
    push its outbox before an event; otherwise ADR 0009 needs an exception;
  - WEEK-08 regenerates the whole week, not only from the next session that
    has not started;
  - the reserve stays optional in the app; a set left unrated is a lower
    bound. The app does not read the plan yet, so it cannot insist on a
    calibration set;
  - no exercise carries `restriction_tags`, so a professional's
    restrictions from the onboarding are stored but do not yet exclude
    anything;
  - after a withdrawal the decision log and past plans still name regions
    and states; whether they must be redacted is a legal question (ENT-4).
- **Not yet built:**
  - starting a session offline through sync;
  - the sync of pain reports;
  - a record of the consent text agreed to;
  - `?explain=trace`.
- **Still blocking production:**
  - The content review of the knowledge base (ENT-10). Until then it stays
    `draft_placeholder`, and production refuses it.
  - The legal questions on minors, health data and screening wording.

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

## Design pass: direction B, "Glut" (merged in #11 and #12; grey pickers awaiting review)

The owner chose direction B of three mockups. The decisions are in ADR 0020.
It is developed on `claude/awesome-wright-gqa9pe`, next to the planner track,
and touches only the app's views, its project file and its strings.

- **One dark, warm theme** for every screen: warm graphite, ember for
  actions, gold only for achievement, Barlow and Barlow Condensed (bundled,
  SIL Open Font License), sizes that follow Dynamic Type.
- **Today** shows XP and the streak, "rest days count", and the sessions with
  their dates as blocks.
- **The logger** has the rest timer as a large card with its progress and
  numbers in large type. Combos are joined by a line. "Log a set" and "Add a
  block" sit at the bottom, in reach of the thumb.
- **The map** is in warm colours, and its first tap shows the skill in a card
  below. **The skill page** shows a level ladder and level cards.
- **The celebration, history and sign-in** follow the theme. The celebration
  no longer reads the generated API type.
- **A demo mode** (Debug builds only) and a screenshot workflow on the
  self-hosted Mac capture every screen in German for review.
- **The set composer** is themed in full now that the planner's reserve
  section has landed (#10). Its sections carry the same caps labels as every
  other list in the app. The composer can open on an exercise already chosen,
  and the demo's `reserve` screen uses that to open it on a hold, so the
  screenshots show the reserve question. Merged in #12.
- **Pickers show their choice in grey**, the secondary text colour, like any
  other value. Ember stays for actions (the owner's call on review, ADR 0020).

### Verification

On the self-hosted Mac (Xcode 26.4.1, iOS 26.4 simulator):
- At 7f3cd01, merged in #11: `ios` (run 37) passed 47 tests in 13 suites and
  built the app for the iOS Simulator under Swift 6 strict concurrency;
  `ios-screenshots` (run 3) captured all 12 demo screens in German on an
  iPhone 17 Pro simulator.
- At c720d95, the set composer on top of the planner (#10): `ios` (run 48)
  passes 63 tests in 15 suites and builds the app; `ios-screenshots` (run 7)
  captures all 13 demo screens, the new `reserve` screen among them.
- At 6d0936f, grey pickers: `ios` (run 51) and `ios-screenshots` (run 9)
  pass. On the `reserve` screen, the pickers' values and the added load now
  measure #A89F95, the palette's secondary text; they were ember (#FF7A33)
  and the system's grey.

### Open questions for review

1. **Light mode.** There is none now (ADR 0020). Should a light palette
   follow for training outdoors?
