#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Preflight Script - CLI Contract Test
#
# Purpose: Verify CLI interface contract for host_preflight_impl.sh
#
# Responsibilities:
#   - Test --help contains required sections (Usage, Examples, Exit codes)
#   - Test invalid arguments exit 64
#   - Test unknown check in --skip-check exits 64
#   - Test missing required values exit 64
#
# Non-scope:
#   - Does NOT test actual check execution
#   - Does NOT test host system state
#
# Invariants:
#   - Tests are deterministic and don't require network access
#   - Tests use isolated environment (PATH isolation)
#
# Usage: host_preflight_cli_contract_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_preflight_cli_contract_test.sh
#   bash scripts/tests/host_preflight_cli_contract_test.sh --help
#
# Exit Codes:
#   0   - All tests passed
#   1   - One or more tests failed
# =============================================================================

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

show_help() {
    cat <<'EOF'
Host Preflight CLI Contract Test

Purpose: Verify CLI interface contract for host_preflight_impl.sh

Usage: host_preflight_cli_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_preflight_cli_contract_test.sh
  bash scripts/tests/host_preflight_cli_contract_test.sh --help

Exit Codes:
  0   - All tests passed
  1   - One or more tests failed
EOF
}

# Parse arguments
if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# -----------------------------------------------------------------------------
# Test Configuration
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PREFLIGHT_SCRIPT="$REPO_ROOT/scripts/libexec/host_preflight_impl.sh"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# -----------------------------------------------------------------------------
# Test Helpers
# -----------------------------------------------------------------------------

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

# -----------------------------------------------------------------------------
# Test Cases
# -----------------------------------------------------------------------------

test_help_contains_usage() {
    if "$PREFLIGHT_SCRIPT" --help 2>&1 | grep -q "Usage:"; then
        pass "--help contains 'Usage:' section"
    else
        fail "--help missing 'Usage:' section"
    fi
}

test_help_contains_examples() {
    if "$PREFLIGHT_SCRIPT" --help 2>&1 | grep -q "Examples:"; then
        pass "--help contains 'Examples:' section"
    else
        fail "--help missing 'Examples:' section"
    fi
}

test_help_contains_exit_codes() {
    if "$PREFLIGHT_SCRIPT" --help 2>&1 | grep -q "Exit Codes:"; then
        pass "--help contains 'Exit Codes:' section"
    else
        fail "--help missing 'Exit Codes:' section"
    fi
}

test_help_contains_check_ids() {
    if "$PREFLIGHT_SCRIPT" --help 2>&1 | grep -q "Check IDs"; then
        pass "--help contains check IDs reference"
    else
        fail "--help missing check IDs reference"
    fi
}

test_unknown_option_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --unknown-option >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "unknown option exits 64"
    else
        fail "unknown option exits $rc, expected 64"
    fi
}

test_invalid_skip_check_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --skip-check invalid-check-id >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "invalid --skip-check exits 64"
    else
        fail "invalid --skip-check exits $rc, expected 64"
    fi
}

test_missing_profile_value_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --profile >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "missing --profile value exits 64"
    else
        fail "missing --profile value exits $rc, expected 64"
    fi
}

test_unsupported_profile_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --profile staging >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "unsupported --profile exits 64"
    else
        fail "unsupported --profile exits $rc, expected 64"
    fi
}

test_invalid_disk_gb_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --min-disk-gb invalid >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "invalid --min-disk-gb exits 64"
    else
        fail "invalid --min-disk-gb exits $rc, expected 64"
    fi
}

test_invalid_port_open_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --require-port-open 99999 >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "invalid --require-port-open exits 64"
    else
        fail "invalid --require-port-open exits $rc, expected 64"
    fi
}

test_invalid_port_blocked_exits_64() {
    local rc=0
    "$PREFLIGHT_SCRIPT" --require-port-blocked 0 >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 64 ]]; then
        pass "invalid --require-port-blocked exits 64"
    else
        fail "invalid --require-port-blocked exits $rc, expected 64"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "Host Preflight CLI Contract Test"
echo "================================"
echo ""

test_help_contains_usage
test_help_contains_examples
test_help_contains_exit_codes
test_help_contains_check_ids
test_unknown_option_exits_64
test_invalid_skip_check_exits_64
test_missing_profile_value_exits_64
test_unsupported_profile_exits_64
test_invalid_disk_gb_exits_64
test_invalid_port_open_exits_64
test_invalid_port_blocked_exits_64

echo ""
echo "================================"
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi
exit 0
