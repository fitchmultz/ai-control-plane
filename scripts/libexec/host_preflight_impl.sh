#!/usr/bin/env bash
#
# AI Control Plane - Host Preflight Bridge
#
# Purpose:
#   Validate the local host-first runtime contract before operators install or
#   start the production service.
#
# Responsibilities:
#   - Check required local binaries for host-first operations.
#   - Validate the production deployment contract against the canonical secrets file.
#   - Confirm the systemd unit template and Compose env path are present.
#
# Non-scope:
#   - Does not mutate system state.
#   - Does not perform remote Ansible deployment.
#
# Invariants/Assumptions:
#   - The command runs on the target Linux host where the service will operate.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

show_help() {
    cat <<'EOF'
Usage: host_preflight_impl.sh [OPTIONS]

Run local host-first preflight checks for systemd-managed Docker Compose
deployment.

Options:
  --profile NAME            Deployment profile (default: production; only production is supported)
  --secrets-env-file PATH   Canonical secrets file (default: /etc/ai-control-plane/secrets.env)
  --compose-env-file PATH   Compose runtime env file (default: demo/.env)
  --help                    Show this help message

Examples:
  host_preflight_impl.sh
  host_preflight_impl.sh --secrets-env-file /etc/ai-control-plane/secrets.env

Exit Codes:
  0   Success
  2   Usage or prerequisite failure
  3   Runtime validation failure
EOF
}

profile="production"
secrets_env_file="/etc/ai-control-plane/secrets.env"
compose_env_file="demo/.env"

while [[ $# -gt 0 ]]; do
    case "$1" in
    --profile)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --profile\n' >&2
            exit 2
        }
        profile="$2"
        shift 2
        ;;
    --secrets-env-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --secrets-env-file\n' >&2
            exit 2
        }
        secrets_env_file="$2"
        shift 2
        ;;
    --compose-env-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --compose-env-file\n' >&2
            exit 2
        }
        compose_env_file="$2"
        shift 2
        ;;
    --help | -h)
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

[[ "${profile}" == "production" ]] || {
    printf 'ERROR: unsupported profile: %s\n' "${profile}" >&2
    exit 2
}

repo_root="$(bridge_repo_root)"
acpctl_bin="$(bridge_acpctl_bin)" || exit $?
secrets_env_file="$(bridge_abspath "${secrets_env_file}")"
compose_env_file="$(bridge_abspath "${compose_env_file}")"

bridge_require_command docker
bridge_require_command make
bridge_require_command systemctl
bridge_detect_compose_bin >/dev/null

[[ -f "${repo_root}/deploy/systemd/ai-control-plane.service.tmpl" ]] || {
    printf 'ERROR: missing systemd template: %s\n' "${repo_root}/deploy/systemd/ai-control-plane.service.tmpl" >&2
    exit 3
}

compose_env_dir="$(dirname "${compose_env_file}")"
[[ -d "${compose_env_dir}" ]] || {
    printf 'ERROR: compose env parent directory not found: %s\n' "${compose_env_dir}" >&2
    exit 3
}

(
    cd "${repo_root}"
    "${acpctl_bin}" validate config --production --secrets-env-file "${secrets_env_file}"
)

printf 'Host preflight passed for %s\n' "${repo_root}"
