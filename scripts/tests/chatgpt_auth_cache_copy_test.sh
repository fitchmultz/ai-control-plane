#!/usr/bin/env bash
#
# AI Control Plane - ChatGPT Auth Cache Copy Contract Tests
#
# Purpose:
#   Validate chatgpt_auth_cache_copy_impl.sh normalizes auth caches safely.
#
# Responsibilities:
#   - Verify help output remains available.
#   - Verify Codex auth cache conversion writes normalized JSON locally.
#   - Verify best-effort live sync failures do not corrupt the persisted cache.
#
# Non-scope:
#   - Does not perform a real Codex login.
#   - Does not require a live container.
#
# Invariants/Assumptions:
#   - Tests run in a temporary fixture repo.
#   - Stubbed jq/docker binaries provide deterministic behavior.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: chatgpt_auth_cache_copy_test.sh [OPTIONS]

Run contract tests for scripts/libexec/chatgpt_auth_cache_copy_impl.sh.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/chatgpt_auth_cache_copy_test.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

test_results_init
test_fixture_init chatgpt-auth-cache-copy-test
TMP_ROOT="${TEST_TMP_ROOT}"
test_fixture_repo_init "${TMP_ROOT}"
AUTH_FILE="${TMP_ROOT}/auth.json"
DEST_FILE="${TEST_REPO}/demo/auth/chatgpt/auth.json"

test_fixture_copy_libexec "chatgpt_auth_cache_copy_impl.sh"
mkdir -p "${TEST_BIN_DIR}"

cat >"${AUTH_FILE}" <<'EOF'
{"tokens":{"access_token":"access-token","refresh_token":"refresh-token","id_token":"id-token","account_id":"acct-123"}}
EOF

test_install_stub "jq_contract_stub.py" "${TEST_BIN_DIR}" "jq"

cat >"${TEST_BIN_DIR}/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
    inspect)
        printf 'true'
        ;;
    exec)
        exit 1
        ;;
esac
EOF
chmod +x "${TEST_BIN_DIR}/docker"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/chatgpt_auth_cache_copy_impl.sh"

run_script() {
    PATH="${TEST_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

test_help_contract() {
    echo "Test: script help..."
    local output
    output="$(run_script --help 2>&1)"

    if grep -Fq "Usage: chatgpt_auth_cache_copy_impl.sh [options]" <<<"${output}"; then
        test_pass "help contains usage"
    else
        test_fail "help missing usage"
    fi
}

test_normalizes_codex_cache() {
    echo "Test: auth cache normalization..."
    local output
    output="$(run_script --auth-file "${AUTH_FILE}" --dest-file "${DEST_FILE}" 2>&1)" || {
        test_fail "script should succeed on normalized cache write"
        return
    }

    if [[ -f "${DEST_FILE}" ]]; then
        test_pass "destination cache written"
    else
        test_fail "destination cache missing"
    fi

    if grep -Fq '"access_token": "access-token"' "${DEST_FILE}" || grep -Fq '"access_token":"access-token"' "${DEST_FILE}"; then
        test_pass "destination cache contains normalized token payload"
    else
        test_fail "destination cache missing normalized token payload"
    fi

    if grep -Fq "Wrote normalized auth cache" <<<"${output}"; then
        test_pass "script reports normalized cache write"
    else
        test_fail "script should report normalized cache write"
    fi
}

main() {
    echo "ChatGPT Auth Cache Copy Contract Tests"
    echo "====================================="
    echo ""

    test_help_contract
    test_normalizes_codex_cache

    test_results_exit "All chatgpt auth cache copy tests passed." "One or more chatgpt auth cache copy tests failed."
}

main "$@"
