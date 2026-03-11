#!/usr/bin/env bash
set -euo pipefail

# DLP Health Contract Test
#
# Purpose:
#   - Verify the Make-backed DLP runtime health target checks the active runtime
#     instead of delegating to static repository validation.
#
# Responsibilities:
#   - Assert `make dlp-health` fails when the DLP overlay is absent.
#   - Assert `make dlp-health` succeeds when both Presidio services are healthy.
#   - Assert `make dlp-health` fails when a Presidio service is unhealthy.
#
# Scope:
#   - Makefile DLP overlay health wrapper behavior only.
#
# Usage:
#   - bash scripts/tests/dlp_health_contract_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally with a stubbed docker CLI and do not require live
#     containers.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: dlp_health_contract_test.sh [OPTIONS]

Validate that `make dlp-health` checks live Presidio container health.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
test_fixture_init dlp-health-contract-test

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
if [[ "$*" == *" ps -q presidio-analyzer" ]]; then
    if [[ -n "${ACP_TEST_PRESIDIO_ANALYZER_ID:-}" ]]; then
        printf '%s\n' "${ACP_TEST_PRESIDIO_ANALYZER_ID}"
    fi
    exit 0
fi
if [[ "$*" == *" ps -q presidio-anonymizer" ]]; then
    if [[ -n "${ACP_TEST_PRESIDIO_ANONYMIZER_ID:-}" ]]; then
        printf '%s\n' "${ACP_TEST_PRESIDIO_ANONYMIZER_ID}"
    fi
    exit 0
fi
if [[ "${1:-}" == "inspect" ]]; then
    container_id="${@: -1}"
    case "${container_id}" in
        analyzer-id)
            printf '%s\n' "${ACP_TEST_PRESIDIO_ANALYZER_STATUS:-healthy}"
            ;;
        anonymizer-id)
            printf '%s\n' "${ACP_TEST_PRESIDIO_ANONYMIZER_STATUS:-healthy}"
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

run_dlp_health() {
    local analyzer_id="$1"
    local anonymizer_id="$2"
    local analyzer_status="$3"
    local anonymizer_status="$4"
    local exit_code=0

    : >"${CAPTURE_FILE}"
    set +e
    (
        cd "${REPO_ROOT}"
        PATH="${TEST_TMP_ROOT}:${PATH}" \
            ACP_DOCKER_TEST_CAPTURE_FILE="${CAPTURE_FILE}" \
            ACP_TEST_PRESIDIO_ANALYZER_ID="${analyzer_id}" \
            ACP_TEST_PRESIDIO_ANONYMIZER_ID="${anonymizer_id}" \
            ACP_TEST_PRESIDIO_ANALYZER_STATUS="${analyzer_status}" \
            ACP_TEST_PRESIDIO_ANONYMIZER_STATUS="${anonymizer_status}" \
            COMPOSE_ENV_FILE="${ENV_FILE}" \
            make --silent dlp-health
    ) >"${STDOUT_FILE}" 2>"${STDERR_FILE}"
    exit_code=$?
    set -e
    return "${exit_code}"
}

printf 'DLP Health Contract Test\n'
printf '========================\n'

if run_dlp_health "" "" "healthy" "healthy"; then
    printf '  ✗ dlp-health should fail when the DLP overlay is absent\n'
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "DLP overlay is not active" "dlp-health fails clearly when overlay is absent"

if ! run_dlp_health "analyzer-id" "anonymizer-id" "healthy" "healthy"; then
    printf '  ✗ dlp-health should succeed when both Presidio services are healthy\n'
    cat "${STDOUT_FILE}"
    cat "${STDERR_FILE}" >&2
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "DLP overlay services are healthy" "dlp-health succeeds when both services are healthy"
test_assert_file_contains "${CAPTURE_FILE}" "-f docker-compose.yml -f docker-compose.dlp.yml" "dlp-health checks the base plus DLP compose surfaces"
test_assert_file_contains "${CAPTURE_FILE}" "ps -q presidio-analyzer" "dlp-health inspects presidio-analyzer"
test_assert_file_contains "${CAPTURE_FILE}" "ps -q presidio-anonymizer" "dlp-health inspects presidio-anonymizer"
test_assert_file_contains "${CAPTURE_FILE}" "inspect --format {{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}} analyzer-id" "dlp-health inspects analyzer container health"
test_assert_file_contains "${CAPTURE_FILE}" "inspect --format {{if .State.Health}}{{.State.Health.Status}}{{else}}missing-healthcheck{{end}} anonymizer-id" "dlp-health inspects anonymizer container health"

if run_dlp_health "analyzer-id" "anonymizer-id" "healthy" "unhealthy"; then
    printf '  ✗ dlp-health should fail when a Presidio service is unhealthy\n'
    exit 1
fi
test_assert_file_contains "${STDOUT_FILE}" "presidio-anonymizer=unhealthy" "dlp-health reports unhealthy Presidio services"
