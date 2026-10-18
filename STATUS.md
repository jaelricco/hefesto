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
- **The plan's screens in the iOS app (this checkpoint, ADR 0022):**
  - A new module, `HefestoPlan`, maps the kept plan to the screens' own
    types in one place; views never see the API's types (ADR 0020). The
    plan is fetched again after every sync.
  - Today leads with the plan: starting today's open session is the main
    action, a free session the second. Without a plan, on a rest day of the
    plan or once the session is done, a free session leads again. The week
    follows, day by day, rest days included. A stop shows the plan's
    reasons verbatim and adds no advice.
  - A planned session shows every block and item with sets, target,
    reserve, rest and the knowledge base's reasons.
  - Starting asks the two optional check-in questions and says what they
    change and that they are not stored. The outbox goes out first, so the
    server plans with the latest log.
  - The logger shows open planned sets as targets. "Next set" opens the
    next one with its targets filled in. The first working set of a
    calibration item says why its reserve matters; the reserve stays
    optional. History shows only what was done.
  - A planned band set cannot be performed until the app can name a band.
    Performed without one, it would read as unassisted and count for the
    map.
  - The demo holds a plan of the week; four new screens (`plansession`,
    `checkin`, `planned`, `target`) join the screenshots.
  - Not built: offers on active choice, the pain question on monitored
    items, the onboarding in the app, choosing a band, the offline start.
- **Choosing a band in the iOS app (this checkpoint, ADR 0024):**
  - The app keeps the server's bands, the athlete's own and the
    catalogue's, and fetches them after every sync; bands are not in the
    sync feed (ADR 0009). A band is added and removed online.
  - The set composer offers band assistance again: the band and how many at
    once. A band set cannot be saved without its band.
  - Planned band sets open with the band assistance and can be performed;
    ADR 0022's lock is lifted (U-75 → U-80).
  - Logger and history name a set's band. The demo holds two bands of its
    own; a new screen (`band`) joins the screenshots.
  - Not built: bands offline, editing a band, the anchor.
- **The body in the iOS app (this checkpoint, ADR 0028):**
  - Symptoms during training, pain reports, a region's warning signs and a
    professional's clearance go to the server at once, each with its own id;
    a retry after a lost answer keeps it. The plan and the regions are
    fetched again after each.
  - A new screen, Body, shows every region with a complaint or a
    constraint: its state as what the plan does, a rest until a date, a
    recommended assessment, a professional's restrictions and the
    disclaimer, verbatim. Today leads there; the logger reports pain or
    symptoms from the session.
  - A pain report asks what the planner reads at its moment (spec §8.6).
    When the planner asks for warning signs, they open next.
  - A stopped plan offers the clearance of training as a whole, except
    for a minor (SAFE-07). Until now no stop could be lifted in the app.
  - The demo holds two regions; three new screens (`body`, `pain`,
    `symptoms`) join the screenshots.
  - Not built: changing the consent, the questions before and after a
    session on monitored regions, the list of one's pain reports, the sync
    of pain reports.
- **Starting a planned session offline (this checkpoint, ADR 0027):**
  - Sync now carries `planned_session_id` on a session and
    `planned_item_id` on a set. The server takes them when it creates the
    row and keeps them after; an unknown planned session is dropped.
  - The app writes the draft as the server would (Materialize, ported) and
    queues it parents first. There is no check-in offline: lightening a
    session needs the server. A test pins the schema on both sides.
  - The planner links an offline start after the commit: after every push,
    before a plan is read, before a start and before a completion. The day's
    session in the current plan points at it, started or completed. A draft
    from a replaced plan follows the current one; without a session on the
    day its open sets go. A day started first elsewhere keeps that session.
  - Today shows a start or completion kept on the device before the plan
    knows it. Without a connection the check-in sheet offers to start as
    planned; a new demo screen (`offlinestart`) shows it.
  - Findings: an offline start split over two pushes after the plan changed
    can leave late sets on their old item; a second device's offline start
    of the same session stays unlinked; there is no check-in offline.
- **The onboarding in the iOS app (this checkpoint, ADR 0023):**
  - The knowledge base now words the six screening questions (`SCR-01` to
    `SCR-06`), checked by KB-10 and KB-12. The planner catalogue serves them
    with the exertion question (RF-10) and the minor age, so the app asks
    exactly what the server validates.
  - Without health-data consent the screening may be empty: the design
    does not ask it then, and the server discards it anyway.
  - `HefestoPlan` holds the answers in a form that decides which questions
    apply (onboarding.md §3.6), when a page is complete and what is sent:
    only answers that still apply, RF-10 once for every region. A retry of
    the same answers keeps its idempotency key.
  - The app asks page by page: frame, goals, time, equipment, background,
    where the athlete stands, health, one page per current complaint with
    its warning signs, then the result. Knowledge-base texts are shown
    verbatim; a yes to a red flag shows its advice at once. Clarifications
    come back on the last page.
  - Without a plan, Today leads with the onboarding. The demo holds the
    catalogue; three new screens (`onboarding`, `standing`, `complaint`)
    join the screenshots.
  - Not built: the test day, «refine profile», preferred days, free text
    for «other», confirming a clearance in the app.
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
  plan's foundation in the iOS app in ADR 0021 (U-69 to U-71), the plan's
  screens in ADR 0022 (U-72 to U-75), the onboarding in the app in
  ADR 0023 (U-76 to U-79), the band choice in ADR 0024 (U-80), the
  offline start in ADR 0027 (U-81 to U-85), the body in the app in
  ADR 0028 (U-86 to U-89), and these findings in spec §15.4:
  - sets with partner or machine assistance reach neither the capacities
    nor the load history;
  - a set performed offline after the server replaced or removed it is
    refused by sync, because deletions are final (ADR 0009). The app should
    push its outbox before an event; otherwise ADR 0009 needs an exception;
  - WEEK-08 regenerates the whole week, not only from the next session that
    has not started;
  - the reserve stays optional in the app; a set left unrated is a lower
    bound. The app now marks a calibration set and says why its reserve
    matters, but does not insist on it;
  - no exercise carries `restriction_tags`, so a professional's
    restrictions from the onboarding are stored but do not yet exclude
    anything;
  - after a withdrawal the decision log and past plans still name regions
    and states; whether they must be redacted is a legal question (ENT-4);
  - the catalogue does not say which goals bring which onboarding
    questions; the app holds onboarding.md §3.6 with the knowledge base's
    skill slugs (ADR 0023);
  - a minor answers the whole onboarding before learning there is no plan
    (SAFE-07); an early notice waits on OE-1;
  - the planner does not tell bands apart: a light and a strong band count
    as the same band capacity of an exercise (ADR 0024);
  - the offline start's limits: a start split over two pushes, two devices
    starting the same session offline, no check-in offline (ADR 0027);
  - a professional's clearance is the athlete's word, and symptoms reported
    without a connection pause the plan only once they reach the server
    (ADR 0028).
- **Not yet built:**
  - the sync of pain reports;
  - a record of the consent text agreed to;
  - `?explain=trace`.
- **Still blocking production:**
  - The content review of the knowledge base (ENT-10). Until then it stays
    `draft_placeholder`, and production refuses it.
  - The legal questions on minors, health data and screening wording.


## Current phase: 7 — polish (complete, awaiting review)

The decisions are in ADR 0025 (numbered after the planner track's 0022–0024). This phase was built before the planner and the
Glut design pass reached `main`, and merged with both afterwards.

- **Localization**
  - `ios/scripts/check-strings.py` checks:
    - every user-facing literal is in the string catalog;
    - every key has a translated German entry with matching placeholders;
    - no key is unused;
    - counts before nouns have plural forms in both languages.
  - It runs in CI on every push.
  - With the planner and design pass merged it found stale keys and two
    "of N levels" strings that read "1 of 1 levels"; those now use plural
    substitutions. There are 205 keys, all translated. `DemoMode.swift` is
    fixture data and is not checked.
- **Accessibility**
  - The skill map:
    - respects Reduce Motion;
    - reads top to bottom with VoiceOver;
    - has a **list view** grouped by standing, in the Glut theme, remembered
      per device.
  - Level rows and the rest banner no longer swallow their buttons into one
    VoiceOver element.
  - The rest clock scales with Dynamic Type. The effort sheet drops to three
    columns at accessibility sizes and scrolls.
  - Chart bars read as "date, value".
  - Logging a set, a finished rest and sign-in errors are announced.
  - A UI test target runs Apple's accessibility audit on every demo screen
    (`DemoMode.swift`, opened by `-HefestoDemoScreen`), in English, at the
    largest accessibility size, and in German. It runs on the simulator the
    screenshot job already keeps ("Hefesto Screenshots") and creates no other.
  - Contrast and Dynamic Type for the rest of the app came with the Glut
    design pass (ADR 0020).
- **TestFlight**
  - `testflight.yml` runs on a GitHub-hosted Mac: automatic signing with an
    App Store Connect API key, build number = run number, upload with
    `xcodebuild -exportArchive`.
  - The app now has an icon (a generated placeholder), an export-compliance
    declaration, a privacy manifest, and version 0.7.0.
  - Setup is in `docs/DEPLOYMENT.md`.
- **Content import**
  - `contentlint` reports how much content is still placeholder (today all 3
    skills and 56 exercises, including the 49 generated from the planner's
    knowledge base).
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
- **String check**: 205 keys, 0 problems. It caught every string added in this
  phase before its translation existed.
- **Swift** — on the self-hosted Mac runner, ios run 59 on `1b6b432` (this
  phase merged with the planner and the Glut design pass):
  - `swift test` passes for HefestoKit;
  - the app and the accessibility UI test target build for the iOS Simulator
    under Swift 6 strict concurrency (`build-for-testing`);
  - the audit itself runs only when the ios workflow is dispatched with
    `ui_tests`, and has not run yet.
- **The Mac runner** lost contact with GitHub ten minutes into cold Xcode
  builds four times. The long steps now run under `caffeinate -i` at lower
  priority; the first run after that change passed.

### Deliberately not in this phase

- **The skill research itself.** The planner's research (`docs/research`,
  `content/training`) is in, but the skills, levels and unlock criteria the map
  needs are still placeholders, and nothing may be invented (CLAUDE.md). The
  import path and the release gate are ready for them.
- **A TestFlight upload.** The workflow needs the App Store Connect app record
  and four repository secrets, which only the owner can create. Until then it
  stops at its first step and names what is missing.
- **Running the accessibility audit on every push.** It runs when the ios
  workflow is dispatched with `ui_tests`. See the open questions.
- **A designed app icon.** The generated one is a placeholder.

### Open questions for review

1. **The skill research.** When do the skills, levels and unlock criteria
   arrive, and in what form? The import expects YAML per
   `CONTENT_AUTHORING.md`, keeping today's skill, level and exercise slugs.
2. **The accessibility audit on every push.** It reuses the screenshot job's
   simulator, so it adds nothing to the Mac beyond run time (39 launches,
   several minutes). Should it run on every push instead of on request?
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

## Design pass: direction B, "Glut" (done; merged in #11, #12 and #13)

The owner chose direction B of three mockups. The decisions are in ADR 0020.
It was developed on `claude/awesome-wright-gqa9pe`, next to the planner track,
and touched only the app's views, its project file and its strings.

- **One dark, warm theme** for every screen: warm graphite, ember for
  actions, gold only for achievement, Barlow and Barlow Condensed (bundled,
  SIL Open Font License), sizes that follow Dynamic Type. There is no light
  mode; on review the owner chose to stay dark.
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
  Merged in #13.

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
