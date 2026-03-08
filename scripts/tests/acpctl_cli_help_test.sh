#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Help Contract Test
#
# Purpose:
#   - Verify help output exposes the expected typed command surface.
#
# Responsibilities:
#   - Build an isolated acpctl binary for contract testing.
#   - Assert required help sections and core commands remain visible.
#
# Scope:
#   - Help output and unknown-command behavior only.
#
# Usage:
#   - bash scripts/tests/acpctl_cli_help_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require Docker or make delegation.

show_help() {
	cat <<'EOF'
Usage: acpctl_cli_help_test.sh [OPTIONS]

Validate scripts/acpctl.sh help output and usage behavior.

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

ACPCTL_BIN_TEST="${TMP_DIR}/acpctl-bin"
GO_SHIM="${TMP_DIR}/acpctl-go-shim.sh"

go build -trimpath -o "${ACPCTL_BIN_TEST}" "${REPO_ROOT}/cmd/acpctl"

cat >"${GO_SHIM}" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec "${ACPCTL_BIN_TEST}" "\$@"
EOF
chmod +x "${GO_SHIM}"

assert_contains() {
	local haystack="$1"
	local needle="$2"
	local description="$3"
	if grep -Fq "${needle}" <<<"${haystack}"; then
		printf '  ✓ %s\n' "${description}"
	else
		printf '  ✗ %s\n' "${description}"
		exit 1
	fi
}

printf 'ACPCTL CLI Help Contract Test\n'
printf '=============================\n'

help_output="$(ACPCTL_BIN="${GO_SHIM}" "${SCRIPT_UNDER_TEST}" --help 2>&1)"
assert_contains "${help_output}" "Usage: acpctl" "--help includes usage"
assert_contains "${help_output}" "Commands:" "--help includes Commands section"
assert_contains "${help_output}" "Examples:" "--help includes Examples section"
assert_contains "${help_output}" "env" "--help lists env command"
assert_contains "${help_output}" "chargeback" "--help lists chargeback command"
assert_contains "${help_output}" "deploy" "--help lists deploy command"
assert_contains "${help_output}" "validate" "--help lists validate command"
assert_contains "${help_output}" "db" "--help lists db command"
assert_contains "${help_output}" "key" "--help lists key command"
assert_contains "${help_output}" "doctor" "--help lists doctor command"

unknown_rc=0
ACPCTL_BIN="${GO_SHIM}" "${SCRIPT_UNDER_TEST}" unknown-command >/dev/null 2>&1 || unknown_rc=$?
if [[ "${unknown_rc}" -ne 64 ]]; then
	printf '  ✗ unknown command should exit 64 (got %s)\n' "${unknown_rc}"
	exit 1
fi
printf '  ✓ unknown command exits 64\n'
