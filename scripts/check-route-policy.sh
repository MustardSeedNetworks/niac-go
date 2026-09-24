#!/usr/bin/env bash
# check-route-policy.sh — capability-registry enforcement gate.
#
# Every route MUST be registered through foundation's route.Registrar
# (Register / RegisterAll; niac's table is internal/api/routes.go), which
# composes its per-route policy — rate limiting, auth, method gate, CSRF, admin
# scope, body cap — in ONE canonical order. A ServeMux of our own, or a direct
# Handle/HandleFunc with a pattern, bypasses that composition and is how a
# mutating route can silently ship without CSRF or scope enforcement.
#
# This is foundation's pkg/httpserver/route/check-route-policy.sh rule with the
# registration pattern anchored to a string literal: the shared script's bare
# `\.Handle\(` also matches slog.Handler.Handle (internal/api/sse), so niac
# keeps this copy until foundation#70 lands and then runs the shared one.
#
# Run locally: scripts/check-route-policy.sh
set -euo pipefail

API_DIR="internal/api"

violations=$(grep -rnE --include='*.go' --exclude='*_test.go' \
	'http\.NewServeMux\(|\.Handle(Func)?\("' "$API_DIR" || true)

if [[ -n "$violations" ]]; then
	echo "❌ Route-policy gate: register every route through route.Registrar"
	echo "   (Register / RegisterAll), not a ServeMux of your own. A direct"
	echo "   registration skips the rate-limit/auth/CSRF/scope composition."
	echo "   Add a route.Route entry to the appropriate register*Routes function."
	echo ""
	echo "$violations"
	exit 1
fi

echo "✓ Route-policy gate: all routes go through the capability registry."
