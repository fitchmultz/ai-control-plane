#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Delegation Contract Test
#
# Purpose:
#   - Verify delegated flow commands invoke the expected make targets.
#
# Responsibilities:
#   - Build an isolated acpctl binary and stub make for capture.
#   - Assert delegated command paths stay mapped to the intended targets.
#
# Scope:
#   - Make delegation behavior only.
#
# Usage:
#   - bash scripts/tests/acpctl_cli_delegation_test.sh
#
# Invariants/Assumptions:
#   - Tests run in a temp fixture and do not invoke real make targets.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: acpctl_cli_delegation_test.sh [OPTIONS]

Validate delegated make targets for scripts/acpctl.sh.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
SCRIPT_UNDER_TEST="${REPO_ROOT}/scripts/acpctl.sh"

test_fixture_init acpctl-cli-delegation-test
TMP_DIR="${TEST_TMP_ROOT}"

CAPTURE_FILE="${TMP_DIR}/make-args.txt"
ACPCTL_BIN_TEST="${TMP_DIR}/acpctl-bin"
GO_SHIM="${TMP_DIR}/acpctl-go-shim.sh"
MAKE_STUB_DIR="${TMP_DIR}/bin"
MAKE_STUB="${MAKE_STUB_DIR}/make-stub.sh"

mkdir -p "${MAKE_STUB_DIR}"
test_build_acpctl_binary "${ACPCTL_BIN_TEST}"
test_create_exec_shim "${ACPCTL_BIN_TEST}" "${GO_SHIM}"
test_install_stub "make_capture_stub.sh" "${MAKE_STUB_DIR}" "make-stub.sh"

run_with_make_stub() {
    : >"${CAPTURE_FILE}"
    ACPCTL_BIN="${GO_SHIM}" \
        ACPCTL_MAKE_BIN="${MAKE_STUB}" \
        ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
        ACP_REPO_ROOT="${REPO_ROOT}" \
        "${SCRIPT_UNDER_TEST}" "$@"
}

assert_make_target() {
    local expected_target="$1"
    shift
    run_with_make_stub "$@" >/dev/null 2>&1
    actual_target="$(head -n1 "${CAPTURE_FILE}" || true)"
    if [[ "${actual_target}" != "${expected_target}" ]]; then
        printf '  ✗ expected target %s, got %s\n' "${expected_target}" "${actual_target}"
        exit 1
    fi
    printf '  ✓ %s -> %s\n' "$*" "${expected_target}"
}

printf 'ACPCTL CLI Delegation Contract Test\n'
printf '===================================\n'

assert_make_target "up" deploy up
assert_make_target "demo-all" demo all
assert_make_target "up-tls" deploy up-tls
assert_make_target "tf-plan" terraform plan

missing_rc=0
ACPCTL_BIN="${GO_SHIM}" \
    ACPCTL_MAKE_BIN="${TMP_DIR}/missing-make" \
    ACP_REPO_ROOT="${REPO_ROOT}" \
    "${SCRIPT_UNDER_TEST}" deploy up >/dev/null 2>&1 || missing_rc=$?
test_assert_exit_code "${missing_rc}" "2" "missing ACPCTL_MAKE_BIN exits 2" || exit 1
