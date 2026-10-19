#!/bin/sh
#
# Base backups for point-in-time recovery (ADR 0026), as a long-lived
# sidecar beside Postgres. Postgres itself archives every WAL segment
# (archive_command = pgbackrest archive-push); this takes the base backups
# that WAL is replayed onto: a full backup once a week, a differential every
# day. pgBackRest expires old ones itself (PGBACKREST_REPO1_RETENTION_FULL).
#
# Runs as the postgres user, with the data directory mounted read-only and
# Postgres's socket directory shared, so the backup reads the files directly
# and talks to the server over the socket. All repository settings come from
# the PGBACKREST_* environment.

set -eu

INTERVAL_S="${HEFESTO_PITR_INTERVAL_S:-86400}"
FULL_EVERY_S="${HEFESTO_PITR_FULL_EVERY_S:-604800}"

log() { printf '%s pitr: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

until pg_isready -q -h /var/run/postgresql; do
    log "waiting for postgres"
    sleep 2
done

# Creating the stanza is safe to repeat: it checks an existing one instead.
log "stanza ${PGBACKREST_STANZA}"
pgbackrest stanza-create
pgbackrest check
log "archiving works"

last_full() { # epoch at which the newest full backup finished, 0 if none
    pgbackrest info --output=json \
        | jq '[.[0].backup[]? | select(.type == "full") | .timestamp.stop] | max // 0'
}

while :; do
    now=$(date -u +%s)
    full=$(last_full)
    kind="diff"
    if [ $((now - full)) -ge "$FULL_EVERY_S" ]; then
        kind="full"
    fi
    # A broken archive (a rotated key, a missing bucket) shows here daily,
    # not only when a recovery needs the WAL.
    if ! pgbackrest check; then
        log "ERROR archiving is failing: WAL is not reaching the repository"
    fi
    log "taking a $kind backup"
    if pgbackrest backup --type="$kind"; then
        log "$kind backup done"
        pgbackrest info | sed 's/^/    /'
    else
        log "ERROR $kind backup failed"
    fi
    log "sleeping ${INTERVAL_S}s"
    sleep "$INTERVAL_S"
done
