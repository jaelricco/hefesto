# ADR 0011 — The skill map and stats on the device

- Status: accepted
- Date: 2026-09-26
- Deciders: Jaelricco

## Context

Phase 6 adds the skill map constellation, skill detail, the unlock
celebration, and history and stats. ADR 0010 says the UI reads only the local
database. The map, though, comes from `GET /v1/me/skill-map`, which is content
plus the athlete's level states, evaluated on the server. The brief's API
surface lists `GET /v1/me/stats/exercises/{id}` for stats, but that endpoint
was never built. The brief also asks the map to "animate edges on unlock".

## Decisions

### The skill map is cached locally

The map's content (skills, levels with their criteria, edges), the level
states, XP and the streak are all stored in the local database.

- **Views read these tables**, like every other screen. The map opens offline
  and never waits for the network.
- **Content is replaced whole when its version changes**, and states on every
  refresh. The app refreshes after every sync, because completing a session is
  what changes states.
- **Injury notes** are fetched when a skill is opened and kept for offline
  reading, together with the disclaimer the API sends with them.
- **Unlocks are never revoked here either.** If a refresh ever shows an
  unlocked level as anything else, the local copy keeps the unlock and takes
  the rest, such as current best values.
- **The server stays the judge.** The app never evaluates unlock criteria. It
  shows them in words, from the same DSL the server evaluates.

### "Newly unlocked" is local state

Each level state has a local `seenAt`, stamped once the map has shown the
unlock. A level that is unlocked but has no `seenAt` gets its lines lit with
an animation, then is marked seen.

- The first map refresh on a device marks every existing unlock as seen, so a
  new phone does not replay the athlete's whole history.
- This is presentation state, so it never syncs.

### Stats are computed on the device

Per-exercise stats (volume per day, and bests) are computed from the local
sets, with no endpoint.

- **The device already holds every set.** Sync pulls them all, so the numbers
  are the same numbers the server would compute.
- **Stats work offline** and update the moment a set is logged, before it has
  synced.
- **Bests count only clean repetitions**: full range, not eccentric-only, and
  no assistance. Added load still counts. Everything else counts as volume.

`GET /v1/me/stats/exercises/{id}` stays unbuilt until a client without the
full log needs it, such as a web view or a coach.

### History follows ISO weeks

Sessions are grouped by the ISO week of their local date, Monday first (brief
§10). The local date is the day the athlete trained where they were, so
travel never moves a session to another week.

## Consequences

- One more migration (`v2-skill-map`) in the local database, and a refresh call
  after each sync.
- If stats grow expensive over years of logs, they can be cached. The rules
  would stay the same.
- The map does not show `stale_since`. The API marks it as a hint, and showing
  it well without implying the athlete is falling behind (ADR 0003) is a
  design question for review.
