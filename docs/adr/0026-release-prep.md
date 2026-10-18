# ADR 0026 — Release prep: staging, rehearsed deploys and recovery

- Status: accepted
- Date: 2026-10-19
- Deciders: Jaelricco

## Context

Phase 8 prepares the first external release. When it started, production had
never been deployed. ADR 0004 had left four things for later:
- a staging environment, "once the iOS app is in TestFlight";
- WAL archiving, "a Phase 7 commitment, not a maybe";
- a rehearsed restore, as part of Phase 7's definition of done;
- an internal TestFlight build that real data cannot be stranded by.

The owner chose:
- staging on the production host, not a second server;
- internal TestFlight builds against staging, external ones against
  production;
- a recommendations page for the open review questions
  (`docs/REVIEW_DECISIONS.md`);
- a step-by-step first deploy that the owner runs, since the server and its
  accounts are theirs.

## Decisions

### Staging is a second Compose project on the production host

`docker-compose.staging.yml` runs the same image as production under the
project name `hefesto-staging`:
- its own Postgres volume, its own `.env.staging`, its own media bucket;
- smaller Postgres memory settings, since it shares the host;
- no Caddy and no backup sidecar.

Production's Caddy serves it at `api-staging.hefesto.fit` over a shared
external Docker network, `hefesto-edge`. `scripts/deploy.sh` takes
`DEPLOY_ENV=staging` and creates that network when it is missing, so either
environment can be deployed first.

`deploy-staging.yml` deploys every commit on `main` whose CI passed, as image
tag `sha-<commit>`. It stays off until the repository variable
`STAGING_DEPLOY` is `true`, so `main` does not go red before the server
exists.

A failed staging deploy never touches production. A runaway staging process
can slow production; a second host is the answer when that happens, not
before.

### TestFlight: Staging and Release configurations

The app has a third build configuration, `Staging`. It is a release build,
so it carries no Debug code (no demo mode), and it talks to the staging API.
The TestFlight workflow archives:
- `Staging` for internal testers, the default and on `ios-v*` tags;
- `Release` for external testers, which also requires
  `make content-release-check`.

The Mac CI job now builds `Staging` on every run. A release-type build had
never been compiled before a TestFlight archive would have been the first.

### Every pull request rehearses a deploy

`scripts/deploy-rehearsal.sh` runs on a GitHub runner's Docker, through the
real scripts, Compose files and Caddyfile, with throwaway secrets and
`*.localhost` domains (Caddy's own CA, no Let's Encrypt). In order:
1. deploy production, and check it through Caddy;
2. deploy staging beside it, and check that its database is its own;
3. redeploy production, which must be a no-op;
4. wait for the first logical dump and the first full base backup;
5. restore the dump into a scratch database, and compare row counts;
6. restore the dump over the live database, and check the API comes back;
7. recover to a point in time, and check that a row written after it is gone.
It drives Compose through `scripts/compose.sh`, as an operator on the server
would.

Building it and running it found three bugs, each of which would have broken
the first deploy or the first restore:
- the Caddyfile used `exclude` inside `log`, which Caddy rejects, so no
  public endpoint would have come up;
- `backup.sh` called `mc`, but Alpine installs the MinIO client as `mcli`, so
  no backup would ever have been uploaded;
- the Compose files require `HEFESTO_TAG`, which only `deploy.sh` set, so
  `make prod-ps`, `prod-logs`, `prod-psql` and `restore.sh` failed on the
  server. `scripts/compose.sh` now reads the live tag from `.deployed-tag`.

CI validates the Caddyfile with the `caddy:2-alpine` image production runs.

### Restores: a script, scratch database by default

`scripts/restore.sh` restores a logical dump into a new database
(`hefesto_restore_<stamp>`) unless asked otherwise. `--into-live` stops the
API, restores with `--clean`, and starts it again, after a typed
confirmation unless `--yes` is given.

### Point-in-time recovery with pgBackRest

Production Postgres is `postgres:16-alpine` plus Alpine's pgBackRest
(`docker/Dockerfile.postgres`), built on the server by Compose. With that
file unchanged, the build is cached and the image id is the same, so deploys
do not restart the database.

- **WAL archiving.** Postgres archives every WAL segment with
  `archive_command = pgbackrest archive-push`. `archive_timeout` is 60 s, so
  a recovery loses at most about a minute.
- **Base backups.** A `pitr` sidecar runs as the postgres user, with the data
  volume mounted read-only and the socket directory shared. It takes a full
  backup weekly and a differential daily. pgBackRest expires old backups
  after two fulls, about two weeks of recovery window.
- **Encryption.** The repository is encrypted with aes-256-cbc. The
  passphrase lives only in `.env.prod`, plus a copy the owner keeps off the
  server: without it the backups cannot be read.
- **Restore.** `scripts/pitr-restore.sh` recovers to a timestamp, or to the
  end of the archive.
- **Why pgBackRest.** WAL-G's current releases cannot be pinned through the
  Go module proxy, because its module path never took the `/v3` suffix.
  pgBackRest is packaged by Alpine, and it is what ADR 0004 named.
- **Its constraint.** pgBackRest requires TLS to object storage. Hetzner
  serves TLS; the rehearsal runs a second MinIO with a throwaway certificate.

The nightly logical dump stays: it is a second, independent path in a
different format.

### A preflight before the first deploy

`make preflight` runs `scripts/preflight.sh` on the server over SSH. It
checks what the rehearsal cannot, because it uses throwaway values: the real
env files, DNS, ports, and every real object-storage credential. Each bucket
gets a probe written, read back and deleted, as the API and the sidecars will
do. It changes nothing and prints no secret.

Writing it turned up one trap. The templates said to generate
`POSTGRES_PASSWORD` with `openssl rand -base64`, but that password is also
pasted into `DATABASE_URL`, where a `/` breaks the URL. The templates now say
`openssl rand -hex 32`, and the preflight rejects a password a URL cannot
hold as it is.

### The device id rotates on sign-out

The server binds a device id to one account (ADR 0007), but the app kept one
id per install. A second account signing in on the same phone was refused.
The app now starts a new device id whenever it signs out, by hand or because
the server ended the session.

### Sign-out erases the phone's copy

Once a second account could sign in, the local database became a leak. The
next account would see the last one's log, and the last one's queued changes
would sync as its own. `docs/REVIEW_DECISIONS.md` had already recommended
wiping on sign-out (D-16). The device-id fix made it a bug, so it ships here:
- **Signing out** erases every table (`AppDatabase.eraseAll`). When changes
  are unsynced, Today says how many and offers to sync first. "Sync and sign
  out" signs out only when nothing is left to send.
- **When the server ends the session**, nothing is erased: the same athlete
  signs in again and their queue syncs. The app records which account the
  database holds, and a sign-in by any other account erases it first.

## Consequences

- **The first deploy is the owner's steps.** `docs/DEPLOYMENT.md` lists them
  in order: server, DNS, both env files, GitHub secrets and variable, App
  Store Connect. Everything after that is a tag or a merge.
- **The rehearsal adds CI time.** It runs about ten minutes on every pull
  request, mostly building MinIO and the image.
- **Staging shares production's CPU and memory.**
- **Losing the pgBackRest passphrase loses the archive.** The logical dumps
  are not encrypted by pgBackRest and stay readable.
