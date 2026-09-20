#!/usr/bin/env bash
set -Eeuo pipefail

# Exergism Commons — deployment-attestation handoff for id.exergism.org
#
# This is a one-time/re-runnable host handoff from the initial id resolver
# bootstrap to the reviewed deployment-attestation release pinned below.
# Routine resolver updates after a successful handoff are owned by the agent
# through the id repository's rolling runtime-main release.
#
# Run on the production Droplet as root:
#   sudo /path/to/deploy/install-deployment-attestation.sh
#
# Do not replace the pinned release, source commit, or SHA-256 values without a
# reviewed repository change.

DA_VERSION="v0.1.1"
DA_COMMIT="80c0fa06b8fac71332f0a2fdd5c409e666d91955"

DA_MANIFEST_SHA256="5fc66e50b542d0d9c2cf4ffb704f201b1ba58ce91fd5cb005684cae84535495d"
DA_INSTALLER_SHA256="14b70c91f6cff93943e4d03dc33bb9c41ab42195e1ab7ff922a0c677f3c8dd2f"
DA_AGENT_SHA256="7fbb723a87cc677721b6fdffd78465f0f31d9b6a83784ac1a5d8d7b58b42c5cf"
DA_SUMS_SHA256="8918b7b016c0fdf58c2c35fe0219d0b0093793dab7d46849efcae9433d8f5309"

DA_REPOSITORY="Exergism-Commons/deployment-attestation"
DA_BASE_URL="https://github.com/${DA_REPOSITORY}/releases/download/${DA_VERSION}"

SERVICE="id.exergism.org"
TARGET_UNIT="id-exergism.service"
AGENT_RUN_UNIT="ec-deployment-attestation@${SERVICE}.service"
AGENT_TIMER_UNIT="ec-deployment-attestation@${SERVICE}.timer"

APP_DIR="/srv/id.exergism.org"
APP_BIN="/usr/local/bin/idresolver"
ENV_FILE="/etc/ec-deployment-attestation/${SERVICE}.env"
AGENT="/usr/local/libexec/ec-deployment-agent"
SMOKE="/usr/local/libexec/id.exergism.org-smoke.sh"
ARTIFACT_FENCE_AUDITOR="/usr/local/libexec/ec-id-production-artifact-fence"
ATTESTATION_DOC_DIR="/usr/local/share/doc/ec-deployment-attestation"

log()  { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
die()  { printf '\n\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

WORKDIR=""
STAGE=""
cleanup() {
  local rc=$?
  trap - EXIT
  [[ -n "$STAGE" ]] && rm -rf -- "$STAGE" || true
  [[ -n "$WORKDIR" ]] && rm -rf -- "$WORKDIR" || true
  exit "$rc"
}
trap cleanup EXIT

[[ "${EUID}" -eq 0 ]] || die "Run this script as root (or with sudo)."
[[ "$(uname -m)" == "x86_64" ]] || die "deployment-attestation ${DA_VERSION} is currently pinned to linux-x64 on this host."

[[ -r /etc/os-release ]] || die "/etc/os-release not found."
# shellcheck disable=SC1091
source /etc/os-release
case "${ID:-}" in
  debian|ubuntu) ;;
  *) die "Unsupported host OS: ${ID:-unknown}. Expected Debian/Ubuntu." ;;
esac

export DEBIAN_FRONTEND=noninteractive

log "Installing deployment-attestation host prerequisites"
apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  git \
  jq \
  python3 \
  util-linux

log "Checking existing id.exergism.org production baseline"
systemctl cat "$TARGET_UNIT" >/dev/null || die "$TARGET_UNIT is not installed; run the base Droplet bootstrap first."
[[ -d "$APP_DIR" && ! -L "$APP_DIR" ]] || die "$APP_DIR is missing or not a real directory."
[[ -x "$APP_BIN" && ! -L "$APP_BIN" ]] || die "$APP_BIN is missing, non-executable, or a symlink."
systemctl is-active --quiet "$TARGET_UNIT" || die "$TARGET_UNIT is not active before handoff."
curl -q -fsS --max-time 15 http://127.0.0.1:8080/ >/dev/null \
  || die "Local resolver baseline is unhealthy."
curl -q -fsS --max-time 15 https://id.exergism.org/ >/dev/null \
  || die "Public resolver baseline is unhealthy."

WORKDIR="$(mktemp -d /var/tmp/id-deployment-attestation.XXXXXX)"
chmod 0700 "$WORKDIR"
SOURCE_ROOT="$WORKDIR/source"
RELEASE_DIR="$WORKDIR/release"
mkdir -m 0700 "$RELEASE_DIR"

download_release_asset() {
  local name="$1"
  curl -q \
    --retry 8 \
    --retry-all-errors \
    --retry-delay 3 \
    --connect-timeout 10 \
    -fsSL \
    "${DA_BASE_URL}/${name}" \
    -o "${RELEASE_DIR}/${name}"
}

log "Downloading deployment-attestation ${DA_VERSION} release metadata and binaries"
download_release_asset DEPLOYMENT_MANIFEST.json
download_release_asset SHA256SUMS
download_release_asset install-id-exergism.sh
download_release_asset ec-deployment-agent-linux-x64

log "Verifying pinned GitHub release asset digests"
printf '%s  %s\n' "$DA_MANIFEST_SHA256" "$RELEASE_DIR/DEPLOYMENT_MANIFEST.json" | sha256sum -c -
printf '%s  %s\n' "$DA_SUMS_SHA256" "$RELEASE_DIR/SHA256SUMS" | sha256sum -c -
printf '%s  %s\n' "$DA_INSTALLER_SHA256" "$RELEASE_DIR/install-id-exergism.sh" | sha256sum -c -
printf '%s  %s\n' "$DA_AGENT_SHA256" "$RELEASE_DIR/ec-deployment-agent-linux-x64" | sha256sum -c -

(
  cd "$RELEASE_DIR"
  sha256sum -c SHA256SUMS
)

jq -e \
  --arg repo "$DA_REPOSITORY" \
  --arg tag "$DA_VERSION" \
  --arg commit "$DA_COMMIT" \
  --arg agent "$DA_AGENT_SHA256" \
  --arg installer "$DA_INSTALLER_SHA256" \
  '.schema_version == "0.1"
   and .repository == $repo
   and .release_tag == $tag
   and .source_commit == $commit
   and .assets.amd64.name == "ec-deployment-agent-linux-x64"
   and .assets.amd64.sha256 == $agent
   and .assets.installer.name == "install-id-exergism.sh"
   and .assets.installer.sha256 == $installer' \
  "$RELEASE_DIR/DEPLOYMENT_MANIFEST.json" >/dev/null \
  || die "deployment-attestation release manifest does not match the reviewed handoff pins."

chmod 0700 "$RELEASE_DIR/ec-deployment-agent-linux-x64"

log "Fetching exact reviewed deployment-attestation source"
git clone \
  --quiet \
  --branch "$DA_VERSION" \
  --depth 1 \
  "https://github.com/${DA_REPOSITORY}.git" \
  "$SOURCE_ROOT"

SOURCE_COMMIT="$(git -C "$SOURCE_ROOT" rev-parse HEAD)"
[[ "$SOURCE_COMMIT" == "$DA_COMMIT" ]] \
  || die "Reviewed tag resolved to $SOURCE_COMMIT, expected $DA_COMMIT."

log "Creating root-owned trusted installer stage"
STAGE="$(mktemp -d /run/ec-deployment-attestation-installer.XXXXXX)"
chmod 0700 "$STAGE"
chown root:root "$STAGE"
install -o root -g root -m 0500 \
  "$RELEASE_DIR/install-id-exergism.sh" \
  "$STAGE/install-id-exergism.sh"

printf '%s  %s\n' "$DA_INSTALLER_SHA256" "$STAGE/install-id-exergism.sh" | sha256sum -c -

log "Installing deployment-attestation ${DA_VERSION}"
env -i \
  PATH=/usr/sbin:/usr/bin:/sbin:/bin \
  EC_INSTALLER_TRUSTED_STAGE="$STAGE" \
  EC_INSTALLER_SOURCE_ROOT="$SOURCE_ROOT" \
  EC_INSTALLER_SHA256="$DA_INSTALLER_SHA256" \
  EC_NATIVE_AGENT_BINARY="$RELEASE_DIR/ec-deployment-agent-linux-x64" \
  EC_NATIVE_AGENT_SHA256="$DA_AGENT_SHA256" \
  "$STAGE/install-id-exergism.sh"

# The installer owns and normally removes its trusted stage after success.
STAGE=""

log "Validating installed production configuration"
[[ -f "$ENV_FILE" && ! -L "$ENV_FILE" ]] || die "Installed environment file is missing or unsafe: $ENV_FILE"
[[ "$(stat -c '%u' -- "$ENV_FILE")" == "0" ]] || die "$ENV_FILE is not root-owned."
env_mode_raw="$(stat -c '%a' -- "$ENV_FILE")"
env_mode=$((8#$env_mode_raw))
(( (env_mode & 0037) == 0 )) \
  || die "$ENV_FILE permissions are too broad: mode=$env_mode_raw (group-write/execute and all other access are forbidden)."

required_config=(
  "EC_SERVICE=id.exergism.org"
  "EC_REPOSITORY=Exergism-Commons/id"
  "EC_ENVIRONMENT=production"
  "EC_RELEASE_TAG=runtime-main"
  "EC_RELEASE_MANIFEST=DEPLOYMENT_MANIFEST.json"
  "EC_APP_DIR=/srv/id.exergism.org"
  "EC_APP_BIN=/usr/local/bin/idresolver"
  "EC_SERVICE_UNIT=id-exergism"
  "EC_LOCAL_URL=http://127.0.0.1:8080/"
  "EC_PUBLIC_URL=https://id.exergism.org/"
  "EC_CHECK_PUBLIC=1"
)

for expected in "${required_config[@]}"; do
  grep -Fxq "$expected" "$ENV_FILE" \
    || die "Installed deployment-attestation configuration is missing: $expected"
done

[[ -x "$AGENT" && ! -L "$AGENT" ]] || die "Installed deployment agent is missing or unsafe."
[[ -x "$SMOKE" && ! -L "$SMOKE" ]] || die "Installed smoke script is missing or unsafe."
[[ -x "$ARTIFACT_FENCE_AUDITOR" && ! -L "$ARTIFACT_FENCE_AUDITOR" ]] \
  || die "Installed artifact-fence auditor is missing or unsafe."

log "Running first managed runtime-main cycle"
systemctl start "$AGENT_RUN_UNIT"
[[ "$(systemctl show "$AGENT_RUN_UNIT" --property=Result --value)" == "success" ]] \
  || die "First deployment-attestation cycle did not complete successfully."

log "Validating resolver, smoke checks, artifact fence and agent self-health"
systemctl is-active --quiet "$TARGET_UNIT" || die "$TARGET_UNIT is not active after handoff."
systemctl is-enabled --quiet "$AGENT_TIMER_UNIT" || die "$AGENT_TIMER_UNIT is not enabled."
systemctl is-active --quiet "$AGENT_TIMER_UNIT" || die "$AGENT_TIMER_UNIT is not active."

curl -q -fsS --max-time 15 http://127.0.0.1:8080/ >/dev/null
curl -q -fsS --max-time 15 https://id.exergism.org/ >/dev/null
EC_LOCAL_URL=http://127.0.0.1:8080 "$SMOKE"
"$ARTIFACT_FENCE_AUDITOR"

EC_ATTESTATION_CONFIG="$ENV_FILE" "$AGENT" validate-config
EC_ATTESTATION_CONFIG="$ENV_FILE" "$AGENT" health
EC_ATTESTATION_CONFIG="$ENV_FILE" "$AGENT" status

source_revision="$(cat /usr/local/share/doc/idresolver/source-revision.txt)"
[[ "$source_revision" =~ ^[0-9a-f]{40}$ ]] \
  || die "Deployed idresolver source revision is malformed: $source_revision"

binary_sha256="$(sha256sum "$APP_BIN" | awk '{print $1}')"
[[ "$binary_sha256" =~ ^[0-9a-f]{64}$ ]] \
  || die "Deployed idresolver binary digest is malformed."

log "Recording reviewed deployment-attestation handoff"
install -d -o root -g root -m 0755 "$ATTESTATION_DOC_DIR"
printf '%s\n' "$DA_VERSION" > "$ATTESTATION_DOC_DIR/release-tag.txt"
printf '%s\n' "$DA_COMMIT" > "$ATTESTATION_DOC_DIR/source-revision.txt"
printf '%s\n' "$DA_MANIFEST_SHA256" > "$ATTESTATION_DOC_DIR/release-manifest-sha256.txt"
printf '%s\n' "$DA_INSTALLER_SHA256" > "$ATTESTATION_DOC_DIR/installer-sha256.txt"
printf '%s\n' "$DA_AGENT_SHA256" > "$ATTESTATION_DOC_DIR/agent-sha256.txt"
chmod 0644 "$ATTESTATION_DOC_DIR"/*.txt

printf '\n\033[1;32mDeployment-attestation handoff complete.\033[0m\n\n'
printf 'Agent release:     %s\n' "$DA_VERSION"
printf 'Agent source:      %s\n' "$DA_COMMIT"
printf 'Managed channel:   runtime-main\n'
printf 'Resolver source:   %s\n' "$source_revision"
printf 'Resolver SHA-256:  %s\n' "$binary_sha256"
printf 'Timer:             %s\n' "$AGENT_TIMER_UNIT"
printf '\nRoutine id.exergism.org deployments are now owned by deployment-attestation.\n'
printf 'Do not use setup-digitalocean.sh, manual git pull, or manual binary replacement for normal updates.\n'
