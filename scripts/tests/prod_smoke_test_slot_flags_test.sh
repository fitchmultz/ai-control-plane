#!/usr/bin/env bash
#
# Production Smoke Test - Slot Flags Test
#
# Purpose: Test slot-related flags for prod_smoke_test_impl.sh
#
# Responsibilities:
#   - Test --admin-url, --slot, --evidence-file flags work
#   - Test backward compatibility (no new flags)
#   - Test dry-run with new flags
#
# Non-scope:
#   - Does NOT test actual API calls
#   - Does NOT require running services
#
# Invariants:
#   - Tests are deterministic
#   - Tests verify CLI argument parsing
#
# Usage: prod_smoke_test_slot_flags_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/prod_smoke_test_slot_flags_test.sh
#   bash scripts/tests/prod_smoke_test_slot_flags_test.sh --help
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
Production Smoke Test - Slot Flags Test

Purpose: Test slot-related flags for prod_smoke_test_impl.sh

Usage: prod_smoke_test_slot_flags_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/prod_smoke_test_slot_flags_test.sh
  bash scripts/tests/prod_smoke_test_slot_flags_test.sh --help

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
SMOKE_TEST_SCRIPT="$REPO_ROOT/scripts/libexec/prod_smoke_test_impl.sh"

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

test_script_exists_and_executable() {
    if [[ -f "$SMOKE_TEST_SCRIPT" ]]; then
        pass "Script exists: $SMOKE_TEST_SCRIPT"
    else
        fail "Script not found: $SMOKE_TEST_SCRIPT"
        return 1
    fi

    if [[ -x "$SMOKE_TEST_SCRIPT" ]]; then
        pass "Script is executable"
    else
        fail "Script is not executable"
    fi
}

test_help_contains_slot_flag() {
    if "$SMOKE_TEST_SCRIPT" --help 2>&1 | grep -qE "\-\-slot"; then
        pass "--help contains --slot flag"
    else
        fail "--help missing --slot flag"
    fi
}

test_help_contains_evidence_file_flag() {
    if "$SMOKE_TEST_SCRIPT" --help 2>&1 | grep -qE "\-\-evidence-file"; then
        pass "--help contains --evidence-file flag"
    else
        fail "--help missing --evidence-file flag"
    fi
}

test_help_contains_admin_url_flag() {
    if "$SMOKE_TEST_SCRIPT" --help 2>&1 | grep -qE "\-\-admin-url"; then
        pass "--help contains --admin-url flag"
    else
        fail "--help missing --admin-url flag"
    fi
}

test_dry_run_with_slot_flag() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --slot standby 2>/dev/null || exit_code=$?

    # Should not be 64 (usage error)
    if [[ $exit_code -ne 64 ]]; then
        pass "--dry-run --slot standby accepted (exit code $exit_code)"
    else
        fail "--dry-run --slot standby should not exit 64"
    fi
}

test_dry_run_with_evidence_file() {
    local temp_evidence="/tmp/test_smoke_evidence_$$.log"
    local exit_code=0

    "$SMOKE_TEST_SCRIPT" --dry-run --evidence-file "$temp_evidence" 2>/dev/null || exit_code=$?

    # Should not be 64 (usage error)
    if [[ $exit_code -ne 64 ]]; then
        pass "--dry-run --evidence-file accepted (exit code $exit_code)"
    else
        fail "--dry-run --evidence-file should not exit 64"
    fi

    # Cleanup
    rm -f "$temp_evidence"
}

test_dry_run_with_admin_url() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --admin-url https://example.com 2>/dev/null || exit_code=$?

    # Should not be 64 (usage error)
    if [[ $exit_code -ne 64 ]]; then
        pass "--dry-run --admin-url accepted (exit code $exit_code)"
    else
        fail "--dry-run --admin-url should not exit 64"
    fi
}

test_backward_compatibility_no_new_flags() {
    # Test that old basic invocation still works
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: basic --dry-run works"
    else
        fail "Backward compatibility: basic --dry-run should not exit 64"
    fi
}

test_backward_compatibility_public_url() {
    # Test that --public-url still works
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --public-url http://127.0.0.1:4000 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: --public-url works"
    else
        fail "Backward compatibility: --public-url should not exit 64"
    fi
}

test_backward_compatibility_admin_port() {
    # Test that --admin-port still works
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --admin-port 4000 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: --admin-port works"
    else
        fail "Backward compatibility: --admin-port should not exit 64"
    fi
}

test_backward_compatibility_admin_host() {
    # Test that --admin-host still works
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --admin-host 127.0.0.1 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: --admin-host works"
    else
        fail "Backward compatibility: --admin-host should not exit 64"
    fi
}

test_slot_flag_validates_active() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --slot active 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "--slot active accepted (exit code $exit_code)"
    else
        fail "--slot active should not exit 64"
    fi
}

test_slot_flag_validates_standby() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --slot standby 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "--slot standby accepted (exit code $exit_code)"
    else
        fail "--slot standby should not exit 64"
    fi
}

test_combined_flags() {
    local temp_evidence="/tmp/test_smoke_evidence_combined_$$.log"
    local exit_code=0

    "$SMOKE_TEST_SCRIPT" \
        --dry-run \
        --slot standby \
        --admin-url https://example.com \
        --evidence-file "$temp_evidence" \
        2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Combined flags work (exit code $exit_code)"
    else
        fail "Combined flags should not exit 64"
    fi

    rm -f "$temp_evidence"
}

test_insecure_flag_backward_compat() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --insecure 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: --insecure works"
    else
        fail "Backward compatibility: --insecure should not exit 64"
    fi
}

test_timeout_flag_backward_compat() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --dry-run --timeout-seconds 30 2>/dev/null || exit_code=$?

    if [[ $exit_code -ne 64 ]]; then
        pass "Backward compatibility: --timeout-seconds works"
    else
        fail "Backward compatibility: --timeout-seconds should not exit 64"
    fi
}

test_help_exits_zero() {
    local exit_code=0
    "$SMOKE_TEST_SCRIPT" --help >/dev/null 2>&1 || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "--help exits with code 0"
    else
        fail "--help should exit 0, got $exit_code"
    fi
}

test_script_has_slot_handling() {
    # Check that the script actually processes slot-related arguments
    if grep -qE "\-\-slot|--evidence-file|--admin-url" "$SMOKE_TEST_SCRIPT"; then
        pass "Script contains slot-related flag handling"
    else
        fail "Script missing slot-related flag handling"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Production Smoke Test - Slot Flags Test ==="
echo ""

# Run all tests
test_script_exists_and_executable
test_help_contains_slot_flag
test_help_contains_evidence_file_flag
test_help_contains_admin_url_flag
test_dry_run_with_slot_flag
test_dry_run_with_evidence_file
test_dry_run_with_admin_url
test_backward_compatibility_no_new_flags
test_backward_compatibility_public_url
test_backward_compatibility_admin_port
test_backward_compatibility_admin_host
test_slot_flag_validates_active
test_slot_flag_validates_standby
test_combined_flags
test_insecure_flag_backward_compat
test_timeout_flag_backward_compat
test_help_exits_zero
test_script_has_slot_handling

# Summary
echo ""
echo "=== Test Summary ==="
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All slot flags tests passed"
    exit 0
else
    echo ""
    echo "✗ Some slot flags tests failed"
    exit 1
fi
