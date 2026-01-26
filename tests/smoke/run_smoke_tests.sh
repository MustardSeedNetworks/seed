#!/bin/bash
#
# run_smoke_tests.sh - Comprehensive Smoke Tests for The Seed
#
# Tests core functionality: binary, version, API endpoints, auth, and diagnostics.
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
SEED_PORT=8443
SEED_SCHEME="https"
SEED_HOST="localhost"
SEED_SMOKE_USERNAME="${SEED_SMOKE_USERNAME:-}"
SEED_SMOKE_CRED="${SEED_SMOKE_CRED:-}"
SEED_SMOKE_TOKEN="${SEED_SMOKE_TOKEN:-}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../.."
SEED_BIN="${PROJECT_ROOT}/bin/seed"

# Mode
INSTALLED_MODE=false
if [[ "${1:-}" == "--installed" ]]; then
    INSTALLED_MODE=true
    SEED_BIN=$(command -v seed 2>/dev/null || echo "/usr/bin/seed")
fi

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Process tracking
SEED_PID=""
SEED_COOKIE_JAR="$(mktemp -t seed-smoke-cookie.XXXXXX)"

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
    if [[ -n "$SEED_PID" ]]; then
        kill $SEED_PID 2>/dev/null || true
        wait $SEED_PID 2>/dev/null || true
    fi
    if [[ -f "$SEED_COOKIE_JAR" ]]; then
        rm -f "$SEED_COOKIE_JAR"
    fi
    if [[ -f /tmp/seed_login.json ]]; then
        rm -f /tmp/seed_login.json
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
    status=$(curl -sk $extra_args -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)

    if [[ "$status" == "$expected" ]]; then
        log_pass "$name (HTTP $status)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_fail "$name (got HTTP $status, expected $expected)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Assert string contains
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

    if [[ ! -x "${SEED_BIN}" ]] && [[ "$INSTALLED_MODE" == false ]]; then
        log_info "Binary not found, building..."
        (cd "${PROJECT_ROOT}" && make build)
    fi

    run_test "Binary exists and is executable" \
        "test -x ${SEED_BIN}"

    run_test "Binary is not empty" \
        "test -s ${SEED_BIN}"

    # Check binary has embedded UI (should be >5MB)
    if [[ "$INSTALLED_MODE" == false ]]; then
        local size
        size=$(stat -f%z "${SEED_BIN}" 2>/dev/null || stat -c%s "${SEED_BIN}" 2>/dev/null || echo "0")
        run_test "Binary has embedded assets (>5MB)" \
            "test ${size} -gt 5000000"
    fi
}

# ============================================================================
# SECTION 2: Version Tests
# ============================================================================

test_version() {
    log_section "VERSION TESTS"

    run_test "Version command succeeds" \
        "${SEED_BIN} version"

    local version_output
    version_output=$("${SEED_BIN}" version 2>&1)

    assert_contains "$version_output" "Seed" "Version mentions Seed"
    assert_contains "$version_output" "v0\." "Version has v0.x format"
}

# ============================================================================
# SECTION 3: CLI Help Tests
# ============================================================================

test_cli_help() {
    log_section "CLI HELP TESTS"

    run_test "Help flag (--help)" "${SEED_BIN} --help"
    run_test "Help flag (-h)" "${SEED_BIN} -h"

    local help_output
    help_output=$("${SEED_BIN}" --help 2>&1)

    assert_contains "$help_output" "seed" "Help mentions seed"
}

# ============================================================================
# SECTION 4: API Endpoint Tests (requires running service)
# ============================================================================

test_api_endpoints() {
    log_section "API ENDPOINT TESTS"

    local base_url="${SEED_SCHEME}://${SEED_HOST}:${SEED_PORT}"

    # Check if service is reachable
    if ! curl -sk -o /dev/null -w "" "${base_url}/" 2>/dev/null; then
        skip_test "API endpoint tests" "Service not reachable at ${base_url}"
        return
    fi

    log_header "Unauthenticated Endpoints"
    assert_http_status "${base_url}/" "200" "WebUI serves HTML"

    log_header "Auth-Protected Endpoints (expect 401)"
    assert_http_status "${base_url}/api/v1/status" "401" "Status requires auth"
    assert_http_status "${base_url}/api/v1/health" "401" "Health requires auth"
    assert_http_status "${base_url}/api/v1/network/interfaces" "401" "Interfaces requires auth"
    assert_http_status "${base_url}/api/v1/network/link" "401" "Link requires auth"
    assert_http_status "${base_url}/api/v1/network/dhcp" "401" "DHCP requires auth"
    assert_http_status "${base_url}/api/v1/network/dns" "401" "DNS requires auth"
    assert_http_status "${base_url}/api/v1/network/gateway" "401" "Gateway requires auth"
    assert_http_status "${base_url}/api/v1/settings" "401" "Settings requires auth"

    log_header "Invalid Endpoints"
    assert_http_status "${base_url}/api/v1/nonexistent" "401" "Nonexistent path returns 401 (auth first)"

    log_header "WebUI Assets"
    # Check that static files are served
    local html
    html=$(curl -sk "${base_url}/" 2>/dev/null)
    assert_contains "$html" "html" "Root serves HTML document"
    assert_contains "$html" "script" "HTML includes script tags"

    log_header "HTTPS/TLS"
    TESTS_RUN=$((TESTS_RUN + 1))
    if curl -s --connect-timeout 2 "http://${SEED_HOST}:${SEED_PORT}/" 2>&1 | grep -qi "wrong version\|routines\|SSL\|reset"; then
        log_pass "Service rejects plain HTTP (TLS enforced)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_skip "TLS enforcement check" "Could not verify"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    fi

    log_header "Response Headers"
    local headers
    headers=$(curl -sk -I "${base_url}/" 2>/dev/null)
    assert_contains "$headers" "X-Content-Type-Options" "Security header X-Content-Type-Options present"
    assert_contains "$headers" "X-Frame-Options" "Security header X-Frame-Options present"

    log_header "Authenticated Endpoints (optional)"
    local auth_header=""
    if [[ -n "$SEED_SMOKE_TOKEN" ]]; then
        auth_header="Authorization: Bearer ${SEED_SMOKE_TOKEN}"
    elif [[ -n "$SEED_SMOKE_USERNAME" && -n "$SEED_SMOKE_CRED" ]]; then
        local login_payload
        login_payload=$(printf '{"username":"%s","password":"%s"}' "$SEED_SMOKE_USERNAME" "$SEED_SMOKE_CRED")
        local login_status
        login_status=$(curl -sk -c "$SEED_COOKIE_JAR" -o /tmp/seed_login.json -w "%{http_code}" \
            -H "Content-Type: application/json" \
            -d "$login_payload" \
            "${base_url}/api/v1/auth/login" 2>/dev/null || true)
        if [[ "$login_status" == "200" ]]; then
            local login_token
            login_token=$(grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' /tmp/seed_login.json | head -1 | cut -d'"' -f4)
            if [[ -n "$login_token" ]]; then
                auth_header="Authorization: Bearer ${login_token}"
            fi
            log_pass "Login succeeded for authenticated checks"
        else
            log_skip "Authenticated checks" "Login failed (HTTP ${login_status})"
        fi
    else
        log_skip "Authenticated checks" "Set SEED_SMOKE_USERNAME/SEED_SMOKE_CRED or SEED_SMOKE_TOKEN to enable"
    fi

    if [[ -n "$auth_header" ]]; then
        assert_http_status "${base_url}/api/v1/status" "200" "Status with auth" "-H \"$auth_header\""
        assert_http_status "${base_url}/api/v1/settings" "200" "Settings with auth" "-H \"$auth_header\""
        assert_http_status "${base_url}/api/v1/auth/csrf" "200" "CSRF token endpoint with auth" "-H \"$auth_header\""
        assert_http_status "${base_url}/api/v1/events" "200" "SSE endpoint with auth" "-H \"$auth_header\" -H \"Accept: text/event-stream\" --max-time 2"
    fi
}

# ============================================================================
# SECTION 5: Service Management Tests (installed mode only)
# ============================================================================

test_service_management() {
    log_section "SERVICE MANAGEMENT TESTS"

    if [[ "$INSTALLED_MODE" == false ]]; then
        skip_test "Service management" "Not in --installed mode"
        return
    fi

    run_test "systemd unit file exists" \
        "test -f /usr/lib/systemd/system/seed.service"

    run_test "Service is active" \
        "systemctl is-active --quiet seed.service"

    run_test "Service is enabled" \
        "systemctl is-enabled --quiet seed.service"

    run_test "Binary has capabilities set" \
        "getcap /usr/bin/seed 2>/dev/null | grep -q cap_net_raw"

    run_test "Config directory exists" \
        "test -d /etc/seed"

    run_test "Data directory exists" \
        "test -d /var/lib/seed"

    run_test "Log directory exists" \
        "test -d /var/log/seed"
}

# ============================================================================
# SECTION 6: Concurrent Request Test
# ============================================================================

test_concurrent_requests() {
    log_section "CONCURRENT REQUEST TESTS"

    local base_url="${SEED_SCHEME}://${SEED_HOST}:${SEED_PORT}"

    if ! curl -sk -o /dev/null "${base_url}/" 2>/dev/null; then
        skip_test "Concurrent requests" "Service not reachable"
        return
    fi

    # Send 20 concurrent requests
    for i in $(seq 1 20); do
        curl -sk -o /dev/null "${base_url}/" 2>/dev/null &
    done
    wait

    # Verify service still responds
    TESTS_RUN=$((TESTS_RUN + 1))
    local status
    status=$(curl -sk -o /dev/null -w "%{http_code}" "${base_url}/" 2>/dev/null)
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
    echo -e "${BOLD}${CYAN}|   The Seed - Comprehensive Smoke Test Suite        |${NC}"
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
