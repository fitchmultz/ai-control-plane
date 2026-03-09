#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Help Contract Test
#
# Purpose:
#   - Verify help output exposes the expected typed command surface.
#
# Responsibilities:
#   - Build an isolated acpctl binary for contract testing.
#   - Assert required help sections and core commands remain visible.
#
# Scope:
#   - Help output and unknown-command behavior only.
#
# Usage:
#   - bash scripts/tests/acpctl_cli_help_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require Docker or make delegation.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: acpctl_cli_help_test.sh [OPTIONS]

Validate scripts/acpctl.sh help output and usage behavior.

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

test_fixture_init acpctl-cli-help-test
TMP_DIR="${TEST_TMP_ROOT}"

ACPCTL_BIN_TEST="${TMP_DIR}/acpctl-bin"
GO_SHIM="${TMP_DIR}/acpctl-go-shim.sh"

test_build_acpctl_binary "${ACPCTL_BIN_TEST}"
test_create_exec_shim "${ACPCTL_BIN_TEST}" "${GO_SHIM}"

printf 'ACPCTL CLI Help Contract Test\n'
printf '=============================\n'

help_output="$(ACPCTL_BIN="${GO_SHIM}" "${SCRIPT_UNDER_TEST}" --help 2>&1)"
test_assert_contains "${help_output}" "Usage: acpctl" "--help includes usage" || exit 1
test_assert_contains "${help_output}" "Commands:" "--help includes Commands section" || exit 1
test_assert_contains "${help_output}" "Examples:" "--help includes Examples section" || exit 1
test_assert_contains "${help_output}" "env" "--help lists env command" || exit 1
test_assert_contains "${help_output}" "chargeback" "--help lists chargeback command" || exit 1
test_assert_contains "${help_output}" "deploy" "--help lists deploy command" || exit 1
test_assert_contains "${help_output}" "validate" "--help lists validate command" || exit 1
test_assert_contains "${help_output}" "db" "--help lists db command" || exit 1
test_assert_contains "${help_output}" "key" "--help lists key command" || exit 1
test_assert_contains "${help_output}" "doctor" "--help lists doctor command" || exit 1

unknown_rc=0
ACPCTL_BIN="${GO_SHIM}" "${SCRIPT_UNDER_TEST}" unknown-command >/dev/null 2>&1 || unknown_rc=$?
test_assert_exit_code "${unknown_rc}" "64" "unknown command exits 64" || exit 1
