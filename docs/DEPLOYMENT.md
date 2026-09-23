# Deployment

Production is a single Hetzner Cloud instance running the same Compose topology
as development, behind Caddy. Deploys go through GitHub Actions; nothing is
built or pushed from a laptop.

```
git tag v0.2.0  ──▶  GitHub Actions ──▶  ghcr.io/jaelricco/hefesto:v0.2.0
                          │
                          └── ssh ──▶  /srv/hefesto/scripts/deploy.sh v0.2.0
                                            │
                                            ├─ pull image
                                            ├─ goose up          (fails ⇒ abort, nothing changed)
                                            ├─ recreate api      
                                            ├─ poll /readyz      (fails ⇒ roll back to previous tag)
                                            └─ record .deployed-tag
```

## One-time setup

### 1. The server

Hetzner CAX21 (ARM, 4 vCPU / 8 GB) in Falkenstein or Nuremberg, Ubuntu 24.04.

```sh
make server-bootstrap host=<ip>
```

That installs Docker, creates the `deploy` user, opens 22/80/443, disables SSH
password and root login, turns on unattended security upgrades, and creates
`/srv/hefesto`.

### 2. DNS

| Record | Points to |
|---|---|
| `api.hefesto.fit` | server IP (A, and AAAA if you use IPv6) |
| `hefesto.fit` | server IP |
| `hefesto.ch` | server IP |

Caddy obtains certificates for all three on first boot. `hefesto.ch` serves a
permanent redirect to `hefesto.fit`.

### 3. `/srv/hefesto/.env.prod`

Copy `.env.prod.example` to the server, fill in every blank, then:

```sh
chmod 600 /srv/hefesto/.env.prod
chown deploy:deploy /srv/hefesto/.env.prod
```

Generate each secret with `openssl rand -base64 48`. This file is the only
place production secrets exist. It is not in the repository, not in CI, and not
in any backup that leaves the host.

Use **two separate object-storage credentials**: one for media (the API needs
it) and one for backups with write access to the backup bucket only. A
compromised API key must not be able to delete the backups.

### 4. GitHub repository secrets

Settings → Secrets and variables → Actions:

| Secret | What it is |
|---|---|
| `DEPLOY_HOST` | server IP or hostname |
| `DEPLOY_USER` | `deploy` |
| `DEPLOY_PORT` | optional, defaults to 22 |
| `DEPLOY_SSH_KEY` | private half of an ed25519 key whose public half is in the deploy user's `authorized_keys`. Generate it for CI only; do not reuse a personal key. |
| `GHCR_PULL_TOKEN` | fine-grained PAT with `read:packages`, so the server can pull the private image |

Pushing to GHCR uses the built-in `GITHUB_TOKEN`; no secret needed for that.

### 5. Optional but recommended: require an approval

Settings → Environments → new environment `production` → add yourself as a
required reviewer. The deploy job already targets that environment, so it will
then wait for a click before touching production.

## Deploying

```sh
make release version=v0.2.0     # tag, push, build, deploy
make deploy-watch               # follow the run
make prod-version               # what is live right now
```

Rolling back is deploying an older tag — the same path, not a special script:

```sh
make deploy tag=v0.1.9
```

**Rollback caveat:** the schema does not roll back. Migrations from the failed
release have already been applied when the API is reverted, which is exactly
why migrations must be backward compatible for one release (expand, deploy,
contract). If you truly need to undo a migration, do it deliberately with
`goose down` and a restore plan, not as part of a rollback.

Emergency deploys can skip the test job:

```sh
gh workflow run deploy.yml -f tag=v0.2.1 -f skip_tests=true
```

Use it when production is already broken and the fix is obvious. Not otherwise.

## Backups

The `backup` sidecar takes a `pg_dump --format=custom` every 24h to
`hefesto-backups/daily/` in Hetzner Object Storage and expires dumps older than
30 days. It refuses to upload a dump under 4 KB, which is the usual signature
of a dump that ran against the wrong or an empty database.

This is a **logical** backup: you can restore to the last nightly dump and
nothing between. Point-in-time recovery via WAL archiving (pgBackRest) is a
Phase 7 item, and the moment there is real user data it stops being optional.

### Restoring — rehearse this before you need it

```sh
# on the server
mc cp backup/hefesto-backups/daily/hefesto-<stamp>.dump /tmp/restore.dump
docker compose -f docker-compose.prod.yml --env-file .env.prod stop api
docker compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres \
  pg_restore --clean --if-exists --no-owner --no-privileges \
             -U hefesto -d hefesto < /tmp/restore.dump
docker compose -f docker-compose.prod.yml --env-file .env.prod start api
```

A backup that has never been restored is a hypothesis, not a backup. Do this
once against a scratch database before the first real user exists, and note the
date in `STATUS.md`.

## Operating

```sh
make prod-ps            # what is running
make prod-logs          # tail the api
make prod-psql          # psql on production — think before you type
make prod-backup-now    # force a backup off-schedule
```

Logs are JSON on stdout, capped at 10 MB × 5 files per service by Docker's
json-file driver. There is no log aggregation yet; when it is needed, that gets
its own ADR rather than an agent silently added to the box.

## What is deliberately not here

- **No blue/green or rolling deploy.** One container is replaced, which means a
  few seconds of downtime per deploy. For a single-user-scale app this is the
  right trade; revisit when downtime costs something.
- **No secret manager.** `.env.prod` with mode 0600 is proportionate at this
  size. When there is a second host or a second person, this changes.
- **No staging environment.** The verify job runs the test suite against the
  exact tagged tree, and migrations are gated. A staging box is worth adding
  once the iOS client is in TestFlight and a bad schema change can strand real
  devices.
