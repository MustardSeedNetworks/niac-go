#!/usr/bin/env bash
# Deployment validation: install a released NIAC package on a host and prove the
# service that comes up is the artifact the release published.
#
# The pre-1.0 build contract requires this after every install. It asserts the
# three things a build can silently get wrong:
#
#   1. /__version reports the version that was installed — not a stale service
#      that never restarted.
#   2. uiBuildHash is non-empty — the UI really is embedded. A binary built
#      outside the make pipeline (`go build`, `make quick`) reports "" here and
#      serves no web interface, and nothing else catches it.
#   3. Installing over an existing configuration does not crash-loop the
#      service. A fresh install exercises none of the upgrade path; the second
#      install runs against the certs, database and config the first one left.
#
# The assertions are made ON the host over ssh, against the loopback listener,
# so a closed firewall is not mistaken for a broken deployment.
#
# Usage:
#   scripts/lab/deploy-validate.sh --host <ssh-target> [--version vX.Y.Z]
#                                  [--port 8445] [--package <local file>]
#
#   --version   release tag to install (default: the latest GitHub release)
#   --package   install this local .deb/.rpm instead of downloading a release;
#               --version then names the version it is expected to report
#
# Requires: ssh access with passwordless sudo on the host, and `gh` locally
# unless --package is given.
set -euo pipefail

PORT=8445
HOST=""
VERSION=""
PACKAGE=""
SETTLE_SECONDS="${SETTLE_SECONDS:-30}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"

die() {
	printf '\033[31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
pass() { printf '\033[32mPASS\033[0m %s\n' "$1"; }

while [[ $# -gt 0 ]]; do
	case "$1" in
	--host) HOST="${2:-}" && shift 2 ;;
	--version) VERSION="${2:-}" && shift 2 ;;
	--port) PORT="${2:-}" && shift 2 ;;
	--package) PACKAGE="${2:-}" && shift 2 ;;
	-h | --help)
		sed -n '2,29p' "$0"
		exit 0
		;;
	*) die "unknown argument: $1" ;;
	esac
done

[[ -n "$HOST" ]] || die "--host is required (an ssh target, e.g. dev-srv-ubuntu)"

on_host() { ssh -o BatchMode=yes "$HOST" "$@"; }

# The version string the daemon reports carries no leading "v"; the release tag
# does. Compare the same shape at both ends.
strip_v() { printf '%s' "${1#v}"; }

resolve_version() {
	[[ -n "$VERSION" ]] && return 0
	command -v gh >/dev/null 2>&1 || die "gh is required to resolve the latest release (or pass --version)"
	VERSION="$(gh release view --json tagName --jq .tagName)"
	[[ -n "$VERSION" ]] || die "could not resolve the latest release tag"
}

# deb or rpm, decided by what the host actually has rather than by its name.
host_package_format() {
	if on_host 'command -v dpkg' >/dev/null 2>&1; then
		printf 'deb'
	elif on_host 'command -v rpm' >/dev/null 2>&1; then
		printf 'rpm'
	else
		die "host $HOST has neither dpkg nor rpm"
	fi
}

# The release asset names are set by .goreleaser.yml's nfpms file_name_template.
asset_pattern() {
	local format="$1" arch="$2" version
	version="$(strip_v "$VERSION")"
	case "$format" in
	deb) printf 'niac_%s_%s.deb' "$version" "$arch" ;;
	rpm)
		case "$arch" in
		amd64) arch=x86_64 ;;
		arm64) arch=aarch64 ;;
		esac
		printf 'niac-%s-1.%s.rpm' "$version" "$arch"
		;;
	esac
}

host_arch() {
	case "$(on_host uname -m)" in
	x86_64) printf 'amd64' ;;
	aarch64 | arm64) printf 'arm64' ;;
	*) die "unsupported host architecture" ;;
	esac
}

# The second pass must actually re-run the package's scripts. Both package
# managers treat "install the version that is already installed" as a no-op and
# exit 0, which would make the upgrade assertion below pass without an upgrade
# having happened — so ask for a reinstall explicitly.
install_package() {
	local remote="$1" format="$2" mode="${3:-install}"
	case "$format:$mode" in
	deb:install) on_host "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y '$remote'" ;;
	deb:reinstall) on_host "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --reinstall '$remote'" ;;
	rpm:install) on_host "sudo dnf install -y --allowerasing '$remote'" ;;
	rpm:reinstall) on_host "sudo dnf reinstall -y '$remote'" ;;
	esac
}

# systemd's own restart counter is the crash-loop evidence: a service that keeps
# dying and being restarted by Restart=on-failure increments it.
restart_count() { on_host 'systemctl show niac.service -p NRestarts --value'; }

fetch_version_json() {
	on_host "curl -sk --max-time 10 https://127.0.0.1:${PORT}/__version"
}

wait_healthy() {
	local deadline=$((SECONDS + HEALTH_TIMEOUT))
	while ((SECONDS < deadline)); do
		if fetch_version_json | grep -q '"version"'; then
			return 0
		fi
		sleep 3
	done
	on_host 'sudo journalctl -u niac.service -n 40 --no-pager' >&2 || true
	die "/__version did not answer on ${HOST}:${PORT} within ${HEALTH_TIMEOUT}s"
}

# Both assertions read one response, so they describe the same instant.
assert_version_payload() {
	local json expected
	json="$(fetch_version_json)"
	expected="$(strip_v "$VERSION")"
	printf '%s\n' "$json"
	printf '%s' "$json" | python3 -c '
import json, sys
expected = sys.argv[1]
doc = json.load(sys.stdin)
reported = doc.get("version", "")
if reported.lstrip("v") != expected:
    sys.exit(f"/__version reports {reported!r}, expected {expected!r}")
if not doc.get("uiBuildHash"):
    sys.exit("/__version reports an empty uiBuildHash: the UI is not embedded")
' "$expected"
}

resolve_version
step "Validating NIAC $VERSION on $HOST"

FORMAT="$(host_package_format)"
ARCH="$(host_arch)"

if [[ -n "$PACKAGE" ]]; then
	[[ -f "$PACKAGE" ]] || die "no such package: $PACKAGE"
	LOCAL_PACKAGE="$PACKAGE"
else
	command -v gh >/dev/null 2>&1 || die "gh is required to download the release asset"
	DOWNLOAD_DIR="$(mktemp -d)"
	trap 'rm -rf "$DOWNLOAD_DIR"' EXIT
	PATTERN="$(asset_pattern "$FORMAT" "$ARCH")"
	step "Downloading $PATTERN from $VERSION"
	gh release download "$VERSION" --pattern "$PATTERN" --dir "$DOWNLOAD_DIR"
	LOCAL_PACKAGE="$DOWNLOAD_DIR/$PATTERN"
fi

REMOTE_PACKAGE="/tmp/$(basename "$LOCAL_PACKAGE")"
scp -o BatchMode=yes -q "$LOCAL_PACKAGE" "$HOST:$REMOTE_PACKAGE"

step "Install 1 of 2 (this host may or may not already carry a configuration)"
install_package "$REMOTE_PACKAGE" "$FORMAT"
wait_healthy
assert_version_payload
pass "/__version reports $(strip_v "$VERSION") with a non-empty uiBuildHash"

# Everything the first install left behind — /etc/niac, the database, the
# self-signed certificate — is now the "existing configuration" the second
# install has to survive.
step "Install 2 of 2, over the configuration the first one created"
BEFORE_RESTARTS="$(restart_count)"
install_package "$REMOTE_PACKAGE" "$FORMAT" reinstall
wait_healthy
assert_version_payload
pass "/__version still correct after installing over an existing configuration"

step "Watching for a restart loop for ${SETTLE_SECONDS}s"
sleep "$SETTLE_SECONDS"
AFTER_RESTARTS="$(restart_count)"
on_host 'systemctl is-active --quiet niac.service' ||
	die "niac.service is not active after the upgrade"
[[ "$BEFORE_RESTARTS" == "$AFTER_RESTARTS" ]] ||
	die "niac.service restarted during the upgrade window (NRestarts $BEFORE_RESTARTS -> $AFTER_RESTARTS): crash loop"
pass "niac.service active, NRestarts unchanged at $AFTER_RESTARTS"

printf '\n\033[32mdeploy-validate: %s on %s\033[0m\n' "$VERSION" "$HOST"
