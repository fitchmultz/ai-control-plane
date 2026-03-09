#!/usr/bin/env bash
set -euo pipefail

# Onboard Export Contract Test
#
# Purpose:
#   - Verify onboarding export output, key redaction, and tool defaults.
#
# Responsibilities:
#   - Create an isolated onboard fixture with stubbed acpctl behavior.
#   - Assert exported variables and default models by tool/mode.
#
# Scope:
#   - Export rendering and tool-default behavior only.
#
# Usage:
#   - bash scripts/tests/onboard_export_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require network access.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: onboard_export_contract_test.sh [OPTIONS]

Validate onboarding export output and defaults.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

test_fixture_init onboard-export-contract-test
TMP_ROOT="${TEST_TMP_ROOT}"
test_fixture_repo_init "${TMP_ROOT}"
test_fixture_copy_libexec "onboard_impl.sh"

test_write_fixture_env <<EOF
EVIL=\$(touch "${TMP_ROOT}/env-pwned")
LITELLM_MASTER_KEY=sk-master-test-12345
EOF

cat >"${TEST_BIN_DIR}/acpctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
redact_key() {
    local key="$1"
    printf '%s...%s\n' "${key:0:8}" "${key: -4}"
}
if [[ "${1:-}" != "onboard" ]]; then
    exit 1
fi
shift
tool="${1:-}"
shift || true
mode=""
alias_name="${tool}-cli"
model=""
show_key="false"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --mode) mode="${2:-}"; shift 2 ;;
        --alias) alias_name="${2:-}"; shift 2 ;;
        --model) model="${2:-}"; shift 2 ;;
        --show-key) show_key="true"; shift ;;
        *) shift ;;
    esac
done
case "${tool}" in
    codex) mode="${mode:-subscription}"; model="${model:-$([[ "${mode}" == "subscription" ]] && printf 'chatgpt-gpt5.3-codex' || printf 'openai-gpt5.2')}" ;;
    claude) mode="${mode:-api-key}"; model="${model:-claude-haiku-4-5}" ;;
    opencode|cursor|copilot) mode="${mode:-api-key}"; model="${model:-openai-gpt5.2}" ;;
    *) echo "ERROR: unsupported tool: ${tool}" >&2; exit 64 ;;
esac
key_value="sk-test-full-key-1234567890-abcdef"
printed_key="$(redact_key "${key_value}")"
if [[ "${show_key}" == "true" ]]; then
    printed_key="${key_value}"
fi
printf 'Tool: %s\n' "${tool}"
printf 'Mode: %s\n' "${mode}"
printf 'Model: %s\n' "${model}"
if [[ "${tool}" == "claude" ]]; then
    printf 'export ANTHROPIC_BASE_URL="http://127.0.0.1:4000"\n'
    printf 'export ANTHROPIC_API_KEY="%s"\n' "${printed_key}"
else
    printf 'export OPENAI_BASE_URL="http://127.0.0.1:4000"\n'
    printf 'export OPENAI_API_KEY="%s"\n' "${printed_key}"
fi
EOF
chmod +x "${TEST_BIN_DIR}/acpctl"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/onboard_impl.sh"
run_onboard() {
    PATH="${TEST_STUB_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

assert_contains() {
    test_assert_contains "$1" "$2" "$3" || exit 1
}

printf 'Onboard Export Contract Test\n'
printf '============================\n'

output="$(run_onboard codex --mode api-key --alias redaction 2>&1)"
assert_contains "${output}" 'export OPENAI_API_KEY="sk-test-...cdef"' "default output redacts OpenAI key"
if grep -Fq "sk-test-full-key-1234567890-abcdef" <<<"${output}"; then
    printf '  ✗ default output must not print full key\n'
    exit 1
fi
printf '  ✓ default output hides full key\n'

output="$(run_onboard codex --mode api-key --show-key 2>&1)"
assert_contains "${output}" 'export OPENAI_API_KEY="sk-test-full-key-1234567890-abcdef"' "--show-key prints full key"

output="$(run_onboard codex --mode subscription 2>&1)"
assert_contains "${output}" "Mode: subscription" "codex subscription mode selected"
assert_contains "${output}" "Model: chatgpt-gpt5.3-codex" "codex subscription default model"

output="$(run_onboard claude 2>&1)"
assert_contains "${output}" "Model: claude-haiku-4-5" "claude default model"
assert_contains "${output}" 'export ANTHROPIC_BASE_URL="http://127.0.0.1:4000"' "claude exports anthropic base url"

rm -f "${TMP_ROOT}/env-pwned"
run_onboard codex --mode api-key >/dev/null 2>&1 || true
if [[ -e "${TMP_ROOT}/env-pwned" ]]; then
    printf '  ✗ .env payload must not execute\n'
    exit 1
fi
printf '  ✓ .env payload is treated as data\n'
