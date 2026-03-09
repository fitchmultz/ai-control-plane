#!/usr/bin/env bash
#
# AI Control Plane - Bridge Script Common Helpers
#
# Purpose:
#   Share repository resolution, command lookup, and small portability helpers
#   across internal bridge scripts.
#
# Responsibilities:
#   - Resolve the repository root and canonical acpctl binary.
#   - Provide portable command/path utilities for Linux host workflows.
#   - Keep bridge wrappers thin and consistent.
#
# Non-scope:
#   - Does not own any host workflow business logic.
#   - Does not mutate runtime state by itself.
#
# Invariants/Assumptions:
#   - Scripts are executed from inside the repository checkout or with
#     ACP_REPO_ROOT exported by the typed acpctl bridge.

set -euo pipefail

bridge_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bridge_repo_root_default="$(cd "${bridge_lib_dir}/../.." && pwd)"

# shellcheck source=scripts/libexec/exitcodes.sh
source "${bridge_lib_dir}/exitcodes.sh"

bridge_repo_root() {
    printf '%s\n' "${ACP_REPO_ROOT:-${bridge_repo_root_default}}"
}

bridge_log_info() {
    printf 'INFO: %s\n' "$*"
}

bridge_log_error() {
    printf 'ERROR: %s\n' "$*" >&2
}

bridge_log_debug() {
    if [[ "${BRIDGE_DEBUG:-}" == "1" ]]; then
        printf 'DEBUG: %s\n' "$*" >&2
    fi
}

bridge_acpctl_bin() {
    local repo_root
    repo_root="$(bridge_repo_root)"

    if [[ -n "${ACPCTL_BIN:-}" ]]; then
        if [[ -x "${ACPCTL_BIN}" ]]; then
            printf '%s\n' "${ACPCTL_BIN}"
            return 0
        fi
        bridge_log_error "ACPCTL_BIN is set but not executable: ${ACPCTL_BIN}"
        return "${ACP_EXIT_PREREQ}"
    fi
    if [[ -x "${repo_root}/.bin/acpctl" ]]; then
        printf '%s\n' "${repo_root}/.bin/acpctl"
        return 0
    fi
    if [[ -x "${repo_root}/acpctl" ]]; then
        printf '%s\n' "${repo_root}/acpctl"
        return 0
    fi
    if command -v acpctl >/dev/null 2>&1; then
        command -v acpctl
        return 0
    fi

    bridge_log_error "acpctl binary not found. Run: make install-binary"
    return "${ACP_EXIT_PREREQ}"
}

bridge_require_command() {
    local command_name="$1"
    if ! command -v "${command_name}" >/dev/null 2>&1; then
        bridge_log_error "required command not found: ${command_name}"
        return "${ACP_EXIT_PREREQ}"
    fi
}

bridge_abspath() {
    local repo_root input
    repo_root="$(bridge_repo_root)"
    input="${1:-}"
    if [[ -z "${input}" ]]; then
        printf '%s\n' "${repo_root}"
        return 0
    fi
    if [[ "${input}" = /* ]]; then
        printf '%s\n' "${input}"
        return 0
    fi
    printf '%s\n' "${repo_root}/${input}"
}

bridge_detect_compose_bin() {
    if docker compose version >/dev/null 2>&1; then
        printf 'docker compose\n'
        return 0
    fi
    if command -v docker-compose >/dev/null 2>&1; then
        printf 'docker-compose\n'
        return 0
    fi

    bridge_log_error "Docker Compose not found. Install docker compose v2 or docker-compose."
    return "${ACP_EXIT_PREREQ}"
}

bridge_detect_compose_bin_optional() {
    bridge_detect_compose_bin 2>/dev/null || true
}

bridge_portable_stat_mode() {
    local path="$1"
    if stat -f '%Lp' "${path}" >/dev/null 2>&1; then
        stat -f '%Lp' "${path}"
        return 0
    fi
    stat -c '%a' "${path}"
}

bridge_escape_sed_replacement() {
    printf '%s' "${1}" | sed -e 's/[|&]/\\&/g'
}
