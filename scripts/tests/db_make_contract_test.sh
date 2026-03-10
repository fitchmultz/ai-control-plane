#!/usr/bin/env bash
set -euo pipefail

# DB Make Contract Test
#
# Purpose:
#   - Verify Make-backed database entrypoints invoke the typed CLI once and do
#     not recurse back into Make.
#
# Responsibilities:
#   - Stub the `ACPCTL_BIN` target for `make db-status` and `make db-shell`.
#   - Assert each target executes the expected typed subcommand exactly once.
#
# Scope:
#   - Makefile database wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/db_make_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally without requiring Docker or PostgreSQL.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: db_make_contract_test.sh [OPTIONS]

Validate that `make db-status` and `make db-shell` call the typed CLI once.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
test_fixture_init db-make-contract-test

CAPTURE_FILE="${TEST_TMP_ROOT}/acpctl-calls.txt"
ACPCTL_STUB="${TEST_TMP_ROOT}/acpctl-stub.sh"

cat >"${ACPCTL_STUB}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${ACPCTL_TEST_CAPTURE_FILE}"
EOF
chmod +x "${ACPCTL_STUB}"

run_make_target() {
    local target="$1"
    : >"${CAPTURE_FILE}"
    (
        cd "${REPO_ROOT}"
        ACPCTL_BIN="${ACPCTL_STUB}" \
        ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
        make --silent "${target}"
    ) >/dev/null
}

assert_single_invocation() {
    local target="$1"
    local expected="$2"
    run_make_target "${target}"
    local actual
    actual="$(tr -d '\r' <"${CAPTURE_FILE}")"
    if [[ "${actual}" != "${expected}"$'\n' && "${actual}" != "${expected}" ]]; then
        printf '  ✗ %s should invoke "%s" once (got %q)\n' "${target}" "${expected}" "${actual}"
        exit 1
    fi
    if [[ "$(wc -l <"${CAPTURE_FILE}")" -ne 1 ]]; then
        printf '  ✗ %s should invoke ACPCTL_BIN exactly once\n' "${target}"
        exit 1
    fi
    printf '  ✓ %s -> %s\n' "${target}" "${expected}"
}

printf 'DB Make Contract Test\n'
printf '=====================\n'

assert_single_invocation "db-status" "db status"
assert_single_invocation "db-shell" "db shell"
