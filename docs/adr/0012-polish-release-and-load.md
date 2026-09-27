# ADR 0012 — Polish, the TestFlight build and the load smoke test

- Status: accepted
- Date: 2026-09-27
- Deciders: Jaelricco

## Context

Phase 7 is polish (brief §12):
- localization;
- an accessibility pass;
- a TestFlight build;
- importing the researched seed content;
- a load smoke test.

Four constraints shape it:
- **The self-hosted Mac runner is the owner's machine.** It must not change
  anything outside its workspace.
- **Content must not be invented** (CLAUDE.md), and the research has not
  landed in `content/` yet.
- **Nothing touches production** from CI except the deploy workflow.
- **No secret goes into the repository.**

## Decisions

### Localization is checked, not trusted

`ios/scripts/check-strings.py` runs on every push, in `ci.yml`. It checks four
things:
- every user-facing literal in the app is a key in `Localizable.xcstrings`;
- every key has a German translation, marked translated, with the same
  placeholders;
- no key is left unused;
- a count in front of a noun has plural forms in both languages.

The script compares strings by shape, with each interpolation reduced to one
token, so it needs no Xcode and runs on Linux. German follows Swiss usage (no
ß). Server problem titles stay English: they are for developers, and the app
shows its own words for every error it handles.

### Accessibility: fix what the code shows, and audit the rest

The code-visible problems are fixed:
- **Buttons merged into combined rows.** VoiceOver could not reach "I can
  already do this" or the rest timer's Skip as buttons.
- **Fixed-size text.** The rest clock now scales with the text size.
- **Screens that clip at accessibility sizes.** The effort grid drops to three
  columns and scrolls.
- **Motion that ignored Reduce Motion.** The map's unlock animation now does.
- **Low-contrast labels.** Locked labels on the map are now above 4.5:1
  contrast.
- **Silent changes.** Logging a set, a finished rest and sign-in errors are
  now announced.

The skill map also gets a **list view**, grouped by standing. A canvas of
positioned nodes is hard to read with VoiceOver or at large text sizes, and
the list carries the same information.

A UI test target runs Apple's `performAccessibilityAudit` on every main screen
three times: at the default size, at the largest accessibility size, and in
German. It runs on a **Debug-only offline fixture** (`-uiTestFixture`: signed
in, in-memory data, no server).

CI **builds** the UI tests on every run. It **runs** them only when the ios
workflow is dispatched with `ui_tests`, because booting a simulator writes
simulator state outside the runner's workspace.

### TestFlight builds run on a GitHub-hosted Mac

`testflight.yml` archives the app and uploads it. It runs by hand, or on an
`ios-v*` tag. It runs on `macos-15`, not the self-hosted runner, because
signing creates certificates and downloads provisioning profiles. That is
exactly the kind of change the owner's machine must not see. A hosted machine
is discarded after the run.

- **Signing is automatic**, with an App Store Connect API key
  (`-allowProvisioningUpdates`). With the Admin role, Xcode uses
  cloud-managed distribution certificates, so no private key or `.p12` is
  stored anywhere.
- **The secrets live in GitHub**: the team id, the key id, the issuer id and
  the `.p8` contents. The key file is written to the runner's temporary
  directory and removed at the end.
- **The build number is the workflow's run number**, so it always increases.
  The marketing version comes from `project.yml` (0.7.0 for this phase).
- **Export compliance** is declared in `Info.plist`
  (`ITSAppUsesNonExemptEncryption: false`). The app uses only HTTPS.
- **The privacy manifest** declares no tracking and what the app collects:
  email, name, user id and the training log. Everything collected is linked to
  the account and used only for the app's own function. It also declares one
  required-reason API, UserDefaults, for the install's device id.
- **The icon is a placeholder**, drawn by `ios/scripts/make-icon.py`: a
  constellation with one star lit. App Store Connect refuses a build without
  an icon. A designed icon replaces the PNG and nothing else.

### Content: gate the release, not the pipeline

The research is not in the repository, so this phase does not import it. It
makes the import safe instead:
- **contentlint reports how much content is still placeholder.**
- **It warns when a researched skill uses a placeholder exercise.** CI fails
  on warnings.
- **With `-release` (`make content-release-check`), any placeholder is an
  error.** The TestFlight workflow runs this check when dispatched for
  external testers.

`CONTENT_AUTHORING.md` describes the import, including the slugs production
may already hold, which the research must keep.

Placeholders are allowed in internal builds. Nothing may be invented to
replace them.

### The load smoke test is a small Go program, run in CI

`cmd/loadsmoke` registers virtual athletes. Then they train at the same time:
- **Odd rounds:** a whole session as one sync batch, as the app's outbox sends
  it.
- **Even rounds:** one REST write at a time.
- **After each round:** a full sync pull, the skill map, progress, the session
  list, and a conditional catalogue read.

It reports latency per route. It fails on any error, or on a p95 over its
class's budget:

| Class | p95 budget |
|---|---|
| Reads | 300 ms |
| Single writes | 400 ms |
| Completion and sync batches | 1.5 s |

- **It is a Go program, not k6 or Locust.** That adds no dependency or
  language, reuses `google/uuid`, and CI already has Go.
- **It is a smoke test, not a benchmark.** It asks whether a realistic handful
  of athletes gets correct, fast answers from the release build. CI runs 25
  athletes for 60 seconds on a fresh, seeded Postgres.
- **Registration is not budgeted.** It hashes with argon2 on purpose.
- **`HEFESTO_AUTH_PER_MINUTE`** makes the per-address auth limit
  configurable (default still 10), so the athletes can register from one host.
  It stays at 10 in production.
- **Never point it at production.** It creates accounts.

## Consequences

- **The accessibility audit can drift unnoticed** between dispatched runs.
  Running it before each TestFlight upload is the habit to keep.
- **The first TestFlight run needs one-time setup** before it can pass: the app
  record in App Store Connect, the Sign in with Apple capability on
  `fit.hefesto.ios`, and the four secrets. Until then the workflow fails at its
  first step and names what is missing.
- **External testing waits for the research.** `make content-release-check`
  fails while any skill or exercise is a placeholder.
- **The load budgets are guesses** from a first run: p95 was under 60 ms for
  everything except sync batches, which reached about 230 ms. Tighten them
  once CI has a few weeks of numbers.
