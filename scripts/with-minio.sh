#!/usr/bin/env bash
#
# Run a command with a real MinIO server for the integration tests.
#
#   ./scripts/with-minio.sh go test -tags=integration ./...
#
# MinIO no longer publishes container images on Docker Hub, so the server is
# built from its pinned source with the Go toolchain (about two minutes the
# first time, cached in ./bin after that) and started on localhost with
# throwaway credentials. The command sees HEFESTO_TEST_S3_ENDPOINT and its
# keys, and the server stops when the command ends.
#
# If HEFESTO_TEST_S3_ENDPOINT is already set, the command runs against that
# server instead and nothing is started.

set -Eeuo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <command> [args...]" >&2
  exit 2
fi

if [[ -n "${HEFESTO_TEST_S3_ENDPOINT:-}" ]]; then
  exec "$@"
fi

# A pinned commit of the official source. MinIO's community edition is no
# longer published as images or binaries.
MINIO_MODULE="github.com/minio/minio@v0.0.0-20260212201848-7aac2a2c5b7c"
PORT="${HEFESTO_TEST_MINIO_PORT:-9123}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin="$root/bin/minio-${MINIO_MODULE##*-}" # the commit, so a new pin rebuilds
if [[ ! -x "$bin" ]]; then
  echo "building MinIO from source ($MINIO_MODULE); this happens once" >&2
  tmpbin="$(mktemp -d)"
  GOBIN="$tmpbin" go install "$MINIO_MODULE"
  mkdir -p "$root/bin"
  mv "$tmpbin/minio" "$bin"
  rmdir "$tmpbin"
fi

data="$(mktemp -d)"
access="test-$(od -An -N8 -tx1 /dev/urandom | tr -d ' \n')"
secret="$(od -An -N24 -tx1 /dev/urandom | tr -d ' \n')"

MINIO_ROOT_USER="$access" MINIO_ROOT_PASSWORD="$secret" MINIO_BROWSER=off \
  "$bin" server "$data" --address "127.0.0.1:$PORT" --quiet >"$data.log" 2>&1 &
pid=$!
cleanup() {
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  rm -rf "$data" "$data.log"
}
trap cleanup EXIT

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:$PORT/minio/health/live" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "MinIO did not start:" >&2
    cat "$data.log" >&2
    exit 1
  fi
  sleep 0.5
done

export HEFESTO_TEST_S3_ENDPOINT="http://127.0.0.1:$PORT"
export HEFESTO_TEST_S3_ACCESS_KEY="$access"
export HEFESTO_TEST_S3_SECRET_KEY="$secret"
"$@"
