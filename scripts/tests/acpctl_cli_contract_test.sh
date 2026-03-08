#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Contract Test
#
# Purpose:
#   Verify the operator CLI contract for scripts/acpctl.sh and cmd/acpctl.
#
# Responsibilities:
#   - Validate help output sections and command visibility.
#   - Validate delegated command-to-make target mapping.
#   - Validate typed command paths stay make-independent.
#   - Validate ACPCTL_MAKE_BIN override behavior for testability.
#   - Validate CI runtime-scope command path remains make-independent.
#
# Non-scope:
#   - Does NOT test real Docker services or host workflows.
#   - Does NOT test every Make target implementation.
#
# Invariants:
#   - Tests run with isolated temporary stubs.
#   - No network access is required.
#   - Exit code 0 means all tests passed.

show_help() {
    cat <<'EOF'
ACPCTL CLI Contract Test

Purpose: Validate scripts/acpctl.sh and typed acpctl command delegation.

Usage: acpctl_cli_contract_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/acpctl_cli_contract_test.sh
  bash scripts/tests/acpctl_cli_contract_test.sh --help

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
SCRIPT_UNDER_TEST="$REPO_ROOT/scripts/acpctl.sh"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_DIR="$(mktemp -d)"
CAPTURE_FILE="$TMP_DIR/make-args.txt"
GO_SHIM="$TMP_DIR/acpctl-go-shim.sh"
ACPCTL_BIN_TEST="$TMP_DIR/acpctl-bin"
MAKE_STUB="$TMP_DIR/make-stub.sh"

trap 'rm -rf "$TMP_DIR"' EXIT

go build -trimpath -o "$ACPCTL_BIN_TEST" "$REPO_ROOT/cmd/acpctl"

cat >"$GO_SHIM" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "$ACPCTL_BIN_TEST" "\$@"
EOF
chmod +x "$GO_SHIM"

cat >"$MAKE_STUB" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
capture_file="${ACPCTL_TEST_CAPTURE_FILE:?missing ACPCTL_TEST_CAPTURE_FILE}"
printf '%s\n' "$@" >"$capture_file"
exit "${ACPCTL_TEST_MAKE_EXIT_CODE:-0}"
EOF
chmod +x "$MAKE_STUB"

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

run_with_make_stub() {
    : >"$CAPTURE_FILE"
    ACPCTL_BIN="$GO_SHIM" \
        ACPCTL_MAKE_BIN="$MAKE_STUB" \
        ACPCTL_TEST_CAPTURE_FILE="$CAPTURE_FILE" \
        ACP_REPO_ROOT="$REPO_ROOT" \
        "$SCRIPT_UNDER_TEST" "$@"
}

assert_make_target() {
    local expected_target="$1"
    local description="$2"
    shift 2

    local rc=0
    if ! run_with_make_stub "$@" >/dev/null 2>&1; then
        rc=$?
    fi
    if [[ "$rc" -ne 0 ]]; then
        fail "$description (command failed with exit $rc)"
        return
    fi

    local actual_target
    actual_target="$(head -n1 "$CAPTURE_FILE" || true)"
    if [[ "$actual_target" == "$expected_target" ]]; then
        pass "$description"
    else
        fail "$description (expected target '$expected_target', got '$actual_target')"
    fi
}

assert_typed_no_delegation() {
    local description="$1"
    shift

    : >"$CAPTURE_FILE"
    local rc=0
    if ! run_with_make_stub "$@" >/dev/null 2>&1; then
        rc=$?
    fi
    if [[ "$rc" -ne 0 ]]; then
        fail "$description (command failed with exit $rc)"
        return
    fi

    if [[ -s "$CAPTURE_FILE" ]]; then
        fail "$description (unexpected make delegation detected)"
        return
    fi

    pass "$description"
}

echo "ACPCTL CLI Contract Test"
echo "========================"
echo ""

echo "Test: --help contains required sections and flow commands..."
HELP_OUTPUT="$(ACPCTL_BIN="$GO_SHIM" "$SCRIPT_UNDER_TEST" --help 2>&1)"
if echo "$HELP_OUTPUT" | grep -q "Usage: acpctl"; then
    pass "--help includes Usage"
else
    fail "--help missing Usage"
fi
if echo "$HELP_OUTPUT" | grep -q "Commands:"; then
    pass "--help includes Commands section"
else
    fail "--help missing Commands section"
fi
if echo "$HELP_OUTPUT" | grep -q "Examples:"; then
    pass "--help includes Examples section"
else
    fail "--help missing Examples section"
fi
if echo "$HELP_OUTPUT" | grep -q "env"; then
    pass "--help lists env command"
else
    fail "--help missing env command"
fi
if echo "$HELP_OUTPUT" | grep -q "chargeback"; then
    pass "--help lists chargeback command"
else
    fail "--help missing chargeback command"
fi
if echo "$HELP_OUTPUT" | grep -q "deploy"; then
    pass "--help lists deploy command"
else
    fail "--help missing deploy command"
fi
if echo "$HELP_OUTPUT" | grep -q "validate"; then
    pass "--help lists validate command"
else
    fail "--help missing validate command"
fi
if echo "$HELP_OUTPUT" | grep -q "db"; then
    pass "--help lists db command"
else
    fail "--help missing db command"
fi
if echo "$HELP_OUTPUT" | grep -q "key"; then
    pass "--help lists key command"
else
    fail "--help missing key command"
fi
if echo "$HELP_OUTPUT" | grep -q "host"; then
    pass "--help lists host command"
else
    fail "--help missing host command"
fi
if echo "$HELP_OUTPUT" | grep -q "demo"; then
    pass "--help lists demo command"
else
    fail "--help missing demo command"
fi
if echo "$HELP_OUTPUT" | grep -q "terraform"; then
    pass "--help lists terraform command"
else
    fail "--help missing terraform command"
fi
if echo "$HELP_OUTPUT" | grep -q "doctor"; then
    pass "--help lists doctor command"
else
    fail "--help missing doctor command"
fi
echo ""

echo "Test: env command is typed and exposes help..."
ENV_HELP_OUTPUT="$(ACPCTL_BIN="$GO_SHIM" "$SCRIPT_UNDER_TEST" env get --help 2>&1)"
if echo "$ENV_HELP_OUTPUT" | grep -q "Usage: acpctl env get"; then
    pass "env get help includes usage"
else
    fail "env get help missing usage"
fi
assert_typed_no_delegation "env get help stays make-independent" env get --help
echo ""

echo "Test: unknown command exits 64..."
UNKNOWN_RC=0
ACPCTL_BIN="$GO_SHIM" "$SCRIPT_UNDER_TEST" unknown-command >/dev/null 2>&1 || UNKNOWN_RC=$?
if [[ "$UNKNOWN_RC" -eq 64 ]]; then
    pass "unknown command exits 64"
else
    fail "unknown command should exit 64 (got $UNKNOWN_RC)"
fi
echo ""

echo "Test: flow commands delegate to expected make targets..."
assert_make_target "up" "deploy up delegates to make up" deploy up
assert_make_target "db-status" "db status delegates to make db-status" db status
assert_make_target "host-preflight" "host preflight delegates to make host-preflight" host preflight
assert_make_target "demo-all" "demo all delegates to make demo-all" demo all
assert_make_target "up-tls" "deploy up-tls delegates to make up-tls" deploy up-tls
assert_make_target "tf-plan" "terraform plan delegates to make tf-plan" terraform plan
echo ""

echo "Test: typed command paths do not delegate to make..."
assert_typed_no_delegation "validate config runs via typed path" validate config
assert_typed_no_delegation "validate config --production stays make-independent" validate config --production --secrets-env-file /tmp/secrets.env
assert_typed_no_delegation "chargeback report help stays make-independent" chargeback report --help

: >"$CAPTURE_FILE"
KEY_DRY_RUN_RC=0
KEY_DRY_RUN_OUTPUT="$(
    ACPCTL_BIN="$GO_SHIM" \
        ACPCTL_MAKE_BIN="$MAKE_STUB" \
        ACPCTL_TEST_CAPTURE_FILE="$CAPTURE_FILE" \
        ACP_REPO_ROOT="$REPO_ROOT" \
        "$SCRIPT_UNDER_TEST" key gen contract-test --budget 1.00 --dry-run 2>&1
)" || KEY_DRY_RUN_RC=$?
if [[ "$KEY_DRY_RUN_RC" -eq 0 ]]; then
    pass "key gen dry-run succeeds via typed path"
else
    fail "key gen dry-run should exit 0 (got $KEY_DRY_RUN_RC)"
fi
if [[ ! -s "$CAPTURE_FILE" ]]; then
    pass "key gen dry-run does not invoke make delegation"
else
    fail "key gen dry-run should not invoke make delegation"
fi
if echo "$KEY_DRY_RUN_OUTPUT" | grep -q "Alias: contract-test" && echo "$KEY_DRY_RUN_OUTPUT" | grep -q "Budget: \$1.00"; then
    pass "key gen dry-run output includes alias and budget"
else
    fail "key gen dry-run should print alias and budget details"
fi
echo ""

echo "Test: validate unknown subcommand exits 64 without make delegation..."
: >"$CAPTURE_FILE"
UNKNOWN_VALIDATE_RC=0
ACPCTL_BIN="$GO_SHIM" \
    ACPCTL_MAKE_BIN="$MAKE_STUB" \
    ACPCTL_TEST_CAPTURE_FILE="$CAPTURE_FILE" \
    ACP_REPO_ROOT="$REPO_ROOT" \
    "$SCRIPT_UNDER_TEST" validate network-contract-check >/dev/null 2>&1 || UNKNOWN_VALIDATE_RC=$?
if [[ "$UNKNOWN_VALIDATE_RC" -eq 64 ]]; then
    pass "validate network-contract-check exits 64 (unknown subcommand)"
else
    fail "validate network-contract-check should exit 64 (got $UNKNOWN_VALIDATE_RC)"
fi
if [[ ! -s "$CAPTURE_FILE" ]]; then
    pass "unknown validate subcommand does not invoke make delegation"
else
    fail "unknown validate subcommand should not invoke make delegation"
fi
echo ""

echo "Test: ACPCTL_MAKE_BIN override missing path exits 2..."
MISSING_MAKE_RC=0
ACPCTL_BIN="$GO_SHIM" \
    ACPCTL_MAKE_BIN="$TMP_DIR/make-does-not-exist" \
    ACP_REPO_ROOT="$REPO_ROOT" \
    "$SCRIPT_UNDER_TEST" deploy up >/dev/null 2>&1 || MISSING_MAKE_RC=$?
if [[ "$MISSING_MAKE_RC" -eq 2 ]]; then
    pass "missing ACPCTL_MAKE_BIN executable exits 2"
else
    fail "missing ACPCTL_MAKE_BIN executable should exit 2 (got $MISSING_MAKE_RC)"
fi
echo ""

echo "Test: ci should-run-runtime path does not delegate to make..."
: >"$CAPTURE_FILE"
CI_RC=0
ACPCTL_BIN="$GO_SHIM" \
    ACPCTL_MAKE_BIN="$MAKE_STUB" \
    ACPCTL_TEST_CAPTURE_FILE="$CAPTURE_FILE" \
    ACP_REPO_ROOT="$REPO_ROOT" \
    "$SCRIPT_UNDER_TEST" ci should-run-runtime --path docs/README.md --quiet >/dev/null 2>&1 || CI_RC=$?

if [[ "$CI_RC" -eq 0 || "$CI_RC" -eq 1 ]]; then
    pass "ci should-run-runtime returns domain decision exit code"
else
    fail "ci should-run-runtime should exit 0 or 1 (got $CI_RC)"
fi

if [[ ! -s "$CAPTURE_FILE" ]]; then
    pass "ci should-run-runtime does not invoke make delegation"
else
    fail "ci should-run-runtime should not invoke make delegation"
fi
echo ""

echo "========================"
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi
exit 0
