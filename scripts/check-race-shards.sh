#!/bin/sh
# scripts/check-race-shards.sh
#
# The three race shards must together cover `go list ./...` exactly once. A
# package that fell out of every shard would silently lose its race coverage.
set -eu

dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

go list ./... | sort >"$dir/all"
for shard in 1 2 3; do
  ./scripts/race-shard.sh --list "$shard"
done | sort >"$dir/union"

if ! diff -u "$dir/all" "$dir/union" >&2; then
  echo "check-race-shards: shards do not cover go list ./... exactly once" >&2
  exit 1
fi
echo "check-race-shards: $(wc -l <"$dir/all" | tr -d ' ') packages, each in exactly one shard"
