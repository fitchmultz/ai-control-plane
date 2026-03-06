#!/usr/bin/env bash
#
# Production Smoke Test CLI Contract Test
#
# Purpose: Validate the CLI contract for prod smoke test implementation
#   - --help exits 0 and contains Usage, Examples, Exit codes
#   - Unknown option exits 64
#   - Missing required inputs exits 64
#   - --dry-run exits 0 without requiring curl/jq
#
# Usage: ./prod_smoke_test_cli_test.sh [--help]
#
# Exit codes:
#   0   - All tests passed
#   1   - One or more tests failed

set -euo pipefail

#------------------------------------------------------------------------------
# Setup
#------------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source the test harness
# shellcheck source=../../demo/scripts/tests/lib/test_harness.sh
source "$REPO_ROOT/demo/scripts/tests/lib/test_harness.sh"

# Initialize test counters
acp_test_init

# Path to script under test
SCRIPT_UNDER_TEST="$REPO_ROOT/scripts/libexec/prod_smoke_test_impl.sh"

echo "${COLOR_BOLD}Production Smoke Test CLI Contract Test${COLOR_RESET}"
echo ""

#------------------------------------------------------------------------------
# Test: --help exits 0 and contains required sections
#------------------------------------------------------------------------------

echo "Test: --help exits 0 with required sections..."

set +e
HELP_OUTPUT=$("$SCRIPT_UNDER_TEST" --help 2>&1)
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
    echo "  ${SYMBOL_PASS} --help exits 0"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --help should exit 0 (exit: $EXIT_CODE)"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$HELP_OUTPUT" | grep -q "Usage:"; then
    echo "  ${SYMBOL_PASS} --help contains Usage:"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --help missing Usage: section"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$HELP_OUTPUT" | grep -qi "Example"; then
    echo "  ${SYMBOL_PASS} --help contains Examples"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --help missing Examples section"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$HELP_OUTPUT" | grep -q "Exit codes:"; then
    echo "  ${SYMBOL_PASS} --help contains Exit codes:"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --help missing Exit codes: section"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Test: Unknown option exits 64
#------------------------------------------------------------------------------

echo "Test: Unknown option exits 64..."

set +e
"$SCRIPT_UNDER_TEST" --unknown-option 2>/dev/null
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 64 ]; then
    echo "  ${SYMBOL_PASS} Unknown option exits 64"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} Unknown option should exit 64 (exit: $EXIT_CODE)"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Test: Missing LITELLM_MASTER_KEY exits 64 (usage error)
#------------------------------------------------------------------------------

echo "Test: Missing LITELLM_MASTER_KEY exits 64..."

# Unset LITELLM_MASTER_KEY for this test
set +e
(
    unset LITELLM_MASTER_KEY
    "$SCRIPT_UNDER_TEST" 2>/dev/null
)
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 64 ]; then
    echo "  ${SYMBOL_PASS} Missing master key exits 64"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} Missing master key should exit 64 (exit: $EXIT_CODE)"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Test: --dry-run exits 0 without requiring curl/jq
#------------------------------------------------------------------------------

echo "Test: --dry-run exits 0 without network calls..."

# Create isolated PATH without curl/jq
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Create a fake curl that fails if called
cat >"$TEMP_DIR/curl" <<'EOF'
#!/usr/bin/env bash
echo "ERROR: curl should not be called in dry-run mode" >&2
exit 1
EOF
chmod +x "$TEMP_DIR/curl"

# Create a fake jq that fails if called
cat >"$TEMP_DIR/jq" <<'EOF'
#!/usr/bin/env bash
echo "ERROR: jq should not be called in dry-run mode" >&2
exit 1
EOF
chmod +x "$TEMP_DIR/jq"

set +e
PATH="$TEMP_DIR:$PATH" "$SCRIPT_UNDER_TEST" --dry-run 2>/dev/null
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
    echo "  ${SYMBOL_PASS} --dry-run exits 0 without calling curl/jq"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --dry-run should exit 0 (exit: $EXIT_CODE)"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Test: --dry-run output contains expected sections
#------------------------------------------------------------------------------

echo "Test: --dry-run output contains expected information..."

DRY_RUN_OUTPUT=$("$SCRIPT_UNDER_TEST" --dry-run 2>&1)

if echo "$DRY_RUN_OUTPUT" | grep -q "Dry Run"; then
    echo "  ${SYMBOL_PASS} --dry-run indicates dry run mode"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --dry-run should indicate dry run mode"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$DRY_RUN_OUTPUT" | grep -q "/health"; then
    echo "  ${SYMBOL_PASS} --dry-run shows health check"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --dry-run should show health check"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$DRY_RUN_OUTPUT" | grep -q "/v1/models"; then
    echo "  ${SYMBOL_PASS} --dry-run shows models endpoint check"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} --dry-run should show models endpoint check"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Test: Smoke failure exit propagation contract
#------------------------------------------------------------------------------

echo "Test: Smoke failure propagation contract..."

SCRIPT_CONTENT="$(cat "$SCRIPT_UNDER_TEST")"

if echo "$SCRIPT_CONTENT" | grep -q "if run_smoke_tests"; then
    echo "  ${SYMBOL_PASS} main flow captures smoke exit status without negation"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} main flow should use direct run_smoke_tests status capture"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
fi

if echo "$SCRIPT_CONTENT" | grep -q "if ! run_smoke_tests"; then
    echo "  ${SYMBOL_FAIL} negated smoke status handling is present (can mask failures)"
    ACP_TESTS_FAILED=$((ACP_TESTS_FAILED + 1))
else
    echo "  ${SYMBOL_PASS} negated smoke status handling is not present"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
fi
echo ""

#------------------------------------------------------------------------------
# Summary
#------------------------------------------------------------------------------

acp_finish "Production Smoke Test CLI"
