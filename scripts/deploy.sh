#!/usr/bin/env bash
#
# Deploy one image tag on the production host, to production or to staging.
#
# Runs ON THE SERVER, from the deploy directory (default /srv/hefesto). The
# GitHub Actions workflows invoke it over SSH; you can also run it by hand.
#
#   ./scripts/deploy.sh v1.4.2                                # production
#   DEPLOY_ENV=staging ./scripts/deploy.sh sha-<commit>       # staging
#
# Staging (ADR 0026) is its own Compose project on the same host, with its own
# database and .env.staging. It has no Caddy and no backup sidecar; production's
# Caddy serves it over the shared hefesto-edge network.
#
# What it does, in order:
#   1. records the currently running tag so a rollback has somewhere to go
#   2. pulls the new image
#   3. runs migrations to completion and aborts if they fail
#   4. imports the content baked into the image (a no-op if unchanged)
#   5. recreates the API and reloads Caddy
#   6. polls /readyz until healthy, and rolls back if it never gets there
#
# Migrations run BEFORE the new API starts, which means every migration must be
# backward compatible with the currently running version for the length of one
# deploy. That is a constraint on how migrations are written, not a bug here:
# expand, deploy, contract.

set -Eeuo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
DEPLOY_ENV="${DEPLOY_ENV:-production}"
case "$DEPLOY_ENV" in
    production)
        COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
        ENV_FILE="${ENV_FILE:-.env.prod}"
        STATE_FILE="${STATE_FILE:-.deployed-tag}"
        PULL_SERVICES=(api migrate seed backup)
        BUILD_SERVICES=(postgres pitr)
        UP_SERVICES=(api caddy backup pitr)
        ;;
    staging)
        COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.staging.yml}"
        ENV_FILE="${ENV_FILE:-.env.staging}"
        STATE_FILE="${STATE_FILE:-.deployed-tag-staging}"
        PULL_SERVICES=(api migrate seed)
        BUILD_SERVICES=()
        UP_SERVICES=(api)
        ;;
    *)
        printf 'DEPLOY_ENV must be production or staging, not %s\n' "$DEPLOY_ENV" >&2
        exit 1
        ;;
esac
EDGE_NETWORK="hefesto-edge"
HEALTH_RETRIES="${HEALTH_RETRIES:-30}"
HEALTH_INTERVAL_S="${HEALTH_INTERVAL_S:-2}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

[[ $# -eq 1 ]] || die "usage: $0 <image-tag>   (e.g. $0 v1.4.2)"
NEW_TAG="$1"

cd "$DEPLOY_DIR" || die "deploy directory $DEPLOY_DIR does not exist"
[[ -f "$ENV_FILE" ]] || die "$DEPLOY_DIR/$ENV_FILE is missing — see docs/DEPLOYMENT.md"
[[ -f "$COMPOSE_FILE" ]] || die "$DEPLOY_DIR/$COMPOSE_FILE is missing"

compose() {
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" "$@"
}

PREVIOUS_TAG=""
if [[ -f "$STATE_FILE" ]]; then
    PREVIOUS_TAG="$(cat "$STATE_FILE")"
fi
log "$DEPLOY_ENV: current tag: ${PREVIOUS_TAG:-<none>}  →  new tag: $NEW_TAG"

export HEFESTO_TAG="$NEW_TAG"

# ---------------------------------------------------------------- pull
log "pulling image"
compose pull --quiet "${PULL_SERVICES[@]}" \
    || die "could not pull ${HEFESTO_IMAGE:-image}:$NEW_TAG — is the tag published and is the registry login still valid?"

# ------------------------------------------------------------ network
# Production's Caddy and the staging API meet on this network. Whichever
# environment is deployed first creates it.
if ! docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1; then
    log "creating network $EDGE_NETWORK"
    docker network create "$EDGE_NETWORK" >/dev/null
fi

# --------------------------------------------------------------- build
# Production's Postgres image is postgres:16-alpine plus pgBackRest, built
# here (docker/Dockerfile.postgres). Unchanged, the build is cached and the
# image id stays the same, so the database is not restarted.
if (( ${#BUILD_SERVICES[@]} )); then
    log "building ${BUILD_SERVICES[*]}"
    compose build --quiet "${BUILD_SERVICES[@]}"
fi

# ---------------------------------------------------------- database up
log "ensuring database is up"
compose up -d postgres
# shellcheck disable=SC2016  # $POSTGRES_* must expand inside the container, not here
compose exec -T postgres sh -c 'until pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do sleep 1; done' >/dev/null

# ------------------------------------------------------------ migrate
log "running migrations"
if ! compose run --rm migrate; then
    die "migrations failed — the running version was not touched"
fi

# ---------------------------------------------------------------- seed
# Content ships inside the image. The import is one transaction and a no-op
# when nothing changed; the running API only ever sees the old tree or the new.
log "importing content"
if ! compose run --rm seed -dir /srv/content -applied-by "deploy:$NEW_TAG"; then
    die "content import failed — the running version was not touched (migrations from $NEW_TAG are applied)"
fi

# ------------------------------------------------------------- release
log "starting api"
compose up -d --remove-orphans "${UP_SERVICES[@]}"

# -------------------------------------------------------------- verify
log "waiting for readiness (max $((HEALTH_RETRIES * HEALTH_INTERVAL_S))s)"
healthy=false
for _ in $(seq 1 "$HEALTH_RETRIES"); do
    if compose exec -T api /usr/local/bin/api -healthcheck >/dev/null 2>&1; then
        healthy=true
        break
    fi
    sleep "$HEALTH_INTERVAL_S"
done

if [[ "$healthy" != true ]]; then
    warn "new version never became healthy"
    compose logs --tail 80 api >&2 || true
    if [[ -n "$PREVIOUS_TAG" ]]; then
        warn "rolling back to $PREVIOUS_TAG"
        export HEFESTO_TAG="$PREVIOUS_TAG"
        compose up -d api
        die "rolled back to $PREVIOUS_TAG — deploy of $NEW_TAG failed. NOTE: migrations from $NEW_TAG have already been applied."
    fi
    die "deploy of $NEW_TAG failed and there is no previous tag to roll back to"
fi

echo "$NEW_TAG" > "$STATE_FILE"
log "deployed $NEW_TAG to $DEPLOY_ENV"

# ------------------------------------------------------------- cleanup
log "pruning images older than 7 days"
docker image prune --force --filter "until=168h" >/dev/null || warn "image prune failed (not fatal)"

compose ps
