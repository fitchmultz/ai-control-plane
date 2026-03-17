#!/usr/bin/env bash
#
# AI Control Plane - Host Install Bridge Test
#
# Purpose:
#   - Verify the host install bridge renders and manages both the runtime
#     service and automated backup timer units.
#
# Responsibilities:
#   - Exercise install, service-status, and uninstall flows with stubbed systemd.
#   - Assert rendered unit files contain the expected backup schedule contract.
#
# Scope:
#   - scripts/libexec/host_install_impl.sh only.
#
# Usage:
#   - bash scripts/tests/host_install_impl_test.sh
#
# Invariants/Assumptions:
#   - Tests run locally without requiring real systemd or Docker.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: host_install_impl_test.sh [OPTIONS]

Validate that host_install_impl.sh manages the runtime service and backup timer together.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

test_fixture_init host-install-bridge-test

REPO_ROOT="$(test_repo_root)"
FIXTURE_REPO="${TEST_TMP_ROOT}/fixture-repo"
UNIT_DIR="${TEST_TMP_ROOT}/units"
BIN_DIR="${TEST_TMP_ROOT}/bin"
SYSTEMCTL_CAPTURE="${TEST_TMP_ROOT}/systemctl-calls.txt"
SECRETS_FILE="${TEST_TMP_ROOT}/secrets.env"
SCRIPT_PATH="${REPO_ROOT}/scripts/libexec/host_install_impl.sh"

mkdir -p "${FIXTURE_REPO}/deploy/systemd" "${UNIT_DIR}" "${BIN_DIR}"
cp "${REPO_ROOT}/deploy/systemd/ai-control-plane.service.tmpl" "${FIXTURE_REPO}/deploy/systemd/"
cp "${REPO_ROOT}/deploy/systemd/ai-control-plane-backup.service.tmpl" "${FIXTURE_REPO}/deploy/systemd/"
cp "${REPO_ROOT}/deploy/systemd/ai-control-plane-backup.timer.tmpl" "${FIXTURE_REPO}/deploy/systemd/"

cat >"${SECRETS_FILE}" <<'EOF'
LITELLM_MASTER_KEY=0123456789abcdef0123456789abcdef
LITELLM_SALT_KEY=abcdef0123456789abcdef0123456789
EOF
chmod 600 "${SECRETS_FILE}"

cat >"${BIN_DIR}/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${ACP_TEST_SYSTEMCTL_CAPTURE}"
case "${1:-}" in
  is-active)
    exit 1
    ;;
  *)
    exit 0
    ;;
esac
EOF
chmod +x "${BIN_DIR}/systemctl"

cat >"${BIN_DIR}/systemd-analyze" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
exit 0
EOF
chmod +x "${BIN_DIR}/systemd-analyze"

: >"${SYSTEMCTL_CAPTURE}"

run_bridge() {
    ACP_REPO_ROOT="${FIXTURE_REPO}" \
        ACP_TEST_SYSTEMCTL_CAPTURE="${SYSTEMCTL_CAPTURE}" \
        PATH="${BIN_DIR}:${PATH}" \
        bash "${SCRIPT_PATH}" "$@"
}

printf 'Host Install Bridge Test\n'
printf '========================\n'

run_bridge install \
    --unit-dir "${UNIT_DIR}" \
    --env-file "${SECRETS_FILE}" \
    --compose-bin 'docker compose' \
    --backup-on-calendar 'Mon *-*-* 02:00:00' \
    --backup-randomized-delay 5m \
    --backup-retention-keep 9 >/dev/null

test_assert_file_contains "${UNIT_DIR}/ai-control-plane-backup.service" 'db backup-retention --apply --keep 9' 'backup service renders retention keep value'
test_assert_file_contains "${UNIT_DIR}/ai-control-plane-backup.timer" 'OnCalendar=Mon *-*-* 02:00:00' 'backup timer renders OnCalendar value'
test_assert_file_contains "${UNIT_DIR}/ai-control-plane-backup.timer" 'RandomizedDelaySec=5m' 'backup timer renders RandomizedDelaySec value'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'enable ai-control-plane.service' 'install enables main service'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'enable ai-control-plane-backup.timer' 'install enables backup timer'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'start ai-control-plane.service' 'install starts main service'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'start ai-control-plane-backup.timer' 'install starts backup timer'

: >"${SYSTEMCTL_CAPTURE}"
run_bridge service-status --unit-dir "${UNIT_DIR}" >/dev/null

test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'status --no-pager ai-control-plane.service' 'service-status checks main service'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'status --no-pager ai-control-plane-backup.timer' 'service-status checks backup timer'

: >"${SYSTEMCTL_CAPTURE}"
run_bridge uninstall --unit-dir "${UNIT_DIR}" >/dev/null

if [[ -e "${UNIT_DIR}/ai-control-plane.service" || -e "${UNIT_DIR}/ai-control-plane-backup.service" || -e "${UNIT_DIR}/ai-control-plane-backup.timer" ]]; then
    printf '  ✗ uninstall should remove rendered units\n'
    exit 1
fi

test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'stop ai-control-plane-backup.timer' 'uninstall stops backup timer'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'disable ai-control-plane-backup.timer' 'uninstall disables backup timer'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'stop ai-control-plane.service' 'uninstall stops main service'
test_assert_file_contains "${SYSTEMCTL_CAPTURE}" 'disable ai-control-plane.service' 'uninstall disables main service'
