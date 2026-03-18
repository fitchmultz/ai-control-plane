#!/usr/bin/env bash
set -euo pipefail

# DB Make Contract Test
#
# Purpose:
#   - Verify Make-backed database entrypoints invoke the typed CLI once and do
#     not recurse back into Make.
#
# Responsibilities:
#   - Stub the `ACPCTL_BIN` target for `make db-status`, `make db-shell`, and retention wrappers.
#   - Assert each target executes the expected typed subcommand exactly once.
#
# Scope:
#   - Makefile database wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/db_make_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally without requiring Docker or PostgreSQL.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: db_make_contract_test.sh [OPTIONS]

Validate that `make db-status`, `make db-shell`, and backup-retention entrypoints call the typed CLI once.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
test_fixture_init db-make-contract-test

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
            make --silent "${target}"
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

printf 'DB Make Contract Test\n'
printf '=====================\n'

assert_single_invocation "db-status" "db status"
assert_single_invocation "db-shell" "db shell"
assert_single_invocation "db-backup-retention" "db backup-retention --check"
run_make_target_with_manifest() {
    : >"${CAPTURE_FILE}"
    (
        cd "${REPO_ROOT}"
        ACPCTL_BIN="${ACPCTL_STUB}" \
            ACPCTL_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
            OFF_HOST_RECOVERY_MANIFEST="demo/logs/recovery-inputs/off_host_recovery.yaml" \
            make --silent db-off-host-drill
    ) >/dev/null
}

run_make_target_with_manifest
actual="$(tr -d '\r' <"${CAPTURE_FILE}")"
expected="db off-host-drill --manifest demo/logs/recovery-inputs/off_host_recovery.yaml"
if [[ "${actual}" != "${expected}"$'\n' && "${actual}" != "${expected}" ]]; then
    printf '  ✗ db-off-host-drill should invoke "%s" once (got %q)\n' "${expected}" "${actual}"
    exit 1
fi
printf '  ✓ db-off-host-drill -> %s\n' "${expected}"

printf '\n'
printf 'Production Runtime Contract Test\n'
printf '===============================\n'

DOCKER_STUB="${TEST_TMP_ROOT}/docker"
MAKE_CAPTURE="${TEST_TMP_ROOT}/docker-compose-calls.txt"
SECRETS_FILE="${TEST_TMP_ROOT}/secrets.env"

cat >"${SECRETS_FILE}" <<'EOF'
ACP_DATABASE_MODE=embedded
POSTGRES_USER=litellm
POSTGRES_PASSWORD=supersecurepostgres1
POSTGRES_DB=litellm
DATABASE_URL=postgresql://litellm:supersecurepostgres1@postgres:5432/litellm
LITELLM_MASTER_KEY=0123456789abcdef0123456789abcdef
LITELLM_SALT_KEY=abcdef0123456789abcdef0123456789
EOF
chmod 600 "${SECRETS_FILE}"

cat >"${DOCKER_STUB}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    exit 0
fi
printf '%s\n' "$*" >>"${ACP_DOCKER_TEST_CAPTURE_FILE}"
exit 0
EOF
chmod +x "${DOCKER_STUB}"

: >"${MAKE_CAPTURE}"
(
    cd "${REPO_ROOT}"
    PATH="${TEST_TMP_ROOT}:${PATH}" \
        ACPCTL_BIN="${TEST_TMP_ROOT}/acpctl-bin" \
        ACP_DOCKER_TEST_CAPTURE_FILE="${MAKE_CAPTURE}" \
        SECRETS_ENV_FILE="${SECRETS_FILE}" \
        make --silent up-production
) >/dev/null

test_assert_file_contains "${MAKE_CAPTURE}" "--profile embedded-db" "up-production includes embedded-db profile"
test_assert_file_contains "${MAKE_CAPTURE}" "--profile production" "up-production includes production profile"
test_assert_file_contains "${MAKE_CAPTURE}" "-f docker-compose.yml -f docker-compose.tls.yml" "up-production uses base plus TLS compose files"
test_assert_file_contains "${MAKE_CAPTURE}" "up -d --timeout 120 postgres litellm caddy otel-collector" "up-production starts postgres, litellm, caddy, and otel-collector"
