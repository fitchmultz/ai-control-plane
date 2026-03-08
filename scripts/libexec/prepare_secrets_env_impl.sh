#!/usr/bin/env bash
#
# AI Control Plane - Host Secrets Refresh Bridge
#
# Purpose:
#   Validate the canonical host secrets file and atomically sync it into the
#   Docker Compose runtime env path consumed by host-first deployments.
#
# Responsibilities:
#   - Optionally execute a secrets fetch hook before validation.
#   - Enforce the canonical production config contract via typed validation.
#   - Copy the validated secrets file into the Compose env path with 0600 perms.
#
# Non-scope:
#   - Does not generate secrets.
#   - Does not print secret values.
#
# Invariants/Assumptions:
#   - The source secrets file is the source of truth.
#   - The destination Compose env file is a runtime sync target only.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

show_help() {
    cat <<'EOF'
Usage: prepare_secrets_env_impl.sh [OPTIONS]

Validate the canonical host secrets file and sync it into the Compose runtime
env file used by host-first deployments.

Options:
  --secrets-file PATH       Canonical secrets file (default: /etc/ai-control-plane/secrets.env)
  --compose-env-file PATH   Compose runtime env file (default: demo/.env)
  --fetch-hook PATH         Optional executable hook run before validation
  --service-user USER       Optional owner for the synced env file
  --help                    Show this help message

Examples:
  prepare_secrets_env_impl.sh
  prepare_secrets_env_impl.sh --secrets-file /etc/ai-control-plane/secrets.env --compose-env-file demo/.env
  prepare_secrets_env_impl.sh --fetch-hook /usr/local/bin/refresh-secrets.sh

Exit Codes:
  0   Success
  2   Usage or prerequisite failure
  3   Runtime validation or sync failure
EOF
}

secrets_file="/etc/ai-control-plane/secrets.env"
compose_env_file="demo/.env"
fetch_hook=""
service_user=""

while [[ $# -gt 0 ]]; do
    case "$1" in
    --secrets-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --secrets-file\n' >&2
            exit 2
        }
        secrets_file="$2"
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
    --fetch-hook)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --fetch-hook\n' >&2
            exit 2
        }
        fetch_hook="$2"
        shift 2
        ;;
    --service-user)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --service-user\n' >&2
            exit 2
        }
        service_user="$2"
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

repo_root="$(bridge_repo_root)"
acpctl_bin="$(bridge_acpctl_bin)" || exit $?
secrets_file="$(bridge_abspath "${secrets_file}")"
compose_env_file="$(bridge_abspath "${compose_env_file}")"

if [[ -n "${fetch_hook}" ]]; then
    fetch_hook="$(bridge_abspath "${fetch_hook}")"
    [[ -f "${fetch_hook}" ]] || {
        printf 'ERROR: fetch hook not found: %s\n' "${fetch_hook}" >&2
        exit 2
    }
    [[ -x "${fetch_hook}" ]] || {
        printf 'ERROR: fetch hook is not executable: %s\n' "${fetch_hook}" >&2
        exit 2
    }
    "${fetch_hook}"
fi

[[ -f "${secrets_file}" ]] || {
    printf 'ERROR: secrets file not found: %s\n' "${secrets_file}" >&2
    exit 2
}
[[ ! -L "${secrets_file}" ]] || {
    printf 'ERROR: secrets file must not be a symlink: %s\n' "${secrets_file}" >&2
    exit 3
}

mode="$(bridge_portable_stat_mode "${secrets_file}")"
case "${mode}" in
600 | 640) ;;
*)
    printf 'ERROR: secrets file permissions must be 600 or 640, got %s for %s\n' "${mode}" "${secrets_file}" >&2
    exit 3
    ;;
esac

(
    cd "${repo_root}"
    "${acpctl_bin}" validate config --production --secrets-env-file "${secrets_file}" >/dev/null
)

compose_env_dir="$(dirname "${compose_env_file}")"
mkdir -p "${compose_env_dir}"
tmp_file="$(mktemp "${compose_env_dir}/.env.tmp.XXXXXX")"
trap 'rm -f "${tmp_file:-}"' EXIT

cp "${secrets_file}" "${tmp_file}"
chmod 0600 "${tmp_file}"
if [[ -n "${service_user}" ]] && [[ "${EUID}" -eq 0 ]]; then
    chown "${service_user}" "${tmp_file}"
fi
mv "${tmp_file}" "${compose_env_file}"
trap - EXIT

printf 'Synced canonical secrets into %s\n' "${compose_env_file}"
