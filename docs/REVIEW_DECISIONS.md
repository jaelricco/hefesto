# Open review questions — recommendations

Phases 1 to 7 each ended with questions for review. This page collects the
ones still open, with a recommendation for each. Approve or change each one:
the approved ones are implemented in Phase 8 (or the phase named), and the
outcome moves into the relevant ADR. Each item marked **Keep** confirms
today's behaviour and needs no code.

The planner track's own review items (spec §15) are not repeated here.

## Summary

| ID | Question | Recommendation | Work if approved |
|---|---|---|---|
| D-1 | Implicit level ladder | Keep | none |
| D-2 | Levels cannot be removed or renamed | Keep | none |
| D-3 | Criteria `assistance` is `none \| any` | Keep | none |
| D-4 | Bands are not syncable | Defer to the bands screen | none now |
| D-5 | Registration reveals a taken email | Keep for v1 | none |
| D-6 | Rate limits are in memory | Keep | none |
| D-7 | Evidence for an advanced level does not unlock earlier levels | Keep | none |
| D-8 | Self-attested unlocks earn no XP | Keep | none |
| D-9 | The streak is daily | **Change** to a weekly streak | engine, API, app |
| D-10 | `occurrences` counts sets | **Change** to distinct sessions | engine, golden tests |
| D-11 | The athlete's clock decides between two copies of a set | Keep | none |
| D-12 | Sync applies ops one by one | Keep | none |
| D-13 | A photo stays when its set element is deleted | **Change**: delete it after 7 days | reaper |
| D-14 | "Log a rest day" | Keep | none |
| D-15 | Bodyweight entries have no time zone in the feed | **Change**: add `timezone` | additive API field |
| D-16 | Signing out keeps the local data | **Change**: wipe on sign-out | done in Phase 8, as a bug fix |
| D-17 | `stale_since` is not shown | Keep hidden | none |
| D-18 | Any open level can be self-attested | Keep | none |
| D-19 | Stats are computed on the device | Keep for v1 | none |
| D-20 | The accessibility audit runs only on request | **Change**: also on every push to `main` | workflow |
| D-21 | TestFlight builds use GitHub-hosted Mac minutes | Keep | none |
| D-22 | Load test p95 budgets | Keep; review after four weeks of CI numbers | none |

D-9 and D-10 change how unlocks and streaks are earned. Everything else is
small.

One item is not a question but a bug, and Phase 8 fixes it either way: the
server binds a device id to one account (Phase 2), but the app keeps one
device id per install. A second account signing in on the same phone is
refused with "Please check your details". The app now starts a new device id
on sign-out.

Fixing that turned D-16 from a question into a bug too. Once a second account
could sign in, it would have seen the first account's log, and the first
account's unsynced changes would have been uploaded as the second account's
own. Phase 8 therefore wipes the local database on sign-out, as recommended
below.

## Content (Phase 1)

**D-1 — Implicit level ladder.** Level *n* of a skill requires level *n − 1*
without being written (ADR 0006).
*Recommendation: keep.* It matches how every progression is coached, and
authors cannot forget a step. A level that genuinely skips one can get an
explicit opt-out field when it exists.

**D-2 — Levels are permanent.** The seed refuses to remove or rename a level,
because progress points at it.
*Recommendation: keep.* `CONTENT_AUTHORING.md` already tells the research
import to keep today's level slugs. A rename becomes a deliberate migration
when one is needed.

**D-3 — Criteria `assistance` is `none | any`.**
*Recommendation: keep.* An unlock records unassisted achievement. Assisted
work is a level of its own, with its own exercise (`band-assisted-tuck-fl`),
not a criterion on another.

**D-4 — Bands are not syncable** (also Phase 4, offline band creation).
*Recommendation: defer* until the bands screen ships. Adding sync columns is
an expand-only migration that can land with the screen that needs it.

## Accounts (Phase 2)

**D-5 — Registration reveals a taken email** (`409 email-taken`).
*Recommendation: keep for v1.* Sign in with Apple is the main path, and
credential endpoints are rate limited. Email verification, which would close
the probe, is worth it once email sign-up matters.

**D-6 — Rate limits are in memory, per process.**
*Recommendation: keep.* There is one API process. Staging is a separate
process with its own limits.

## Unlocks, XP and streaks (Phase 3)

**D-7 — Prerequisite gating.** A level whose criteria are met stays locked
until its prerequisites unlock. One completion can unlock a whole chain.
*Recommendation: keep.* Prerequisites are the safety ladder. An athlete who
arrives strong self-attests the earlier levels.

**D-8 — Self-attested unlocks earn no XP.**
*Recommendation: keep.* XP rewards logged work. Self-attestation stays
available without becoming a shortcut.

**D-9 — The streak is daily.** A three-times-a-week athlete must log rest days
or spend freezes to keep it.
*Recommendation: change to a weekly streak*: a week counts when the athlete
trained on as many days as they planned (from the planner when there is a
plan, two days otherwise). Planned rest and deload weeks count as kept
(ADR 0003). This removes the daily pressure the brief warns against.
*Work:* the streak calculation in `internal/domain/progress`, its golden
tests, the meaning of `streak.current_days` (renamed by expand and contract:
add `current_weeks`, keep the old field for one release), and the app's
labels.

**D-10 — `occurrences` counts set entries**, so two qualifying sets in one
session meet `occurrences: 2`.
*Recommendation: change to distinct sessions.* "Twice" in an unlock standard
means repeatable on separate days, not two sets while fresh.
*Work:* the evaluator in `internal/domain/progress` and its golden files.
Unlocks are never revoked, so nothing already unlocked changes.

## Sync and media (Phase 4)

**D-11 — The set conflict rule.** The athlete's clock decides between two
copies of a set; an older copy is `superseded`.
*Recommendation: keep.* Plain last-received-wins would let a phone that was
offline overwrite a later correction.

**D-12 — Per-op results.** One bad op is rejected without holding back the
queue.
*Recommendation: keep.* A rejected op can only come from a client bug, and
holding back every later write for it would be worse.

**D-13 — Media retention.** A photo stays when its set element is deleted,
until the athlete deletes it or the account goes.
*Recommendation: change*: the reaper deletes a photo seven days after its
element is deleted (a window for undo). Photos are personal data, and an
orphan has no way back into the app.

## The logger (Phase 5)

**D-14 — "Log a rest day"** creates and completes a rest-day session at once.
*Recommendation: keep*, next to the rest days a plan schedules. Athletes
without a plan still need a way to say "this was rest".

**D-15 — Bodyweight time zone.** The sync feed does not carry an entry's time
zone, so an edit on another device takes that device's zone.
*Recommendation: change*: add `timezone` to `SyncBodyweight`. It is an
additive API field.

**D-16 — Signing out keeps the local database**, so the next account on the
device would see it.
*Recommendation: change*: sign-out wipes the local database. When changes are
still unsynced, it first says how many and offers to sync. This goes with the
device-id fix above.
*Done in Phase 8* (ADR 0026):
- **Signing out** erases the database. With unsynced changes, Today first
  offers "Sync and sign out", which signs out only if everything was sent.
- **When the server ends a session**, the data stays, so the same athlete
  loses nothing on signing back in. A different account signing in erases it
  first.

## The map and stats (Phase 6)

**D-17 — `stale_since` is not shown.**
*Recommendation: keep it hidden.* Any "you have not practised this lately"
reads as falling behind (ADR 0003), and unlocks are never revoked. The
planner can use it to suggest maintenance work without a label on the map.

**D-18 — Self-attestation scope.** Any open level can be self-attested, not
only levels the log cannot show.
*Recommendation: keep.* It is the path for athletes who arrive strong, and it
earns no XP (D-8).

**D-19 — Stats on the device.**
*Recommendation: keep for v1.* The endpoint waits for a client without the
full log, such as a web view or a coach.

## Release (Phase 7)

**D-20 — The accessibility audit runs only on request.**
*Recommendation: change*: also run it on every push to `main`. That catches
regressions after merge without slowing every branch push.

**D-21 — TestFlight builds on GitHub-hosted Macs.**
*Recommendation: keep.* Builds are rare, and signing never touches the
owner's machine.

**D-22 — Load test budgets** (p95: reads 300 ms, writes 400 ms, completion and
sync batches 1.5 s).
*Recommendation: keep*, and review them after four weeks of CI numbers. A run
against production-sized data belongs before the App Store release.

The skill research itself (Phase 7, question 1) is not a decision: it waits
on the research.
