#!/usr/bin/env bash
#
# AI Control Plane - Shell Test Helpers
#
# Purpose:
#   Share common shell-test setup, fixture, assertion, and stub-install helpers.
#
# Responsibilities:
#   - Resolve the repository root through the canonical bridge helpers.
#   - Manage temporary fixture lifecycle and reusable test result accounting.
#   - Install reusable stub binaries and fixture repo layouts for script tests.
#
# Scope:
#   - Supports scripts/tests/*.sh contract tests only.
#   - Does not own production bridge or onboarding business logic.
#
# Usage:
#   - Source from shell tests under scripts/tests/.
#   - Call helper functions to build temp fixtures, assertions, and stubs.
#
# Invariants/Assumptions:
#   - Relies on scripts/libexec/common.sh for canonical repo-root behavior.
#   - Expects callers to run under bash with `set -euo pipefail`.

set -euo pipefail

TEST_HELPERS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${TEST_HELPERS_DIR}/../libexec/common.sh"

test_repo_root() {
    bridge_repo_root
}

test_fixture_init() {
    local prefix="${1:-acp-test}"
    TEST_TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/${prefix}.XXXXXX")"
    trap 'rm -rf "${TEST_TMP_ROOT:-}"' EXIT
}

test_fixture_repo_init() {
    local fixture_root="$1"
    TEST_FIXTURE_REPO="${fixture_root}/repo"
    TEST_FIXTURE_BIN_DIR="${TEST_FIXTURE_REPO}/.bin"
    TEST_FIXTURE_SCRIPT_DIR="${TEST_FIXTURE_REPO}/scripts/libexec"
    TEST_FIXTURE_DEMO_DIR="${TEST_FIXTURE_REPO}/demo"
    TEST_FIXTURE_STUB_BIN_DIR="${fixture_root}/bin"
    TEST_REPO="${TEST_FIXTURE_REPO}"
    TEST_BIN_DIR="${TEST_FIXTURE_BIN_DIR}"
    TEST_SCRIPT_DIR="${TEST_FIXTURE_SCRIPT_DIR}"
    TEST_DEMO_DIR="${TEST_FIXTURE_DEMO_DIR}"
    TEST_STUB_BIN_DIR="${TEST_FIXTURE_STUB_BIN_DIR}"
    mkdir -p "${TEST_FIXTURE_BIN_DIR}" "${TEST_FIXTURE_SCRIPT_DIR}" "${TEST_FIXTURE_DEMO_DIR}" "${TEST_FIXTURE_STUB_BIN_DIR}"
}

test_fixture_copy_libexec() {
    local script_name
    mkdir -p "${TEST_FIXTURE_SCRIPT_DIR}"
    cp "$(test_repo_root)/scripts/libexec/common.sh" "${TEST_FIXTURE_SCRIPT_DIR}/common.sh"
    chmod +x "${TEST_FIXTURE_SCRIPT_DIR}/common.sh"
    for script_name in "$@"; do
        cp "$(test_repo_root)/scripts/libexec/${script_name}" "${TEST_FIXTURE_SCRIPT_DIR}/${script_name}"
        chmod +x "${TEST_FIXTURE_SCRIPT_DIR}/${script_name}"
    done
}

test_write_fixture_env() {
    cat >"${TEST_FIXTURE_DEMO_DIR}/.env"
}

test_assert_contains() {
    local haystack="$1"
    local needle="$2"
    local description="$3"
    if grep -Fq "${needle}" <<<"${haystack}"; then
        printf '  ✓ %s\n' "${description}"
        return 0
    fi
    printf '  ✗ %s\n' "${description}"
    return 1
}

test_assert_file_contains() {
    local file_path="$1"
    local needle="$2"
    local description="$3"
    if grep -Fq "${needle}" "${file_path}"; then
        printf '  ✓ %s\n' "${description}"
        return 0
    fi
    printf '  ✗ %s\n' "${description}"
    return 1
}

test_assert_exit_code() {
    local actual="$1"
    local expected="$2"
    local description="$3"
    if [[ "${actual}" == "${expected}" ]]; then
        printf '  ✓ %s\n' "${description}"
        return 0
    fi
    printf '  ✗ %s (expected %s, got %s)\n' "${description}" "${expected}" "${actual}"
    return 1
}

test_build_acpctl_binary() {
    local output_path="$1"
    go build -trimpath -o "${output_path}" "$(test_repo_root)/cmd/acpctl"
}

test_create_exec_shim() {
    local target="$1"
    local shim_path="$2"
    cat >"${shim_path}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "${target}" "\$@"
EOF
    chmod +x "${shim_path}"
}

test_install_stub() {
    local stub_name="$1"
    local destination_dir="$2"
    local destination_name="${3:-${stub_name}}"
    install -m 755 "${TEST_HELPERS_DIR}/stubs/${stub_name}" "${destination_dir}/${destination_name}"
}

test_results_init() {
    TESTS_PASSED=0
    TESTS_FAILED=0
}

test_pass() {
    printf '  ✓ %s\n' "$1"
    ((TESTS_PASSED++)) || true
}

test_fail() {
    printf '  ✗ %s\n' "$1"
    ((TESTS_FAILED++)) || true
}

test_results_summary() {
    printf '\nResults\n'
    printf '%s\n' '-------'
    printf '  Passed: %s\n' "${TESTS_PASSED}"
    printf '  Failed: %s\n' "${TESTS_FAILED}"
}

test_results_exit() {
    local success_message="$1"
    local failure_message="$2"
    test_results_summary
    if [[ "${TESTS_FAILED}" -eq 0 ]]; then
        printf '%s\n' "${success_message}"
        exit 0
    fi
    printf '%s\n' "${failure_message}"
    exit 1
}
