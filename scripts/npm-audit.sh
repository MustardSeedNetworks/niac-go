#!/usr/bin/env bash
# Run npm audit, retrying when the registry itself is the problem.
#
# `npm audit` asks registry.npmjs.org for advisories, so its exit status
# conflates two very different answers: "this dependency tree has a known
# vulnerability" and "the advisory endpoint returned 503". Both failed the
# build identically, and the second one blocked PRs that changed no
# dependencies at all.
#
# Retrying a registry error is right; retrying a real finding is not, so the
# two are told apart by npm's own message before deciding. A registry that
# stays down still fails the build -- silently passing would turn the supply
# chain check into decoration, which is exactly what removing an earlier
# `|| true` was meant to stop.
set -uo pipefail

ATTEMPTS="${NPM_AUDIT_ATTEMPTS:-3}"
DELAY="${NPM_AUDIT_RETRY_DELAY:-15}"
LEVEL="${NPM_AUDIT_LEVEL:-high}"

for attempt in $(seq 1 "$ATTEMPTS"); do
	output=$(npm audit --audit-level="$LEVEL" 2>&1)
	status=$?

	if [[ $status -eq 0 ]]; then
		printf '%s\n' "$output"
		exit 0
	fi

	# npm reports a registry problem as an audit-endpoint error rather than a
	# finding; a finding names the vulnerabilities instead.
	if ! grep -qiE 'audit endpoint returned an error|Service Unavailable|ENOTFOUND|EAI_AGAIN|ETIMEDOUT|socket hang up' <<<"$output"; then
		printf '%s\n' "$output"
		exit "$status"
	fi

	printf 'npm audit could not reach the advisory endpoint (attempt %d/%d)\n' \
		"$attempt" "$ATTEMPTS" >&2
	if [[ $attempt -lt $ATTEMPTS ]]; then
		sleep $((DELAY * attempt))
	fi
done

printf '%s\n' "$output" >&2
printf '::error::npm audit could not reach the advisory endpoint after %d attempts; not treating that as a pass\n' \
	"$ATTEMPTS" >&2
exit 1
