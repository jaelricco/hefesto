#!/usr/bin/env bash
#
# Point-in-time recovery from the pgBackRest repository (ADR 0026): bring the
# production database back to how it was at a moment, or to the newest WAL
# the archive holds.
#
# Runs ON THE SERVER, from the deploy directory (default /srv/hefesto).
#
#   ./scripts/pitr-restore.sh "2026-10-19 03:12:00+00"   # to that moment
#   ./scripts/pitr-restore.sh latest                      # to the end of the archive
#
# It stops the API, both backup sidecars and Postgres, restores the newest
# base backup before the target with --delta, starts Postgres to replay WAL
# up to the target and promote, then starts everything again. Everything
# written after the target is gone afterwards. It asks first unless --yes.
#
# For a whole-dump restore into a scratch database, use scripts/restore.sh.

set -Eeuo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
export DEPLOY_DIR DEPLOY_ENV=production
RECOVERY_TIMEOUT_S="${RECOVERY_TIMEOUT_S:-900}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

target=""
yes=false
for arg in "$@"; do
    case "$arg" in
        --yes) yes=true ;;
        -*) die "unknown option $arg" ;;
        *) [[ -z "$target" ]] || die "usage: $0 <timestamp|latest> [--yes]"; target="$arg" ;;
    esac
done
[[ -n "$target" ]] || die "usage: $0 <timestamp|latest> [--yes]   e.g. \"2026-10-19 03:12:00+00\""

cd "$DEPLOY_DIR" || die "deploy directory $DEPLOY_DIR does not exist"
[[ -f .env.prod ]] || die "$DEPLOY_DIR/.env.prod is missing"
compose() { "$DEPLOY_DIR/scripts/compose.sh" "$@"; }

restore_args=(--delta)
if [[ "$target" != latest ]]; then
    restore_args+=(--type=time "--target=$target" --target-action=promote)
fi

if [[ "$yes" != true ]]; then
    read -r -p "Recover the production database to '$target'? Everything after it is lost. Type 'recover' to go on: " answer
    [[ "$answer" == recover ]] || die "nothing changed"
fi

log "stopping the api, the backup sidecars and postgres"
compose stop api backup pitr
compose stop postgres

log "restoring the base backup (${restore_args[*]})"
compose run --rm --no-deps -T -u postgres --entrypoint pgbackrest postgres "${restore_args[@]}" restore

log "starting postgres to replay WAL up to '$target'"
compose up -d postgres
deadline=$(( $(date +%s) + RECOVERY_TIMEOUT_S ))
# shellcheck disable=SC2016  # $POSTGRES_* expand inside the container
until [[ "$(compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "SELECT pg_is_in_recovery()"' 2>/dev/null)" == f ]]; do
    (( $(date +%s) < deadline )) || die "postgres is still recovering after ${RECOVERY_TIMEOUT_S}s; see: ./scripts/compose.sh logs postgres"
    sleep 2
done
log "recovered and promoted"

log "starting the api and the backup sidecars"
compose up -d api backup pitr
for _ in $(seq 1 30); do
    if compose exec -T api /usr/local/bin/api -healthcheck >/dev/null 2>&1; then
        log "recovered to '$target'; the pitr sidecar takes a new base backup on its next run"
        exit 0
    fi
    sleep 2
done
die "the database is recovered, but the api did not become healthy; see: ./scripts/compose.sh logs api"
