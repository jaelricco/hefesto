# ADR 0003 — Gamification guardrails

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

The dominant injury mode in calisthenics is not acute trauma. It is chronic
overuse of connective tissue — medial and lateral elbow tendinopathy from lever
and planche work, long head of biceps and anterior shoulder from front lever and
straight-arm pulling, wrist from handstand volume. Tendon adapts on a slower
timescale than muscle, so the athlete's *strength* outruns their *tolerance*, and
the failure shows up weeks after the training that caused it.

A gamified trainer is therefore not a neutral container for this content. The
standard mechanics of the genre — daily streaks, XP per unit of volume,
comparison against peers, notifications that frame absence as loss — all push in
exactly the direction that produces this injury. A streak that breaks on a rest
day is a mechanic that rewards training through a warning sign.

## Decision

Ship the motivating parts and refuse the coercive ones.

**Shipped:**

- Skill map states, with unlock events and a celebration payload returned by
  `POST /v1/sessions/{id}/complete`.
- XP weighted by **quality, not volume**: form quality, hitting planned targets,
  consistency across weeks, and progression events. Not sets, not reps, not
  minutes under tension.
- Badges for milestones and for process (e.g. completing a planned deload).
- Training streaks, under the constraints below.

**Constraints, binding on all future work:**

1. A streak counts a **planned rest day or a deload week as maintained**, not as
   a gap. Rest is training. `user_training_days` records a day's intent, not just
   its activity, which is why the streak calculation reads that table rather than
   counting sessions.
2. **Streak freezes** exist, accrue automatically, and require no purchase and no
   ad.
3. **No notification may imply the user is falling behind, losing something, or
   disappointing anyone.** Notifications announce facts the user asked for
   (a rest timer finishing, a planned session) or celebrate something achieved.
4. **No leaderboard in v1**, and none afterwards without its own ADR. Comparison
   against strangers is the single mechanic most likely to drive maximal attempts
   on an under-prepared joint.
5. **XP is never awarded for attempting a maximal hold**, and the criteria
   evaluator requires `occurrences >= 2` for most unlocks precisely so that a
   single grinding rep does not unlock anything.
6. **Unlocks are never revoked.** The map records what the athlete achieved.
   Current form is tracked separately (`best_value`, `stale_since`) and surfaced
   as information, never as a penalty.

## Consequences

- Streak logic is more complex than "did they train today", because it must model
  planned rest. The `user_training_days` rollup exists for this.
- XP cannot be computed from raw set volume, so it needs its own weighting
  function in `internal/domain/progress` with its own tests.
- Some engagement is left on the table. That is the point.
- Injury content in the app is educational and carries
  `disclaimer: educational_only` in the API payload itself, not merely in a
  footer, so that no client can render it without the disclaimer available.
  Hefesto makes no medical claims and is not a substitute for a clinician.
