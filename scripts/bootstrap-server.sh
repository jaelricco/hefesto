#!/usr/bin/env bash
#
# Prepare a fresh Hetzner Cloud instance to receive deploys.
#
# Run ONCE, as root, on a clean Debian 12 or Ubuntu 24.04 box:
#   ssh root@<ip> 'bash -s' < scripts/bootstrap-server.sh
#
# Afterwards, upload .env.prod, docker-compose.prod.yml, docker/Caddyfile and
# scripts/ to /srv/hefesto (the deploy workflow does this for you), then run
# scripts/deploy.sh <tag>.
#
# Deliberately not done here: creating .env.prod. Secrets are never generated
# by a script that also has network access and shell history.

set -Eeuo pipefail

DEPLOY_USER="${DEPLOY_USER:-deploy}"
DEPLOY_DIR="${DEPLOY_DIR:-/srv/hefesto}"
SSH_PUBKEY="${SSH_PUBKEY:-}"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31mxx\033[0m %s\n' "$*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "run as root"

log "installing packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg ufw fail2ban unattended-upgrades

log "installing docker engine"
# shellcheck source=/dev/null  # /etc/os-release exists on the target host, not here
if ! command -v docker >/dev/null; then
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/"$(. /etc/os-release && echo "$ID")"/gpg \
        -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
https://download.docker.com/linux/$(. /etc/os-release && echo "$ID") \
$(. /etc/os-release && echo "$VERSION_CODENAME") stable" > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi
systemctl enable --now docker

log "creating $DEPLOY_USER"
if ! id -u "$DEPLOY_USER" >/dev/null 2>&1; then
    adduser --disabled-password --gecos "" "$DEPLOY_USER"
fi
usermod -aG docker "$DEPLOY_USER"

if [[ -n "$SSH_PUBKEY" ]]; then
    log "installing deploy public key"
    install -d -m 0700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh"
    echo "$SSH_PUBKEY" >> "/home/$DEPLOY_USER/.ssh/authorized_keys"
    chown "$DEPLOY_USER:$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh/authorized_keys"
    chmod 0600 "/home/$DEPLOY_USER/.ssh/authorized_keys"
else
    log "SSH_PUBKEY not set — add the CI deploy key to /home/$DEPLOY_USER/.ssh/authorized_keys yourself"
fi

log "creating $DEPLOY_DIR"
install -d -m 0750 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$DEPLOY_DIR"

log "hardening sshd"
sed -i \
    -e 's/^#*PermitRootLogin.*/PermitRootLogin no/' \
    -e 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' \
    -e 's/^#*KbdInteractiveAuthentication.*/KbdInteractiveAuthentication no/' \
    /etc/ssh/sshd_config
systemctl reload ssh 2>/dev/null || systemctl reload sshd

log "configuring firewall"
ufw --force reset >/dev/null
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    comment 'ssh'
ufw allow 80/tcp    comment 'http (acme + redirect)'
ufw allow 443/tcp   comment 'https'
ufw allow 443/udp   comment 'http/3'
ufw --force enable

log "enabling unattended security upgrades"
dpkg-reconfigure -f noninteractive unattended-upgrades

cat <<EOF

Bootstrap complete.

Remaining manual steps:
  1. Point DNS at this host:
       api.hefesto.fit   A/AAAA -> this IP
       hefesto.fit       A/AAAA -> this IP
       hefesto.ch        A/AAAA -> this IP
  2. Create $DEPLOY_DIR/.env.prod from .env.prod.example, with real secrets.
     chmod 600 it and chown it to $DEPLOY_USER.
  3. Log the deploy user into the registry once:
       sudo -u $DEPLOY_USER docker login ghcr.io -u <github-user> --password-stdin
  4. Push a tag, or run: gh workflow run deploy.yml -f tag=<tag>

Nothing on this host generates or stores a secret for you. That is on purpose.
EOF
