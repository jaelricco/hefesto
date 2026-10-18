#!/usr/bin/env bash
#
# docker compose for a deployed environment: its Compose file, its env file,
# and the image tag that is live.
#
# Runs ON THE SERVER, from the deploy directory (default /srv/hefesto), like
# deploy.sh.
#
#   ./scripts/compose.sh ps
#   ./scripts/compose.sh logs -f --tail 200 api
#   ./scripts/compose.sh exec postgres psql -U hefesto -d hefesto
#   DEPLOY_ENV=staging ./scripts/compose.sh logs -f api
#
# The Compose files require HEFESTO_TAG, so a bare `docker compose ps` fails.
# deploy.sh records the tag it deployed in .deployed-tag
# (.deployed-tag-staging); this reads it back.

set -Eeuo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
case "${DEPLOY_ENV:-production}" in
    production) file=docker-compose.prod.yml env=.env.prod state=.deployed-tag ;;
    staging) file=docker-compose.staging.yml env=.env.staging state=.deployed-tag-staging ;;
    *)
        printf 'DEPLOY_ENV must be production or staging, not %s\n' "$DEPLOY_ENV" >&2
        exit 1
        ;;
esac

cd "$DEPLOY_DIR"
if [[ -z "${HEFESTO_TAG:-}" ]]; then
    [[ -f "$state" ]] || { printf 'xx nothing is deployed: %s/%s is missing\n' "$DEPLOY_DIR" "$state" >&2; exit 1; }
    HEFESTO_TAG="$(cat "$state")"
fi
export HEFESTO_TAG
exec docker compose -f "$file" --env-file "$env" "$@"
