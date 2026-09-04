#!/usr/bin/env bash
# Refuse to ship a starter walk that still carries real-world data.
#
# The starter walks are content, not code, and nothing stopped an unsanitised
# one landing in the tree: four of them were shipping tcpConnTable and LLDP
# management addresses in OID index positions, which record who a real device
# was actually talking to.
#
# `niac sanitize --check` reports those properties; this wires it into CI over
# every shipped walk.
set -euo pipefail

WALKS_DIR="${1:-internal/library/starter/walks}"
NIAC_BIN="${NIAC_BIN:-./niac}"

if [[ ! -x "$NIAC_BIN" ]]; then
	printf 'building %s to run the check\n' "$NIAC_BIN" >&2
	go build -o "$NIAC_BIN" ./cmd/niac
fi

shopt -s nullglob
walks=("$WALKS_DIR"/*.walk)
shopt -u nullglob

if [[ ${#walks[@]} -eq 0 ]]; then
	# An empty directory passing silently is how this gate would rot: a moved
	# walks directory would report success forever.
	printf '::error::no walks found under %s; the gate is checking nothing\n' "$WALKS_DIR" >&2
	exit 1
fi

"$NIAC_BIN" sanitize --check "${walks[@]}"
