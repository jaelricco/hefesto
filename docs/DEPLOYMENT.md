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

## First deploy, in order

Production has never been deployed. Start to finish it takes about an hour,
most of it waiting for DNS. Each step is detailed below.

1. **Server:** `make server-bootstrap host=<ip>`, then
   `git config hefesto.prodhost <ip>` so the `make prod-*` targets find it.
2. **DNS:** the four records in step 2. Start this early; it propagates while
   you do the rest.
3. **Object storage:** in Hetzner Object Storage, create the buckets
   `hefesto-media`, `hefesto-backups`, `hefesto-pitr` and
   `hefesto-media-staging`, all private. Then create credentials: one for
   production media, one for backups (the dump and pgBackRest) and one for
   staging media.
4. **Env files:** fill in `.env.prod` and `.env.staging` from their templates.
   Copy them to `/srv/hefesto/` and `chmod 600` both (steps 3 and 3b). Put
   `HEFESTO_PITR_CIPHER_PASS` in your password manager too.
5. **`make preflight`:** run it until it ends "Ready for the first deploy"
   (step 3c).
6. **GitHub:** add the deploy secrets (step 4). Optionally, add a
   `production` environment that needs your approval (step 5).
7. **Deploy:** `make release version=v0.8.0`, then `make deploy-watch`.
   `curl https://api.hefesto.fit/healthz` should answer `"status":"ok"`.
8. **Staging:** set the repository variable `STAGING_DEPLOY` to `true`, then
   `gh workflow run deploy-staging.yml -f ref=main`.
9. **Backups:** within a few minutes, `make prod-logs` shows the API.
   `./scripts/compose.sh logs backup pitr` on the server shows
   `uploaded hefesto-…` and `full backup done`. Then rehearse one restore:
   `./scripts/restore.sh`, look at the counts, and drop the scratch database.
10. **TestFlight:** set up App Store Connect and its four secrets (below).
    Run the `testflight` workflow, which builds Staging for internal testers.

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
| `api-staging.hefesto.fit` | server IP |
| `hefesto.fit` | server IP |
| `hefesto.ch` | server IP |

Caddy obtains certificates for all four on first boot. `hefesto.ch` serves a
permanent redirect to `hefesto.fit`.

### 3. `/srv/hefesto/.env.prod`

Copy `.env.prod.example` to the server, fill in every blank, then:

```sh
chmod 600 /srv/hefesto/.env.prod
chown deploy:deploy /srv/hefesto/.env.prod
```

Generate each secret with `openssl rand -base64 48`, except
`POSTGRES_PASSWORD`. Generate that one with `openssl rand -hex 32`: it also goes
into `DATABASE_URL`, where a `/` or `+` would break the URL. This file is the
only place production secrets exist. It is not in the repository, not in CI,
and not in any backup that leaves the host.

Use **separate object-storage credentials**: one for media, which the API
needs, and one for backups, with access to the backup and pitr buckets only. A
compromised API key must not be able to delete the backups.

`HEFESTO_PITR_CIPHER_PASS` encrypts the point-in-time archive. Keep a copy off
the server, in your password manager. Without it, that archive cannot be read.

The media bucket (`HEFESTO_S3_BUCKET`) must exist and stay **private**: phones
upload and download through short-lived presigned URLs the API signs
(`HEFESTO_S3_PRESIGN_TTL`, 15 minutes by default), never through public reads.
Outside dev the API refuses to start without `HEFESTO_S3_ENDPOINT`. Objects live
under `u/<user id>/`; deleting an account removes that prefix.

### 3b. `/srv/hefesto/.env.staging`

Staging (ADR 0026) runs on the same server as its own Compose project. Copy
`.env.staging.example`, fill in every blank, and `chmod 600` it like the
production file. Use values of its own: a different database password, a
different JWT signing key, and a separate media bucket
(`hefesto-media-staging`) with its own credentials. A staging token must not
work against production.

### 3c. Check the server

```sh
make preflight
```

This runs `scripts/preflight.sh` on the server over SSH. It changes nothing
and prints no secret. It checks:
- Docker and Compose;
- both env files: their mode, no blank or missing key, and a `DATABASE_URL`
  that matches the Postgres settings;
- that staging shares no secret or bucket with production;
- that every domain resolves to this server, and that ports 80 and 443 are
  free;
- for every bucket, that its credential can list, write, read and delete;
- that the media bucket refuses anonymous reads.

Fix each `FAIL` it prints and run it again until it ends "Ready for the first
deploy".

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

Staging uses the same secrets. It stays off until you set the repository
**variable** `STAGING_DEPLOY` to `true` (Settings → Secrets and variables →
Actions → Variables), after `.env.staging` is on the server.
`STAGING_API_DOMAIN` overrides `api-staging.hefesto.fit` if you choose another
name.

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

## Staging

Every commit on `main` whose CI passed deploys to staging
(`.github/workflows/deploy-staging.yml`). It uses the same image as production
and the same `scripts/deploy.sh`, with `DEPLOY_ENV=staging`. It has its own
database and media bucket, no backup sidecar (its data is disposable), and no
Caddy: production's Caddy serves it at `api-staging.hefesto.fit` over the
shared `hefesto-edge` Docker network. Internal TestFlight builds talk to it.

```sh
gh workflow run deploy-staging.yml -f ref=<branch-or-sha>   # deploy any ref by hand
DEPLOY_ENV=staging ./scripts/compose.sh logs -f api       # on the server, in /srv/hefesto
```

A failed staging deploy never touches production. While staging is down,
`api-staging.hefesto.fit` answers 502.

## Rehearsed on every pull request

The `deploy rehearsal` CI job (`scripts/deploy-rehearsal.sh`) runs all of this
on a runner's Docker before it ever runs on the server:
- `deploy.sh` for production and then staging, through Caddy;
- a redeploy;
- the backup sidecar's first dump to object storage (MinIO);
- a restore into a scratch database, with row counts compared;
- a restore over the live database;
- the first pgBackRest full backup, and a point-in-time recovery that removes
  a row written after the target.

## Backups

The `backup` sidecar takes a `pg_dump --format=custom` every 24h to
`hefesto-backups/daily/` in Hetzner Object Storage and expires dumps older than
30 days. It refuses to upload a dump under 4 KB, which is the usual signature
of a dump that ran against the wrong or an empty database.

A dump is a **logical** backup: it restores to the moment it was taken and
nothing in between. Point-in-time recovery (below) covers the gap. The dump
stays as a second, independent path in a different format.

### Restoring

`scripts/restore.sh` restores a dump. By default it restores into a **new**
database and never touches the live one:

```sh
# on the server, in /srv/hefesto
./scripts/restore.sh                         # latest dump → hefesto_restore_<stamp>
./scripts/restore.sh hefesto-<stamp>.dump    # a chosen dump → a new database
./scripts/restore.sh latest --into-live      # over the live database (asks first)
```

It prints row counts of the restored database. `--into-live` stops the API,
restores with `--clean`, and starts it again. CI rehearses both paths on every
pull request (above). Drop a scratch database when you are done with it:
`./scripts/compose.sh exec postgres dropdb -U hefesto hefesto_restore_<stamp>`.

### Point-in-time recovery

Production Postgres archives every WAL segment to an encrypted pgBackRest
repository in `hefesto-pitr`, at least once a minute (ADR 0026). The `pitr`
sidecar takes a full base backup weekly and a differential daily, and keeps
two weeks of fulls. Production's Postgres image is `postgres:16-alpine` plus
pgBackRest (`docker/Dockerfile.postgres`), built on the server by
`deploy.sh`. Unchanged, the build is cached and a deploy does not restart the
database.

```sh
# on the server, in /srv/hefesto
./scripts/compose.sh exec pitr pgbackrest info      # backups and the WAL range
./scripts/pitr-restore.sh "2026-10-19 03:12:00+00"  # back to that moment (asks first)
./scripts/pitr-restore.sh latest                     # to the newest WAL archived
```

`pitr-restore.sh` stops the API, both sidecars and Postgres. It restores the
newest base backup before the target, replays WAL up to the target and
promotes, then starts everything again. **Everything written after the target
is gone.** If you are unsure of the moment, restore a logical dump into a
scratch database first (`restore.sh`) and look.

If archiving fails (a rotated key, a missing bucket), the `pitr` log says
`ERROR archiving is failing` on its next cycle. Postgres keeps the WAL it
could not send, up to 4 GiB. Past that, pgBackRest drops it so the disk never
fills. Recovery then has a gap until the next backup, but the database stays
up.

Deploys reuse the built Postgres image, so they never restart the database,
but they also don't pick up security updates to `postgres:16-alpine`. Take
those deliberately, about monthly:
`./scripts/compose.sh build --pull postgres pitr && ./scripts/compose.sh up -d postgres pitr`.
That restarts Postgres for a few seconds.

## Operating

```sh
make prod-ps            # what is running
make prod-logs          # tail the api
make prod-psql          # psql on production — think before you type
make prod-backup-now    # force a backup off-schedule
```

On the server, run Compose through `scripts/compose.sh` (`./scripts/compose.sh
ps`, `DEPLOY_ENV=staging ./scripts/compose.sh logs api`). The Compose files
need the live image tag, which a bare `docker compose` does not have; the
wrapper reads it from `.deployed-tag`.

Logs are JSON on stdout, capped at 10 MB × 5 files per service by Docker's
json-file driver. There is no log aggregation yet; when it is needed, that gets
its own ADR rather than an agent silently added to the box.

## The iOS app on TestFlight

`.github/workflows/testflight.yml` archives the app and uploads it to App
Store Connect (ADR 0025). It runs on a GitHub-hosted Mac, never on the
self-hosted runner, because signing writes certificates and provisioning
profiles.

### One-time setup

1. In App Store Connect, create the app with the bundle id `fit.hefesto.ios`,
   and enable **Sign in with Apple** on that identifier in the developer
   portal.
2. Create an App Store Connect API key with the **Admin** role (Users and
   Access → Integrations). Admin lets Xcode sign with cloud-managed
   certificates, so no signing key is ever stored.
3. Add four repository secrets:

   | Secret | Value |
   |---|---|
   | `APPLE_TEAM_ID` | the ten-character team id |
   | `APP_STORE_CONNECT_KEY_ID` | the key's id |
   | `APP_STORE_CONNECT_ISSUER_ID` | the issuer id shown above the keys |
   | `APP_STORE_CONNECT_KEY_P8` | the contents of the downloaded `.p8` file |

### Uploading a build

Run the `testflight` workflow from the Actions tab, or push a tag:

```sh
git tag -a ios-v0.7.0 -m "iOS 0.7.0" && git push origin ios-v0.7.0
```

- **Versions.** The build number is the workflow's run number, and the version
  is `MARKETING_VERSION` in `ios/project.yml`.
- **Release builds** talk to `https://api.hefesto.fit`, so deploy the API first
  when a build needs a new endpoint.
- **Before inviting external testers**, dispatch the workflow with `external`
  ticked. It then fails while any skill or exercise is still a placeholder
  (`make content-release-check`).
- **Before each upload**, dispatch the `ios` workflow with `ui_tests` ticked
  to run the accessibility audit.

## What is deliberately not here

- **No blue/green or rolling deploy.** One container is replaced, which means a
  few seconds of downtime per deploy. For a single-user-scale app this is the
  right trade; revisit when downtime costs something.
- **No secret manager.** `.env.prod` with mode 0600 is proportionate at this
  size. When there is a second host or a second person, this changes.
- **No separate staging server.** Staging shares the production host as its
  own Compose project (ADR 0026). It is cheap and close to production, and a
  runaway staging process can slow production; a second host is worth it once
  that happens.
