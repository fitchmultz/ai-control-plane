#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Delegation Contract Test
#
# Purpose:
#   - Verify delegated flow commands invoke the expected make targets.
#
# Responsibilities:
#   - Build an isolated acpctl binary and stub make for capture.
#   - Assert delegated command paths stay mapped to the intended targets.
#
# Scope:
#   - Make delegation behavior only.
#
# Usage:
#   - bash scripts/tests/acpctl_cli_delegation_test.sh
#
# Invariants/Assumptions:
#   - Tests run in a temp fixture and do not invoke real make targets.

show_help() {
    cat <<'EOF'
Usage: acpctl_cli_delegation_test.sh [OPTIONS]

Validate delegated make targets for scripts/acpctl.sh.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCRIPT_UNDER_TEST="${REPO_ROOT}/scripts/acpctl.sh"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

CAPTURE_FILE="${TMP_DIR}/make-args.txt"
ACPCTL_BIN_TEST="${TMP_DIR}/acpctl-bin"
GO_SHIM="${TMP_DIR}/acpctl-go-shim.sh"
MAKE_STUB="${TMP_DIR}/make-stub.sh"

go build -trimpath -o "${ACPCTL_BIN_TEST}" "${REPO_ROOT}/cmd/acpctl"

cat >"${GO_SHIM}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "${ACPCTL_BIN_TEST}" "\$@"
EOF
chmod +x "${GO_SHIM}"

cat >"${MAKE_STUB}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
capture_file="${ACPCTL_TEST_CAPTURE_FILE:?missing ACPCTL_TEST_CAPTURE_FILE}"
printf '%s\n' "$@" >"${capture_file}"
exit 0
EOF
chmod +x "${MAKE_STUB}"

run_with_make_stub() {
    : >"${CAPTURE_FILE}"
    ACPCTL_BIN="${GO_SHIM}" \
        ACPCTL_MAKE_BIN="${MAKE_STUB}" \
        ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
        ACP_REPO_ROOT="${REPO_ROOT}" \
        "${SCRIPT_UNDER_TEST}" "$@"
}

assert_make_target() {
    local expected_target="$1"
    shift
    run_with_make_stub "$@" >/dev/null 2>&1
    actual_target="$(head -n1 "${CAPTURE_FILE}" || true)"
    if [[ "${actual_target}" != "${expected_target}" ]]; then
        printf '  ✗ expected target %s, got %s\n' "${expected_target}" "${actual_target}"
        exit 1
    fi
    printf '  ✓ %s -> %s\n' "$*" "${expected_target}"
}

printf 'ACPCTL CLI Delegation Contract Test\n'
printf '===================================\n'

assert_make_target "up" deploy up
assert_make_target "db-status" db status
assert_make_target "demo-all" demo all
assert_make_target "up-tls" deploy up-tls
assert_make_target "tf-plan" terraform plan

missing_rc=0
ACPCTL_BIN="${GO_SHIM}" \
    ACPCTL_MAKE_BIN="${TMP_DIR}/missing-make" \
    ACP_REPO_ROOT="${REPO_ROOT}" \
    "${SCRIPT_UNDER_TEST}" deploy up >/dev/null 2>&1 || missing_rc=$?
if [[ "${missing_rc}" -ne 2 ]]; then
    printf '  ✗ missing ACPCTL_MAKE_BIN should exit 2 (got %s)\n' "${missing_rc}"
    exit 1
fi
printf '  ✓ missing ACPCTL_MAKE_BIN exits 2\n'
