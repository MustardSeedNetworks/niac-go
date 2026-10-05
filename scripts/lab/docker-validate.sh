#!/usr/bin/env bash
# Deployment validation for the container install path (deploy/docker): build
# the image from a published release, run it the way docs/DEPLOYMENT.md says
# to, and prove the container that comes up is that release and can simulate.
#
# The same four assertions deploy-validate.sh makes for the .deb and .rpm:
#   1. /__version reports the release with a non-empty uiBuildHash.
#   2. A simulation starts, so the capabilities reach the unprivileged user.
#   3. The next release, run over the first one's volume, comes up without a
#      restart loop and recovers the simulation that volume recorded.
#   4. A new simulation still starts after that upgrade.
#
# Simulations run on throwaway dummy interfaces, created for the run and
# deleted afterwards, so nothing this script starts reaches the host network.
#
# Usage:
#   scripts/lab/docker-validate.sh [--version vX.Y.Z]
#                                  [--from-version vX.Y.Z|none] [--port 8445]
#
#   --version       release to validate (default: the latest GitHub release)
#   --from-version  release run first on the same volume, so the second run is
#                   an upgrade (default: the release before --version; "none"
#                   reruns the same image over its own volume)
#   --port          loopback port the daemon listens on; must be free
#
# Requires: docker, gh (unless both versions are given), passwordless sudo for
# `ip link`, curl and python3. Runs on the Linux host it validates.
set -euo pipefail

PORT=8445
VERSION=""
FROM_VERSION=""
SETTLE_SECONDS="${SETTLE_SECONDS:-30}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-60}"
CONTEXT="$(cd "$(dirname "$0")/../../deploy/docker" && pwd)"
CONTAINER="niac-docker-validate-$$"
VOLUME="niac-docker-validate-$$"
PROBE_INTERFACES=()

die() {
	printf '\033[31merror:\033[0m %s\n' "$1" >&2
	exit 1
}

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
pass() { printf '\033[32mPASS\033[0m %s\n' "$1"; }

while [[ $# -gt 0 ]]; do
	case "$1" in
	--version) VERSION="${2:-}" && shift 2 ;;
	--from-version) FROM_VERSION="${2:-}" && shift 2 ;;
	--port) PORT="${2:-}" && shift 2 ;;
	-h | --help)
		awk 'NR > 1 && /^#/ { print; next } NR > 1 { exit }' "$0"
		exit 0
		;;
	*) die "unknown argument: $1" ;;
	esac
done

strip_v() { printf '%s' "${1#v}"; }

[[ -n "$VERSION" ]] || VERSION="$(gh release view --repo MustardSeedNetworks/niac-go --json tagName --jq .tagName)"
[[ -n "$VERSION" ]] || die "could not resolve the latest release tag"
if [[ -z "$FROM_VERSION" ]]; then
	FROM_VERSION="$(gh release list --repo MustardSeedNetworks/niac-go --limit 30 --json tagName --jq '[.[].tagName]' |
		python3 -c '
import json, sys

target = "v" + sys.argv[1].lstrip("v")
tags = json.load(sys.stdin)
i = tags.index(target) if target in tags else -1
print(tags[i + 1] if 0 <= i < len(tags) - 1 else "none")
' "$VERSION")"
fi
[[ "$FROM_VERSION" == none ]] && FROM_VERSION="$VERSION"

# The daemon walks to the next port when its own is taken, and host networking
# shares the host's ports, so a busy port would validate whatever already
# answers there rather than the container.
if ss -Htln "sport = :$PORT" | grep -q .; then
	die "port $PORT is already in use on this host; pass --port with a free one"
fi

cleanup() {
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	docker volume rm "$VOLUME" >/dev/null 2>&1 || true
	for iface in "${PROBE_INTERFACES[@]}"; do
		sudo ip link delete "$iface" 2>/dev/null || true
	done
}
trap cleanup EXIT

build_image() {
	local version
	version="$(strip_v "$1")"
	docker build --quiet --build-arg "NIAC_VERSION=$version" -t "niac:$version" "$CONTEXT" >/dev/null ||
		die "building the image for $1 failed"
}

# Exactly the documented run line, plus the listen address that pins the port.
run_container() {
	docker run -d --name "$CONTAINER" --network host \
		--cap-add NET_RAW --cap-add NET_ADMIN --restart on-failure \
		-e "NIAC_LISTEN_ADDR=127.0.0.1:$PORT" \
		-v "$VOLUME:/var/lib/niac" "niac:$(strip_v "$1")" >/dev/null
}

api() { curl -sk --max-time 10 "https://127.0.0.1:${PORT}$1"; }

wait_healthy() {
	local deadline=$((SECONDS + HEALTH_TIMEOUT))
	while ((SECONDS < deadline)); do
		api /__version | grep -q '"version"' && return 0
		sleep 2
	done
	docker logs --tail 40 "$CONTAINER" >&2 || true
	die "/__version did not answer on 127.0.0.1:${PORT} within ${HEALTH_TIMEOUT}s"
}

assert_version_payload() {
	local json
	json="$(api /__version)"
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
' "$(strip_v "$1")"
}

probe_interface() {
	sudo ip link delete "$1" 2>/dev/null || true
	if ! sudo ip link add "$1" type dummy || ! sudo ip link set "$1" up; then
		die "could not create the dummy interface $1 (is the dummy module available?)"
	fi
	PROBE_INTERFACES+=("$1")
}

# Started through the CLI inside the container, which is the operator's path.
start_simulation() {
	docker exec -e "NIAC_API_URL=https://127.0.0.1:$PORT" "$CONTAINER" \
		niac simulation start -i "$1" --config "$SCENARIO.yaml" --session "$2" --insecure >/dev/null ||
		die "starting $SCENARIO on $1 failed"
}

assert_sessions_running() {
	api /api/v1/sessions | python3 -c '
import json, sys

want = set(sys.argv[1:])
running = {session["sessionId"] for session in json.load(sys.stdin)}
missing = sorted(want - running)
if missing:
    sys.exit(f"sessions {missing} are not running; the daemon reports {sorted(running)}")
' "$@"
}

step "Validating the NIAC image for $VERSION (first run: $FROM_VERSION)"
build_image "$FROM_VERSION"
[[ "$FROM_VERSION" == "$VERSION" ]] || build_image "$VERSION"
pass "built niac:$(strip_v "$FROM_VERSION") and niac:$(strip_v "$VERSION") from their release archives"

step "Run 1 of 2: $FROM_VERSION on a new volume"
run_container "$FROM_VERSION"
wait_healthy
assert_version_payload "$FROM_VERSION"
pass "/__version reports $(strip_v "$FROM_VERSION") with a non-empty uiBuildHash"

SCENARIO="$(api /api/v1/library/networks | python3 -c 'import json, sys; print(json.load(sys.stdin)[0]["name"])')"
probe_interface niac-dk-pre
start_simulation niac-dk-pre docker-validate-pre
assert_sessions_running docker-validate-pre
pass "$SCENARIO runs as docker-validate-pre on niac-dk-pre"

step "Run 2 of 2: $VERSION over the volume run 1 left"
docker rm -f "$CONTAINER" >/dev/null
run_container "$VERSION"
wait_healthy
assert_version_payload "$VERSION"
pass "/__version reports $(strip_v "$VERSION") over the existing volume"

step "Watching for a restart loop for ${SETTLE_SECONDS}s"
sleep "$SETTLE_SECONDS"
read -r state restarts < <(docker inspect -f '{{.State.Status}} {{.RestartCount}}' "$CONTAINER")
[[ "$state" == running && "$restarts" == 0 ]] ||
	die "the container is $state after $restarts restarts: crash loop"
pass "container running, RestartCount 0"

assert_sessions_running docker-validate-pre
pass "docker-validate-pre recovered across the upgrade"

probe_interface niac-dk-post
start_simulation niac-dk-post docker-validate-post
assert_sessions_running docker-validate-pre docker-validate-post
pass "a new simulation starts after the upgrade"

step "Docker deployment validation PASSED for $VERSION"
