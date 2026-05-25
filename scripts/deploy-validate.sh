#!/bin/bash
# =============================================================================
# deploy-validate.sh - Validate NIAC deployment
# =============================================================================
#
# Verifies that a deployed NIAC instance is running the expected version
# with the correct UI build hash.
#
# Usage: ./scripts/deploy-validate.sh <expected-version> <expected-commit> [host] [port]
#
# Examples:
#   ./scripts/deploy-validate.sh v0.66.25 abc1234
#   ./scripts/deploy-validate.sh v0.66.25 abc1234 niac-srv-ubuntu 8445
#
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
EXPECTED_VERSION="${1:-}"
EXPECTED_COMMIT="${2:-}"
HOST="${3:-localhost}"
PORT="${4:-8445}"
SCHEME="${NIAC_VALIDATE_SCHEME:-https}"
ENDPOINT="${SCHEME}://${HOST}:${PORT}/__version"
MAX_RETRIES=5
RETRY_DELAY=3
CURL_FLAGS=()
if [ "$SCHEME" = "https" ]; then
    # NIAC ships with a self-signed cert by default; skip TLS verify for
    # validation. Override NIAC_VALIDATE_SCHEME=http for plaintext setups
    # (legacy HTTP listener via `daemon --http` on port 8080).
    CURL_FLAGS+=("-k")
fi

# Functions
success() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠ $1${NC}"; }
info() { echo -e "${CYAN}ℹ $1${NC}"; }

usage() {
    echo "Usage: $0 <expected-version> <expected-commit> [host] [port]"
    echo ""
    echo "Arguments:"
    echo "  expected-version  The version string to expect (e.g., v0.66.25)"
    echo "  expected-commit   The short commit hash to expect (e.g., abc1234)"
    echo "  host              Target hostname (default: localhost)"
    echo "  port              Target port (default: 8445)"
    echo ""
    echo "Environment:"
    echo "  NIAC_VALIDATE_SCHEME  http | https (default: https)"
    echo ""
    echo "Examples:"
    echo "  $0 v0.66.25 abc1234"
    echo "  $0 v0.66.25 abc1234 niac-srv-ubuntu 8445"
    exit 1
}

# Validate arguments
if [ -z "$EXPECTED_VERSION" ] || [ -z "$EXPECTED_COMMIT" ]; then
    usage
fi

echo "=== NIAC Deployment Validation ==="
echo ""
info "Target:   $ENDPOINT"
info "Expected: version=$EXPECTED_VERSION commit=$EXPECTED_COMMIT"
echo ""

# Wait for service to be available
echo "Waiting for service to be available..."
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -sf "${CURL_FLAGS[@]}" -o /dev/null "$ENDPOINT" 2>/dev/null; then
        success "Service is responding"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        echo "  Retry $RETRY_COUNT/$MAX_RETRIES in ${RETRY_DELAY}s..."
        sleep $RETRY_DELAY
    fi
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    fail "Service not responding after $MAX_RETRIES attempts"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check if niac is running: systemctl status niac"
    echo "  2. Check service logs: journalctl -u niac -f"
    echo "  3. Check if port $PORT is listening: ss -tlnp | grep $PORT"
    exit 1
fi

# Fetch version info
echo ""
echo "Fetching version information..."
RESPONSE=$(curl -sf "${CURL_FLAGS[@]}" "$ENDPOINT" 2>&1) || {
    fail "Failed to fetch /__version endpoint"
    exit 1
}

# Parse response (requires jq)
if ! command -v jq &> /dev/null; then
    fail "jq is required but not installed"
    echo "Install with: apt-get install jq (Ubuntu) or dnf install jq (Fedora)"
    exit 1
fi

ACTUAL_VERSION=$(echo "$RESPONSE" | jq -r '.version // empty')
ACTUAL_COMMIT=$(echo "$RESPONSE" | jq -r '.commit // empty')
ACTUAL_COMMIT_FULL=$(echo "$RESPONSE" | jq -r '.commitFull // empty')
ACTUAL_UI_HASH=$(echo "$RESPONSE" | jq -r '.uiBuildHash // empty')
ACTUAL_BUILD_TIME=$(echo "$RESPONSE" | jq -r '.buildTime // empty')
ACTUAL_PLATFORM=$(echo "$RESPONSE" | jq -r '.platform // empty')
ACTUAL_GO_VERSION=$(echo "$RESPONSE" | jq -r '.goVersion // empty')

echo ""
echo "Build Information:"
echo "  Version:      $ACTUAL_VERSION"
echo "  Commit:       $ACTUAL_COMMIT ($ACTUAL_COMMIT_FULL)"
echo "  Build Time:   $ACTUAL_BUILD_TIME"
echo "  UI Hash:      $ACTUAL_UI_HASH"
echo "  Platform:     $ACTUAL_PLATFORM"
echo "  Go Version:   $ACTUAL_GO_VERSION"
echo ""

# Validation checks
VALIDATION_FAILED=0

# Check version
echo "Running validation checks..."
if [ "$ACTUAL_VERSION" = "$EXPECTED_VERSION" ]; then
    success "Version matches: $ACTUAL_VERSION"
else
    fail "Version MISMATCH: expected=$EXPECTED_VERSION actual=$ACTUAL_VERSION"
    VALIDATION_FAILED=1
fi

# Check commit (compare first 7 characters)
EXPECTED_COMMIT_SHORT="${EXPECTED_COMMIT:0:7}"
ACTUAL_COMMIT_SHORT="${ACTUAL_COMMIT:0:7}"
if [ "$ACTUAL_COMMIT_SHORT" = "$EXPECTED_COMMIT_SHORT" ]; then
    success "Commit matches: $ACTUAL_COMMIT_SHORT"
else
    fail "Commit MISMATCH: expected=$EXPECTED_COMMIT_SHORT actual=$ACTUAL_COMMIT_SHORT"
    VALIDATION_FAILED=1
fi

# Check UI build hash is present
if [ -n "$ACTUAL_UI_HASH" ] && [ "$ACTUAL_UI_HASH" != "null" ] && [ "$ACTUAL_UI_HASH" != "" ]; then
    success "UI build hash present: $ACTUAL_UI_HASH"
else
    fail "UI build hash MISSING (frontend may not be embedded)"
    VALIDATION_FAILED=1
fi

# Check build time is present
if [ -n "$ACTUAL_BUILD_TIME" ] && [ "$ACTUAL_BUILD_TIME" != "null" ]; then
    success "Build time present: $ACTUAL_BUILD_TIME"
else
    warn "Build time missing (non-critical)"
fi

echo ""

# Final result
if [ $VALIDATION_FAILED -eq 0 ]; then
    echo "======================================="
    success "DEPLOYMENT VALIDATION PASSED"
    echo "======================================="
    exit 0
else
    echo "======================================="
    fail "DEPLOYMENT VALIDATION FAILED"
    echo "======================================="
    echo ""
    echo "The deployed version does not match the expected build."
    echo "This could indicate:"
    echo "  1. Package installation failed"
    echo "  2. Service not restarted after installation"
    echo "  3. Wrong package version installed"
    echo ""
    echo "Remediation:"
    echo "  1. Reinstall the package"
    echo "  2. Restart the service: systemctl restart niac"
    echo "  3. Check logs: journalctl -u niac -f"
    exit 1
fi
