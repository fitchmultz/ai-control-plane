#!/usr/bin/env bash
set -euo pipefail

# Onboard Help Contract Test
#
# Purpose:
#   - Verify the live onboarding help surface reflects the guided wizard cutover.
#
# Responsibilities:
#   - Build an isolated acpctl binary for the test.
#   - Assert help text matches the wizard-first contract.
#   - Ensure removed flags and deleted compatibility shim stay gone.
#
# Scope:
#   - Help and command-surface contract validation only.
#
# Usage:
#   - bash scripts/tests/onboard_help_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require live gateway services.
#   - The onboarding shim must not exist after the cutover.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: onboard_help_contract_test.sh [OPTIONS]

Validate onboarding wizard help and removed legacy surfaces.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

PROJECT_ROOT="$(test_repo_root)"
test_fixture_init onboard-help-contract-test
BINARY_PATH="${TEST_TMP_ROOT}/acpctl"
test_build_acpctl_binary "${BINARY_PATH}"

printf 'Onboard Help Contract Test\n'
printf '==========================\n'

help_output="$(cd "${PROJECT_ROOT}" && ACPCTL_BIN="${BINARY_PATH}" ./scripts/acpctl.sh onboard --help 2>&1)"

test_assert_contains "${help_output}" "Interactive guided setup" "help describes the wizard" || exit 1
test_assert_contains "${help_output}" "acpctl onboard" "help shows canonical entrypoint" || exit 1
test_assert_contains "${help_output}" "acpctl onboard codex" "help shows tool preselection" || exit 1
test_assert_contains "${help_output}" "codex" "help lists codex" || exit 1
test_assert_contains "${help_output}" "claude" "help lists claude" || exit 1
test_assert_contains "${help_output}" "opencode" "help lists opencode" || exit 1
test_assert_contains "${help_output}" "cursor" "help lists cursor" || exit 1

for legacy_flag in --mode --verify --write-config --show-key --host --port --tls; do
    if grep -Fq -- "${legacy_flag}" <<<"${help_output}"; then
        printf '  ✗ legacy flag %s must not appear in onboarding help\n' "${legacy_flag}"
        exit 1
    fi
done
printf '  ✓ onboarding help omits removed legacy flags\n'

if grep -Fq -- "copilot" <<<"${help_output}"; then
    printf '  ✗ unsupported copilot onboarding must not appear in wizard help\n'
    exit 1
fi
printf '  ✓ onboarding help omits unsupported copilot onboarding\n'

if [[ -e "${PROJECT_ROOT}/scripts/libexec/onboard_impl.sh" ]]; then
    printf '  ✗ legacy onboarding shim should be removed\n'
    exit 1
fi
printf '  ✓ legacy onboarding shim is removed\n'
