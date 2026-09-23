#!/bin/sh
#
# Nightly logical backup of the production database to S3-compatible object
# storage (Hetzner Object Storage in production).
#
# Runs as a long-lived sidecar rather than a host cron job so that backups are
# part of the deployed unit and cannot be silently missing on a rebuilt host.
#
# This is deliberately a LOGICAL backup (pg_dump), which gives a consistent
# snapshot per run and nothing between runs. Point-in-time recovery via WAL
# archiving (pgBackRest) is a Phase 7 item; until this project has real user
# data, a nightly dump with a documented, rehearsed restore is the honest
# trade. See docs/DEPLOYMENT.md.

set -eu

: "${PGHOST:?}" "${PGUSER:?}" "${PGDATABASE:?}"
: "${HEFESTO_BACKUP_BUCKET:?}" "${HEFESTO_BACKUP_ENDPOINT:?}"
: "${HEFESTO_BACKUP_ACCESS_KEY:?}" "${HEFESTO_BACKUP_SECRET_KEY:?}"
INTERVAL_S="${HEFESTO_BACKUP_INTERVAL_S:-86400}"
RETAIN_DAYS="${HEFESTO_BACKUP_RETAIN_DAYS:-30}"

log() { printf '%s backup: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# The runtime image is Alpine and carries neither pg_dump nor mc; install once
# at container start rather than baking a backup toolchain into the API image.
if ! command -v pg_dump >/dev/null 2>&1; then
    log "installing postgresql-client and minio client"
    apk add --no-cache postgresql16-client minio-client >/dev/null
fi

mc alias set backup "$HEFESTO_BACKUP_ENDPOINT" \
    "$HEFESTO_BACKUP_ACCESS_KEY" "$HEFESTO_BACKUP_SECRET_KEY" >/dev/null
mc mb --ignore-existing "backup/$HEFESTO_BACKUP_BUCKET" >/dev/null

while :; do
    stamp="$(date -u +%Y%m%dT%H%M%SZ)"
    file="/tmp/hefesto-${stamp}.dump"

    log "dumping $PGDATABASE"
    if pg_dump --format=custom --compress=9 --no-owner --no-privileges \
               --file="$file" "$PGDATABASE"; then
        size="$(wc -c < "$file")"
        # A dump that is suspiciously small means pg_dump succeeded against an
        # empty or wrong database. Better to shout than to archive garbage.
        if [ "$size" -lt 4096 ]; then
            log "ERROR dump is only ${size} bytes — not uploading"
        elif mc cp "$file" "backup/$HEFESTO_BACKUP_BUCKET/daily/hefesto-${stamp}.dump" >/dev/null; then
            log "uploaded hefesto-${stamp}.dump (${size} bytes)"
        else
            log "ERROR upload failed"
        fi
    else
        log "ERROR pg_dump failed"
    fi
    rm -f "$file"

    log "expiring backups older than ${RETAIN_DAYS} days"
    mc rm --recursive --force --older-than "${RETAIN_DAYS}d" \
        "backup/$HEFESTO_BACKUP_BUCKET/daily/" >/dev/null 2>&1 || \
        log "retention sweep failed (not fatal)"

    log "sleeping ${INTERVAL_S}s"
    sleep "$INTERVAL_S"
done
