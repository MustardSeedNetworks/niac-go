#!/usr/bin/env bash
# check-output-escaping.sh — output-escaping / XSS regression gate (#742).
#
# Audited 2026-05-28: classified all fmt.Fprintf(w, ...) sites.
#   - internal/converter (PrintSummary -> *bufio.Writer): CLI/file output.
#   - internal/protocols/snmp/synth/format.go (io.Writer): SNMP walk-file
#     generation, CLI/file output.
#   - internal/api/handlers_metrics.go: Prometheus exposition format,
#     Content-Type text/plain, metric names are server constants.
#   - internal/api/sse_serve.go: SSE frame interpolating a fixed
#     SSEStream enum ("packets"/"logs"/"stats"), never user input.
# No site renders user-supplied data as HTML. This gate prevents
# regressions.
#
# Two checks:
#   1. No raw innerHTML injection in ui/src (React XSS vector).
#   2. No NEW value-interpolating fmt.Fprintf(w, "...%s...") in
#      internal/api outside the audited allow-list. HTTP responses
#      should use writeJSON / json.Encode, never raw format strings.
#
# Run locally: scripts/check-output-escaping.sh

set -uo pipefail

FAIL=0
INNER_HTML_RE='dangerously[S]etInnerHTML'

# Files in internal/api whose Fprintf(w,...) sites were audited safe.
# handlers_metrics.go: Prometheus text/plain, server-constant names.
# sse_serve.go: SSE frame with a fixed server-side enum, not user input.
ALLOWLIST_RE='internal/api/(handlers_metrics|sse_serve)\.go'

UI_DIR=""
if [ -d "ui/src" ]; then
  UI_DIR="ui/src"
elif [ -d "src" ] && [ -f "package.json" ]; then
  UI_DIR="src"
fi

if [ -n "$UI_DIR" ]; then
  DANGER=$(grep -rEn "$INNER_HTML_RE" "$UI_DIR" \
    --include='*.tsx' --include='*.ts' 2>/dev/null \
    | grep -vE '\.(test|spec|stories)\.(ts|tsx):' || true)
  if [ -n "$DANGER" ]; then
    echo "============================================================"
    echo "[XSS] raw innerHTML injection found in UI app code:"
    echo "$DANGER"
    echo "Use plain text/JSX, or sanitize with DOMPurify and justify in review."
    echo ""
    FAIL=1
  fi
fi

if [ -d "internal/api" ]; then
  HTTP_FMT=$(grep -rEn 'fmt\.Fprintf\(w,[^)]*%[svqd]' internal/api 2>/dev/null \
    | grep -v '_test.go' \
    | grep -vE "$ALLOWLIST_RE" || true)
  if [ -n "$HTTP_FMT" ]; then
    echo "============================================================"
    echo "[XSS] value-interpolating fmt.Fprintf(w, ...) in internal/api —"
    echo "HTTP responses must use writeJSON / json.NewEncoder, not raw"
    echo "format strings:"
    echo "$HTTP_FMT"
    echo ""
    echo "If a new site is deliberately safe (text/plain or a"
    echo "server-controlled enum), add it to ALLOWLIST_RE with a note."
    echo ""
    FAIL=1
  fi
fi

if [ "$FAIL" -ne 0 ]; then
  echo "FAIL: output-escaping gate (#742)."
  exit 1
fi

echo "OK: no raw innerHTML injection; internal/api Fprintf sites are within the audited allow-list."
