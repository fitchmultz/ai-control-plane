#!/usr/bin/env bash
#
# AI Control Plane - Chargeback DB Script Contract Tests
#
# Purpose:
#   Validate the chargeback DB helper uses argv-safe command execution.
#
# Responsibilities:
#   - Verify path-only test overrides work.
#   - Verify shell-expanded command injection is rejected.
#
# Non-scope:
#   - Does not connect to a real database.
#   - Does not test report rendering flows.
#
# Invariants/Assumptions:
#   - Tests source the canonical demo library.
#   - Temp fixtures are removed on exit.

set -euo pipefail

show_help() {
    cat <<'EOF'
Usage: chargeback_db_test.sh [OPTIONS]

Run contract tests for demo/scripts/lib/chargeback_db.sh.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/chargeback_db_test.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIB_UNDER_TEST="${PROJECT_ROOT}/demo/scripts/lib/chargeback_db.sh"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

ARGS_LOG="${TMP_DIR}/args.log"
PWNED="${TMP_DIR}/pwned"

cat >"${TMP_DIR}/psql-stub" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >>"${ARGS_LOG}"
if [[ "\$*" == *" -t -A "* || "\$*" == -t\ -A* ]]; then
    echo "42"
else
    echo "(1 row)"
fi
EOF
chmod +x "${TMP_DIR}/psql-stub"

# shellcheck source=/dev/null
source "${LIB_UNDER_TEST}"

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

test_path_only_override() {
    echo "Test: valid CHARGEBACK_PSQL_BIN override..."
    CHARGEBACK_PSQL_BIN="${TMP_DIR}/psql-stub"
    local result
    result="$(query 'SELECT 42;')" || true

    if [[ "${result}" == "42" ]]; then
        pass "query returned stubbed scalar result"
    else
        fail "query should return stubbed scalar result"
    fi

    if grep -Fq -- "-X -v ON_ERROR_STOP=1 -P pager=off -t -A -c SELECT 42;" "${ARGS_LOG}"; then
        pass "query executed argv-safe psql invocation"
    else
        fail "query should execute argv-safe psql invocation"
    fi
}

test_scalar_trimming_preserves_internal_content() {
    echo "Test: scalar query trimming is deterministic..."
    cat >"${TMP_DIR}/psql-stub-trim" <<EOF
#!/usr/bin/env bash
set -euo pipefail
if [[ "\${1:-}" == "-X" ]]; then
    printf '  Alpha  Beta  \n'
else
    printf 'unexpected args\n' >&2
    exit 1
fi
EOF
    chmod +x "${TMP_DIR}/psql-stub-trim"

    CHARGEBACK_PSQL_BIN="${TMP_DIR}/psql-stub-trim"
    local result
    result="$(query 'SELECT $$Alpha  Beta$$;')" || true

    if [[ "${result}" == "Alpha  Beta" ]]; then
        pass "scalar trimming removes only leading/trailing whitespace"
    else
        fail "scalar trimming should preserve internal spacing"
    fi
}

test_shell_expansion_is_rejected() {
    echo "Test: injected CHARGEBACK_PSQL_BIN is rejected..."
    CHARGEBACK_PSQL_BIN="/bin/sh -c touch ${PWNED}"

    if query 'SELECT 1;' >/dev/null 2>&1; then
        fail "invalid CHARGEBACK_PSQL_BIN should not succeed"
    else
        pass "invalid CHARGEBACK_PSQL_BIN is rejected"
    fi

    if [[ ! -e "${PWNED}" ]]; then
        pass "injection payload was not executed"
    else
        fail "injection payload must never execute"
    fi
}

main() {
    echo "Chargeback DB Script Contract Tests"
    echo "==================================="
    echo ""

    test_path_only_override
    test_scalar_trimming_preserves_internal_content
    test_shell_expansion_is_rejected

    echo ""
    echo "Results"
    echo "-------"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"

    if [[ "${TESTS_FAILED}" -eq 0 ]]; then
        echo "All chargeback DB tests passed."
        exit 0
    fi

    echo "One or more chargeback DB tests failed."
    exit 1
}

main "$@"
