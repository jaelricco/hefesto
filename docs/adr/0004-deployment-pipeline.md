# ADR 0004 — Deployment pipeline

- Status: accepted
- Date: 2026-09-23
- Deciders: Jaelricco

## Context

ADR 0002 chose Hetzner Cloud. This ADR settles how code gets there. The
constraints that shaped it:

- one developer, working with Claude Code, who wants to ship without leaving
  the terminal;
- no users yet, so downtime is currently free — but the pipeline should not
  need rewriting when that changes;
- secrets must never pass through CI or the repository;
- a bad release must be reversible in under a minute, by a path that gets
  exercised often enough to still work.

## Decision

**Tag-triggered deploys through GitHub Actions**, with `workflow_dispatch` for
redeploys and rollbacks. `main` stays continuously deployable; nothing reaches
production because a merge happened.

**One image, built once.** `ghcr.io/jaelricco/hefesto:<tag>` carries the API,
the seeder, the linter, goose, and the migration files. The migrations that run
against production are provably the ones built alongside the binary, rather
than whatever is checked out on the server.

**The pipeline:**

1. `verify` — re-runs vet, unit tests and contentlint against the exact tagged
   tree. CI already ran against the PR merge commit; this runs against what is
   actually shipping, which is not the same thing.
2. `build` — buildx, push to GHCR with the tag, the version and the long SHA.
3. `deploy` — rsync the compose file, Caddyfile and scripts to the host, then
   run `scripts/deploy.sh <tag>` over SSH, then smoke-test the public HTTPS
   endpoint from outside.

**`scripts/deploy.sh` owns the release sequence**, not the workflow YAML. It
records the live tag, pulls, migrates to completion, recreates the API, polls
readiness, and rolls back to the recorded tag if readiness never arrives. Two
reasons for putting it in a script: it can be run by hand when GitHub is down,
and it is reviewable as a program rather than as forty lines of `run:`.

**Migrations run before the new API starts.** This makes every migration
obliged to be backward compatible with the currently running version for the
duration of one deploy — expand, deploy, contract. The alternative (migrate
after) breaks the new code instead, which is worse because the failure is
subtler.

**Secrets live only in `/srv/hefesto/.env.prod`**, mode 0600. CI holds the SSH
key, the host, the user and a registry pull token — nothing about the
application. The deploy workflow explicitly does not ship `.env.prod`.

**Caddy terminates TLS** for `api.hefesto.fit`, `hefesto.fit` and
`hefesto.ch`, obtaining and renewing certificates itself. Postgres publishes no
port at all; it is reachable only on the Compose network.

**Backups are a sidecar, not a host cron job**, so they are part of the
deployed unit and cannot go quietly missing on a rebuilt host.

## Consequences

- **Deploys cause a few seconds of downtime.** One container is replaced. This
  is the right trade at this size and the wrong one later; blue/green gets its
  own ADR when downtime starts costing something.
- **Rollback reverts code, not schema.** The failed release's migrations stay
  applied. The expand/deploy/contract discipline is what makes that safe, and
  it is a real constraint on how migrations get written — recorded in
  `CLAUDE.md` so it is in front of whoever writes the next one.
- **No staging environment.** Justified while the only client is a simulator.
  Once the iOS app is in TestFlight, a bad schema change can strand real
  devices, and staging stops being optional.
- **Backups are nightly logical dumps.** Recovery point objective is 24 hours.
  Acceptable with no users; not acceptable with them. WAL archiving is a Phase 7
  commitment, not a maybe.
- **`skip_tests` exists.** It is a loaded gun, documented as such: for when
  production is already broken and the fix is known.
- **A rehearsed restore is part of the definition of done for Phase 7.** A
  backup that has never been restored is a hypothesis.
