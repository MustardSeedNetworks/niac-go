#!/usr/bin/env bash
# Deployment validation: install a released NIAC package on a host and prove the
# service that comes up is the artifact the release published.
#
# The pre-1.0 build contract requires this after every install. It asserts the
# four things a build can silently get wrong:
#
#   1. /__version reports the version that was installed — not a stale service
#      that never restarted.
#   2. uiBuildHash is non-empty — the UI really is embedded. A binary built
#      outside the make pipeline (`go build`, `make quick`) reports "" here and
#      serves no web interface, and nothing else catches it.
#   3. Upgrading over an existing configuration does not crash-loop the
#      service. The first install is the PREVIOUS release, so the second one
#      crosses a version boundary and runs against the certs, database, config
#      and recovery state the older build left — an upgrade, not a reinstall of
#      the same bytes.
#   4. The daemon can still start a simulation afterwards, and the one that was
#      running before the upgrade survived it. Clauses 1-3 all passed on CT304
#      while #2092 left the simulator unable to start anything: state written
#      under an older recovery schema stayed on disk and turned every start
#      into a generic 500. A deployment that answers /__version and cannot
#      simulate is not a working deployment, and only a real start says so —
#      reading recovery's own verdict cannot, because a host where nothing was
#      running reports a clean one either way.
#
# The simulations run on throwaway dummy interfaces created for the run and
# deleted afterwards, so nothing this script starts reaches the host's network.
#
# The assertions are made ON the host over ssh, against the loopback listener,
# so a closed firewall is not mistaken for a broken deployment.
#
# Usage:
#   scripts/lab/deploy-validate.sh --host <ssh-target> [--version vX.Y.Z]
#                                  [--from-version vX.Y.Z|none] [--port 8445]
#                                  [--package <local file>]
#
#   --version   release tag to install (default: the latest GitHub release)
#   --from-version
#               release installed first, so the second install is a real
#               version-to-version upgrade (default: the release immediately
#               before --version; "none" forces a same-version reinstall)
#   --package   install this local .deb/.rpm instead of downloading a release;
#               --version then names the version it is expected to report
#
# Requires: ssh access with passwordless sudo on the host, and `gh` locally
# unless --package is given.
set -euo pipefail

PORT=8445
HOST=""
VERSION=""
FROM_VERSION=""
PACKAGE=""
SETTLE_SECONDS="${SETTLE_SECONDS:-30}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"

die() {
	printf '\033[31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
pass() { printf '\033[32mPASS\033[0m %s\n' "$1"; }
skip() { printf '\033[33mSKIP\033[0m %s\n' "$1"; }

while [[ $# -gt 0 ]]; do
	case "$1" in
	--host) HOST="${2:-}" && shift 2 ;;
	--version) VERSION="${2:-}" && shift 2 ;;
	--port) PORT="${2:-}" && shift 2 ;;
	--package) PACKAGE="${2:-}" && shift 2 ;;
	--from-version) FROM_VERSION="${2:-}" && shift 2 ;;
	-h | --help)
		# The whole leading comment block, however long it grows: a fixed
		# line range silently truncated the usage the moment a flag was added.
		awk 'NR > 1 && /^#/ { print; next } NR > 1 { exit }' "$0"
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

# The release before --version, so the first install is genuinely an older
# build and the second one is an upgrade across a version boundary rather than
# a reinstall of the same bytes. That is what carries recovery state written by
# one schema into a daemon that may read another (#2092), which a same-version
# reinstall cannot exercise. Falls back to the same version -- the old
# behaviour -- when there is no earlier release to install.
resolve_from_version() {
	[[ -n "$FROM_VERSION" ]] && return 0
	[[ -n "$PACKAGE" ]] && FROM_VERSION=none && return 0
	command -v gh >/dev/null 2>&1 || { FROM_VERSION=none && return 0; }
	FROM_VERSION="$(gh release list --limit 30 --json tagName --jq '[.[].tagName]' |
		python3 -c '
import json, sys

target = sys.argv[1]
tags = json.load(sys.stdin)
print(tags[tags.index(target) + 1] if target in tags and tags.index(target) + 1 < len(tags) else "none")
' "$VERSION")"
	[[ -n "$FROM_VERSION" ]] || FROM_VERSION=none
}

# deb or rpm, decided by the tool that INSTALLS rather than the one that queries:
# dev-srv-fedora carries /usr/bin/dpkg with no apt-get behind it, and asking for
# dpkg there picked the .deb and then failed at "sudo: apt-get: command not found".
host_package_format() {
	if on_host 'command -v apt-get' >/dev/null 2>&1; then
		printf 'deb'
	elif on_host 'command -v dnf' >/dev/null 2>&1 || on_host 'command -v yum' >/dev/null 2>&1; then
		printf 'rpm'
	else
		die "host $HOST has neither apt-get nor dnf/yum"
	fi
}

# The release asset names are set by .goreleaser.yml's nfpms file_name_template.
asset_pattern() {
	local format="$1" arch="$2" version
	version="$(strip_v "${3:-$VERSION}")"
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
	# The baseline install puts the PREVIOUS release on, and the host may
	# already be carrying a newer one from an earlier run — which both package
	# managers refuse by default rather than silently going backwards.
	deb:baseline) on_host "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --allow-downgrades '$remote'" ;;
	rpm:install) on_host "sudo \$(command -v dnf || command -v yum) install -y --allowerasing '$remote'" ;;
	rpm:reinstall) on_host "sudo \$(command -v dnf || command -v yum) reinstall -y '$remote'" ;;
	rpm:baseline) on_host "sudo \$(command -v dnf || command -v yum) install -y --allowerasing '$remote' ||
		sudo \$(command -v dnf || command -v yum) downgrade -y '$remote'" ;;
	esac
}

# systemd's own restart counter is the crash-loop evidence: a service that keeps
# dying and being restarted by Restart=on-failure increments it.
restart_count() { on_host 'systemctl show niac.service -p NRestarts --value'; }

# Everything the simulation assertions create, removed on any exit path so a
# failed run leaves no session holding an interface and no dummy link behind.
PROBE_INTERFACES=()
PROBE_SESSIONS=()

cleanup() {
	release_probes
	[[ -n "${DOWNLOAD_DIR:-}" ]] && rm -rf "$DOWNLOAD_DIR"
	return 0
}

release_probes() {
	local session iface
	for session in ${PROBE_SESSIONS+"${PROBE_SESSIONS[@]}"}; do
		on_host "niac simulation stop '$session' --insecure" >/dev/null 2>&1 || true
	done
	for iface in ${PROBE_INTERFACES+"${PROBE_INTERFACES[@]}"}; do
		on_host "sudo ip link delete '$iface'" >/dev/null 2>&1 || true
	done
}

# A dummy link is a real interface to libpcap, attached to nothing. Starting a
# pack on the host's own NIC would put a fleet of invented devices onto the lab
# network, which is not something a validation run may do.
probe_interface() {
	local iface="$1"
	on_host "sudo ip link delete '$iface' 2>/dev/null; sudo ip link add '$iface' type dummy && sudo ip link set '$iface' up" ||
		die "could not create the dummy interface $iface on $HOST (is the dummy module available?)"
	PROBE_INTERFACES+=("$iface")
}

# The first scenario the daemon itself offers, so this script never needs to
# know where the library lives.
library_scenario() {
	on_host "curl -sk --max-time 10 https://127.0.0.1:${PORT}/api/v1/library/networks" |
		python3 -c '
import json, sys

entries = json.load(sys.stdin)
if not entries:
    sys.exit("the daemon offers no library scenario to start")
print(entries[0]["name"])
'
}

# Start by name, which is the only spelling an operator has: where the library
# sits is the daemon's own business (#2124).
#
# A release predating that fix refuses the name outright. On the BASELINE
# install that is a property of the old build rather than a deployment failure,
# so it returns non-zero with a named reason and the caller decides; on the
# release under test it is fatal, because a release that cannot start a
# scenario an operator can name is not shippable.
start_simulation() {
	local iface="$1" session="$2" scenario="$3" out
	if out="$(on_host "niac simulation start -i '$iface' --config '$scenario.yaml' --session '$session' --insecure" 2>&1)"; then
		PROBE_SESSIONS+=("$session")
		return 0
	fi
	printf '%s\n' "$out" >&2
	case "$out" in
	*"NIAC-managed storage"*) return 2 ;;
	esac
	return 1
}

assert_sessions_running() {
	on_host "curl -sk --max-time 10 https://127.0.0.1:${PORT}/api/v1/sessions" |
		python3 -c '
import json, sys

want = set(sys.argv[1:])
running = {session["sessionId"] for session in json.load(sys.stdin)}
missing = sorted(want - running)
if missing:
    sys.exit(f"sessions {missing} are not running; the daemon reports {sorted(running)}")
' "$@"
}

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
	expected="$(strip_v "${1:-$VERSION}")"
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
resolve_from_version
step "Validating NIAC $VERSION on $HOST"

FORMAT="$(host_package_format)"
ARCH="$(host_arch)"
trap cleanup EXIT

# Stage a release's package on the host and echo its remote path.
stage_release() {
	local version="$1" pattern local_package
	pattern="$(asset_pattern "$FORMAT" "$ARCH" "$version")"
	step "Downloading $pattern from $version" >&2
	gh release download "$version" --pattern "$pattern" --dir "$DOWNLOAD_DIR" >&2
	local_package="$DOWNLOAD_DIR/$pattern"
	scp -o BatchMode=yes -q "$local_package" "$HOST:/tmp/$pattern"
	printf '/tmp/%s' "$pattern"
}

if [[ -n "$PACKAGE" ]]; then
	[[ -f "$PACKAGE" ]] || die "no such package: $PACKAGE"
	REMOTE_PACKAGE="/tmp/$(basename "$PACKAGE")"
	scp -o BatchMode=yes -q "$PACKAGE" "$HOST:$REMOTE_PACKAGE"
	REMOTE_FROM_PACKAGE="$REMOTE_PACKAGE"
else
	command -v gh >/dev/null 2>&1 || die "gh is required to download the release asset"
	DOWNLOAD_DIR="$(mktemp -d)"
	REMOTE_PACKAGE="$(stage_release "$VERSION")"
	if [[ "$FROM_VERSION" == "none" ]]; then
		REMOTE_FROM_PACKAGE="$REMOTE_PACKAGE"
	else
		REMOTE_FROM_PACKAGE="$(stage_release "$FROM_VERSION")"
	fi
fi

if [[ "$REMOTE_FROM_PACKAGE" == "$REMOTE_PACKAGE" ]]; then
	INSTALLED_FIRST="$VERSION"
	UPGRADE_MODE=reinstall
else
	INSTALLED_FIRST="$FROM_VERSION"
	UPGRADE_MODE=install
fi

step "Install 1 of 2: $INSTALLED_FIRST (this host may or may not already carry a configuration)"
install_package "$REMOTE_FROM_PACKAGE" "$FORMAT" baseline
wait_healthy
assert_version_payload "$INSTALLED_FIRST"
pass "/__version reports $(strip_v "$INSTALLED_FIRST") with a non-empty uiBuildHash"

# A running simulation is the rest of the "existing configuration": it writes
# recovery state, and that state is what the upgrade has to carry forward.
step "Start a simulation on $INSTALLED_FIRST, so the upgrade runs against recovery state"
SCENARIO="$(library_scenario)"
probe_interface niac-dv-pre
CARRIES_STATE=yes
start_simulation niac-dv-pre deploy-validate-pre "$SCENARIO" || case $? in
2)
	CARRIES_STATE=no
	skip "$INSTALLED_FIRST predates #2124 and cannot start a scenario named without a path, so this run upgrades an install carrying no recovery state"
	;;
*) die "starting $SCENARIO on niac-dv-pre failed" ;;
esac
if [[ "$CARRIES_STATE" == "yes" ]]; then
	assert_sessions_running deploy-validate-pre
	pass "$SCENARIO runs as deploy-validate-pre on niac-dv-pre"
fi

# Everything the first install left behind — /etc/niac, the database, the
# self-signed certificate, and now the recovery state of a running session — is
# the "existing configuration" the second install has to survive.
step "Install 2 of 2: $VERSION, over the configuration the first one created"
BEFORE_RESTARTS="$(restart_count)"
install_package "$REMOTE_PACKAGE" "$FORMAT" "$UPGRADE_MODE"
wait_healthy
assert_version_payload "$VERSION"
pass "/__version reports $(strip_v "$VERSION") after installing over an existing configuration"

step "Watching for a restart loop for ${SETTLE_SECONDS}s"
sleep "$SETTLE_SECONDS"
AFTER_RESTARTS="$(restart_count)"
on_host 'systemctl is-active --quiet niac.service' ||
	die "niac.service is not active after the upgrade"
[[ "$BEFORE_RESTARTS" == "$AFTER_RESTARTS" ]] ||
	die "niac.service restarted during the upgrade window (NRestarts $BEFORE_RESTARTS -> $AFTER_RESTARTS): crash loop"
pass "niac.service active, NRestarts unchanged at $AFTER_RESTARTS"

step "Checking recovery left nothing that blocks a simulation start"
# recovery is a field on the simulation status, not its own route.
SIM_STATE="$(on_host "curl -sk --max-time 10 https://127.0.0.1:${PORT}/api/v1/simulation" 2>/dev/null || true)"
RECOVERY_VERDICT="$(printf '%s' "$SIM_STATE" | python3 -c '
import json, sys

raw = sys.stdin.read().strip()
if not raw:
    print("UNREADABLE no simulation status")
    raise SystemExit
try:
    status = json.loads(raw)
except json.JSONDecodeError as err:
    print(f"UNREADABLE {err}")
    raise SystemExit
recovery = status.get("recovery") or {}
state = recovery.get("state", "")
message = recovery.get("message", "")
if status.get("running"):
    print("RUNNING a simulation is running after the upgrade")
elif state == "failed" and "set aside" not in message:
    # This is the #2092 shape: recovery refused the state and left it on disk,
    # so every later start returns a generic 500 while /__version looks fine.
    print(f"BLOCKED {message}")
else:
    print("CLEAR no simulation was running and nothing blocks a start")
')"
case "$RECOVERY_VERDICT" in
RUNNING*) pass "${RECOVERY_VERDICT#RUNNING }" ;;
CLEAR*) pass "${RECOVERY_VERDICT#CLEAR }" ;;
*) die "recovery left state that blocks every simulation start: ${RECOVERY_VERDICT#* }" ;;
esac

if [[ "$CARRIES_STATE" == "yes" ]]; then
	step "The simulation running before the upgrade survived it"
	assert_sessions_running deploy-validate-pre
	pass "deploy-validate-pre recovered across the upgrade"
fi

# The assertion #2092 needed and nothing here made: a start, after the upgrade,
# against the recovery state the upgrade inherited. A second dummy interface,
# because one interface carries one session.
step "A simulation still starts after the upgrade"
probe_interface niac-dv-post
start_simulation niac-dv-post deploy-validate-post "$SCENARIO" ||
	die "$VERSION cannot start a scenario named without a path"
if [[ "$CARRIES_STATE" == "yes" ]]; then
	assert_sessions_running deploy-validate-pre deploy-validate-post
else
	assert_sessions_running deploy-validate-post
fi
pass "$SCENARIO started as deploy-validate-post after the upgrade"

printf '\n\033[32mdeploy-validate: %s on %s\033[0m\n' "$VERSION" "$HOST"
