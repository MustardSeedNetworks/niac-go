#!/usr/bin/env bash

set -uo pipefail

fail=0

report_matches() {
  local label=$1
  shift
  local matches
  matches=$(rg -n "$@" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    printf '%s\n%s\n\n' "$label" "$matches"
    fail=1
  fi
}

report_matches \
  "Alternate API or metrics listener support found:" \
  'MetricsAddr|EnableTLS|metricsServer|\.Serve\(' \
  internal/api/server.go internal/api/tls_fingerprint.go internal/daemon/daemon.go

report_matches \
  "Obsolete run-mode listener flag found:" \
  '"(api-listen|metrics-listen)"|--(api-listen|metrics-listen|web)([ =`]|$)|127\.0\.0\.1:8080|os\.Getenv\("NIAC_LISTEN"\)' \
  --glob '!**/*_test.go' cmd/niac

report_matches \
  "Plaintext or obsolete API example found in active operator documentation:" \
  'http://localhost:8080|localhost:9090/metrics|niac daemon.*--(api|web)' \
  docs/API_EXAMPLES.md docs/CI_CD.md docs/FAQ.md docs/MONITORING.md \
  docs/PERFORMANCE.md docs/README.md docs/WEBUI.md docs/openapi.yaml

report_matches \
  "Plaintext or obsolete API endpoint found in packaging or smoke tests:" \
  'http://localhost:8080|NIAC_PORT=8080|NIAC_SCHEME="http"|LocalPort 8080' \
  deploy/macos/build-pkg.sh deploy/windows/build.ps1 tests/smoke/run_smoke_tests.sh

report_matches \
  "Plaintext development API proxy found:" \
  'http://localhost:8080' \
  ui/vite.config.ts

if [ "$fail" -ne 0 ]; then
  echo "FAIL: HTTPS-only API contract regressed."
  exit 1
fi

echo "OK: API, Web UI, and metrics are restricted to the HTTPS listener."
