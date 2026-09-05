#!/usr/bin/env bash
set -Eeuo pipefail

# Exergism Commons — id.exergism.org DigitalOcean Droplet bootstrap
#
# Target: fresh Ubuntu 26.04 LTS or Debian 13 Droplet, run as root.
# Other currently supported Debian/Ubuntu releases may work, but production
# deployments should use a currently supported stable/LTS image.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Exergism-Commons/id/main/deploy/setup-digitalocean.sh | sudo bash
#
# Optional environment variables:
#   REPO_URL=https://github.com/Exergism-Commons/id.git
#   REPO_REF=main
#   DOMAIN=id.exergism.org
#   APP_USER=idexergism
#   APP_DIR=/srv/id.exergism.org
#   APP_BIN=/usr/local/bin/idresolver
#   LISTEN_ADDR=127.0.0.1:8080
#   ENABLE_UFW=1
#
# DNS is intentionally NOT changed by this script. Point id.exergism.org to the
# Droplet only after the local resolver and Caddy configuration are healthy.

REPO_URL="${REPO_URL:-https://github.com/Exergism-Commons/id.git}"
REPO_REF="${REPO_REF:-main}"
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

resolve_stable_go() {
  local metadata version major minor
  metadata="$(curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL 'https://go.dev/dl/?mode=json')"
  version="$(jq -r '[.[] | select(.stable == true)][0].version // empty' <<<"$metadata")"
  [[ "$version" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || die "Could not determine a valid stable Go version from go.dev."

  major="${version#go}"
  minor="${major#*.}"
  major="${major%%.*}"
  minor="${minor%%.*}"
  if (( major < 1 || (major == 1 && minor < 27) )); then
    die "Resolved Go version ${version} is older than the repository minimum Go 1.27."
  fi

  printf '%s\n' "$version"
}

install_go() {
  local version="$1" arch go_arch tarball checksum metadata tmpdir target primary_url fallback_url
  arch="$(dpkg --print-architecture)"
  case "$arch" in
    amd64) go_arch="amd64" ;;
    arm64) go_arch="arm64" ;;
    *) die "Unsupported CPU architecture for Go bootstrap: $arch" ;;
  esac

  tarball="${version}.linux-${go_arch}.tar.gz"
  metadata="$(curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL 'https://go.dev/dl/?mode=json&include=all')"
  checksum="$(jq -r --arg version "$version" --arg filename "$tarball" '
    [.[] | select(.version == $version) | .files[] | select(.filename == $filename)][0].sha256 // empty
  ' <<<"$metadata")"
  [[ "$checksum" =~ ^[0-9a-f]{64}$ ]] || die "Could not resolve the official SHA-256 for ${tarball}."

  tmpdir="$(mktemp -d)"
  target="${tmpdir}/${tarball}"
  primary_url="https://go.dev/dl/${tarball}"
  fallback_url="https://dl.google.com/go/${tarball}"

  if ! curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL "$primary_url" -o "$target"; then
    warn "Go download failed via ${primary_url}; retrying with the official download mirror."
    if ! curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL "$fallback_url" -o "$target"; then
      rm -rf "$tmpdir"
      die "Could not download ${tarball} from either official Go download endpoint."
    fi
  fi

  printf '%s  %s\n' "$checksum" "$target" | sha256sum -c - >/dev/null

  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$target"
  rm -rf "$tmpdir"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
}

stable_go="$(resolve_stable_go)"
if command -v go >/dev/null 2>&1; then
  current_go="$(go env GOVERSION 2>/dev/null || true)"
else
  current_go=""
fi

if [[ "$current_go" == "$stable_go" ]]; then
  log "Using current stable Go toolchain: $current_go"
else
  if [[ -n "$current_go" ]]; then
    log "Replacing Go ${current_go} with current stable ${stable_go}"
  else
    log "Installing current stable Go toolchain: ${stable_go}"
  fi
  install_go "$stable_go"
fi

go version

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

log "Fetching Exergism identifier service"
if [[ -d "${APP_DIR}/.git" ]]; then
  git -C "$APP_DIR" fetch --prune origin
  git -C "$APP_DIR" checkout "$REPO_REF"
  git -C "$APP_DIR" reset --hard "origin/${REPO_REF}"
else
  if [[ -e "$APP_DIR" && -n "$(find "$APP_DIR" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    die "$APP_DIR exists and is not an empty Git repository. Refusing to overwrite it."
  fi
  rm -rf "$APP_DIR"
  git clone --branch "$REPO_REF" --depth 1 "$REPO_URL" "$APP_DIR"
fi
chown -R root:root "$APP_DIR"
chmod -R a+rX "$APP_DIR"

log "Testing and building resolver"
cd "$APP_DIR"
go test ./...
go vet ./...
go build -trimpath -ldflags='-s -w' -o /tmp/idresolver ./cmd/idresolver
install -o root -g root -m 0755 /tmp/idresolver "$APP_BIN"
rm -f /tmp/idresolver

log "Installing redistribution and license notices"
install -d -o root -g root -m 0755 "$DOC_DIR"
for notice in EULA.md THIRD_PARTY_NOTICES.md LICENSES/Go.txt LICENSES/Apache-2.0.txt; do
  if [[ -f "${APP_DIR}/${notice}" ]]; then
    install -o root -g root -m 0644 "${APP_DIR}/${notice}" "${DOC_DIR}/$(basename "$notice")"
  fi
done
git -C "$APP_DIR" rev-parse HEAD > "${DOC_DIR}/source-revision.txt"
chmod 0644 "${DOC_DIR}/source-revision.txt"

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

printf '\n\033[1;32mSetup complete.\033[0m\n\n'
printf 'Resolver:  http://%s (loopback only)\n' "$LISTEN_ADDR"
printf 'Domain:    https://%s\n' "$DOMAIN"
printf 'Go:        %s\n' "$(go env GOVERSION)"
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
