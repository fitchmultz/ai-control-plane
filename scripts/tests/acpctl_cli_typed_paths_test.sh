#!/usr/bin/env bash
set -euo pipefail

# ACPCTL CLI Typed Path Test
#
# Purpose:
#   - Verify typed command paths remain make-independent.
#
# Responsibilities:
#   - Build an isolated acpctl binary and stub make for capture.
#   - Assert typed validation, key, and CI commands do not delegate to make.
#
# Scope:
#   - Typed command execution paths only.
#
# Usage:
#   - bash scripts/tests/acpctl_cli_typed_paths_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require real runtime services.

show_help() {
	cat <<'EOF'
Usage: acpctl_cli_typed_paths_test.sh [OPTIONS]

Validate make-independent typed acpctl command paths.

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

assert_no_delegation() {
	local description="$1"
	shift
	if ! run_with_make_stub "$@" >/dev/null 2>&1; then
		printf '  ✗ %s\n' "${description}"
		exit 1
	fi
	if [[ -s "${CAPTURE_FILE}" ]]; then
		printf '  ✗ %s\n' "${description}"
		exit 1
	fi
	printf '  ✓ %s\n' "${description}"
}

printf 'ACPCTL CLI Typed Path Test\n'
printf '==========================\n'

assert_no_delegation "env get help stays make-independent" env get --help
assert_no_delegation "validate config stays make-independent" validate config
run_with_make_stub validate config --production --secrets-env-file /tmp/secrets.env >/dev/null 2>&1 || true
if [[ -s "${CAPTURE_FILE}" ]]; then
	printf '  ✗ validate config --production should stay make-independent\n'
	exit 1
fi
printf '  ✓ validate config --production stays make-independent\n'
assert_no_delegation "chargeback report help stays make-independent" chargeback report --help
assert_no_delegation "host preflight help stays make-independent" host preflight --help
assert_no_delegation "bridge host_preflight help stays make-independent" bridge host_preflight --help

key_output="$(
	ACPCTL_BIN="${GO_SHIM}" \
		ACPCTL_MAKE_BIN="${MAKE_STUB}" \
		ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
		ACP_REPO_ROOT="${REPO_ROOT}" \
		"${SCRIPT_UNDER_TEST}" key gen contract-test --budget 1.00 --dry-run 2>&1
)"
if [[ -s "${CAPTURE_FILE}" ]]; then
	printf '  ✗ key gen dry-run should not delegate to make\n'
	exit 1
fi
if ! grep -Fq "Alias: contract-test" <<<"${key_output}" || ! grep -Fq 'Budget: $1.00' <<<"${key_output}"; then
	printf '  ✗ key gen dry-run output should include alias and budget\n'
	exit 1
fi
printf '  ✓ key gen dry-run stays typed and prints details\n'

ci_rc=0
ACPCTL_BIN="${GO_SHIM}" \
	ACPCTL_MAKE_BIN="${MAKE_STUB}" \
	ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
	ACP_REPO_ROOT="${REPO_ROOT}" \
	"${SCRIPT_UNDER_TEST}" ci should-run-runtime --path docs/README.md --quiet >/dev/null 2>&1 || ci_rc=$?
if [[ "${ci_rc}" -ne 0 && "${ci_rc}" -ne 1 ]]; then
	printf '  ✗ ci should-run-runtime should exit 0 or 1 (got %s)\n' "${ci_rc}"
	exit 1
fi
if [[ -s "${CAPTURE_FILE}" ]]; then
	printf '  ✗ ci should-run-runtime should not delegate to make\n'
	exit 1
fi
printf '  ✓ ci should-run-runtime stays make-independent\n'
