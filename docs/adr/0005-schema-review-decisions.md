# ADR 0005 — Schema review decisions

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

`docs/DDL_PROPOSAL.md` was the Phase 0 deliverable. It listed eight
deviations from the brief (D-1 … D-8) and seven open questions, and Phase 1 was
blocked on the answers. This ADR records them, plus the corrections found while
turning the proposal into migrations 00001–00008.

## Decisions

### Accepted as proposed

- **D-1 … D-8** are accepted: one shared `families` table; `session_id`
  denormalised onto set entries and elements; `assistance_class` on set
  elements; `is_rest_day` and `user_training_days`; `server_seq` as the sync
  cursor; a `devices` table behind `client_id`; cycle checks in tooling rather
  than a trigger; `user_exercise_bests` as a rebuildable cache.
- **Circuits** log one `set_entry` per round, with `round_index` saying which.
- **Rest inside a combo** does not exist; rest lives on the `set_entry`.
- **Edges run between levels**, not skills; the map aggregates them.
- **`est_weeks_from_prev` is a display hint** shared by every user, never
  authoritative in the client.

### `load_kg` is added load only, never negative

The brief let a negative `load_kg` mean counterweight. That gave assistance two
representations — a negative load, or a positive `estimated_assist_kg` on the
assistance row — and every reader would have had to reconcile them.
`set_elements.load_kg` and `template_set_elements.target_load_kg` now carry
`CHECK (… >= 0)`. A counterweight is `set_element_assistance.type =
'counterweight'` with `estimated_assist_kg`. One fact, one place.

### `user_bodyweight_log`

A syncable table for bodyweight on days without training. Relative-strength
maths (weighted vs bodyweight PRs) needs a value on the day, and
`workout_sessions.bodyweight_kg` only exists on training days.

### Account deletion: soft, then hard after 30 days

`users.deletion_requested_at` is set with `status = 'deletion_pending'`
(`users_deletion_pending_ck` ties the two together). A reaper, arriving with
auth in Phase 2, hard-deletes 30 days later through the existing
`ON DELETE CASCADE` chain.

### Corrections made while implementing

- **`touch_sync` assigns `server_seq` unconditionally.** The proposal kept a
  client-supplied value on UPDATE if it differed from the old one, which let a
  client write its own cursor position. Now every insert and update takes the
  next per-user value, and a missing user raises.
- **`user_skill_states` evidence FK uses `ON DELETE SET NULL
  (evidence_set_entry_id)`.** A plain `SET NULL` on the composite key would
  also null `user_id` and violate `NOT NULL`.
- **"Unlocks are never revoked" is a trigger**, `user_skill_states_monotonic`:
  once `unlocked`, `state` and `first_achieved_at` are frozen.
- **`skill_unlock_events` is append-only by trigger**, not a rule (rules cannot
  raise). A `DELETE` arriving through a foreign-key cascade is let through, so
  account deletion still works.
- **`content_versions.checksum` is not unique.** Reverting content (A → B → A)
  legitimately records A again as the newest version; the seed compares
  against the newest version, not any version.
- **Every `CHECK` is named**, not only the vocabularies, so a violation always
  maps to a field-level problem at the HTTP edge.
- **User bands are not yet syncable.** `bands` mixes global rows (no owner)
  with user rows, and `touch_sync` needs a user. Phase 4 adds the sync columns
  in an expand-only migration when the sync endpoints exist.

## Consequences

- Assistance has a single representation; the evaluator filters on
  `assistance_class` and never has to interpret a sign.
- The schema's invariants are pinned by integration tests in
  `internal/store/schema_integration_test.go`, so a later migration cannot
  weaken one silently.
- Deleting a user remains a single `DELETE` despite two tables that otherwise
  refuse changes.
