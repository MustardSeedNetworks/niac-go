#!/usr/bin/env bash
# scripts/check-library-stdout.sh
#
# Library code must not write to os.Stdout. `niac daemon --once` prints a JSON
# summary there and redirects internal/logging to stderr so a caller can pipe
# the result; a package under internal/ that writes to stdout directly bypasses
# that redirect and corrupts the output. That is niac#1805: 161 such sites in
# internal/protocols alone made `daemon --once | jq` impossible.
#
# internal/logging owns the writer and is the one allowed exception. Commands
# under cmd/ are not library code and print freely.

set -uo pipefail

# Comments explaining the rule are not violations of it.
matches=$(
  rg -n --glob '!*_test.go' --glob '!internal/logging/**' \
    'fmt\.(Fprintf|Fprintln|Fprint)\(\s*$|fmt\.(Fprintf|Fprintln|Fprint)\([^)]*os\.Stdout' \
    internal 2>/dev/null |
    rg 'os\.Stdout' || true
)

# The multi-line call form puts os.Stdout on its own line, which the single-line
# pattern above cannot see. Catch it by looking for the argument directly.
multiline=$(
  rg -n --glob '!*_test.go' --glob '!internal/logging/**' \
    '^\s*os\.Stdout,\s*$' internal 2>/dev/null || true
)

if [ -n "$matches" ] || [ -n "$multiline" ]; then
  printf 'Library code writing to os.Stdout (use internal/logging; see niac#1805):\n'
  [ -n "$matches" ] && printf '%s\n' "$matches"
  [ -n "$multiline" ] && printf '%s\n' "$multiline"
  printf '\nIf a new package legitimately owns stdout, it belongs in cmd/, not internal/.\n'
  exit 1
fi

echo "OK: no library package under internal/ writes to os.Stdout."
