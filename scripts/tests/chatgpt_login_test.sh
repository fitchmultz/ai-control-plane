#!/usr/bin/env bash
#
# AI Control Plane - ChatGPT Login Script Contract Tests
#
# Purpose:
#   Validate chatgpt_login_impl.sh reads env data safely and keeps its CLI contract.
#
# Responsibilities:
#   - Verify help output remains available.
#   - Verify LITELLM_MASTER_KEY is read without sourcing `.env`.
#   - Verify the happy path does not require real Docker or network access.
#
# Non-scope:
#   - Does not complete a real OAuth flow.
#   - Does not start real containers.
#
# Invariants/Assumptions:
#   - Tests run in a temporary fixture repo.
#   - Stubbed acpctl/docker/curl binaries provide deterministic behavior.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: chatgpt_login_test.sh [OPTIONS]

Run contract tests for scripts/libexec/chatgpt_login_impl.sh.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/chatgpt_login_test.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

test_results_init
test_fixture_init chatgpt-login-test
TMP_ROOT="${TEST_TMP_ROOT}"
test_fixture_repo_init "${TMP_ROOT}"
mkdir -p "${TEST_FIXTURE_DEMO_DIR}/auth/chatgpt"
test_fixture_copy_libexec "chatgpt_login_impl.sh"

test_write_fixture_env <<EOF
EVIL=\$(touch "${TMP_ROOT}/env-pwned")
LITELLM_MASTER_KEY=sk-master-test-12345
EOF

cat >"${TEST_BIN_DIR}/acpctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "env" && "${2:-}" == "get" && "${3:-}" == "--file" ]]; then
    file="${4:?missing file}"
    key="${5:?missing key}"
    awk -F= -v want="${key}" '
        /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
        {
            env_key=$1
            sub(/^[[:space:]]+/, "", env_key)
            sub(/[[:space:]]+$/, "", env_key)
            if (env_key == want) {
                sub(/^[^=]*=/, "", $0)
                gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
                print $0
                found=1
                exit 0
            }
        }
        END {
            if (!found) {
                exit 1
            }
        }
    ' "${file}"
    exit $?
fi

echo "unsupported acpctl stub command: $*" >&2
exit 1
EOF
chmod +x "${TEST_BIN_DIR}/acpctl"

test_install_stub "docker_compose_ok_stub.sh" "${TEST_STUB_BIN_DIR}" "docker"

cat >"${TEST_STUB_BIN_DIR}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$*" == *"%{http_code}"* ]]; then
    printf '200'
    exit 0
fi

case "$*" in
    *"/v1/models"*)
        printf '{"data":[{"id":"chatgpt-gpt5.3-codex"}]}'
        ;;
    *"/v1/responses"*)
        printf '{"output":"ok"}'
        ;;
    *)
        printf '{}'
        ;;
esac
EOF
chmod +x "${TEST_STUB_BIN_DIR}/curl"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/chatgpt_login_impl.sh"

run_script() {
    PATH="${TEST_STUB_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

test_help_contract() {
    echo "Test: script help..."
    local output
    output="$(run_script --help 2>&1)"

    if grep -Fq "Usage: chatgpt_login_impl.sh [options]" <<<"${output}"; then
        test_pass "help contains usage"
    else
        test_fail "help missing usage"
    fi
}

test_env_file_is_not_sourced() {
    echo "Test: env file is parsed as data, not sourced..."
    rm -f "${TMP_ROOT}/env-pwned"

    if output="$(run_script 2>&1)"; then
        :
    else
        test_fail "script should succeed on stubbed happy path"
        return
    fi

    if [[ ! -e "${TMP_ROOT}/env-pwned" ]]; then
        test_pass ".env payload was not executed"
    else
        test_fail ".env payload must never execute"
    fi

    if grep -Fq "Model alias 'chatgpt-gpt5.3-codex' is present" <<<"${output}"; then
        test_pass "script completed authorized model check"
    else
        test_fail "script should complete authorized model check"
    fi
}

main() {
    echo "ChatGPT Login Script Contract Tests"
    echo "==================================="
    echo ""

    test_help_contract
    test_env_file_is_not_sourced

    test_results_exit "All chatgpt login tests passed." "One or more chatgpt login tests failed."
}

main "$@"
