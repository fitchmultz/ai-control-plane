#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Install Script - CLI Contract Test
#
# Purpose: Verify CLI interface contract for host_install_impl.sh
#
# Responsibilities:
#   - Test --help contains required sections (Usage, Options, Examples, Exit Codes)
#   - Test missing command exits 64
#   - Test unknown command exits 64
#   - Test unknown flag exits 64
#   - Test missing option values exit 64
#
# Non-scope:
#   - Does NOT test actual systemd operations
#   - Does NOT test file system mutations
#
# Invariants:
#   - Tests are deterministic and don't require root/systemd
#   - Tests use isolated environment (PATH isolation)
#
# Usage: host_install_cli_contract_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_install_cli_contract_test.sh
#   bash scripts/tests/host_install_cli_contract_test.sh --help
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
Host Install CLI Contract Test

Purpose: Verify CLI interface contract for host_install_impl.sh

Usage: host_install_cli_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_install_cli_contract_test.sh
  bash scripts/tests/host_install_cli_contract_test.sh --help

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
HOST_INSTALL_SCRIPT="$REPO_ROOT/scripts/libexec/host_install_impl.sh"

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
    if "$HOST_INSTALL_SCRIPT" --help 2>&1 | grep -q "Usage:"; then
        pass "--help contains 'Usage:' section"
    else
        fail "--help missing 'Usage:' section"
    fi
}

test_help_contains_options() {
    if "$HOST_INSTALL_SCRIPT" --help 2>&1 | grep -qi "Options:"; then
        pass "--help contains 'Options:' section"
    else
        fail "--help missing 'Options:' section"
    fi
}

test_help_contains_examples() {
    if "$HOST_INSTALL_SCRIPT" --help 2>&1 | grep -qi "Examples:"; then
        pass "--help contains 'Examples:' section"
    else
        fail "--help missing 'Examples:' section"
    fi
}

test_help_contains_exit_codes() {
    if "$HOST_INSTALL_SCRIPT" --help 2>&1 | grep -qi "Exit Codes:"; then
        pass "--help contains 'Exit Codes:' section"
    else
        fail "--help missing 'Exit Codes:' section"
    fi
}

test_help_contains_commands() {
    local help_output
    help_output=$("$HOST_INSTALL_SCRIPT" --help 2>&1)

    local commands=("install" "uninstall" "status" "start" "stop" "restart" "render-unit")
    for cmd in "${commands[@]}"; do
        if echo "$help_output" | grep -q "$cmd"; then
            pass "--help mentions '$cmd' command"
        else
            fail "--help missing '$cmd' command"
        fi
    done
}

test_missing_command_exits_64() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Missing command exits with code 64"
    else
        fail "Missing command should exit 64, got $exit_code"
    fi
}

test_invalid_command_exits_64() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" invalid_cmd 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Invalid command exits with code 64"
    else
        fail "Invalid command should exit 64, got $exit_code"
    fi
}

test_unknown_flag_exits_64() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --unknown-flag 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Unknown flag exits with code 64"
    else
        fail "Unknown flag should exit 64, got $exit_code"
    fi
}

test_service_name_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --service-name 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--service-name without argument exits with code 64"
    else
        fail "--service-name without argument should exit 64, got $exit_code"
    fi
}

test_service_user_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --service-user 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--service-user without argument exits with code 64"
    else
        fail "--service-user without argument should exit 64, got $exit_code"
    fi
}

test_service_group_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --service-group 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--service-group without argument exits with code 64"
    else
        fail "--service-group without argument should exit 64, got $exit_code"
    fi
}

test_repo_root_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --repo-root 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--repo-root without argument exits with code 64"
    else
        fail "--repo-root without argument should exit 64, got $exit_code"
    fi
}

test_compose_file_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --compose-file 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--compose-file without argument exits with code 64"
    else
        fail "--compose-file without argument should exit 64, got $exit_code"
    fi
}

test_env_file_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --env-file 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--env-file without argument exits with code 64"
    else
        fail "--env-file without argument should exit 64, got $exit_code"
    fi
}

test_unit_dir_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --unit-dir 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--unit-dir without argument exits with code 64"
    else
        fail "--unit-dir without argument should exit 64, got $exit_code"
    fi
}

test_compose_env_file_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --compose-env-file 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--compose-env-file without argument exits with code 64"
    else
        fail "--compose-env-file without argument should exit 64, got $exit_code"
    fi
}

test_secrets_fetch_hook_option_requires_argument() {
    local exit_code=0
    "$HOST_INSTALL_SCRIPT" install --secrets-fetch-hook 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "--secrets-fetch-hook without argument exits with code 64"
    else
        fail "--secrets-fetch-hook without argument should exit 64, got $exit_code"
    fi
}

test_help_contains_secrets_options() {
    local help_output
    help_output=$("$HOST_INSTALL_SCRIPT" --help 2>&1)

    if echo "$help_output" | grep -q "compose-env-file"; then
        pass "--help mentions --compose-env-file option"
    else
        fail "--help missing --compose-env-file option"
    fi

    if echo "$help_output" | grep -q "secrets-fetch-hook"; then
        pass "--help mentions --secrets-fetch-hook option"
    else
        fail "--help missing --secrets-fetch-hook option"
    fi
}

test_help_contains_secrets_contract() {
    local help_output
    help_output=$("$HOST_INSTALL_SCRIPT" --help 2>&1)

    # Check that help references production secrets path
    if echo "$help_output" | grep -q "/etc/ai-control-plane/secrets.env"; then
        pass "--help references canonical secrets path"
    else
        fail "--help missing canonical secrets path reference"
    fi
}

test_all_commands_have_help() {
    local commands=("install" "uninstall" "status" "start" "stop" "restart" "render-unit")

    for cmd in "${commands[@]}"; do
        local exit_code=0
        "$HOST_INSTALL_SCRIPT" "$cmd" --help 2>/dev/null || exit_code=$?

        if [[ $exit_code -eq 0 ]]; then
            pass "$cmd --help exits 0"
        else
            fail "$cmd --help should exit 0, got $exit_code"
        fi
    done
}

test_default_env_file_is_canonical() {
    # Check that help shows the canonical secrets path as default
    local help_output
    help_output=$("$HOST_INSTALL_SCRIPT" --help 2>&1)

    # Extract default value for --env-file from help (should be /etc/ai-control-plane/secrets.env)
    if echo "$help_output" | grep -q "default.*ai-control-plane.*secrets.env"; then
        pass "Default env-file is canonical production path"
    else
        fail "Default env-file should be canonical production path"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Install CLI Contract Test ==="
echo ""

# Run all tests
test_help_contains_usage
test_help_contains_options
test_help_contains_examples
test_help_contains_exit_codes
test_help_contains_commands
test_help_contains_secrets_options
test_help_contains_secrets_contract
test_missing_command_exits_64
test_invalid_command_exits_64
test_unknown_flag_exits_64
test_service_name_option_requires_argument
test_service_user_option_requires_argument
test_service_group_option_requires_argument
test_repo_root_option_requires_argument
test_compose_file_option_requires_argument
test_env_file_option_requires_argument
test_unit_dir_option_requires_argument
test_compose_env_file_option_requires_argument
test_secrets_fetch_hook_option_requires_argument
test_all_commands_have_help
test_default_env_file_is_canonical

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
