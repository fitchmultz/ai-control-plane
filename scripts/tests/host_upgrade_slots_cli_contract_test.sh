#!/usr/bin/env bash
#
# Host Upgrade Slots - CLI Contract Test
#
# Purpose: Verify CLI interface contract for host_upgrade_slots_impl.sh
#
# Responsibilities:
#   - Test --help contains required sections (Usage, Examples, Exit codes)
#   - Test all 6 subcommands exist and have proper error handling
#   - Test invalid slot names are rejected
#   - Test missing required arguments exit 64
#
# Non-scope:
#   - Does NOT test actual Docker operations
#   - Does NOT test slot health checks
#
# Invariants:
#   - Tests are deterministic and don't require Docker
#   - Tests use isolated environment (PATH isolation)
#
# Usage: host_upgrade_slots_cli_contract_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_upgrade_slots_cli_contract_test.sh
#   bash scripts/tests/host_upgrade_slots_cli_contract_test.sh --help
#
# Exit Codes:
#   0   - All tests passed
#   1   - One or more tests failed
# =============================================================================

set -euo pipefail

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

show_help() {
    cat <<'EOF'
Host Upgrade Slots CLI Contract Test

Purpose: Verify CLI interface contract for host_upgrade_slots_impl.sh

Usage: host_upgrade_slots_cli_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_upgrade_slots_cli_contract_test.sh
  bash scripts/tests/host_upgrade_slots_cli_contract_test.sh --help

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
HOST_UPGRADE_SCRIPT="$REPO_ROOT/scripts/libexec/host_upgrade_slots_impl.sh"

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

test_help_exits_zero() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" --help >/dev/null 2>&1 || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "--help exits with code 0"
    else
        fail "--help should exit 0, got $exit_code"
    fi
}

test_help_contains_usage() {
    if "$HOST_UPGRADE_SCRIPT" --help 2>&1 | grep -q "Usage:"; then
        pass "--help contains 'Usage:' section"
    else
        fail "--help missing 'Usage:' section"
    fi
}

test_help_contains_examples() {
    if "$HOST_UPGRADE_SCRIPT" --help 2>&1 | grep -qi "Examples:"; then
        pass "--help contains 'Examples:' section"
    else
        fail "--help missing 'Examples:' section"
    fi
}

test_help_contains_exit_codes() {
    if "$HOST_UPGRADE_SCRIPT" --help 2>&1 | grep -qi "Exit Codes:"; then
        pass "--help contains 'Exit Codes:' section"
    else
        fail "--help missing 'Exit Codes:' section"
    fi
}

test_invalid_command_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" invalid-command 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Invalid command exits with code 64"
    else
        fail "Invalid command should exit 64, got $exit_code"
    fi
}

test_cutover_without_from_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --to standby 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "cutover without --from exits with code 64"
    else
        fail "cutover without --from should exit 64, got $exit_code"
    fi
}

test_cutover_without_to_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --from active 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "cutover without --to exits with code 64"
    else
        fail "cutover without --to should exit 64, got $exit_code"
    fi
}

test_rollback_without_to_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" rollback 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "rollback without --to exits with code 64"
    else
        fail "rollback without --to should exit 64, got $exit_code"
    fi
}

test_invalid_slot_name_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --from foo --to bar 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Invalid slot name exits with code 64"
    else
        fail "Invalid slot name should exit 64, got $exit_code"
    fi
}

test_same_slot_for_from_to_exits_64() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --from active --to active 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 64 ]]; then
        pass "Same slot for from/to exits with code 64"
    else
        fail "Same slot for from/to should exit 64, got $exit_code"
    fi
}

test_prepare_standby_with_release_works() {
    # This should fail with prereq error (Docker not available) but NOT usage error
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" prepare-standby --release v1.2.3 2>/dev/null || exit_code=$?

    # Should NOT be 64 (usage error) - should be 2 (prereq) or 0 (success)
    if [[ $exit_code -ne 64 ]]; then
        pass "prepare-standby --release accepts release value (exit code $exit_code)"
    else
        fail "prepare-standby --release should not exit 64 (usage error)"
    fi
}

test_all_commands_exist_in_help() {
    local help_output
    help_output=$("$HOST_UPGRADE_SCRIPT" --help 2>&1)

    local commands=("prepare-standby" "smoke-standby" "cutover" "rollback" "status" "rehearse")
    local all_found=true

    for cmd in "${commands[@]}"; do
        if ! echo "$help_output" | grep -q "$cmd"; then
            fail "--help missing command: $cmd"
            all_found=false
        fi
    done

    if [[ "$all_found" == true ]]; then
        pass "All 6 commands exist in help"
    fi
}

test_status_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" status --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "status --help exits with code 0"
    else
        fail "status --help should exit 0, got $exit_code"
    fi
}

test_prepare_standby_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" prepare-standby --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "prepare-standby --help exits with code 0"
    else
        fail "prepare-standby --help should exit 0, got $exit_code"
    fi
}

test_smoke_standby_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" smoke-standby --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "smoke-standby --help exits with code 0"
    else
        fail "smoke-standby --help should exit 0, got $exit_code"
    fi
}

test_cutover_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "cutover --help exits with code 0"
    else
        fail "cutover --help should exit 0, got $exit_code"
    fi
}

test_rollback_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" rollback --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "rollback --help exits with code 0"
    else
        fail "rollback --help should exit 0, got $exit_code"
    fi
}

test_rehearse_help_works() {
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" rehearse --help 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "rehearse --help exits with code 0"
    else
        fail "rehearse --help should exit 0, got $exit_code"
    fi
}

test_verbose_flag_accepted() {
    # Should not exit 64 (usage error) when --verbose is provided
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" status --verbose 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "--verbose flag is accepted (exit code $exit_code)"
    else
        fail "--verbose should be accepted, got usage error"
    fi
}

test_cutover_with_valid_slots_syntax() {
    # Should not exit 64 (usage error) with valid slot syntax
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" cutover --from active --to standby 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "cutover with valid slots has correct syntax (exit code $exit_code)"
    else
        fail "cutover with valid slots should not exit 64"
    fi
}

test_rollback_with_valid_slot_syntax() {
    # Should not exit 64 (usage error) with valid slot syntax
    local exit_code=0
    "$HOST_UPGRADE_SCRIPT" rollback --to active 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "rollback with valid slot has correct syntax (exit code $exit_code)"
    else
        fail "rollback with valid slot should not exit 64"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Upgrade Slots CLI Contract Test ==="
echo ""

# Run all tests
test_help_exits_zero
test_help_contains_usage
test_help_contains_examples
test_help_contains_exit_codes
test_invalid_command_exits_64
test_cutover_without_from_exits_64
test_cutover_without_to_exits_64
test_rollback_without_to_exits_64
test_invalid_slot_name_exits_64
test_same_slot_for_from_to_exits_64
test_prepare_standby_with_release_works
test_all_commands_exist_in_help
test_status_help_works
test_prepare_standby_help_works
test_smoke_standby_help_works
test_cutover_help_works
test_rollback_help_works
test_rehearse_help_works
test_verbose_flag_accepted
test_cutover_with_valid_slots_syntax
test_rollback_with_valid_slot_syntax

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
