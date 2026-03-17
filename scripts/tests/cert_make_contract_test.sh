#!/usr/bin/env bash
set -euo pipefail

# Certificate Make Contract Test
#
# Purpose:
#   - Verify Make-backed certificate entrypoints invoke the typed CLI once.
#
# Responsibilities:
#   - Stub `ACPCTL_BIN` for certificate make targets.
#   - Assert each target forwards the expected typed certificate command.
#
# Scope:
#   - Makefile certificate wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/cert_make_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally without requiring Docker or systemd.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: cert_make_contract_test.sh [OPTIONS]

Validate that `make cert-*` entrypoints call the typed CLI once.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/cert_make_contract_test.sh

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
test_fixture_init cert-make-contract-test

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
            SECRETS_ENV_FILE="/etc/ai-control-plane/secrets.env" \
            DOMAIN="gateway.example.com" \
            THRESHOLD_DAYS="30" \
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

printf 'Certificate Make Contract Test\n'
printf '==============================\n'

assert_single_invocation "cert-status" "cert check --domain gateway.example.com --threshold-days 30"
assert_single_invocation "cert-renew" "cert renew --domain gateway.example.com --threshold-days 30"
assert_single_invocation "cert-renew-install" "cert renew-auto --env-file /etc/ai-control-plane/secrets.env --threshold-days 30"
