#!/usr/bin/env bash
set -Eeuo pipefail

# Exergism Commons — id.exergism.org DigitalOcean Droplet bootstrap
#
# Target: fresh Ubuntu/Debian Droplet, run as root.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id/main/deploy/setup-digitalocean.sh | sudo bash
#
# Optional environment variables:
#   REPO_URL=https://github.com/Exergism-Commons/id.git
#   RELEASE_TAG=runtime-main
#   DOMAIN=id.exergism.org
#   APP_USER=idexergism
#   APP_DIR=/srv/id.exergism.org
#   APP_BIN=/usr/local/bin/idresolver
#   LISTEN_ADDR=127.0.0.1:8080
#   ENABLE_UFW=1
#
# Runtime binaries are built by GitHub Actions and published as GitHub Release
# assets. This host does not need a Go toolchain and does not compile source.
# DNS is intentionally NOT changed by this script. Point id.exergism.org to the
# Droplet only after the local resolver and Caddy configuration are healthy.

REPO_URL="${REPO_URL:-https://github.com/Exergism-Commons/id.git}"
RELEASE_TAG="${RELEASE_TAG:-runtime-main}"
DOMAIN="${DOMAIN:-id.exergism.org}"
APP_USER="${APP_USER:-idexergism}"
APP_DIR="${APP_DIR:-/srv/id.exergism.org}"
APP_BIN="${APP_BIN:-/usr/local/bin/idresolver}"
LISTEN_ADDR="${LISTEN_ADDR:-127.0.0.1:8080}"
ENABLE_UFW="${ENABLE_UFW:-1}"
SERVICE_NAME="id-exergism"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CADDY_FILE="/etc/caddy/Caddyfile"
DOC_DIR="/usr/local/share/doc/idresolver"
RELEASE_BASE="https://github.com/Exergism-Commons/id/releases/download/${RELEASE_TAG}"

log()  { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\n\033[1;33mWARN:\033[0m %s\n' "$*" >&2; }
die()  { printf '\n\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

trap 'die "Setup failed at line $LINENO. Inspect the output above before re-running."' ERR

if [[ "${EUID}" -ne 0 ]]; then
  die "Run this script as root (or with sudo)."
fi

if [[ ! -r /etc/os-release ]]; then
  die "Unsupported system: /etc/os-release not found."
fi

# shellcheck disable=SC1091
source /etc/os-release
case "${ID:-}" in
  debian|ubuntu) ;;
  *) die "This bootstrap supports Debian/Ubuntu only (detected: ${ID:-unknown})." ;;
esac

[[ "$RELEASE_TAG" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ ]] \
  || die "Invalid RELEASE_TAG: ${RELEASE_TAG}"

export DEBIAN_FRONTEND=noninteractive

log "Updating operating-system packages"
apt-get update
apt-get upgrade -y

log "Installing base packages"
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  debian-keyring \
  debian-archive-keyring \
  apt-transport-https \
  git \
  gnupg \
  jq \
  ufw

arch="$(dpkg --print-architecture)"
case "$arch" in
  amd64|arm64) ;;
  *) die "Unsupported CPU architecture for idresolver release assets: $arch" ;;
esac
asset="idresolver-linux-${arch}"

download_release_asset() {
  local name="$1" target="$2"
  curl \
    --retry 8 \
    --retry-all-errors \
    --retry-delay 3 \
    --connect-timeout 10 \
    -fsSL \
    "${RELEASE_BASE}/${name}" \
    -o "$target"
}

log "Downloading prebuilt idresolver release ${RELEASE_TAG}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

download_release_asset SOURCE_COMMIT "${tmpdir}/SOURCE_COMMIT"
download_release_asset SHA256SUMS "${tmpdir}/SHA256SUMS"
download_release_asset "$asset" "${tmpdir}/${asset}"

source_commit="$(tr -d '\r\n' < "${tmpdir}/SOURCE_COMMIT")"
[[ "$source_commit" =~ ^[0-9a-f]{40}$ ]] \
  || die "Release SOURCE_COMMIT is not a valid Git commit SHA."

expected_checksum="$(awk -v asset="$asset" '$2 == asset {print $1}' "${tmpdir}/SHA256SUMS")"
[[ "$expected_checksum" =~ ^[0-9a-f]{64}$ ]] \
  || die "Could not find a valid SHA-256 for ${asset} in the release checksum file."
actual_checksum="$(sha256sum "${tmpdir}/${asset}" | awk '{print $1}')"
[[ "$actual_checksum" == "$expected_checksum" ]] \
  || die "SHA-256 verification failed for ${asset}."
chmod 0755 "${tmpdir}/${asset}"

log "Installing latest Caddy stable from the official repository"
install -d -m 0755 /usr/share/keyrings
curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
  | gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
  -o /etc/apt/sources.list.d/caddy-stable.list
apt-get update
apt-get install -y caddy
caddy version

log "Creating service account"
if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd \
    --system \
    --user-group \
    --home-dir "$APP_DIR" \
    --shell /usr/sbin/nologin \
    "$APP_USER"
fi

log "Fetching release-matched repository state"
if [[ -d "${APP_DIR}/.git" ]]; then
  git -C "$APP_DIR" remote set-url origin "$REPO_URL"
else
  if [[ -e "$APP_DIR" && -n "$(find "$APP_DIR" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    die "$APP_DIR exists and is not an empty Git repository. Refusing to overwrite it."
  fi
  rm -rf "$APP_DIR"
  git init "$APP_DIR"
  git -C "$APP_DIR" remote add origin "$REPO_URL"
fi

git -C "$APP_DIR" fetch --force --depth 1 origin \
  "refs/tags/${RELEASE_TAG}:refs/tags/${RELEASE_TAG}"
git -C "$APP_DIR" checkout --detach "refs/tags/${RELEASE_TAG}"
git -C "$APP_DIR" reset --hard "refs/tags/${RELEASE_TAG}"
git -C "$APP_DIR" clean -fdx

checked_out_commit="$(git -C "$APP_DIR" rev-parse HEAD)"
[[ "$checked_out_commit" == "$source_commit" ]] \
  || die "Release binary source ${source_commit} does not match checked-out repository ${checked_out_commit}."

chown -R root:root "$APP_DIR"
chmod -R a+rX "$APP_DIR"

log "Installing verified resolver binary"
install -o root -g root -m 0755 "${tmpdir}/${asset}" "$APP_BIN"
"$APP_BIN" -h >/dev/null 2>&1 || true

log "Installing redistribution and license notices"
install -d -o root -g root -m 0755 "$DOC_DIR"
for notice in EULA.md THIRD_PARTY_NOTICES.md LICENSES/Go.txt LICENSES/Apache-2.0.txt; do
  if [[ -f "${APP_DIR}/${notice}" ]]; then
    install -o root -g root -m 0644 "${APP_DIR}/${notice}" "${DOC_DIR}/$(basename "$notice")"
  fi
done
printf '%s\n' "$source_commit" > "${DOC_DIR}/source-revision.txt"
printf '%s\n' "$RELEASE_TAG" > "${DOC_DIR}/release-tag.txt"
printf '%s  %s\n' "$expected_checksum" "$asset" > "${DOC_DIR}/binary-sha256.txt"
chmod 0644 "${DOC_DIR}/source-revision.txt" "${DOC_DIR}/release-tag.txt" "${DOC_DIR}/binary-sha256.txt"

log "Installing systemd service"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Exergism Commons persistent identifier resolver
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${APP_USER}
Group=${APP_USER}
WorkingDirectory=${APP_DIR}
ExecStart=${APP_BIN} -listen ${LISTEN_ADDR} -root ${APP_DIR} -registry ${APP_DIR}/resolver/registry.json
Restart=on-failure
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadOnlyPaths=${APP_DIR}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME"

log "Waiting for local resolver"
for _ in {1..30}; do
  if curl -fsS -H 'Accept: text/html' "http://${LISTEN_ADDR}/" >/dev/null; then
    break
  fi
  sleep 1
done
curl -fsS -H 'Accept: text/html' "http://${LISTEN_ADDR}/" >/dev/null \
  || die "Resolver did not become healthy on http://${LISTEN_ADDR}/"

log "Installing Caddy reverse proxy"
cat > "$CADDY_FILE" <<EOF
${DOMAIN} {
    encode zstd gzip
    reverse_proxy ${LISTEN_ADDR}
}
EOF

caddy validate --config "$CADDY_FILE"
systemctl enable caddy
systemctl restart caddy

if [[ "$ENABLE_UFW" == "1" ]]; then
  log "Configuring firewall"
  ufw allow OpenSSH
  ufw allow 80/tcp
  ufw allow 443/tcp
  ufw --force enable
fi

log "Running local negotiation smoke tests"
curl -fsSI -H 'Accept: text/html' "http://${LISTEN_ADDR}/funding" >/dev/null
curl -fsSI -H 'Accept: text/turtle' "http://${LISTEN_ADDR}/ontology/funding" >/dev/null

PUBLIC_IP="$(curl -4 -fsS --max-time 5 https://api.ipify.org || true)"

rm -rf "$tmpdir"
trap - EXIT

printf '\n\033[1;32mSetup complete.\033[0m\n\n'
printf 'Release:   %s\n' "$RELEASE_TAG"
printf 'Source:    %s\n' "$source_commit"
printf 'Binary:    %s\n' "$asset"
printf 'Resolver:  http://%s (loopback only)\n' "$LISTEN_ADDR"
printf 'Domain:    https://%s\n' "$DOMAIN"
printf 'Caddy:     %s\n' "$(caddy version | head -n1)"
printf 'Service:   systemctl status %s\n' "$SERVICE_NAME"
printf 'Logs:      journalctl -u %s -f\n' "$SERVICE_NAME"
printf 'Caddy:     systemctl status caddy\n'
printf 'Notices:   %s\n' "$DOC_DIR"
if [[ -n "$PUBLIC_IP" ]]; then
  printf '\nNext DNS step in Spaceship:\n'
  printf '  A    id    %s    TTL 300\n' "$PUBLIC_IP"
else
  printf '\nNext DNS step: point %s to this Droplet public IPv4 address.\n' "$DOMAIN"
fi
printf '\nDo not remove the existing GitHub Pages DNS target until this server is healthy.\n'
printf 'After DNS propagates, verify:\n'
printf "  curl -i -H 'Accept: text/html' https://%s/funding\n" "$DOMAIN"
printf "  curl -i -H 'Accept: text/turtle' https://%s/ontology/funding\n" "$DOMAIN"
