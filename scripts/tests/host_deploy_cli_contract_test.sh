#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Deploy Script - CLI Contract Test
#
# Purpose: Verify CLI interface contract for host_deploy_impl.sh
#
# Responsibilities:
#   - Test --help contains required sections (Usage, Examples, Exit codes)
#   - Test missing required arguments exit 64
#   - Test unknown options exit 64
#   - Test invalid command exits 64
#
# Non-scope:
#   - Does NOT test actual Ansible execution
#   - Does NOT test host connectivity
#
# Invariants:
#   - Tests are deterministic and don't require network access
#   - Tests use isolated environment (PATH isolation)
#
# Usage: host_deploy_cli_contract_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_deploy_cli_contract_test.sh
#   bash scripts/tests/host_deploy_cli_contract_test.sh --help
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
Host Deploy CLI Contract Test

Purpose: Verify CLI interface contract for host_deploy_impl.sh

Usage: host_deploy_cli_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_deploy_cli_contract_test.sh
  bash scripts/tests/host_deploy_cli_contract_test.sh --help

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
HOST_DEPLOY_SCRIPT="$REPO_ROOT/scripts/libexec/host_deploy_impl.sh"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TMP_DIR="$(mktemp -d)"
TEST_INVENTORY_PATH="$TMP_DIR/test_inventory.yml"
printf '%s\n' "all:" >"$TEST_INVENTORY_PATH"

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

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
    if "$HOST_DEPLOY_SCRIPT" --help 2>&1 | grep -q "Usage:"; then
        pass "--help contains 'Usage:' section"
    else
        fail "--help missing 'Usage:' section"
    fi
}

test_help_contains_examples() {
    if "$HOST_DEPLOY_SCRIPT" --help 2>&1 | grep -qi "Examples:"; then
        pass "--help contains 'Examples:' section"
    else
        fail "--help missing 'Examples:' section"
    fi
}

test_help_contains_exit_codes() {
    if "$HOST_DEPLOY_SCRIPT" --help 2>&1 | grep -qi "Exit Codes:"; then
        pass "--help contains 'Exit Codes:' section"
    else
        fail "--help missing 'Exit Codes:' section"
    fi
}

test_help_contains_commands() {
    local help_output
    help_output=$("$HOST_DEPLOY_SCRIPT" --help 2>&1)

    if echo "$help_output" | grep -q "check"; then
        pass "--help mentions 'check' command"
    else
        fail "--help missing 'check' command"
    fi

    if echo "$help_output" | grep -q "apply"; then
        pass "--help mentions 'apply' command"
    else
        fail "--help missing 'apply' command"
    fi
}

test_missing_command_exits_64() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Missing command exits with code 64"
    else
        fail "Missing command should exit 64, got $exit_code"
    fi
}

test_invalid_command_exits_64() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" invalid_cmd 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Invalid command exits with code 64"
    else
        fail "Invalid command should exit 64, got $exit_code"
    fi
}

test_missing_inventory_exits_64() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Missing --inventory exits with code 64"
    else
        fail "Missing --inventory should exit 64, got $exit_code"
    fi
}

test_unknown_option_exits_64() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --unknown-option 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Unknown option exits with code 64"
    else
        fail "Unknown option should exit 64, got $exit_code"
    fi
}

test_invalid_tls_mode_exits_64() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory "$TEST_INVENTORY_PATH" --tls-mode invalid 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Invalid --tls-mode exits with code 64"
    else
        fail "Invalid --tls-mode should exit 64, got $exit_code"
    fi
}

test_inventory_option_requires_argument() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--inventory without argument exits with code 64"
    else
        fail "--inventory without argument should exit 64, got $exit_code"
    fi
}

test_limit_option_requires_argument() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory "$TEST_INVENTORY_PATH" --limit 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--limit without argument exits with code 64"
    else
        fail "--limit without argument should exit 64, got $exit_code"
    fi
}

test_repo_path_option_requires_argument() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory "$TEST_INVENTORY_PATH" --repo-path 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--repo-path without argument exits with code 64"
    else
        fail "--repo-path without argument should exit 64, got $exit_code"
    fi
}

test_env_file_option_requires_argument() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory "$TEST_INVENTORY_PATH" --env-file 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--env-file without argument exits with code 64"
    else
        fail "--env-file without argument should exit 64, got $exit_code"
    fi
}

test_public_url_option_requires_argument() {
    local exit_code=0
    "$HOST_DEPLOY_SCRIPT" check --inventory "$TEST_INVENTORY_PATH" --public-url 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--public-url without argument exits with code 64"
    else
        fail "--public-url without argument should exit 64, got $exit_code"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Deploy CLI Contract Test ==="
echo ""

# Run all tests
test_help_contains_usage
test_help_contains_examples
test_help_contains_exit_codes
test_help_contains_commands
test_missing_command_exits_64
test_invalid_command_exits_64
test_missing_inventory_exits_64
test_unknown_option_exits_64
test_invalid_tls_mode_exits_64
test_inventory_option_requires_argument
test_limit_option_requires_argument
test_repo_path_option_requires_argument
test_env_file_option_requires_argument
test_public_url_option_requires_argument

# Summary
echo ""
echo "=== Test Summary ==="
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All CLI contract tests passed"
    exit 0
else
    echo ""
    echo "✗ Some CLI contract tests failed"
    exit 1
fi
