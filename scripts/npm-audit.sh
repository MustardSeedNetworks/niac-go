#!/usr/bin/env bash
# Run npm audit, retrying when the registry itself is the problem.
#
# `npm audit` asks registry.npmjs.org for advisories, so its exit status
# conflates two very different answers: "this dependency tree has a known
# vulnerability" and "the advisory endpoint returned 503". Both failed the
# build identically, and the second one blocked PRs that changed no
# dependencies at all.
#
# Retrying a registry error is right; retrying a real finding is not. The
# first cut of this script (#1772) told the two apart by grepping npm's
# human-readable text for outage-sounding prose (ENOTFOUND, ETIMEDOUT, and
# friends). That is still locale- and npm-version-dependent, and the seed
# equivalent grepped a list ending in the bare word "network" — so a genuine
# High finding whose package name or advisory title contained "network" was
# retried three times and reported as an unreachable endpoint instead of a
# vulnerability (seed#2416). `npm audit --json` gives a deterministic signal
# instead: a transport failure returns a top-level "error" key, a completed
# audit returns "metadata.vulnerabilities". Decide on that, never on prose.
# Converges with stem#1004 / trellis#315 / seed#2416, which implement the
# same distinction.
#
# A registry that stays down still fails the build — silently passing would
# turn the supply chain check into decoration, which is exactly what removing
# an earlier `|| true` was meant to stop.
set -uo pipefail

ATTEMPTS="${NPM_AUDIT_ATTEMPTS:-3}"
DELAY="${NPM_AUDIT_RETRY_DELAY:-15}"
LEVEL="${NPM_AUDIT_LEVEL:-high}"

# verdict decides pass/fail from a completed `npm audit --json` payload (no
# top-level "error" key) on stdin — never from npm's prose. Exposed as a
# function, not inlined into main, so tests can pipe synthetic JSON straight
# at the decision without shelling out to npm or a registry. By the time
# verdict runs, main has already proven the audit completed, so a finding
# whose package or advisory title contains a transport-sounding word cannot
# be misread as an outage.
verdict() {
	local out vulns high critical total
	out="$(cat)"
	vulns="$(jq -e '.metadata.vulnerabilities' <<<"$out" 2>/dev/null)" || {
		printf '::error::npm audit returned no metadata.vulnerabilities -- unexpected output, failing closed\n' >&2
		printf '%s\n' "$out" >&2
		return 1
	}
	high="$(jq -r '.high // 0' <<<"$vulns")"
	critical="$(jq -r '.critical // 0' <<<"$vulns")"
	total="$(jq -r '.total // 0' <<<"$vulns")"
	printf 'npm audit: %s vulnerabilities found (%s high, %s critical)\n' "$total" "$high" "$critical"
	if [[ $((high + critical)) -gt 0 ]]; then
		jq -r '(.vulnerabilities // {}) | to_entries[] | select(.value.severity == "high" or .value.severity == "critical") | "  \(.value.severity): \(.key) (\(.value.range // "unknown range"))"' <<<"$out"
		return 1
	fi
	return 0
}

main() {
	local out endpoint attempt
	for attempt in $(seq 1 "$ATTEMPTS"); do
		out="$(npm audit --json --audit-level="$LEVEL" || true)"

		if ! jq -e 'has("error")' >/dev/null 2>&1 <<<"$out"; then
			verdict <<<"$out"
			return $?
		fi

		endpoint="$(jq -r '(.error.summary | select(. != "")) // .message // "registry.npmjs.org advisory endpoint"' <<<"$out" 2>/dev/null)"
		endpoint="${endpoint:-registry.npmjs.org advisory endpoint}"

		if [[ $attempt -lt $ATTEMPTS ]]; then
			printf 'npm audit could not reach the advisory endpoint (attempt %d/%d): %s\n' \
				"$attempt" "$ATTEMPTS" "$endpoint" >&2
			sleep $((DELAY * attempt))
			continue
		fi

		printf '::error::npm audit could not reach the advisory endpoint (registry.npmjs.org) after %d attempts: %s -- this is not a dependency finding, but the audit did not run so the gate fails rather than claiming a clean scan\n' \
			"$ATTEMPTS" "$endpoint" >&2
		return 1
	done
}

# Sourcing the script (as the test suite does, to call `verdict` directly on
# synthetic JSON) must not also run `main` against the real registry.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
	main
	exit $?
fi
