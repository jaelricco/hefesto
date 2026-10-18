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

Generate each secret with `openssl rand -base64 48`. This file is the only
place production secrets exist. It is not in the repository, not in CI, and not
in any backup that leaves the host.

Use **two separate object-storage credentials**: one for media (the API needs
it) and one for backups with write access to the backup bucket only. A
compromised API key must not be able to delete the backups.

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
- a restore over the live database.

## Backups

The `backup` sidecar takes a `pg_dump --format=custom` every 24h to
`hefesto-backups/daily/` in Hetzner Object Storage and expires dumps older than
30 days. It refuses to upload a dump under 4 KB, which is the usual signature
of a dump that ran against the wrong or an empty database.

This is a **logical** backup: you can restore to the last nightly dump and
nothing between. Point-in-time recovery via WAL archiving (pgBackRest) is a
Phase 7 item, and the moment there is real user data it stops being optional.

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
