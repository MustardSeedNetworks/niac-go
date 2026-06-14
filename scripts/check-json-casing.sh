#!/usr/bin/env bash
# check-json-casing.sh — JSON wire-casing discipline gate (ADR-0007).
#
# Convention (mirrors seed's revised ADR-0010, pure boundary mapping): every
# JSON `json:"..."` wire tag the API emits or accepts is camelCase. snake_case
# lives only OFF the wire — config files (the YAML simulation configs), SQL,
# and internal external-system adapters (e.g. iperf3 `-json` parsing in
# internal/protocols, mapped before emission). There are NO wire-level snake
# exceptions: the baseline (scripts/json-casing-baseline.txt) is EMPTY.
#
# This gate scans `json:"..."` struct tags in internal/api for snake_case keys.
# It is a RATCHET against the (empty) baseline:
#   - FAILS if any snake_case wire tag appears, and
#   - passes when there are none.
#
# Note: this scans struct *tags*, not hand-built `map[string]any` JSON keys.
# Most snake map literals in internal/api build YAML *config* (stay snake);
# hand-built JSON *responses* are camelCased per-handler (see ADR-0007).
#
# Run locally: scripts/check-json-casing.sh   ·   regenerate: --update
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASELINE="scripts/json-casing-baseline.txt"
SCAN_DIRS=("internal/api")

current_violations() {
	# `|| true`: grep exits 1 when there are zero snake tags (the healthy state);
	# under `set -o pipefail` that would otherwise abort. Empty = success.
	grep -rnoE 'json:"[a-z][a-z0-9]*_[a-z0-9_]+[^"]*"' "${SCAN_DIRS[@]}" --include='*.go' 2>/dev/null \
		| grep -v '_test\.go:' \
		| sed -E 's/^([^:]+):[0-9]+:json:"([^",]+).*/\1\t\2/' \
		| sort -u || true
}

if [[ "${1:-}" == "--update" ]]; then
	current_violations >"$BASELINE"
	echo "Wrote $(wc -l <"$BASELINE" | tr -d ' ') baselined snake_case JSON tag(s) to $BASELINE"
	exit 0
fi

if [[ ! -f "$BASELINE" ]]; then
	echo "::error::$BASELINE missing — run scripts/check-json-casing.sh --update" >&2
	exit 1
fi

new=$(comm -23 <(current_violations) <(sort -u "$BASELINE") || true)

if [[ -n "$new" ]]; then
	echo "::error::snake_case JSON wire tag(s) — use camelCase (ADR-0007):" >&2
	echo "$new" | sed 's/^/  /' >&2
	exit 1
fi

echo "JSON casing gate OK — no snake_case wire tags in internal/api (ADR-0007, empty baseline)."
