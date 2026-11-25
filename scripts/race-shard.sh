#!/bin/sh
# scripts/race-shard.sh N | --list N
#
# One slice of `go test -race ./...`, so the race detector runs as three
# parallel CI jobs instead of one 13-minute job. The slow packages are the ones
# that start every shipped pack (api, daemon, scenario, protocols, library) and
# the Link-Live acceptance suite; the first two shards take named packages and
# the last takes every package not named, so a package added later runs in
# exactly one shard without anyone editing this file.
#
# Sizes come from the merge_group race job of 2026-11-22 (per-package seconds):
#   1  api 500, library 119, cmd/niac 70
#   2  daemon 368, acceptance/linklive 293, sanitize 105
#   3  scenario 286, protocols 265, protocols/snmp 88, the rest
set -eu

shard1="./internal/api ./internal/library ./cmd/niac"
shard2="./internal/daemon ./internal/acceptance/linklive ./internal/sanitize"

list_only=false
if [ "${1:-}" = "--list" ]; then
  list_only=true
  shift
fi
shard=${1:?usage: race-shard.sh [--list] 1|2|3}

named=$(mktemp)
trap 'rm -f "$named"' EXIT
# The lists are space-separated package paths, so they are deliberately unquoted.
# shellcheck disable=SC2086
go list $shard1 $shard2 >"$named"

# shellcheck disable=SC2086
case "$shard" in
1) pkgs=$(go list $shard1) ;;
2) pkgs=$(go list $shard2) ;;
3) pkgs=$(go list ./... | grep -vxF -f "$named") ;;
*) echo "race-shard: shard must be 1, 2 or 3" >&2; exit 2 ;;
esac

if [ "$list_only" = true ]; then
  echo "$pkgs"
  exit 0
fi
# shellcheck disable=SC2086 # one package path per word
exec go test -race $pkgs
