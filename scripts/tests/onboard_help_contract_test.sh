#!/usr/bin/env bash
set -euo pipefail

# Onboard Help Contract Test
#
# Purpose:
#   - Verify onboarding bridge help surfaces supported tools and usage text.
#
# Responsibilities:
#   - Create an isolated onboard fixture with stubbed acpctl/curl commands.
#   - Assert main and tool-specific help text remains available.
#
# Scope:
#   - Help and invalid-tool behavior only.
#
# Usage:
#   - bash scripts/tests/onboard_help_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require real gateway services.

show_help() {
    cat <<'EOF'
Usage: onboard_help_contract_test.sh [OPTIONS]

Validate onboarding help and usage behavior.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SOURCE_SCRIPT="${PROJECT_ROOT}/scripts/libexec/onboard_impl.sh"

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_ROOT}"' EXIT

TEST_REPO="${TMP_ROOT}/repo"
TEST_BIN_DIR="${TEST_REPO}/.bin"
TEST_SCRIPT_DIR="${TEST_REPO}/scripts/libexec"
TEST_DEMO_DIR="${TEST_REPO}/demo"
TEST_STUB_BIN_DIR="${TMP_ROOT}/bin"
mkdir -p "${TEST_BIN_DIR}" "${TEST_SCRIPT_DIR}" "${TEST_DEMO_DIR}" "${TEST_STUB_BIN_DIR}"

cp "${SOURCE_SCRIPT}" "${TEST_SCRIPT_DIR}/onboard_impl.sh"
chmod +x "${TEST_SCRIPT_DIR}/onboard_impl.sh"

cat >"${TEST_DEMO_DIR}/.env" <<'EOF'
LITELLM_MASTER_KEY=sk-master-test-12345
EOF

cat >"${TEST_BIN_DIR}/acpctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "onboard" ]]; then
    exit 1
fi
shift
tool="${1:-}"
if [[ -z "${tool}" || "${tool}" == "--help" || "${tool}" == "-h" ]]; then
    cat <<'OUT'
Usage: onboard_impl.sh <tool> [options]
Tools:
  codex
  claude
  opencode
  cursor
  copilot
Examples:
  onboard_impl.sh codex --mode subscription --verify
OUT
    exit 0
fi
shift || true
if [[ "${1:-}" == "--help" || "${2:-}" == "--help" ]]; then
    cat <<'OUT'
Codex notes:
  - For subscription mode, run `make chatgpt-login` on the gateway host first.
  - Codex uses OPENAI_BASE_URL without /v1.
OUT
    exit 0
fi
if [[ "${tool}" == "invalid-tool" ]]; then
    echo "ERROR: unsupported tool: ${tool}" >&2
    exit 64
fi
printf 'stub\n'
EOF
chmod +x "${TEST_BIN_DIR}/acpctl"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/onboard_impl.sh"

run_onboard() {
    PATH="${TEST_STUB_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local description="$3"
    if grep -Fq "${needle}" <<<"${haystack}"; then
        printf '  ✓ %s\n' "${description}"
    else
        printf '  ✗ %s\n' "${description}"
        exit 1
    fi
}

printf 'Onboard Help Contract Test\n'
printf '==========================\n'

help_output="$(run_onboard --help 2>&1)"
assert_contains "${help_output}" "Usage: onboard_impl.sh <tool> [options]" "main help includes usage"
assert_contains "${help_output}" "codex" "main help lists codex"
assert_contains "${help_output}" "claude" "main help lists claude"
assert_contains "${help_output}" "Examples:" "main help includes examples"

codex_help="$(run_onboard codex --help 2>&1)"
assert_contains "${codex_help}" "Codex notes:" "codex help includes notes"
assert_contains "${codex_help}" "subscription mode" "codex help mentions subscription mode"

invalid_rc=0
invalid_output="$(run_onboard invalid-tool 2>&1)" || invalid_rc=$?
if [[ "${invalid_rc}" -ne 64 ]]; then
    printf '  ✗ invalid tool should exit 64 (got %s)\n' "${invalid_rc}"
    exit 1
fi
assert_contains "${invalid_output}" "unsupported tool: invalid-tool" "invalid tool reports explicit error"
