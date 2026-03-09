#!/usr/bin/env bash
set -euo pipefail

# Onboard Verify Mode Test
#
# Purpose:
#   - Verify onboarding `--verify` behavior and make target wiring.
#
# Responsibilities:
#   - Stub curl and acpctl interactions for deterministic verify checks.
#   - Assert verify path hits health/models endpoints and sends auth.
#
# Scope:
#   - Verify-path behavior and onboarding makefile wiring only.
#
# Usage:
#   - bash scripts/tests/onboard_verify_mode_test.sh
#
# Invariants/Assumptions:
#   - Tests run entirely from temp fixtures.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: onboard_verify_mode_test.sh [OPTIONS]

Validate onboarding verify behavior and makefile wiring.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

PROJECT_ROOT="$(test_repo_root)"
test_fixture_init onboard-verify-mode-test
TMP_ROOT="${TEST_TMP_ROOT}"
test_fixture_repo_init "${TMP_ROOT}"
CURL_LOG="${TMP_ROOT}/curl.log"
test_fixture_copy_libexec "onboard_impl.sh"

test_write_fixture_env <<'EOF'
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
shift || true
verify="false"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --verify) verify="true"; shift ;;
        *) shift ;;
    esac
done
printf 'Tool: %s\n' "${tool}"
if [[ "${verify}" == "true" ]]; then
    curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:4000/health" >/dev/null
    curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer sk-test-full-key-1234567890-abcdef" "http://127.0.0.1:4000/v1/models" >/dev/null
    printf 'INFO: Gateway health and authorized model checks passed\n'
fi
EOF
chmod +x "${TEST_BIN_DIR}/acpctl"

test_install_stub "curl_log_http_200_stub.sh" "${TEST_STUB_BIN_DIR}" "curl"

TEST_SCRIPT="${TEST_SCRIPT_DIR}/onboard_impl.sh"
run_onboard() {
    ACP_TEST_CURL_LOG="${CURL_LOG}" \
        PATH="${TEST_STUB_BIN_DIR}:${PATH}" \
        HOME="${TMP_ROOT}/home" \
        "${TEST_SCRIPT}" "$@"
}

printf 'Onboard Verify Mode Test\n'
printf '========================\n'

: >"${CURL_LOG}"
verify_output="$(run_onboard codex --mode api-key --verify 2>&1)"
if ! grep -Fq "Gateway health and authorized model checks passed" <<<"${verify_output}"; then
    printf '  ✗ verify should report success\n'
    exit 1
fi
printf '  ✓ verify reports success\n'

if ! grep -Fq "/health" "${CURL_LOG}"; then
    printf '  ✗ verify should call health endpoint\n'
    exit 1
fi
printf '  ✓ verify calls health endpoint\n'

if ! grep -Fq "/v1/models" "${CURL_LOG}"; then
    printf '  ✗ verify should call models endpoint\n'
    exit 1
fi
printf '  ✓ verify calls models endpoint\n'

if ! grep -Fq "Authorization: Bearer sk-test-full-key-1234567890-abcdef" "${CURL_LOG}"; then
    printf '  ✗ verify should send authorization header\n'
    exit 1
fi
printf '  ✓ verify sends authorization header\n'

mk_file="${PROJECT_ROOT}/mk/onboard.mk"
for target in onboard: onboard-codex: chatgpt-auth-copy:; do
    if ! grep -q "^${target}" "${mk_file}"; then
        printf '  ✗ missing make target %s\n' "${target}"
        exit 1
    fi
done
printf '  ✓ onboarding make targets remain present\n'
