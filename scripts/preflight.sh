#!/usr/bin/env bash
#
# Check the production host before its first deploy (docs/DEPLOYMENT.md,
# "First deploy"): Docker, both env files, DNS, ports 80 and 443, and every
# object-storage credential. It changes nothing on the host and prints no
# secret. Run it from your machine:
#
#   make preflight                  # host from `git config hefesto.prodhost`
#
# which runs this script on the server over SSH and passes it the keys the
# templates require. On the server itself, `bash scripts/preflight.sh` runs
# every check except the missing-key one.
#
# Exits 1 when anything would make the first deploy fail.

# `test && ok || bad` is safe here: ok and warn always succeed.
# shellcheck disable=SC2015

set -uo pipefail # not -e: report every problem, not only the first

DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
# Space-separated keys of .env.prod.example and .env.staging.example.
PROD_KEYS="${PROD_KEYS:-}"
STAGING_KEYS="${STAGING_KEYS:-}"
# The image the runtime uses, for the object-storage checks.
CHECK_IMAGE="${CHECK_IMAGE:-alpine:3.20}"

failed=0
ok()      { printf '  \033[32mok\033[0m    %s\n' "$*"; }
warn()    { printf '  \033[33mwarn\033[0m  %s\n' "$*"; }
bad()     { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; failed=1; }
section() { printf '\n\033[1m%s\033[0m\n' "$*"; }

# get <file> <key>: the value Compose reads, without surrounding quotes.
get() {
    local v
    v=$(sed -n "s/^$2=//p" "$1" | tail -1)
    v="${v%\"}"; v="${v#\"}"; v="${v%\'}"; v="${v#\'}"
    printf '%s' "$v"
}

cd "$DEPLOY_DIR" 2>/dev/null || { bad "$DEPLOY_DIR does not exist: run make server-bootstrap first"; exit 1; }
prod=.env.prod
staging=.env.staging

# ------------------------------------------------------------------ docker
section "Docker"
if ! command -v docker >/dev/null; then
    bad "docker is not installed: run make server-bootstrap"
elif ! docker info >/dev/null 2>&1; then
    bad "$(id -un) cannot reach the Docker daemon: is it in the docker group? (log in again after bootstrap)"
else
    ok "docker $(docker version --format '{{.Server.Version}}' 2>/dev/null)"
    if docker compose version >/dev/null 2>&1; then
        ok "$(docker compose version)"
    else
        bad "the docker compose plugin is missing"
    fi
fi

# --------------------------------------------------------------- env files
check_env() { # check_env <file> <required keys>
    local file=$1 keys=$2 mode blanks missing="" k
    if [[ ! -f "$file" ]]; then
        bad "$file is missing: copy $file.example from the repository and fill it in"
        return 1
    fi
    mode=$(stat -c %a "$file")
    [[ "$mode" == 600 ]] && ok "$file is mode 600" || bad "$file is mode $mode: chmod 600 $file"
    blanks=$(grep -oE '^[A-Z0-9_]+=\s*$' "$file" | tr -d '= ' | tr '\n' ' ')
    [[ -z "$blanks" ]] && ok "$file leaves no key blank" || bad "$file leaves these blank: $blanks"
    if [[ -n "$keys" ]]; then
        for k in $keys; do grep -q "^$k=" "$file" || missing+="$k "; done
        [[ -z "$missing" ]] && ok "$file has every key of its template" || bad "$file lacks: $missing"
    fi
    # The password goes into DATABASE_URL verbatim, where / + @ : and the
    # like would need percent-encoding; a hex one never does.
    local pw url user db
    pw=$(get "$file" POSTGRES_PASSWORD); url=$(get "$file" DATABASE_URL)
    user=$(get "$file" POSTGRES_USER); db=$(get "$file" POSTGRES_DB)
    if [[ -n "$pw" ]]; then
        [[ "$pw" =~ ^[A-Za-z0-9._~-]+$ ]] && ok "POSTGRES_PASSWORD is safe in a URL" \
            || bad "POSTGRES_PASSWORD has characters a URL cannot hold as they are: generate one with openssl rand -hex 32"
        [[ "$url" == "postgres://$user:$pw@postgres:5432/$db?"* || "$url" == "postgres://$user:$pw@postgres:5432/$db" ]] \
            && ok "DATABASE_URL matches POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_DB" \
            || bad "DATABASE_URL must be postgres://$user:<POSTGRES_PASSWORD>@postgres:5432/$db?sslmode=disable"
    fi
    local jwt
    jwt=$(get "$file" HEFESTO_JWT_SIGNING_KEY)
    (( ${#jwt} >= 32 )) && ok "HEFESTO_JWT_SIGNING_KEY is ${#jwt} characters" \
        || bad "HEFESTO_JWT_SIGNING_KEY is shorter than 32 characters: openssl rand -base64 48"
}

section "Production env ($prod)"
have_prod=false
check_env "$prod" "$PROD_KEYS" && have_prod=true
if $have_prod; then
    [[ "$(get "$prod" HEFESTO_ACME_EMAIL)" == *@*.* ]] && ok "HEFESTO_ACME_EMAIL looks like an address" \
        || bad "HEFESTO_ACME_EMAIL must be an address Let's Encrypt can write to"
    media=$(get "$prod" HEFESTO_S3_BUCKET) backup=$(get "$prod" HEFESTO_BACKUP_BUCKET) pitr=$(get "$prod" HEFESTO_PITR_S3_BUCKET)
    if [[ "$media" != "$backup" && "$media" != "$pitr" && "$backup" != "$pitr" ]]; then
        ok "media, backup and pitr buckets are three different buckets"
    else
        bad "media ($media), backup ($backup) and pitr ($pitr) must be three different buckets"
    fi
    [[ "$(get "$prod" HEFESTO_S3_ACCESS_KEY)" != "$(get "$prod" HEFESTO_BACKUP_ACCESS_KEY)" ]] \
        && ok "backups use a credential of their own" \
        || bad "HEFESTO_BACKUP_ACCESS_KEY is the media key: a leaked API key must not reach the backups"
    warn "keep a copy of HEFESTO_PITR_CIPHER_PASS off this server: without it the pgBackRest archive cannot be read"
fi

section "Staging env ($staging)"
have_staging=false
if [[ ! -f "$staging" ]]; then
    warn "$staging is missing: fine until you turn on STAGING_DEPLOY, and the staging domain answers 502 meanwhile"
else
    check_env "$staging" "$STAGING_KEYS" && have_staging=true
fi
if $have_prod && $have_staging; then
    for k in POSTGRES_PASSWORD HEFESTO_JWT_SIGNING_KEY HEFESTO_S3_BUCKET HEFESTO_S3_ACCESS_KEY; do
        [[ "$(get "$prod" "$k")" != "$(get "$staging" "$k")" ]] && ok "staging has its own $k" \
            || bad "staging reuses production's $k: a staging token or upload must never reach production"
    done
    [[ "$(get "$prod" HEFESTO_JWT_ISSUER)" != "$(get "$staging" HEFESTO_JWT_ISSUER)" ]] \
        && ok "the token issuers differ" || bad "HEFESTO_JWT_ISSUER must differ between production and staging"
fi

# --------------------------------------------------------------------- dns
section "DNS"
if $have_prod; then
    mine=" $(hostname -I 2>/dev/null) "
    for k in HEFESTO_API_DOMAIN HEFESTO_SITE_DOMAIN HEFESTO_SITE_DOMAIN_ALT HEFESTO_STAGING_API_DOMAIN; do
        name=$(get "$prod" "$k")
        [[ -n "$name" ]] || continue
        addrs=$(getent ahosts "$name" 2>/dev/null | awk '{print $1}' | sort -u | tr '\n' ' ')
        here=false
        for a in $addrs; do [[ "$mine" == *" $a "* ]] && here=true; done
        if [[ -z "$addrs" ]]; then
            [[ $k == HEFESTO_STAGING_API_DOMAIN ]] && warn "$name does not resolve yet (staging)" \
                || bad "$name does not resolve: Caddy cannot get its certificate"
        elif $here; then
            ok "$name → this host"
        else
            bad "$name → $addrs, not this host ($mine): Caddy cannot get its certificate"
        fi
    done
fi

# ------------------------------------------------------------------- ports
section "Ports"
for port in 80 443; do
    holder=$(ss -ltnH "sport = :$port" 2>/dev/null | head -1)
    container=$(docker ps --filter "publish=$port" --format '{{.Names}}' 2>/dev/null | head -1)
    if [[ -z "$holder" ]]; then
        ok "$port is free for Caddy"
    elif [[ -n "$container" ]]; then
        ok "$port is published by $container (an earlier deploy)"
    else
        bad "$port is taken by something outside Docker: sudo ss -ltnp 'sport = :$port'"
    fi
done

# ---------------------------------------------------------- object storage
# Each check lists the bucket, writes a probe, reads it back and deletes it,
# as the API and the sidecars will. Credentials go to the container through
# its environment, never on a command line.
s3_check() { # s3_check <label> <endpoint URL> <bucket> <access key> <secret key>
    local out
    if out=$(EP="$2" BUCKET="$3" AK="$4" SK="$5" docker run --rm -e EP -e BUCKET -e AK -e SK "$CHECK_IMAGE" sh -c '
        apk add --no-cache -q minio-client >/dev/null 2>&1 || { echo "could not install the MinIO client"; exit 1; }
        mcli alias set t "$EP" "$AK" "$SK" >/dev/null 2>&1 || { echo "the endpoint refused the credentials"; exit 1; }
        mcli ls "t/$BUCKET" >/dev/null 2>&1 || { echo "cannot list the bucket (missing, or no access)"; exit 1; }
        echo preflight | mcli pipe "t/$BUCKET/preflight-probe" >/dev/null 2>&1 || { echo "cannot write to the bucket"; exit 1; }
        mcli cat "t/$BUCKET/preflight-probe" >/dev/null 2>&1 || { echo "cannot read back what it wrote"; exit 1; }
        mcli rm "t/$BUCKET/preflight-probe" >/dev/null 2>&1 || { echo "cannot delete from the bucket"; exit 1; }
    ' 2>&1); then
        ok "$1: list, write, read and delete in $3"
    else
        bad "$1 ($3): ${out:-the check could not run}"
    fi
}
public_check() { # public_check <endpoint URL> <bucket> <path style true|false>
    local url code
    if [[ "$3" == true ]]; then url="${1%/}/$2/"; else url="${1/:\/\//://$2.}"; url="${url%/}/"; fi
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$url" || true)
    case "$code" in
        200) bad "$2 answers anonymous listing: the media bucket must be private" ;;
        000) warn "could not reach $url to check that $2 is private" ;;
        *) ok "$2 refuses anonymous listing ($code)" ;;
    esac
}

section "Object storage"
if ! docker info >/dev/null 2>&1; then
    warn "skipped: Docker is not reachable"
else
    if $have_prod; then
        s3_check "media" "$(get "$prod" HEFESTO_S3_ENDPOINT)" "$(get "$prod" HEFESTO_S3_BUCKET)" \
            "$(get "$prod" HEFESTO_S3_ACCESS_KEY)" "$(get "$prod" HEFESTO_S3_SECRET_KEY)"
        public_check "$(get "$prod" HEFESTO_S3_ENDPOINT)" "$(get "$prod" HEFESTO_S3_BUCKET)" \
            "$(get "$prod" HEFESTO_S3_USE_PATH_STYLE)"
        s3_check "backups" "$(get "$prod" HEFESTO_BACKUP_ENDPOINT)" "$(get "$prod" HEFESTO_BACKUP_BUCKET)" \
            "$(get "$prod" HEFESTO_BACKUP_ACCESS_KEY)" "$(get "$prod" HEFESTO_BACKUP_SECRET_KEY)"
        s3_check "pitr" "https://$(get "$prod" HEFESTO_PITR_S3_ENDPOINT):$(get "$prod" HEFESTO_PITR_S3_PORT)" \
            "$(get "$prod" HEFESTO_PITR_S3_BUCKET)" \
            "$(get "$prod" HEFESTO_PITR_S3_KEY)" "$(get "$prod" HEFESTO_PITR_S3_KEY_SECRET)"
    fi
    if $have_staging; then
        s3_check "staging media" "$(get "$staging" HEFESTO_S3_ENDPOINT)" "$(get "$staging" HEFESTO_S3_BUCKET)" \
            "$(get "$staging" HEFESTO_S3_ACCESS_KEY)" "$(get "$staging" HEFESTO_S3_SECRET_KEY)"
    fi
fi

echo
if (( failed )); then
    printf '\033[31mNot ready:\033[0m fix every FAIL above, then run this again.\n'
    exit 1
fi
printf '\033[32mReady for the first deploy.\033[0m\n'
