#!/usr/bin/env bash
#
# AI Control Plane - Onboarding Script Contract Tests
#
# Purpose:
#   Validate onboarding bridge behavior for supported tools and modes.
#
# Responsibilities:
#   - Verify help/usage contract and error codes.
#   - Verify export output and key redaction behavior.
#   - Verify --verify path performs authorized gateway checks.
#
# Non-scope:
#   - Does NOT hit real Docker services or external networks.
#   - Does NOT validate vendor OAuth/device login flows.
#
# Invariants/Assumptions:
#   - Tests run in an isolated temp fixture.
#   - Stubbed binaries emulate acpctl and curl behavior.
#   - Temp files and binaries are always cleaned up on exit.

set -euo pipefail

show_help() {
    cat <<'EOF'
Usage: onboard_test.sh [OPTIONS]

Run contract tests for scripts/libexec/onboard_impl.sh.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/onboard_test.sh
  make test-onboard
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SOURCE_SCRIPT="${PROJECT_ROOT}/scripts/libexec/onboard_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_ROOT}"' EXIT

TEST_REPO="${TMP_ROOT}/repo"
TEST_BIN_DIR="${TEST_REPO}/.bin"
TEST_SCRIPT_DIR="${TEST_REPO}/scripts/libexec"
TEST_DEMO_DIR="${TEST_REPO}/demo"
TEST_STUB_BIN_DIR="${TMP_ROOT}/bin"
CURL_LOG="${TMP_ROOT}/curl.log"

mkdir -p "${TEST_BIN_DIR}" "${TEST_SCRIPT_DIR}" "${TEST_DEMO_DIR}" "${TEST_STUB_BIN_DIR}"
cp "${SOURCE_SCRIPT}" "${TEST_SCRIPT_DIR}/onboard_impl.sh"
chmod +x "${TEST_SCRIPT_DIR}/onboard_impl.sh"

cat >"${TEST_DEMO_DIR}/.env" <<'EOF'
LITELLM_MASTER_KEY=sk-master-test-12345
EOF

cat >"${TEST_BIN_DIR}/acpctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "key" && "${2:-}" == "gen" ]]; then
    alias_name="${3:-}"
    if [[ "${alias_name}" == "existing-alias" ]]; then
        echo "key alias already exists: ${alias_name}" >&2
        exit 1
    fi
    echo "created key for alias ${alias_name}"
    echo "sk-test-full-key-1234567890-abcdef"
    exit 0
fi

echo "unsupported acpctl stub command: $*" >&2
exit 1
EOF
chmod +x "${TEST_BIN_DIR}/acpctl"

cat >"${TEST_STUB_BIN_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

log_file="${ACP_TEST_CURL_LOG:?missing ACP_TEST_CURL_LOG}"
printf '%s\n' "$*" >>"${log_file}"

http_code="200"
url="${*: -1}"
if [[ "${url}" == *"/health" ]]; then
    http_code="200"
elif [[ "${url}" == *"/v1/models" ]]; then
    http_code="200"
fi

if [[ "$*" == *"%{http_code}"* ]]; then
    printf '%s' "${http_code}"
fi
EOF
chmod +x "${TEST_STUB_BIN_DIR}/curl"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/onboard_impl.sh"

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

run_onboard() {
    ACP_TEST_CURL_LOG="${CURL_LOG}" \
        PATH="${TEST_STUB_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local description="$3"
    if grep -Fq "${needle}" <<<"${haystack}"; then
        pass "${description}"
    else
        fail "${description}"
    fi
}

test_script_contract() {
    echo "Test: script contract and help..."

    if [[ -f "${TEST_SCRIPT}" ]]; then
        pass "onboard_impl.sh exists"
    else
        fail "onboard_impl.sh missing"
    fi

    if [[ -x "${TEST_SCRIPT}" ]]; then
        pass "onboard_impl.sh is executable"
    else
        fail "onboard_impl.sh should be executable"
    fi

    local output
    output="$(run_onboard --help 2>&1)"
    assert_contains "${output}" "Usage: onboard_impl.sh <tool> [options]" "main help contains usage"
    assert_contains "${output}" "Tools:" "main help contains tools section"
    assert_contains "${output}" "codex" "main help lists codex"
    assert_contains "${output}" "claude" "main help lists claude"
    assert_contains "${output}" "Examples:" "main help contains examples"
}

test_tool_specific_help() {
    echo "Test: tool-specific help..."
    local output
    output="$(run_onboard codex --help 2>&1)"
    assert_contains "${output}" "Codex notes:" "codex help includes notes"
    assert_contains "${output}" "subscription" "codex help mentions subscription mode"
}

test_invalid_tool_error() {
    echo "Test: invalid tool handling..."
    local output=""
    local rc=0
    output="$(run_onboard invalid-tool 2>&1)" || rc=$?
    if [[ "${rc}" -eq 64 ]]; then
        pass "invalid tool exits with usage code 64"
    else
        fail "invalid tool should exit 64 (got ${rc})"
    fi
    assert_contains "${output}" "Unsupported tool: invalid-tool" "invalid tool prints explicit error"
}

test_redaction_default_and_show_key() {
    echo "Test: key redaction and show-key..."
    local output
    output="$(run_onboard codex --mode api-key --alias test-redaction 2>&1)"
    assert_contains "${output}" 'export OPENAI_API_KEY="sk-test-...cdef"' "default output redacts key"
    if grep -Fq "sk-test-full-key-1234567890-abcdef" <<<"${output}"; then
        fail "default output must not print full key"
    else
        pass "default output does not print full key"
    fi

    output="$(run_onboard codex --mode api-key --alias test-show --show-key 2>&1)"
    assert_contains "${output}" 'export OPENAI_API_KEY="sk-test-full-key-1234567890-abcdef"' "--show-key prints full key"
}

test_mode_defaults_by_tool() {
    echo "Test: default model and export mapping..."
    local output

    output="$(run_onboard codex --mode subscription --alias codex-sub 2>&1)"
    assert_contains "${output}" "Mode: subscription" "codex subscription mode selected"
    assert_contains "${output}" "Model: chatgpt-gpt5.3-codex" "codex subscription default model"
    assert_contains "${output}" "export OPENAI_BASE_URL=\"http://127.0.0.1:4000\"" "codex uses openai-compatible base url"

    output="$(run_onboard codex --mode api-key --alias codex-api 2>&1)"
    assert_contains "${output}" "Model: openai-gpt5.2" "codex api-key default model"

    output="$(run_onboard claude --alias claude-api 2>&1)"
    assert_contains "${output}" "Model: claude-haiku-4-5" "claude default model"
    assert_contains "${output}" "export ANTHROPIC_BASE_URL=\"http://127.0.0.1:4000\"" "claude export base url"
    assert_contains "${output}" "export ANTHROPIC_API_KEY=\"sk-test-...cdef\"" "claude output redacts key"
}

test_verify_authorized_checks() {
    echo "Test: --verify authorized checks..."
    : >"${CURL_LOG}"
    local output
    output="$(run_onboard codex --mode api-key --alias verify-key --verify 2>&1)"
    assert_contains "${output}" "Gateway health and authorized model checks passed" "--verify reports success"

    if grep -Fq "/health" "${CURL_LOG}"; then
        pass "curl called health endpoint during verify"
    else
        fail "verify should call health endpoint"
    fi

    if grep -Fq "/v1/models" "${CURL_LOG}"; then
        pass "curl called models endpoint during verify"
    else
        fail "verify should call models endpoint"
    fi

    if grep -Fq "Authorization: Bearer sk-test-full-key-1234567890-abcdef" "${CURL_LOG}"; then
        pass "verify sends authorization header"
    else
        fail "verify should send authorization header"
    fi
}

test_makefile_targets_present() {
    echo "Test: make target wiring..."
    local mk_file="${PROJECT_ROOT}/mk/onboard.mk"
    if [[ -f "${mk_file}" ]]; then
        pass "mk/onboard.mk exists"
    else
        fail "mk/onboard.mk missing"
        return
    fi

    if grep -q '^onboard:' "${mk_file}"; then
        pass "onboard target exists"
    else
        fail "onboard target missing"
    fi

    if grep -q '^onboard-codex:' "${mk_file}"; then
        pass "onboard-codex target exists"
    else
        fail "onboard-codex target missing"
    fi

    if grep -q '^chatgpt-auth-copy:' "${mk_file}"; then
        pass "chatgpt-auth-copy target exists"
    else
        fail "chatgpt-auth-copy target missing"
    fi
}

main() {
    echo "Onboarding Script Contract Tests"
    echo "================================"
    echo ""

    test_script_contract
    test_tool_specific_help
    test_invalid_tool_error
    test_redaction_default_and_show_key
    test_mode_defaults_by_tool
    test_verify_authorized_checks
    test_makefile_targets_present

    echo ""
    echo "Results"
    echo "-------"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"

    if [[ "${TESTS_FAILED}" -eq 0 ]]; then
        echo "All onboarding tests passed."
        exit 0
    fi

    echo "One or more onboarding tests failed."
    exit 1
}

main "$@"
