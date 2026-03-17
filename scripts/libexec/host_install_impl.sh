#!/usr/bin/env bash
#
# AI Control Plane - Host Install Bridge
#
# Purpose:
#   Install and manage the systemd units that own the host-first Docker Compose
#   runtime and automated backup timer for AI Control Plane.
#
# Responsibilities:
#   - Render the tracked systemd templates with concrete deployment paths.
#   - Execute systemd lifecycle actions for the installed service and backup timer.
#
# Non-scope:
#   - Does not reimplement Docker Compose orchestration.
#   - Does not manage remote hosts through Ansible.
#
# Invariants/Assumptions:
#   - The tracked templates under deploy/systemd/ are the source of truth.
#   - The service runs against the active host slot only.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

show_help() {
    cat <<'EOF'
Usage: host_install_impl.sh <install|uninstall|service-status|service-start|service-stop|service-restart> [OPTIONS]

Manage the systemd service and automated backup timer for host-first AI Control Plane deployments.

Options:
  --service-name NAME              systemd unit base name (default: ai-control-plane)
  --service-user USER              Service user (default: current user)
  --service-group GROUP            Service group (default: current user's primary group)
  --repo-root PATH                 Repository root (default: current checkout)
  --unit-dir PATH                  systemd unit directory (default: /etc/systemd/system)
  --env-file PATH                  Canonical secrets file (default: /etc/ai-control-plane/secrets.env)
  --compose-file PATH              Compose file for the service (default: demo/docker-compose.yml)
  --compose-bin CMD                Compose binary/command (default: auto-detect)
  --backup-on-calendar SPEC        systemd OnCalendar schedule (default: daily)
  --backup-randomized-delay SPEC   systemd RandomizedDelaySec value (default: 15m)
  --backup-retention-keep N        Number of newest backups to retain (default: 7)
  --no-enable                      Skip `systemctl enable` during install
  --no-start                       Skip `systemctl start` during install
  --help                           Show this help message

Examples:
  host_install_impl.sh install --service-user acp --service-group acp
  host_install_impl.sh install --backup-on-calendar 'Mon *-*-* 02:00:00' --backup-retention-keep 14
  host_install_impl.sh service-status
  host_install_impl.sh uninstall

Exit Codes:
  0   Success
  2   Usage or prerequisite failure
  3   Runtime/systemd failure
EOF
}

[[ $# -ge 1 ]] || {
    show_help >&2
    exit "${ACP_EXIT_USAGE}"
}
subcommand="$1"
shift

case "${subcommand}" in
install | uninstall | service-status | service-start | service-stop | service-restart) ;;
--help | -h)
    show_help
    exit "${ACP_EXIT_SUCCESS}"
    ;;
*)
    printf 'ERROR: unknown host install command: %s\n' "${subcommand}" >&2
    show_help >&2
    exit "${ACP_EXIT_USAGE}"
    ;;
esac

repo_root="$(bridge_repo_root)"
service_name="ai-control-plane"
service_user="$(id -un)"
service_group="$(id -gn)"
unit_dir="/etc/systemd/system"
env_file="/etc/ai-control-plane/secrets.env"
compose_file="demo/docker-compose.yml"
compose_bin=""
backup_on_calendar="daily"
backup_randomized_delay_sec="15m"
backup_retention_keep="7"
enable_service="true"
start_service="true"

while [[ $# -gt 0 ]]; do
    case "$1" in
    --service-name)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --service-name\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        service_name="$2"
        shift 2
        ;;
    --service-user)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --service-user\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        service_user="$2"
        shift 2
        ;;
    --service-group)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --service-group\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        service_group="$2"
        shift 2
        ;;
    --repo-root)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --repo-root\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        repo_root="$(bridge_abspath "$2")"
        shift 2
        ;;
    --unit-dir)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --unit-dir\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        unit_dir="$2"
        shift 2
        ;;
    --env-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --env-file\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        env_file="$2"
        shift 2
        ;;
    --compose-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --compose-file\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        compose_file="$2"
        shift 2
        ;;
    --compose-bin)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --compose-bin\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        compose_bin="$2"
        shift 2
        ;;
    --backup-on-calendar)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --backup-on-calendar\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        backup_on_calendar="$2"
        shift 2
        ;;
    --backup-randomized-delay)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --backup-randomized-delay\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        backup_randomized_delay_sec="$2"
        shift 2
        ;;
    --backup-retention-keep)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --backup-retention-keep\n' >&2
            exit "${ACP_EXIT_USAGE}"
        }
        backup_retention_keep="$2"
        shift 2
        ;;
    --no-enable)
        enable_service="false"
        shift
        ;;
    --no-start)
        start_service="false"
        shift
        ;;
    --help | -h)
        show_help
        exit "${ACP_EXIT_SUCCESS}"
        ;;
    *)
        printf 'ERROR: unknown argument: %s\n' "$1" >&2
        show_help >&2
        exit "${ACP_EXIT_USAGE}"
        ;;
    esac
done

if ! [[ "${backup_retention_keep}" =~ ^[1-9][0-9]*$ ]]; then
    printf 'ERROR: --backup-retention-keep must be a positive integer\n' >&2
    exit "${ACP_EXIT_USAGE}"
fi

bridge_require_command systemctl

env_file="$(bridge_abspath "${env_file}")"
compose_file="$(bridge_abspath "${compose_file}")"
main_template_path="${repo_root}/deploy/systemd/ai-control-plane.service.tmpl"
backup_service_template_path="${repo_root}/deploy/systemd/ai-control-plane-backup.service.tmpl"
backup_timer_template_path="${repo_root}/deploy/systemd/ai-control-plane-backup.timer.tmpl"
unit_path="${unit_dir}/${service_name}.service"
backup_service_unit_path="${unit_dir}/${service_name}-backup.service"
backup_timer_unit_path="${unit_dir}/${service_name}-backup.timer"
backup_service_unit_name="${service_name}-backup.service"
backup_timer_unit_name="${service_name}-backup.timer"

if [[ -z "${compose_bin}" ]]; then
    compose_bin="$(bridge_detect_compose_bin)"
fi

unit_action() {
    local action="$1"
    local unit="$2"
    shift 2 || true
    systemctl "${action}" "$@" "${unit}"
}

main_service_action() {
    local action="$1"
    shift || true
    unit_action "${action}" "${service_name}.service" "$@"
}

backup_timer_action() {
    local action="$1"
    shift || true
    unit_action "${action}" "${backup_timer_unit_name}" "$@"
}

backup_service_action() {
    local action="$1"
    shift || true
    unit_action "${action}" "${backup_service_unit_name}" "$@"
}

backup_timer_installed() {
    [[ -f "${backup_timer_unit_path}" ]]
}

render_main_unit() {
    local rendered_compose_bin
    rendered_compose_bin="$(bridge_escape_sed_replacement "${compose_bin}")"

    sed \
        -e "s|{{SERVICE_USER}}|$(bridge_escape_sed_replacement "${service_user}")|g" \
        -e "s|{{SERVICE_GROUP}}|$(bridge_escape_sed_replacement "${service_group}")|g" \
        -e "s|{{WORKING_DIR}}|$(bridge_escape_sed_replacement "${repo_root}")|g" \
        -e "s|{{ENV_FILE}}|$(bridge_escape_sed_replacement "${env_file}")|g" \
        -e "s|{{COMPOSE_FILE}}|$(bridge_escape_sed_replacement "${compose_file}")|g" \
        -e "s|{{COMPOSE_BIN}}|${rendered_compose_bin}|g" \
        -e "s|{{COMPOSE_PROJECT_NAME}}|ai-control-plane-active|g" \
        "${main_template_path}"
}

render_backup_service_unit() {
    sed \
        -e "s|{{SERVICE_NAME}}|$(bridge_escape_sed_replacement "${service_name}")|g" \
        -e "s|{{SERVICE_USER}}|$(bridge_escape_sed_replacement "${service_user}")|g" \
        -e "s|{{SERVICE_GROUP}}|$(bridge_escape_sed_replacement "${service_group}")|g" \
        -e "s|{{WORKING_DIR}}|$(bridge_escape_sed_replacement "${repo_root}")|g" \
        -e "s|{{ENV_FILE}}|$(bridge_escape_sed_replacement "${env_file}")|g" \
        -e "s|{{BACKUP_RETENTION_KEEP}}|$(bridge_escape_sed_replacement "${backup_retention_keep}")|g" \
        "${backup_service_template_path}"
}

render_backup_timer_unit() {
    sed \
        -e "s|{{SERVICE_NAME}}|$(bridge_escape_sed_replacement "${service_name}")|g" \
        -e "s|{{BACKUP_ON_CALENDAR}}|$(bridge_escape_sed_replacement "${backup_on_calendar}")|g" \
        -e "s|{{RANDOMIZED_DELAY_SEC}}|$(bridge_escape_sed_replacement "${backup_randomized_delay_sec}")|g" \
        "${backup_timer_template_path}"
}

verify_unit_if_possible() {
    local path="$1"
    if command -v systemd-analyze >/dev/null 2>&1; then
        systemd-analyze verify "${path}" >/dev/null
    fi
}

cleanup_temp_units() {
    rm -f "${temp_main_unit:-}" "${temp_backup_service_unit:-}" "${temp_backup_timer_unit:-}"
}

case "${subcommand}" in
install)
    for template_path in "${main_template_path}" "${backup_service_template_path}" "${backup_timer_template_path}"; do
        [[ -f "${template_path}" ]] || {
            printf 'ERROR: systemd template not found: %s\n' "${template_path}" >&2
            exit "${ACP_EXIT_RUNTIME}"
        }
    done
    mkdir -p "${unit_dir}"
    temp_main_unit="$(mktemp "${unit_dir}/${service_name}.service.tmp.XXXXXX")"
    temp_backup_service_unit="$(mktemp "${unit_dir}/${service_name}-backup.service.tmp.XXXXXX")"
    temp_backup_timer_unit="$(mktemp "${unit_dir}/${service_name}-backup.timer.tmp.XXXXXX")"
    trap cleanup_temp_units EXIT
    render_main_unit >"${temp_main_unit}"
    render_backup_service_unit >"${temp_backup_service_unit}"
    render_backup_timer_unit >"${temp_backup_timer_unit}"
    chmod 0644 "${temp_main_unit}" "${temp_backup_service_unit}" "${temp_backup_timer_unit}"
    mv "${temp_main_unit}" "${unit_path}"
    mv "${temp_backup_service_unit}" "${backup_service_unit_path}"
    mv "${temp_backup_timer_unit}" "${backup_timer_unit_path}"
    trap - EXIT
    systemctl daemon-reload
    verify_unit_if_possible "${unit_path}"
    verify_unit_if_possible "${backup_service_unit_path}"
    verify_unit_if_possible "${backup_timer_unit_path}"
    if [[ "${enable_service}" == "true" ]]; then
        main_service_action enable
        backup_timer_action enable
    fi
    if [[ "${start_service}" == "true" ]]; then
        if systemctl is-active --quiet "${service_name}.service" >/dev/null 2>&1; then
            main_service_action restart
        else
            main_service_action start
        fi
        if systemctl is-active --quiet "${backup_timer_unit_name}" >/dev/null 2>&1; then
            backup_timer_action restart
        else
            backup_timer_action start
        fi
    fi
    printf 'Installed %s\n' "${unit_path}"
    printf 'Installed %s\n' "${backup_service_unit_path}"
    printf 'Installed %s\n' "${backup_timer_unit_path}"
    ;;
uninstall)
    if backup_timer_installed; then
        backup_timer_action stop >/dev/null 2>&1 || true
        backup_timer_action disable >/dev/null 2>&1 || true
    fi
    [[ -f "${backup_service_unit_path}" ]] && backup_service_action stop >/dev/null 2>&1 || true
    main_service_action stop >/dev/null 2>&1 || true
    main_service_action disable >/dev/null 2>&1 || true
    rm -f "${unit_path}" "${backup_service_unit_path}" "${backup_timer_unit_path}"
    systemctl daemon-reload
    systemctl reset-failed "${service_name}.service" >/dev/null 2>&1 || true
    systemctl reset-failed "${backup_service_unit_name}" >/dev/null 2>&1 || true
    systemctl reset-failed "${backup_timer_unit_name}" >/dev/null 2>&1 || true
    printf 'Removed %s\n' "${unit_path}"
    printf 'Removed %s\n' "${backup_service_unit_path}"
    printf 'Removed %s\n' "${backup_timer_unit_path}"
    ;;
service-status)
    main_service_action status --no-pager
    if backup_timer_installed; then
        printf '\n'
        backup_timer_action status --no-pager
    fi
    ;;
service-start)
    main_service_action start
    if backup_timer_installed; then
        backup_timer_action start
    fi
    ;;
service-stop)
    if backup_timer_installed; then
        backup_timer_action stop
    fi
    main_service_action stop
    ;;
service-restart)
    main_service_action restart
    if backup_timer_installed; then
        backup_timer_action restart
    fi
    ;;
esac
