#!/usr/bin/env bash
set -euo pipefail

# Supply Chain Allowlist Expiry Check Contract Tests
#
# Purpose:
#   Validate scripts/libexec/check_supply_chain_allowlist_expiry_impl.py behavior.
#
# Responsibilities:
#   - Verify healthy policy exits 0
#   - Verify warning-window entries still exit 0
#   - Verify fail-window/invalid-date entries exit 1
#
# Non-scope:
#   - Does not perform live vulnerability scanning
#
# Exit codes:
#   0  all tests passed
#   1  one or more tests failed

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: supply_chain_allowlist_expiry_check_test.sh [--help]

Run contract tests for scripts/libexec/check_supply_chain_allowlist_expiry_impl.py.

Examples:
  bash scripts/tests/supply_chain_allowlist_expiry_check_test.sh
  bash scripts/tests/supply_chain_allowlist_expiry_check_test.sh --help

Exit codes:
  0  all tests passed
  1  one or more tests failed
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
SCRIPT_UNDER_TEST="$REPO_ROOT/scripts/libexec/check_supply_chain_allowlist_expiry_impl.py"

if [[ ! -f "$SCRIPT_UNDER_TEST" ]]; then
    echo "✗ Missing script under test: $SCRIPT_UNDER_TEST"
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    echo "⚠ python3 not available; skipping"
    exit 0
fi

test_results_init

run_case() {
    local name="$1"
    local expected_exit="$2"
    local policy_file="$3"
    local safe_name="${name// /_}"
    local stdout_file="$TMP_ROOT/$safe_name.out"
    local stderr_file="$TMP_ROOT/$safe_name.err"

    set +e
    python3 "$SCRIPT_UNDER_TEST" \
        --policy "$policy_file" \
        --warn-days 45 \
        --fail-days 14 \
        --today "$FIXED_TODAY" \
        >"$stdout_file" 2>"$stderr_file"
    local exit_code=$?
    set -e

    if [[ "$exit_code" -eq "$expected_exit" ]]; then
        test_pass "$name"
    else
        test_fail "$name (expected exit $expected_exit, got $exit_code)"
        echo "--- stdout ---"
        cat "$stdout_file" || true
        echo "--- stderr ---"
        cat "$stderr_file" || true
    fi
}

test_fixture_init acp-supply-expiry-test
TMP_ROOT="${TEST_TMP_ROOT}"

FIXED_TODAY="2026-03-07"
warn_date="2026-03-27"
healthy_date="2026-07-05"
fail_date="2026-03-07"

cat >"$TMP_ROOT/healthy.json" <<EOF
{"allowlist":[{"id":"CVE-1","package":"pkg","expires_on":"$healthy_date","ticket":"SEC-1"}]}
EOF

cat >"$TMP_ROOT/warn.json" <<EOF
{"allowlist":[{"id":"CVE-2","package":"pkg","expires_on":"$warn_date","ticket":"SEC-2"}]}
EOF

cat >"$TMP_ROOT/fail.json" <<EOF
{"allowlist":[{"id":"CVE-3","package":"pkg","expires_on":"$fail_date","ticket":"SEC-3"}]}
EOF

cat >"$TMP_ROOT/invalid-date.json" <<'EOF'
{"allowlist":[{"id":"CVE-4","package":"pkg","expires_on":"not-a-date","ticket":"SEC-4"}]}
EOF

echo "Supply Chain Allowlist Expiry Check Tests"
run_case "healthy policy passes" 0 "$TMP_ROOT/healthy.json"
run_case "warn-window policy passes with warning" 0 "$TMP_ROOT/warn.json"
run_case "fail-window policy fails" 1 "$TMP_ROOT/fail.json"
run_case "invalid date fails" 1 "$TMP_ROOT/invalid-date.json"

echo
echo "Summary"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi

exit 0
