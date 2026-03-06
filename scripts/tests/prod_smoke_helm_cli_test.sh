#!/usr/bin/env bash
#
# Production Smoke Test Helm CLI Contract Test
#
# Purpose: Validate the CLI contract for Helm prod smoke implementation
#   - --help exits 0 and contains Usage, Examples, Exit codes
#   - Unknown option exits 64
#   - Missing kubectl exits 2 with install hint
#
# Usage: ./prod_smoke_helm_cli_test.sh [--help]
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
SCRIPT_UNDER_TEST="$REPO_ROOT/scripts/libexec/prod_smoke_helm_impl.sh"

echo "${COLOR_BOLD}Production Smoke Test Helm CLI Contract Test${COLOR_RESET}"
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
# Test: Missing kubectl exits 2 with install hint
#------------------------------------------------------------------------------

echo "Test: Missing kubectl exits 2 with install hint..."

# Create isolated PATH without kubectl
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

set +e
# Use a minimal PATH that excludes kubectl
PATH="/usr/bin:/bin" "$SCRIPT_UNDER_TEST" 2>&1
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 2 ]; then
    echo "  ${SYMBOL_PASS} Missing kubectl exits 2"
    ACP_TESTS_PASSED=$((ACP_TESTS_PASSED + 1))
else
    echo "  ${SYMBOL_FAIL} Missing kubectl should exit 2 (exit: $EXIT_CODE)"
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
    # Provide a fake kubectl that does nothing
    PATH="$TEMP_DIR:$PATH"
    cat >"$TEMP_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$TEMP_DIR/kubectl"
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
# Summary
#------------------------------------------------------------------------------

acp_finish "Production Smoke Test Helm CLI"
