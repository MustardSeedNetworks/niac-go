#!/bin/bash
#
# run_smoke_tests.sh - Comprehensive Smoke Tests for NiAC
#
# Tests core functionality: binary, version, API endpoints, and simulations.
# Can run against an installed service or a local binary.
#
# Usage:
#   ./run_smoke_tests.sh              # Test local binary (builds if needed)
#   ./run_smoke_tests.sh --installed  # Test installed system service
#
# Copyright (c) 2025 Mustard Seed Networks. All rights reserved.
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
NIAC_PORT=8080
NIAC_SCHEME="http"
NIAC_HOST="localhost"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../.."
NIAC_BIN="${PROJECT_ROOT}/niac"

# Mode
INSTALLED_MODE=false
if [[ "${1:-}" == "--installed" ]]; then
    INSTALLED_MODE=true
    NIAC_BIN=$(command -v niac 2>/dev/null || echo "/usr/bin/niac")
fi

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Process tracking
NIAC_PID=""

# Timing
START_TIME=$(date +%s)

# Logging
log_info()    { echo -e "${CYAN}[INFO]${NC} $1"; }
log_pass()    { echo -e "${GREEN}[PASS]${NC} $1"; }
log_fail()    { echo -e "${RED}[FAIL]${NC} $1"; }
log_skip()    { echo -e "${YELLOW}[SKIP]${NC} $1"; }
log_header()  { echo -e "\n${BOLD}${CYAN}=== $1 ===${NC}"; }
log_section() { echo -e "\n${BOLD}${CYAN}--- $1 ---${NC}"; }

# Cleanup
cleanup() {
    if [[ -n "$NIAC_PID" ]]; then
        kill $NIAC_PID 2>/dev/null || true
        wait $NIAC_PID 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Run a test
run_test() {
    local name="$1"
    local cmd="$2"
    local expected_exit="${3:-0}"

    TESTS_RUN=$((TESTS_RUN + 1))

    local output exit_code
    set +e
    output=$(eval "$cmd" 2>&1)
    exit_code=$?
    set -e

    if [[ $exit_code -eq $expected_exit ]]; then
        log_pass "$name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "$name (exit: $exit_code, expected: $expected_exit)"
        echo "  Output: $(echo "$output" | head -2 | tr '\n' ' ')"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Assert HTTP status code
assert_http_status() {
    local url="$1"
    local expected="$2"
    local name="$3"
    local extra_args="${4:-}"

    TESTS_RUN=$((TESTS_RUN + 1))

    local status
    status=$(curl -s $extra_args -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)

    if [[ "$status" == "$expected" ]]; then
        log_pass "$name (HTTP $status)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "$name (got HTTP $status, expected $expected)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Assert JSON response contains key
assert_json_key() {
    local url="$1"
    local key="$2"
    local name="$3"

    TESTS_RUN=$((TESTS_RUN + 1))

    local body
    body=$(curl -s "$url" 2>/dev/null)

    if echo "$body" | grep -q "\"$key\""; then
        log_pass "$name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "$name (key '$key' not in response)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

assert_contains() {
    local output="$1"
    local expected="$2"
    local name="$3"

    TESTS_RUN=$((TESTS_RUN + 1))
    if echo "$output" | grep -qi "$expected"; then
        log_pass "$name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "$name (expected to contain: $expected)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

skip_test() {
    local name="$1"
    local reason="$2"
    TESTS_RUN=$((TESTS_RUN + 1))
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    log_skip "$name - $reason"
}

# ============================================================================
# SECTION 1: Binary Tests
# ============================================================================

test_binary() {
    log_section "BINARY TESTS"

    if [[ ! -x "${NIAC_BIN}" ]] && [[ "$INSTALLED_MODE" == false ]]; then
        log_info "Binary not found, building..."
        (cd "${PROJECT_ROOT}" && make build)
    fi

    run_test "Binary exists and is executable" \
        "test -x ${NIAC_BIN}"

    run_test "Binary is not empty" \
        "test -s ${NIAC_BIN}"
}

# ============================================================================
# SECTION 2: Version Tests
# ============================================================================

test_version() {
    log_section "VERSION TESTS"

    run_test "Version flag succeeds" \
        "${NIAC_BIN} --version"

    local version_output
    version_output=$("${NIAC_BIN}" --version 2>&1)

    assert_contains "$version_output" "niac" "Version mentions niac"
    assert_contains "$version_output" "v0\." "Version has v0.x format"
    assert_contains "$version_output" "commit" "Version shows commit"
    assert_contains "$version_output" "built" "Version shows build time"
}

# ============================================================================
# SECTION 3: CLI Help Tests
# ============================================================================

test_cli_help() {
    log_section "CLI HELP TESTS"

    run_test "Help flag (--help)" "${NIAC_BIN} --help"

    local help_output
    help_output=$("${NIAC_BIN}" --help 2>&1)

    assert_contains "$help_output" "niac" "Help mentions niac"
    assert_contains "$help_output" "Network" "Help mentions Network"

    log_header "Subcommand Help"
    run_test "validate --help" "${NIAC_BIN} validate --help"
    run_test "template --help" "${NIAC_BIN} template --help"
    run_test "interactive --help" "${NIAC_BIN} interactive --help"
    run_test "completion --help" "${NIAC_BIN} completion --help"
}

# ============================================================================
# SECTION 4: Template Tests
# ============================================================================

test_templates() {
    log_section "TEMPLATE TESTS"

    run_test "template list command" \
        "${NIAC_BIN} template list"

    local list_output
    list_output=$("${NIAC_BIN}" template list 2>&1)

    assert_contains "$list_output" "router" "Router template available"
    assert_contains "$list_output" "switch" "Switch template available"
}

# ============================================================================
# SECTION 5: API Endpoint Tests (requires running service)
# ============================================================================

test_api_endpoints() {
    log_section "API ENDPOINT TESTS"

    local base_url="${NIAC_SCHEME}://${NIAC_HOST}:${NIAC_PORT}"

    # Check if service is reachable
    if ! curl -s -o /dev/null "${base_url}/" 2>/dev/null; then
        skip_test "API endpoint tests" "Service not reachable at ${base_url}"
        return
    fi

    log_header "WebUI"
    assert_http_status "${base_url}/" "200" "WebUI serves content"

    log_header "Health Endpoint"
    assert_http_status "${base_url}/api/v1/health" "200" "Health endpoint responds"
    assert_json_key "${base_url}/api/v1/health" "status" "Health has status field"

    log_header "Device Endpoints"
    assert_http_status "${base_url}/api/v1/devices" "200" "Devices endpoint responds"
    assert_http_status "${base_url}/api/v1/templates" "200" "Templates endpoint responds"
    assert_http_status "${base_url}/api/v1/simulations" "200" "Simulations endpoint responds"

    log_header "JSON Response Validation"
    local devices_body
    devices_body=$(curl -s "${base_url}/api/v1/devices" 2>/dev/null)
    TESTS_RUN=$((TESTS_RUN + 1))
    if echo "$devices_body" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_pass "Devices endpoint returns valid JSON"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "Devices endpoint returns invalid JSON"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    local templates_body
    templates_body=$(curl -s "${base_url}/api/v1/templates" 2>/dev/null)
    TESTS_RUN=$((TESTS_RUN + 1))
    if echo "$templates_body" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_pass "Templates endpoint returns valid JSON"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "Templates endpoint returns invalid JSON"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    log_header "Invalid Endpoints"
    assert_http_status "${base_url}/api/v1/nonexistent" "404" "Nonexistent path returns 404"

    log_header "WebUI Assets"
    local html
    html=$(curl -s "${base_url}/" 2>/dev/null)
    assert_contains "$html" "html" "Root serves HTML document"
}

# ============================================================================
# SECTION 6: Service Management Tests (installed mode only)
# ============================================================================

test_service_management() {
    log_section "SERVICE MANAGEMENT TESTS"

    if [[ "$INSTALLED_MODE" == false ]]; then
        skip_test "Service management" "Not in --installed mode"
        return
    fi

    run_test "systemd unit file exists" \
        "test -f /usr/lib/systemd/system/niac.service"

    run_test "Service is active" \
        "systemctl is-active --quiet niac.service"

    run_test "Service is enabled" \
        "systemctl is-enabled --quiet niac.service"

    run_test "Binary has capabilities set" \
        "getcap /usr/bin/niac 2>/dev/null | grep -q cap_net_raw"

    run_test "Config directory exists" \
        "test -d /etc/niac"

    run_test "Data directory exists" \
        "test -d /var/lib/niac"

    run_test "Log directory exists" \
        "test -d /var/log/niac"
}

# ============================================================================
# SECTION 7: Concurrent Request Test
# ============================================================================

test_concurrent_requests() {
    log_section "CONCURRENT REQUEST TESTS"

    local base_url="${NIAC_SCHEME}://${NIAC_HOST}:${NIAC_PORT}"

    if ! curl -s -o /dev/null "${base_url}/" 2>/dev/null; then
        skip_test "Concurrent requests" "Service not reachable"
        return
    fi

    # Send 20 concurrent requests
    for i in $(seq 1 20); do
        curl -s -o /dev/null "${base_url}/api/v1/health" &
    done
    wait

    # Verify service still responds
    TESTS_RUN=$((TESTS_RUN + 1))
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "${base_url}/api/v1/health" 2>/dev/null)
    if [[ "$status" == "200" ]]; then
        log_pass "Service stable after 20 concurrent requests"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "Service unstable after concurrent requests (status: $status)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# ============================================================================
# Main
# ============================================================================

main() {
    echo -e "${BOLD}${CYAN}+---------------------------------------------------+${NC}"
    echo -e "${BOLD}${CYAN}|   NiAC - Comprehensive Smoke Test Suite            |${NC}"
    echo -e "${BOLD}${CYAN}|   Copyright (c) 2025 Mustard Seed Networks         |${NC}"
    echo -e "${BOLD}${CYAN}+---------------------------------------------------+${NC}"
    echo ""

    if [[ "$INSTALLED_MODE" == true ]]; then
        log_info "Mode: Testing installed service"
    else
        log_info "Mode: Testing local binary"
    fi

    test_binary
    test_version
    test_cli_help
    test_templates
    test_api_endpoints
    test_service_management
    test_concurrent_requests

    # Summary
    local end_time=$(date +%s)
    local elapsed=$((end_time - START_TIME))

    echo ""
    echo -e "${BOLD}${CYAN}=== TEST SUMMARY ===${NC}"
    echo -e "  Total:   ${TESTS_RUN}"
    echo -e "  ${GREEN}Passed:${NC}  ${TESTS_PASSED}"
    echo -e "  ${RED}Failed:${NC}  ${TESTS_FAILED}"
    echo -e "  ${YELLOW}Skipped:${NC} ${TESTS_SKIPPED}"
    echo -e "  Duration: ${elapsed}s"

    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "\n${RED}${BOLD}SMOKE TESTS FAILED${NC}"
        exit 1
    else
        echo -e "\n${GREEN}${BOLD}ALL SMOKE TESTS PASSED${NC}"
        exit 0
    fi
}

main "$@"
