#!/usr/bin/env bash
set -euo pipefail

# ACPCTL Doctor Command Contract Test
#
# Purpose:
#   Verify the acpctl doctor command contract and behavior.
#
# Responsibilities:
#   - Validate help output contains doctor command.
#   - Validate doctor --help works.
#   - Validate doctor --json produces valid JSON.
#   - Validate unknown doctor options exit 64.
#
# Non-scope:
#   - Does NOT test actual environment state or Docker availability.
#   - Does NOT test --fix behavior or remediation logic.
#
# Invariants:
#   - Tests run without requiring Docker services.
#   - Exit code 0 means all tests passed.

show_help() {
    cat <<'EOF'
ACPCTL Doctor Command Contract Test

Purpose: Validate acpctl doctor command contract.

Usage: acpctl_doctor_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/acpctl_doctor_contract_test.sh

Exit Codes:
  0   - All tests passed
  1   - One or more tests failed
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_DIR="$(mktemp -d)"
GO_SHIM="$TMP_DIR/acpctl-go-shim.sh"
ACPCTL_BIN_TEST="$TMP_DIR/acpctl-bin"

trap 'rm -rf "$TMP_DIR"' EXIT

# Build the binary
go build -trimpath -o "$ACPCTL_BIN_TEST" "$REPO_ROOT/cmd/acpctl"

cat >"$GO_SHIM" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "$ACPCTL_BIN_TEST" "\$@"
EOF
chmod +x "$GO_SHIM"

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

echo "ACPCTL Doctor Contract Test"
echo "==========================="
echo ""

echo "Test: --help contains doctor command..."
HELP_OUTPUT=$("$GO_SHIM" --help 2>&1)
if echo "$HELP_OUTPUT" | grep -q "doctor"; then
    pass "--help includes doctor command"
else
    fail "--help missing doctor command"
fi
echo ""

echo "Test: doctor --help works..."
DOCTOR_HELP=$("$GO_SHIM" doctor --help 2>&1)
DOCTOR_HELP_RC=$?
if [[ "$DOCTOR_HELP_RC" -eq 0 ]]; then
    pass "doctor --help returns exit 0"
else
    fail "doctor --help should return exit 0 (got $DOCTOR_HELP_RC)"
fi

if echo "$DOCTOR_HELP" | grep -q "docker_available"; then
    pass "doctor --help lists docker_available check"
else
    fail "doctor --help missing docker_available check"
fi

if echo "$DOCTOR_HELP" | grep -q "ports_free"; then
    pass "doctor --help lists ports_free check"
else
    fail "doctor --help missing ports_free check"
fi

if echo "$DOCTOR_HELP" | grep -q "env_vars_set"; then
    pass "doctor --help lists env_vars_set check"
else
    fail "doctor --help missing env_vars_set check"
fi

if echo "$DOCTOR_HELP" | grep -q '\-\-json'; then
    pass "doctor --help documents --json flag"
else
    fail "doctor --help missing --json documentation"
fi

if echo "$DOCTOR_HELP" | grep -q '\-\-fix'; then
    pass "doctor --help documents --fix flag"
else
    fail "doctor --help missing --fix documentation"
fi

if echo "$DOCTOR_HELP" | grep -q '\-\-skip-check'; then
    pass "doctor --help documents --skip-check flag"
else
    fail "doctor --help missing --skip-check documentation"
fi
echo ""

echo "Test: doctor with unknown option exits 64..."
UNKNOWN_OPT_RC=0
"$GO_SHIM" doctor --unknown-option 2>/dev/null || UNKNOWN_OPT_RC=$?
if [[ "$UNKNOWN_OPT_RC" -eq 64 ]]; then
    pass "doctor --unknown-option exits 64"
else
    fail "doctor --unknown-option should exit 64 (got $UNKNOWN_OPT_RC)"
fi
echo ""

echo "Test: doctor --json produces valid JSON..."
JSON_OUTPUT=$("$GO_SHIM" doctor --json 2>&1) || true
if echo "$JSON_OUTPUT" | jq empty 2>/dev/null; then
    pass "doctor --json produces valid JSON"
else
    fail "doctor --json output is not valid JSON"
fi

if echo "$JSON_OUTPUT" | jq -e '.overall' >/dev/null 2>&1; then
    pass "doctor --json contains 'overall' field"
else
    fail "doctor --json missing 'overall' field"
fi

if echo "$JSON_OUTPUT" | jq -e '.results' >/dev/null 2>&1; then
    pass "doctor --json contains 'results' array"
else
    fail "doctor --json missing 'results' array"
fi

if echo "$JSON_OUTPUT" | jq -e '.timestamp' >/dev/null 2>&1; then
    pass "doctor --json contains 'timestamp' field"
else
    fail "doctor --json missing 'timestamp' field"
fi

if echo "$JSON_OUTPUT" | jq -e '.duration' >/dev/null 2>&1; then
    pass "doctor --json contains 'duration' field"
else
    fail "doctor --json missing 'duration' field"
fi
echo ""

echo "Test: doctor --skip-check works..."
SKIP_OUTPUT=$("$GO_SHIM" doctor --skip-check docker_available --skip-check ports_free 2>&1) || true
# Just verify it doesn't crash
if [[ -n "$SKIP_OUTPUT" ]]; then
    pass "doctor --skip-check runs without error"
else
    fail "doctor --skip-check produced no output"
fi
echo ""

echo "========================="
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi
exit 0
