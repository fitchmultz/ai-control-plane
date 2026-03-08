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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SOURCE_SCRIPT="${PROJECT_ROOT}/scripts/libexec/chatgpt_auth_cache_copy_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_ROOT}"' EXIT

TEST_REPO="${TMP_ROOT}/repo"
TEST_SCRIPT_DIR="${TEST_REPO}/scripts/libexec"
TEST_BIN_DIR="${TMP_ROOT}/bin"
AUTH_FILE="${TMP_ROOT}/auth.json"
DEST_FILE="${TEST_REPO}/demo/auth/chatgpt/auth.json"

mkdir -p "${TEST_SCRIPT_DIR}" "${TEST_BIN_DIR}"
cp "${SOURCE_SCRIPT}" "${TEST_SCRIPT_DIR}/chatgpt_auth_cache_copy_impl.sh"
chmod +x "${TEST_SCRIPT_DIR}/chatgpt_auth_cache_copy_impl.sh"

cat >"${AUTH_FILE}" <<'EOF'
{"tokens":{"access_token":"access-token","refresh_token":"refresh-token","id_token":"id-token","account_id":"acct-123"}}
EOF

cat >"${TEST_BIN_DIR}/jq" <<'EOF'
#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
if args and args[0] == "-e":
    expr = args[1]
    path = pathlib.Path(args[2])
    data = json.loads(path.read_text())
    ok = False
    if expr == '.tokens.access_token and .tokens.refresh_token and .tokens.id_token':
        ok = isinstance(data.get("tokens"), dict) and all(data["tokens"].get(k) for k in ("access_token", "refresh_token", "id_token"))
    elif expr == '.access_token and .refresh_token and .id_token':
        ok = all(data.get(k) for k in ("access_token", "refresh_token", "id_token"))
    sys.exit(0 if ok else 1)

expr = args[0]
path = pathlib.Path(args[1])
data = json.loads(path.read_text())
if expr.startswith('{'):
    if "tokens." in expr:
        tokens = data["tokens"]
        output = {
            "access_token": tokens["access_token"],
            "refresh_token": tokens["refresh_token"],
            "id_token": tokens["id_token"],
            "account_id": tokens.get("account_id"),
        }
    else:
        output = {
            "access_token": data["access_token"],
            "refresh_token": data["refresh_token"],
            "id_token": data["id_token"],
            "account_id": data.get("account_id"),
            "expires_at": data.get("expires_at"),
        }
    sys.stdout.write(json.dumps(output))
    sys.exit(0)
sys.exit(1)
EOF
chmod +x "${TEST_BIN_DIR}/jq"

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

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

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
        pass "help contains usage"
    else
        fail "help missing usage"
    fi
}

test_normalizes_codex_cache() {
    echo "Test: auth cache normalization..."
    local output
    output="$(run_script --auth-file "${AUTH_FILE}" --dest-file "${DEST_FILE}" 2>&1)" || {
        fail "script should succeed on normalized cache write"
        return
    }

    if [[ -f "${DEST_FILE}" ]]; then
        pass "destination cache written"
    else
        fail "destination cache missing"
    fi

    if grep -Fq '"access_token": "access-token"' "${DEST_FILE}" || grep -Fq '"access_token":"access-token"' "${DEST_FILE}"; then
        pass "destination cache contains normalized token payload"
    else
        fail "destination cache missing normalized token payload"
    fi

    if grep -Fq "Wrote normalized auth cache" <<<"${output}"; then
        pass "script reports normalized cache write"
    else
        fail "script should report normalized cache write"
    fi
}

main() {
    echo "ChatGPT Auth Cache Copy Contract Tests"
    echo "====================================="
    echo ""

    test_help_contract
    test_normalizes_codex_cache

    echo ""
    echo "Results"
    echo "-------"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"

    if [[ "${TESTS_FAILED}" -eq 0 ]]; then
        echo "All chatgpt auth cache copy tests passed."
        exit 0
    fi

    echo "One or more chatgpt auth cache copy tests failed."
    exit 1
}

main "$@"
