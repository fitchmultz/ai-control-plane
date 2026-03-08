#!/usr/bin/env bash
#
# AI Control Plane - Host Install Bridge
#
# Purpose:
#   Install and manage the systemd unit that owns the host-first Docker Compose
#   runtime for AI Control Plane.
#
# Responsibilities:
#   - Render the tracked systemd unit template with concrete deployment paths.
#   - Sync canonical secrets into the Compose runtime env file before install/start.
#   - Execute systemd lifecycle actions for the installed service.
#
# Non-scope:
#   - Does not reimplement Docker Compose orchestration.
#   - Does not manage remote hosts through Ansible.
#
# Invariants/Assumptions:
#   - The tracked template under deploy/systemd/ is the source of truth.
#   - The service runs against the active host slot only.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

show_help() {
    cat <<'EOF'
Usage: host_install_impl.sh <install|uninstall|service-status|service-start|service-stop|service-restart> [OPTIONS]

Manage the systemd service for host-first AI Control Plane deployments.

Options:
  --service-name NAME       systemd unit base name (default: ai-control-plane)
  --service-user USER       Service user (default: current user)
  --service-group GROUP     Service group (default: current user's primary group)
  --repo-root PATH          Repository root (default: current checkout)
  --unit-dir PATH           systemd unit directory (default: /etc/systemd/system)
  --env-file PATH           Canonical secrets file (default: /etc/ai-control-plane/secrets.env)
  --compose-env-file PATH   Compose runtime env file (default: demo/.env)
  --compose-file PATH       Compose file for the service (default: demo/docker-compose.yml)
  --compose-bin CMD         Compose binary/command (default: auto-detect)
  --fetch-hook PATH         Optional secrets fetch hook executed before install/start
  --no-enable               Skip `systemctl enable` during install
  --no-start                Skip `systemctl start` during install
  --help                    Show this help message

Examples:
  host_install_impl.sh install --service-user acp --service-group acp
  host_install_impl.sh service-status
  host_install_impl.sh uninstall

Exit Codes:
  0   Success
  2   Usage or prerequisite failure
  3   Runtime/systemd failure
EOF
}

[[ $# -ge 1 ]] || { show_help >&2; exit 2; }
subcommand="$1"
shift

case "${subcommand}" in
    install|uninstall|service-status|service-start|service-stop|service-restart)
        ;;
    --help|-h)
        show_help
        exit 0
        ;;
    *)
        printf 'ERROR: unknown host install command: %s\n' "${subcommand}" >&2
        show_help >&2
        exit 2
        ;;
esac

repo_root="$(bridge_repo_root)"
service_name="ai-control-plane"
service_user="$(id -un)"
service_group="$(id -gn)"
unit_dir="/etc/systemd/system"
env_file="/etc/ai-control-plane/secrets.env"
compose_env_file="demo/.env"
compose_file="demo/docker-compose.yml"
compose_bin=""
fetch_hook=""
enable_service="true"
start_service="true"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --service-name)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --service-name\n' >&2; exit 2; }
            service_name="$2"
            shift 2
            ;;
        --service-user)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --service-user\n' >&2; exit 2; }
            service_user="$2"
            shift 2
            ;;
        --service-group)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --service-group\n' >&2; exit 2; }
            service_group="$2"
            shift 2
            ;;
        --repo-root)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --repo-root\n' >&2; exit 2; }
            repo_root="$(bridge_abspath "$2")"
            shift 2
            ;;
        --unit-dir)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --unit-dir\n' >&2; exit 2; }
            unit_dir="$2"
            shift 2
            ;;
        --env-file)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --env-file\n' >&2; exit 2; }
            env_file="$2"
            shift 2
            ;;
        --compose-env-file)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --compose-env-file\n' >&2; exit 2; }
            compose_env_file="$2"
            shift 2
            ;;
        --compose-file)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --compose-file\n' >&2; exit 2; }
            compose_file="$2"
            shift 2
            ;;
        --compose-bin)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --compose-bin\n' >&2; exit 2; }
            compose_bin="$2"
            shift 2
            ;;
        --fetch-hook)
            [[ $# -ge 2 ]] || { printf 'ERROR: missing value for --fetch-hook\n' >&2; exit 2; }
            fetch_hook="$2"
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
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            printf 'ERROR: unknown argument: %s\n' "$1" >&2
            show_help >&2
            exit 2
            ;;
    esac
done

bridge_require_command systemctl

env_file="$(bridge_abspath "${env_file}")"
compose_env_file="$(bridge_abspath "${compose_env_file}")"
compose_file="$(bridge_abspath "${compose_file}")"
template_path="${repo_root}/deploy/systemd/ai-control-plane.service.tmpl"
prepare_script="${repo_root}/scripts/libexec/prepare_secrets_env_impl.sh"
unit_path="${unit_dir}/${service_name}.service"

if [[ -z "${compose_bin}" ]]; then
    compose_bin="$(bridge_detect_compose_bin)"
fi

service_action() {
    local action="$1"
    shift || true
    systemctl "${action}" "$@" "${service_name}.service"
}

render_unit() {
    local fetch_hook_arg rendered_compose_bin
    fetch_hook_arg=""
    if [[ -n "${fetch_hook}" ]]; then
        fetch_hook_arg="--fetch-hook $(bridge_abspath "${fetch_hook}")"
    fi
    rendered_compose_bin="$(bridge_escape_sed_replacement "${compose_bin}")"

    sed \
        -e "s|{{SERVICE_USER}}|$(bridge_escape_sed_replacement "${service_user}")|g" \
        -e "s|{{SERVICE_GROUP}}|$(bridge_escape_sed_replacement "${service_group}")|g" \
        -e "s|{{WORKING_DIR}}|$(bridge_escape_sed_replacement "${repo_root}")|g" \
        -e "s|{{ENV_FILE}}|$(bridge_escape_sed_replacement "${env_file}")|g" \
        -e "s|{{COMPOSE_ENV_FILE}}|$(bridge_escape_sed_replacement "${compose_env_file}")|g" \
        -e "s|{{SECRETS_FETCH_HOOK_ARG}}|$(bridge_escape_sed_replacement "${fetch_hook_arg}")|g" \
        -e "s|{{COMPOSE_FILE}}|$(bridge_escape_sed_replacement "${compose_file}")|g" \
        -e "s|{{COMPOSE_BIN}}|${rendered_compose_bin}|g" \
        -e "s|{{COMPOSE_PROJECT_NAME}}|ai-control-plane-active|g" \
        "${template_path}"
}

case "${subcommand}" in
    install)
        [[ -f "${template_path}" ]] || { printf 'ERROR: systemd template not found: %s\n' "${template_path}" >&2; exit 3; }
        [[ -x "${prepare_script}" ]] || { printf 'ERROR: secrets refresh script not executable: %s\n' "${prepare_script}" >&2; exit 3; }
        mkdir -p "${unit_dir}"
        prepare_args=(
            --secrets-file "${env_file}"
            --compose-env-file "${compose_env_file}"
            --service-user "${service_user}"
        )
        if [[ -n "${fetch_hook}" ]]; then
            prepare_args+=(--fetch-hook "${fetch_hook}")
        fi
        "${prepare_script}" "${prepare_args[@]}"
        temp_unit="$(mktemp "${unit_dir}/${service_name}.service.tmp.XXXXXX")"
        trap 'rm -f "${temp_unit:-}"' EXIT
        render_unit >"${temp_unit}"
        chmod 0644 "${temp_unit}"
        mv "${temp_unit}" "${unit_path}"
        trap - EXIT
        systemctl daemon-reload
        if command -v systemd-analyze >/dev/null 2>&1; then
            systemd-analyze verify "${unit_path}" >/dev/null
        fi
        if [[ "${enable_service}" == "true" ]]; then
            service_action enable
        fi
        if [[ "${start_service}" == "true" ]]; then
            if systemctl is-active --quiet "${service_name}.service" >/dev/null 2>&1; then
                service_action restart
            else
                service_action start
            fi
        fi
        printf 'Installed %s\n' "${unit_path}"
        ;;
    uninstall)
        service_action stop >/dev/null 2>&1 || true
        service_action disable >/dev/null 2>&1 || true
        rm -f "${unit_path}"
        systemctl daemon-reload
        systemctl reset-failed "${service_name}.service" >/dev/null 2>&1 || true
        printf 'Removed %s\n' "${unit_path}"
        ;;
    service-status)
        service_action status --no-pager
        ;;
    service-start)
        prepare_args=(
            --secrets-file "${env_file}"
            --compose-env-file "${compose_env_file}"
            --service-user "${service_user}"
        )
        if [[ -n "${fetch_hook}" ]]; then
            prepare_args+=(--fetch-hook "${fetch_hook}")
        fi
        "${prepare_script}" "${prepare_args[@]}"
        service_action start
        ;;
    service-stop)
        service_action stop
        ;;
    service-restart)
        prepare_args=(
            --secrets-file "${env_file}"
            --compose-env-file "${compose_env_file}"
            --service-user "${service_user}"
        )
        if [[ -n "${fetch_hook}" ]]; then
            prepare_args+=(--fetch-hook "${fetch_hook}")
        fi
        "${prepare_script}" "${prepare_args[@]}"
        service_action restart
        ;;
esac
