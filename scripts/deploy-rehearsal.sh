#!/usr/bin/env bash
#
# Rehearse a deploy on a throwaway machine: production, staging, a backup and
# two restores, through the real scripts and Compose files (ADR 0026). CI runs
# it on every pull request, so the first deploy to the real host is not the
# first time any of this has run.
#
#   HEFESTO_TEST_MINIO_HOST=0.0.0.0 scripts/with-minio.sh scripts/deploy-rehearsal.sh
#
# Needs Docker, ports 80, 443 and 5000 free, and MinIO from with-minio.sh. It
# builds the production image, pushes it to a local registry, writes
# .env.prod and .env.staging with throwaway secrets into a scratch deploy
# directory, and then:
#   1. deploys production with scripts/deploy.sh, and checks it through Caddy;
#   2. deploys staging beside it, and checks that its database is its own;
#   3. deploys production again (a redeploy must be a no-op);
#   4. waits for the backup sidecar's first dump in object storage;
#   5. restores it into a scratch database and compares the rows;
#   6. restores it over the live database and checks the API comes back.
# Domains are *.localhost, which Caddy serves from its own CA, so nothing
# asks Let's Encrypt for anything. Everything is removed at the end.

set -Eeuo pipefail

: "${HEFESTO_TEST_S3_ENDPOINT:?run this under scripts/with-minio.sh}"
: "${HEFESTO_TEST_S3_ACCESS_KEY:?}" "${HEFESTO_TEST_S3_SECRET_KEY:?}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
registry="hefesto-rehearsal-registry"
image="localhost:5000/hefesto"
tag="rehearsal-$(date -u +%s)"

diag="$(mktemp)"
log()  { printf '\033[1;35m## %s\033[0m\n' "$*"; echo "## $*" >> "$diag"; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; echo "xx $*" >> "$diag"; exit 1; }
# In CI, a failure also becomes an annotation: the step, the error and the
# last log lines, readable without the job log.
annotate() {
    [[ -n "${GITHUB_ACTIONS:-}" ]] || return 0
    local body
    body=$(tail -n 120 "$diag" | sed -e 's/%/%25/g' | awk '{ printf "%s%%0A", $0 }')
    echo "::error title=deploy rehearsal failed::$body"
}
hex()  { od -An -N"$1" -tx1 /dev/urandom | tr -d ' \n'; }
uuid7() { # time-ordered, as the app makes them
    local ms r
    ms=$(printf '%012x' "$(( $(date +%s%N) / 1000000 ))")
    r=$(hex 10)
    printf '%s-%s-7%s-%x%s-%s\n' "${ms:0:8}" "${ms:8:4}" "${r:0:3}" $(( 8 + RANDOM % 4 )) "${r:3:3}" "${r:6:12}"
}
prod()    { docker compose -f "$work/docker-compose.prod.yml" --env-file "$work/.env.prod" "$@"; }
staging() { docker compose -f "$work/docker-compose.staging.yml" --env-file "$work/.env.staging" "$@"; }
psql_prod() { prod exec -T postgres psql -U hefesto -At -c "$2" "$1"; }

cleanup() {
    status=$?
    if [[ $status -ne 0 ]]; then
        echo "failed at line ${BASH_LINENO[0]:-?} (exit $status); state of both projects" >> "$diag"
        { prod ps -a; staging ps -a; prod logs --no-color --tail 40; staging logs --no-color --tail 20 api; } \
            >> "$diag" 2>&1 || true
        cat "$diag" >&2
        annotate
    fi
    staging down -v --remove-orphans >/dev/null 2>&1 || true
    prod down -v --remove-orphans >/dev/null 2>&1 || true
    docker network rm hefesto-edge >/dev/null 2>&1 || true
    docker rm -f "$registry" >/dev/null 2>&1 || true
    rm -rf "$work" "$diag"
    exit $status
}
trap cleanup EXIT

# Through Caddy, as a phone would reach it.
https() { # https <host> <method> <path> [json]
    local args=(-sS -k --max-time 15 --resolve "$1:443:127.0.0.1" -X "$2" -o /dev/stderr -w '%{http_code}')
    [[ $# -ge 4 ]] && args+=(-H 'Content-Type: application/json' --data "$4")
    curl "${args[@]}" "https://$1$3" 2>/dev/null
}
wait_ready() { # wait_ready <host>
    for _ in $(seq 1 60); do
        [[ "$(https "$1" GET /readyz 2>/dev/null)" == 200 ]] && return 0
        sleep 2
    done
    die "$1 never became ready through Caddy"
}

log "image $image:$tag"
{
    docker run -d --name "$registry" -p 127.0.0.1:5000:5000 registry:2
    docker build -q -f "$root/docker/Dockerfile" -t "$image:$tag" \
        --build-arg VERSION="$tag" --build-arg COMMIT="$(git -C "$root" rev-parse HEAD)" "$root"
    docker push -q "$image:$tag"
} >> "$diag" 2>&1

# The deploy directory, as the deploy workflows rsync it to /srv/hefesto.
mkdir -p "$work/docker"
cp "$root"/docker-compose.prod.yml "$root"/docker-compose.staging.yml "$work/"
cp "$root/docker/Caddyfile" "$work/docker/"
cp -r "$root/scripts" "$work/"

# Containers reach MinIO through the bridge's gateway, which is the host.
gateway=$(docker network inspect bridge -f '{{(index .IPAM.Config 0).Gateway}}')
s3="http://$gateway:${HEFESTO_TEST_S3_ENDPOINT##*:}"

envfile() { # envfile <template> <out> KEY=value...
    local tpl="$1" out="$2"; shift 2
    cp "$tpl" "$out"
    for kv in "$@"; do
        local k="${kv%%=*}" v="${kv#*=}"
        grep -q "^$k=" "$out" || die "$k is not in $tpl"
        sed -i "s|^$k=.*|$k=$v|" "$out"
    done
    chmod 600 "$out"
}

pg_prod="$(hex 24)"; pg_staging="$(hex 24)"
envfile "$root/.env.prod.example" "$work/.env.prod" \
    HEFESTO_IMAGE="$image" \
    HEFESTO_API_DOMAIN=api.localhost HEFESTO_SITE_DOMAIN=hefesto.localhost \
    HEFESTO_SITE_DOMAIN_ALT=alt.hefesto.localhost HEFESTO_STAGING_API_DOMAIN=api-staging.localhost \
    HEFESTO_ACME_EMAIL=ops@example.test \
    POSTGRES_PASSWORD="$pg_prod" \
    DATABASE_URL="postgres://hefesto:$pg_prod@postgres:5432/hefesto?sslmode=disable" \
    HEFESTO_JWT_SIGNING_KEY="$(hex 48)" \
    HEFESTO_S3_ENDPOINT="$s3" HEFESTO_S3_REGION=us-east-1 HEFESTO_S3_USE_PATH_STYLE=true \
    HEFESTO_S3_ACCESS_KEY="$HEFESTO_TEST_S3_ACCESS_KEY" HEFESTO_S3_SECRET_KEY="$HEFESTO_TEST_S3_SECRET_KEY" \
    HEFESTO_BACKUP_ENDPOINT="$s3" HEFESTO_BACKUP_BUCKET=hefesto-backups \
    HEFESTO_BACKUP_ACCESS_KEY="$HEFESTO_TEST_S3_ACCESS_KEY" HEFESTO_BACKUP_SECRET_KEY="$HEFESTO_TEST_S3_SECRET_KEY" \
    HEFESTO_BACKUP_INTERVAL_S=3600
envfile "$root/.env.staging.example" "$work/.env.staging" \
    HEFESTO_IMAGE="$image" \
    POSTGRES_PASSWORD="$pg_staging" \
    DATABASE_URL="postgres://hefesto:$pg_staging@postgres:5432/hefesto_staging?sslmode=disable" \
    HEFESTO_JWT_SIGNING_KEY="$(hex 48)" \
    HEFESTO_S3_ENDPOINT="$s3" HEFESTO_S3_REGION=us-east-1 HEFESTO_S3_USE_PATH_STYLE=true \
    HEFESTO_S3_ACCESS_KEY="$HEFESTO_TEST_S3_ACCESS_KEY" HEFESTO_S3_SECRET_KEY="$HEFESTO_TEST_S3_SECRET_KEY"
# Both env files must leave no blank: the real ones may not either.
! grep -E '^[A-Z_]+=$' "$work/.env.prod" "$work/.env.staging" || die "a key is left blank above"

log "1. deploy production"
DEPLOY_DIR="$work" "$work/scripts/deploy.sh" "$tag" 2>&1 | tee -a "$diag"
wait_ready api.localhost
register='{"email":"rehearsal@example.test","password":"rehearsal correct horse","timezone":"Europe/Zurich","device":{"id":"'"$(uuid7)"'","platform":"ios"}}'
out=$(https api.localhost POST /v1/auth/register "$register" 2>"$work/register.json")
[[ "$out" == 201 ]] || die "register on production answered $out: $(cat "$work/register.json")"
token=$(sed -E 's/.*"access_token":"([^"]+)".*/\1/' "$work/register.json")
session='{"id":"'"$(uuid7)"'","started_at":"2026-10-18T07:00:00Z","timezone":"Europe/Zurich","title":"Rehearsal"}'
code=$(curl -sS -k --resolve api.localhost:443:127.0.0.1 -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
    --data "$session" https://api.localhost/v1/sessions)
[[ "$code" == 201 ]] || die "creating a session answered $code"

log "2. deploy staging beside it"
DEPLOY_ENV=staging DEPLOY_DIR="$work" "$work/scripts/deploy.sh" "$tag" 2>&1 | tee -a "$diag"
wait_ready api-staging.localhost
# Its own database: the address registered on production is free here.
out=$(https api-staging.localhost POST /v1/auth/register "$register" 2>/dev/null)
[[ "$out" == 201 ]] || die "staging shares production's users (register answered $out)"
[[ "$(https api.localhost GET /readyz 2>/dev/null)" == 200 ]] || die "production went down when staging deployed"

log "3. redeploy production"
DEPLOY_DIR="$work" "$work/scripts/deploy.sh" "$tag" 2>&1 | tee -a "$diag"
wait_ready api.localhost

log "4. the first backup"
for _ in $(seq 1 90); do
    prod logs backup 2>/dev/null | grep -q "uploaded hefesto-" && break
    sleep 2
done
prod logs backup | grep "uploaded hefesto-" || die "the backup sidecar never uploaded a dump"

log "5. restore into a scratch database"
(cd "$work" && DEPLOY_DIR="$work" ./scripts/restore.sh latest hefesto_rehearsal) 2>&1 | tee -a "$diag"
for table in users workout_sessions content_versions; do
    live=$(psql_prod hefesto "SELECT count(*) FROM $table")
    restored=$(psql_prod hefesto_rehearsal "SELECT count(*) FROM $table")
    [[ "$live" == "$restored" && "$live" -gt 0 ]] || die "$table: $live live, $restored restored"
    echo "$table: $live rows, restored $restored"
done

log "6. restore over the live database"
(cd "$work" && DEPLOY_DIR="$work" ./scripts/restore.sh latest --into-live --yes) 2>&1 | tee -a "$diag"
wait_ready api.localhost
login='{"email":"rehearsal@example.test","password":"rehearsal correct horse"}'
[[ "$(https api.localhost POST /v1/auth/login "$login" 2>/dev/null)" == 200 ]] \
    || die "the account is gone after restoring over the live database"

log "rehearsal passed: deploy, staging, redeploy, backup, two restores"
