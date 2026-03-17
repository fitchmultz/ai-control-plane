#!/usr/bin/env bash
set -euo pipefail

# Onboard Export Contract Test
#
# Purpose:
#   - Verify the live onboarding wizard produces actionable output for a safe local path.
#
# Responsibilities:
#   - Build an isolated acpctl binary for the test.
#   - Drive the wizard through Codex direct mode without external dependencies.
#   - Assert the resulting exports and completion text remain stable.
#
# Scope:
#   - Interactive wizard shell contract only.
#
# Usage:
#   - bash scripts/tests/onboard_export_contract_test.sh
#
# Invariants/Assumptions:
#   - The direct Codex path requires no live gateway or demo/.env.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: onboard_export_contract_test.sh [OPTIONS]

Validate onboarding wizard output for the direct Codex path.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

PROJECT_ROOT="$(test_repo_root)"
test_fixture_init onboard-export-contract-test
BINARY_PATH="${TEST_TMP_ROOT}/acpctl"
test_build_acpctl_binary "${BINARY_PATH}"

printf 'Onboard Export Contract Test\n'
printf '============================\n'

wizard_input=$'3\n\n\n\nn\n'
output="$(cd "${PROJECT_ROOT}" && printf '%s' "${wizard_input}" | ACPCTL_BIN="${BINARY_PATH}" ./scripts/acpctl.sh onboard codex 2>&1)"

test_assert_contains "${output}" "ACP onboarding wizard" "wizard banner is shown" || exit 1
test_assert_contains "${output}" "Mode: direct" "direct mode was selected" || exit 1
test_assert_contains "${output}" 'export OTEL_EXPORTER_OTLP_ENDPOINT="http://127.0.0.1:4317"' "direct mode exports OTEL endpoint" || exit 1
test_assert_contains "${output}" 'export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"' "direct mode exports OTEL protocol" || exit 1
test_assert_contains "${output}" '[OK] env/config contract: generated OTEL exports are valid for direct mode' "direct mode local lint passes" || exit 1
test_assert_contains "${output}" '[SKIP] gateway reachability: network verification disabled by operator' "direct mode skips network verification when requested" || exit 1
test_assert_contains "${output}" "Onboarding complete." "wizard finishes successfully" || exit 1

if grep -Fq -- 'OPENAI_API_KEY=' <<<"${output}"; then
    printf '  ✗ direct mode must not emit gateway API keys\n'
    exit 1
fi
printf '  ✓ direct mode omits gateway API key exports\n'
