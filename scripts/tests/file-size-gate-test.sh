#!/bin/bash

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/scripts" "$fixture/ui/src"
cp "$repo_root/scripts/check-file-size.sh" "$fixture/scripts/check-file-size.sh"
printf 'ui/src/legacy.ts 6\n' > "$fixture/scripts/file-size-baseline.txt"
printf '1\n2\n3\n4\n5\n6\n' > "$fixture/ui/src/legacy.ts"

SCAN_ROOT="$fixture" RED_FLAG_TS=5 "$fixture/scripts/check-file-size.sh" >/dev/null

printf '7\n' >> "$fixture/ui/src/legacy.ts"
if SCAN_ROOT="$fixture" RED_FLAG_TS=5 "$fixture/scripts/check-file-size.sh" >/dev/null 2>&1; then
    echo "file-size gate accepted growth beyond the baseline"
    exit 1
fi

printf '1\n2\n3\n4\n5\n6\n' > "$fixture/ui/src/new.ts"
if SCAN_ROOT="$fixture" RED_FLAG_TS=5 "$fixture/scripts/check-file-size.sh" >/dev/null 2>&1; then
    echo "file-size gate accepted a new red-flag file"
    exit 1
fi

echo "file-size gate regression tests passed"
