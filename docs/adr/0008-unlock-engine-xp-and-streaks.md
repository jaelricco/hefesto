# ADR 0008 — Unlock engine, XP and streaks

- Status: accepted
- Date: 2026-09-24
- Deciders: Jaelricco

## Context

Phase 3 connects the two halves of the product: the logger's sets become
evidence, and the map's levels unlock from it. The brief (§6, §8) fixes the
criteria format and the guardrails of ADR 0003. What it leaves open are the
rules of evaluation, how states move, and how XP and streaks are counted.
These are the rules an athlete sees, so they are recorded here.

## Decisions

### Evaluation is a pure function over completed sessions

`progress.Evaluate(criteria, history, now)` lives in `internal/domain/progress`
and does no I/O. The store loads the history and passes it in. The golden
files in `testdata/criteria/` pin its behaviour, one case per rule.

- **Only completed sessions are evidence.** Drafts, abandoned sessions,
  planned sets and deleted rows are never read. A session becomes evidence
  when it is completed, and that is when evaluation runs.
- **An occurrence is a set entry.** A condition with `occurrences: 2` needs
  two distinct sets. They may be in one session, and a combo counts once
  however many of its elements match.
- **What never counts:** failed elements, partial range of motion, and
  eccentric-only reps.
- **`assistance: none`** excludes only the assisted class. Weighted sets
  still count, unless the condition caps the load with `max_load_kg`.
- **Time.** Observations after `now` are ignored as clock errors.
  `within_days` counts back from `now`.
- **Evidence** is the most recent qualifying set entry. It is stored on the
  unlock so the celebration can point at it.
- **Progress without an unlock** is shown as the best strict value (right
  exercise, measure, assistance and load; full range; not failed). It
  ignores the threshold, form and window.

### States are derived in graph order and gated on prerequisites

The states are `locked → available → in_progress → unlocked`.
`progress.Advance` walks the levels in topological order:

- A level stays **locked** until all of its prerequisites are unlocked.
  This holds even if the log already meets its criteria.
- Once its prerequisites are unlocked, a level with logged attempts is
  **in progress**, and one without is **available**.
- A level whose criteria are met becomes **unlocked**. This applies in the
  same pass, so one session can unlock a chain. For example, after
  `pull-up/strict-5` unlocks, the front-lever tuck holds logged earlier
  unlock the tuck in the same completion.

Availability is recomputed from the stored rows whenever the map is read. A
level added to the content becomes available without waiting for a session.

**Unlocks are never revoked.** The following are all append-only or frozen
once an unlock is recorded:

- `skill_unlock_events` (a trigger enforces this);
- `first_achieved_at`, `verification` and the evidence on `user_skill_states`.

Deleting the evidence session changes nothing. `best_value` is current form
and keeps moving. `stale_since` is a hint after 90 days without an attempt on
an unlocked level, and never changes the state.

### Self-attest is the fallback, and earns no XP

`POST /v1/me/skills/{levelId}/attest` records `verification: self_attested`.
It serves two cases: levels with no criteria yet, and levels achieved before
the athlete used Hefesto. Prerequisites must already be unlocked; if not, the
endpoint answers 409 `prerequisites-unmet`. Attesting again returns the
existing unlock.

Attested unlocks earn no XP. XP rewards what the log shows, and attesting
for XP would be a way to cheat.

### XP rewards quality and progression, never volume

| Source | Amount | When |
|---|---|---|
| `session_completed` | 10, +5 if at least 3 sets are rated and at least 80 % of them are rated 4 or more | Only the day's first session. Only if it has at least one performed element. Never on a rest day. |
| `skill_unlocked` | 20 + 5 × difficulty tier | Auto unlocks only |
| `streak_milestone` | 20 / 50 / 100 / 250 | At 7 / 30 / 100 / 365 days, once per run |

There is no XP per set, rep or minute. XP events are unique on
`(user, source, ref_type, ref_id)`, so completing a session twice cannot
award twice. A streak milestone's `ref_id` is a UUIDv5 of
(user, run start, days): a new run can earn it again, and the same run
cannot.

### Streaks count rest, and freezes are free

Streaks are recomputed from `user_training_days` by the pure function
`progress.ComputeStreak`, never incremented.

- **Counted days:** a day counts if the athlete completed a session, logged
  a planned rest day, or was in a deload (ADR 0003 §1).
- **Freezes** bridge a gap only when there are enough credits to cover all of
  it. A bridged day keeps the run alive but does not lengthen it.
- **Credits:** an athlete starts with 2. They earn one more per 7 counted
  days in a run, and bank at most 3. Credits are never sold.
- **Today** never breaks a streak, because the day is not over. `/v1/me/progress`
  recomputes on every read, so the streak is right on a day the app was not
  opened.

### Completion is one transaction, serialised per user

`POST /v1/sessions/{id}/complete` does all of the following in one
transaction:

- takes a per-user advisory lock;
- marks the session completed;
- records the training day;
- evaluates every level;
- writes the states and unlock events;
- awards XP;
- recomputes the streak.

Completing an already completed session changes nothing. It returns the
unlocks that session caused, with `already_completed: true`, so a client
retrying after a lost response still shows the celebration. An abandoned
session must go back to draft before it can be completed.

## Consequences

- The engine is testable without a database, and a new criteria rule starts
  as a golden file.
- Evaluation reads the athlete's history for the exercises named in any
  criteria. That is fine for years of logs. If it stops being fine, the
  planned `user_exercise_bests` cache is the answer, and it is not needed
  yet.
- Gating on prerequisites means an athlete who can already do an advanced
  level has to unlock or attest the levels before it. The map stays a
  coherent path, and attesting is quick.
- Streaks are daily. Rest days and freezes make them forgiving, but an
  athlete who trains three times a week and never logs a rest day will not
  build one. This is an open question for review.
