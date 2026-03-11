#!/usr/bin/env bash
set -euo pipefail

# LibreChat Health Contract Test
#
# Purpose:
#   - Verify the Make-backed managed UI runtime health target checks the active
#     runtime instead of probing a localhost port without runtime context.
#
# Responsibilities:
#   - Assert `make librechat-health` fails when the managed UI overlay is absent.
#   - Assert `make librechat-health` succeeds when all managed UI services are healthy.
#   - Assert `make librechat-health` fails when a managed UI service is unhealthy.
#
# Scope:
#   - Makefile managed UI overlay health wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/librechat_health_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally with a stubbed docker CLI and do not require live
#     containers.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: librechat_health_contract_test.sh [OPTIONS]

Validate that `make librechat-health` checks live managed UI overlay container health.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
test_fixture_init librechat-health-contract-test

DOCKER_STUB="${TEST_TMP_ROOT}/docker"
CAPTURE_FILE="${TEST_TMP_ROOT}/docker-calls.txt"
STDOUT_FILE="${TEST_TMP_ROOT}/stdout.txt"
STDERR_FILE="${TEST_TMP_ROOT}/stderr.txt"
ENV_FILE="${TEST_TMP_ROOT}/runtime.env"

cat >"${ENV_FILE}" <<'EOF'
ACP_DATABASE_MODE=embedded
EOF

cat >"${DOCKER_STUB}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${ACP_DOCKER_TEST_CAPTURE_FILE}"
if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    exit 0
fi
if [[ "$*" == *" ps -q librechat " ]] || [[ "$*" == *" ps -q librechat" ]]; then
    if [[ -n "${ACP_TEST_LIBRECHAT_ID:-}" ]]; then
        printf '%s\n' "${ACP_TEST_LIBRECHAT_ID}"
    fi
    exit 0
fi
if [[ "$*" == *" ps -q librechat-mongodb" ]]; then
    if [[ -n "${ACP_TEST_LIBRECHAT_MONGODB_ID:-}" ]]; then
        printf '%s\n' "${ACP_TEST_LIBRECHAT_MONGODB_ID}"
    fi
    exit 0
fi
if [[ "$*" == *" ps -q librechat-meilisearch" ]]; then
    if [[ -n "${ACP_TEST_LIBRECHAT_MEILI_ID:-}" ]]; then
        printf '%s\n' "${ACP_TEST_LIBRECHAT_MEILI_ID}"
    fi
    exit 0
fi
if [[ "${1:-}" == "inspect" ]]; then
    container_id="${@: -1}"
    case "${container_id}" in
        librechat-id)
            printf '%s\n' "${ACP_TEST_LIBRECHAT_STATUS:-healthy}"
            ;;
        mongodb-id)
            printf '%s\n' "${ACP_TEST_LIBRECHAT_MONGODB_STATUS:-healthy}"
            ;;
        meili-id)
            printf '%s\n' "${ACP_TEST_LIBRECHAT_MEILI_STATUS:-healthy}"
            ;;
        *)
            exit 1
            ;;
    esac
    exit 0
fi
exit 0
EOF
chmod +x "${DOCKER_STUB}"

run_librechat_health() {
    local librechat_id="$1"
    local mongodb_id="$2"
    local meili_id="$3"
    local librechat_status="$4"
    local mongodb_status="$5"
    local meili_status="$6"
    local exit_code=0

    : >"${CAPTURE_FILE}"
    set +e
    (
        cd "${REPO_ROOT}"
        PATH="${TEST_TMP_ROOT}:${PATH}" \
            ACP_DOCKER_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
            ACP_TEST_LIBRECHAT_ID="${librechat_id}" \
            ACP_TEST_LIBRECHAT_MONGODB_ID="${mongodb_id}" \
            ACP_TEST_LIBRECHAT_MEILI_ID="${meili_id}" \
            ACP_TEST_LIBRECHAT_STATUS="${librechat_status}" \
            ACP_TEST_LIBRECHAT_MONGODB_STATUS="${mongodb_status}" \
            ACP_TEST_LIBRECHAT_MEILI_STATUS="${meili_status}" \
            COMPOSE_ENV_FILE="${ENV_FILE}" \
            make --silent librechat-health
    ) >"${STDOUT_FILE}" 2>"${STDERR_FILE}"
    exit_code=$?
    set -e
    return "${exit_code}"
}

printf 'LibreChat Health Contract Test\n'
printf '==============================\n'

if run_librechat_health "" "" "" "healthy" "healthy" "healthy"; then
    printf '  ✗ librechat-health should fail when the managed UI overlay is absent\n'
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "Managed UI overlay is not active" "librechat-health fails clearly when overlay is absent"

if ! run_librechat_health "librechat-id" "mongodb-id" "meili-id" "healthy" "healthy" "healthy"; then
    printf '  ✗ librechat-health should succeed when all managed UI services are healthy\n'
    cat "${STDOUT_FILE}"
    cat "${STDERR_FILE}" >&2
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "Managed UI overlay services are healthy" "librechat-health succeeds when all managed UI services are healthy"
test_assert_file_contains "${CAPTURE_FILE}" "-f docker-compose.yml -f docker-compose.ui.yml" "librechat-health checks the base plus UI compose surfaces"
test_assert_file_contains "${CAPTURE_FILE}" "ps -q librechat" "librechat-health inspects librechat"
test_assert_file_contains "${CAPTURE_FILE}" "ps -q librechat-mongodb" "librechat-health inspects librechat-mongodb"
test_assert_file_contains "${CAPTURE_FILE}" "ps -q librechat-meilisearch" "librechat-health inspects librechat-meilisearch"
test_assert_file_contains "${CAPTURE_FILE}" "inspect --format {{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}} librechat-id" "librechat-health inspects librechat container health"
test_assert_file_contains "${CAPTURE_FILE}" "inspect --format {{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}} mongodb-id" "librechat-health inspects mongodb container health"
test_assert_file_contains "${CAPTURE_FILE}" "inspect --format {{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}} meili-id" "librechat-health inspects meilisearch container health"

if run_librechat_health "librechat-id" "mongodb-id" "meili-id" "healthy" "healthy" "unhealthy"; then
    printf '  ✗ librechat-health should fail when a managed UI service is unhealthy\n'
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "librechat-meilisearch=unhealthy" "librechat-health reports unhealthy managed UI services"
