#!/usr/bin/env bash
set -euo pipefail

# Upgrade Make Contract Test
#
# Purpose:
#   - Verify Make-backed upgrade entrypoints invoke the typed CLI once.
#
# Responsibilities:
#   - Stub `ACPCTL_BIN` for upgrade make targets.
#   - Assert each target forwards the expected typed upgrade command.
#
# Scope:
#   - Makefile upgrade wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/upgrade_make_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally without requiring Ansible or PostgreSQL.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: upgrade_make_contract_test.sh [OPTIONS]

Validate that `make upgrade-*` entrypoints call the typed CLI once.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/upgrade_make_contract_test.sh

Exit codes:
  0  success
  1  contract failure
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
test_fixture_init upgrade-make-contract-test

CAPTURE_FILE="${TEST_TMP_ROOT}/acpctl-calls.txt"
ACPCTL_STUB="${TEST_TMP_ROOT}/acpctl-stub.sh"

cat >"${ACPCTL_STUB}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${ACPCTL_TEST_CAPTURE_FILE}"
EOF
chmod +x "${ACPCTL_STUB}"

run_make_target() {
    local target="$1"
    : >"${CAPTURE_FILE}"
    (
        cd "${REPO_ROOT}"
        ACPCTL_BIN="${ACPCTL_STUB}" \
            ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
            FROM_VERSION="0.0.9" \
            UPGRADE_RUN_DIR="demo/logs/upgrades/upgrade-20260317T120000.000000000Z" \
            make --silent -o install-binary "${target}"
    ) >/dev/null
}

assert_single_invocation() {
    local target="$1"
    local expected="$2"
    run_make_target "${target}"
    local actual
    actual="$(tr -d '\r' <"${CAPTURE_FILE}")"
    if [[ "${actual}" != "${expected}"$'\n' && "${actual}" != "${expected}" ]]; then
        printf '  ✗ %s should invoke "%s" once (got %q)\n' "${target}" "${expected}" "${actual}"
        exit 1
    fi
    if [[ "$(wc -l <"${CAPTURE_FILE}")" -ne 1 ]]; then
        printf '  ✗ %s should invoke ACPCTL_BIN exactly once\n' "${target}"
        exit 1
    fi
    printf '  ✓ %s -> %s\n' "${target}" "${expected}"
}

printf 'Upgrade Make Contract Test\n'
printf '==========================\n'

assert_single_invocation "upgrade-plan" "upgrade plan --from 0.0.9 --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env"
assert_single_invocation "upgrade-check" "upgrade check --from 0.0.9 --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env"
assert_single_invocation "upgrade-execute" "upgrade execute --from 0.0.9 --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env"
assert_single_invocation "upgrade-rollback" "upgrade rollback --run-dir demo/logs/upgrades/upgrade-20260317T120000.000000000Z --inventory deploy/ansible/inventory/hosts.yml --env-file /etc/ai-control-plane/secrets.env"
