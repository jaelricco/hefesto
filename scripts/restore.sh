#!/usr/bin/env bash
#
# Restore a logical backup taken by scripts/backup.sh.
#
# Runs ON THE SERVER, from the deploy directory (default /srv/hefesto), like
# deploy.sh. It runs inside the backup service's container, which already
# has the bucket's credentials and reaches Postgres on the Compose network.
#
#   ./scripts/restore.sh                          # latest dump → a new database
#   ./scripts/restore.sh hefesto-<stamp>.dump     # that dump → a new database
#   ./scripts/restore.sh latest hefesto_check     # latest dump → hefesto_check
#   ./scripts/restore.sh latest --into-live       # over the live database
#
# By default it restores into a NEW database (hefesto_restore_<stamp>) and
# never touches the live one: check the restored data there, then decide.
# --into-live stops the API, restores over the live database with --clean,
# and starts the API again. It asks first unless --yes is given.
#
# Point-in-time recovery from the WAL archive is a different procedure:
# docs/DEPLOYMENT.md, "Point-in-time recovery".

set -Eeuo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env.prod}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

dump="latest"
target=""
into_live=false
yes=false
positional=()
for arg in "$@"; do
    case "$arg" in
        --into-live) into_live=true ;;
        --yes) yes=true ;;
        -*) die "unknown option $arg" ;;
        *) positional+=("$arg") ;;
    esac
done
usage="usage: $0 [latest|<dump file>] [target database] [--into-live] [--yes]"
(( ${#positional[@]} <= 2 )) || die "$usage"
if (( ${#positional[@]} >= 1 )); then
    if [[ "${positional[0]}" == latest || "${positional[0]}" == *.dump ]]; then
        dump="${positional[0]}"
        target="${positional[1]:-}"
    else
        (( ${#positional[@]} == 1 )) || die "$usage"
        target="${positional[0]}"
    fi
fi
[[ "$target" =~ ^[a-z_][a-z0-9_]*$ || -z "$target" ]] || die "target database must be lowercase letters, digits and _"

cd "$DEPLOY_DIR" || die "deploy directory $DEPLOY_DIR does not exist"
[[ -f "$ENV_FILE" ]] || die "$DEPLOY_DIR/$ENV_FILE is missing"
compose() { docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" "$@"; }

if [[ "$into_live" == true ]]; then
    [[ -z "$target" ]] || die "--into-live restores over the live database; do not name a target"
    if [[ "$yes" != true ]]; then
        read -r -p "Restore $dump OVER THE LIVE DATABASE? The API stops meanwhile. Type 'restore' to go on: " answer
        [[ "$answer" == "restore" ]] || die "nothing changed"
    fi
    log "stopping the api"
    compose stop api
else
    target="${target:-hefesto_restore_$(date -u +%Y%m%d%H%M%S)}"
fi

log "restoring $dump into ${target:-the live database}"
# The script below runs in the backup container; $PG*, $HEFESTO_BACKUP_* come
# from its environment, DUMP/TARGET from here.
# shellcheck disable=SC2016
compose run --rm --no-deps -T -e DUMP="$dump" -e TARGET="$target" --entrypoint /bin/sh backup -c '
set -eu
apk add --no-cache postgresql16-client minio-client >/dev/null
mcli alias set backup "$HEFESTO_BACKUP_ENDPOINT" "$HEFESTO_BACKUP_ACCESS_KEY" "$HEFESTO_BACKUP_SECRET_KEY" >/dev/null
if [ "$DUMP" = latest ]; then
    DUMP=$(mcli ls "backup/$HEFESTO_BACKUP_BUCKET/daily/" | awk "{print \$NF}" | grep "\.dump$" | sort | tail -1)
    [ -n "$DUMP" ] || { echo "no dump in $HEFESTO_BACKUP_BUCKET/daily/" >&2; exit 1; }
fi
echo "dump: $DUMP"
mcli cp "backup/$HEFESTO_BACKUP_BUCKET/daily/$DUMP" /tmp/restore.dump >/dev/null
if [ -n "$TARGET" ]; then
    psql -v ON_ERROR_STOP=1 -q -d postgres -c "CREATE DATABASE \"$TARGET\""
    pg_restore --no-owner --no-privileges --exit-on-error -d "$TARGET" /tmp/restore.dump
    DB="$TARGET"
else
    pg_restore --clean --if-exists --no-owner --no-privileges --exit-on-error -d "$PGDATABASE" /tmp/restore.dump
    DB="$PGDATABASE"
fi
echo "restored into $DB:"
psql -q -d "$DB" -At -F " " -c "
    SELECT '\''users'\'', count(*) FROM users UNION ALL
    SELECT '\''workout_sessions'\'', count(*) FROM workout_sessions UNION ALL
    SELECT '\''set_entries'\'', count(*) FROM set_entries UNION ALL
    SELECT '\''content_versions'\'', count(*) FROM content_versions"
'

if [[ "$into_live" == true ]]; then
    log "starting the api"
    compose start api
    log "restored $dump over the live database"
else
    log "restored $dump into $target — inspect it, then drop it or swap it in (docs/DEPLOYMENT.md)"
fi
