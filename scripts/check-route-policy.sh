#!/usr/bin/env bash
# check-route-policy.sh — capability-registry enforcement gate.
#
# Every API route MUST be registered through the capability registry
# (register / registerAll in internal/api/route.go), which composes its
# per-route policy — auth, rate limiting, CSRF, admin scope, license feature —
# in ONE canonical order. Hand-wrapping a route directly via
# mux.HandleFunc("/api/...") bypasses that composition and is how a mutating
# route can silently ship without CSRF or scope enforcement (the foot-gun the
# audit flagged in the old registerReadOnlyRoutes mix).
#
# This gate fails if any "/api/..." path is registered directly instead of via
# register(). register() installs routes with a variable path (rt.path), so it
# never matches; only direct literal registrations do. The unauthenticated
# introspection endpoints (/__version, /__capabilities) and the SPA/metrics
# catch-alls are not /api/ literals and are intentionally direct.
#
# Run locally: scripts/check-route-policy.sh
set -euo pipefail

API_DIR="internal/api"

violations=$(grep -rnE 'mux\.HandleFunc\("/api/' "$API_DIR"/*.go \
	| grep -v '_test.go' || true)

if [[ -n "$violations" ]]; then
	echo "❌ Route-policy gate: API routes must be registered through the"
	echo "   capability registry (registerAll/register in route.go), not via a"
	echo "   raw mux.HandleFunc(\"/api/...\"). A direct registration skips the"
	echo "   auth/rate-limit/CSRF/scope composition. Add an apiRoute{} entry to"
	echo "   the appropriate register*Routes function instead."
	echo ""
	echo "$violations"
	exit 1
fi

echo "✓ Route-policy gate: all /api routes go through the capability registry."
