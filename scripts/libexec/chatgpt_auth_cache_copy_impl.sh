#!/usr/bin/env bash
#
# AI Control Plane - ChatGPT Auth Cache Copy Helper
#
# Purpose: Normalize local Codex auth cache and persist it for LiteLLM ChatGPT mode.
# Responsibilities:
#   - Validate local auth cache file exists
#   - Convert Codex cache schema to LiteLLM ChatGPT schema
#   - Write normalized cache to demo/auth/chatgpt/auth.json for overlay mount
#   - Best-effort sync to running LiteLLM container if available
#
# Non-scope:
#   - Does not perform Codex login itself
#   - Does not start/stop Docker services
#
# Invariants/Assumptions:
#   - Source auth file is JSON from Codex CLI local login
#   - Persistent destination is repo-local: demo/auth/chatgpt/auth.json
#   - Never prints token contents

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

REPO_ROOT="$(bridge_repo_root)"

AUTH_FILE="${HOME}/.codex/auth.json"
CONTAINER=""
DEST_FILE="${REPO_ROOT}/demo/auth/chatgpt/auth.json"

show_help() {
    cat <<'EOF'
Usage: chatgpt_auth_cache_copy_impl.sh [options]

Normalize local Codex auth cache for LiteLLM ChatGPT provider.

Options:
  --auth-file <path>   Source auth cache file (default: ~/.codex/auth.json)
  --container <name>   Target container for live sync (default: resolved via docker compose)
  --dest-file <path>   Destination auth cache file (default: demo/auth/chatgpt/auth.json)
  --help, -h           Show help

Examples:
  chatgpt_auth_cache_copy_impl.sh
  chatgpt_auth_cache_copy_impl.sh --auth-file ~/.codex/auth.json
  chatgpt_auth_cache_copy_impl.sh --dest-file demo/auth/chatgpt/auth.json

Exit codes:
  0  success
  1  domain failure
  2  prerequisites missing
  3  runtime/internal error
  64 usage error
EOF
}

resolve_target_container() {
    if [ -n "${CONTAINER}" ]; then
        printf '%s\n' "${CONTAINER}"
        return 0
    fi

    local compose_cmd
    compose_cmd="$(bridge_detect_compose_bin_optional)"
    if [ -n "${compose_cmd}" ]; then
        local container_id
        container_id="$(
            cd "${REPO_ROOT}/demo" &&
                ${compose_cmd} -f docker-compose.yml -f docker-compose.chatgpt.yml ps -q litellm 2>/dev/null | head -n 1
        )"
        if [ -n "${container_id}" ]; then
            printf '%s\n' "${container_id}"
            return 0
        fi
    fi

    printf ''
}

while [ $# -gt 0 ]; do
    case "$1" in
    --auth-file)
        AUTH_FILE="${2:-}"
        shift 2
        ;;
    --container)
        CONTAINER="${2:-}"
        shift 2
        ;;
    --dest-file)
        DEST_FILE="${2:-}"
        shift 2
        ;;
    --help | -h)
        show_help
        exit "${ACP_EXIT_SUCCESS}"
        ;;
    *)
        bridge_log_error "Unknown option: $1"
        exit "${ACP_EXIT_USAGE}"
        ;;
    esac
done

if ! command -v jq >/dev/null 2>&1; then
    bridge_log_error "jq is required"
    exit "${ACP_EXIT_PREREQ}"
fi

if [ ! -f "${AUTH_FILE}" ]; then
    bridge_log_error "Auth cache file not found: ${AUTH_FILE}"
    bridge_log_error "Run 'codex login' locally first (browser-capable machine)."
    exit "${ACP_EXIT_PREREQ}"
fi

if [ ! -s "${AUTH_FILE}" ]; then
    bridge_log_error "Auth cache file is empty: ${AUTH_FILE}"
    exit "${ACP_EXIT_DOMAIN}"
fi

dest_dir="$(dirname "${DEST_FILE}")"
mkdir -p "${dest_dir}"
chmod 700 "${dest_dir}"

normalized_auth_file="$(mktemp)"
trap 'rm -f "${normalized_auth_file}"' EXIT

if jq -e '.tokens.access_token and .tokens.refresh_token and .tokens.id_token' "${AUTH_FILE}" >/dev/null 2>&1; then
    jq '{
        access_token: .tokens.access_token,
        refresh_token: .tokens.refresh_token,
        id_token: .tokens.id_token,
        account_id: (.tokens.account_id // null)
    }' "${AUTH_FILE}" >"${normalized_auth_file}"
elif jq -e '.access_token and .refresh_token and .id_token' "${AUTH_FILE}" >/dev/null 2>&1; then
    jq '{access_token, refresh_token, id_token, account_id: (.account_id // null), expires_at: (.expires_at // null)}' \
        "${AUTH_FILE}" >"${normalized_auth_file}"
else
    bridge_log_error "Unsupported auth cache schema in ${AUTH_FILE}"
    bridge_log_error "Expected Codex auth schema (.tokens.*) or LiteLLM ChatGPT schema (.access_token)."
    exit "${ACP_EXIT_DOMAIN}"
fi

if [ -f "${DEST_FILE}" ]; then
    backup_file="${DEST_FILE}.bak.$(date +%Y%m%d%H%M%S)"
    cp "${DEST_FILE}" "${backup_file}"
    bridge_log_info "Backed up existing cache to ${backup_file}"
fi

cp "${normalized_auth_file}" "${DEST_FILE}"
chmod 600 "${DEST_FILE}"
bridge_log_info "Wrote normalized auth cache to ${DEST_FILE}"

if command -v docker >/dev/null 2>&1; then
    target_container="$(resolve_target_container)"
    if [ -n "${target_container}" ]; then
        container_state="$(docker inspect -f '{{.State.Running}}' "${target_container}" 2>/dev/null || true)"
    else
        container_state="false"
    fi
    if [ "${container_state}" = "true" ]; then
        container_home="$(docker exec "${target_container}" sh -lc 'printf "%s" "${HOME:-/root}"' 2>/dev/null)" || {
            bridge_log_error "Failed to resolve home directory in running LiteLLM container"
            exit "${ACP_EXIT_RUNTIME}"
        }
        if [ -n "${container_home}" ]; then
            live_dest_dir="${container_home}/.config/litellm/chatgpt"
            live_dest_file="${live_dest_dir}/auth.json"
            docker exec "${target_container}" sh -lc "mkdir -p '${live_dest_dir}'" >/dev/null 2>&1 || {
                bridge_log_error "Failed to create live auth cache directory in ${target_container}"
                exit "${ACP_EXIT_RUNTIME}"
            }
            docker exec -i "${target_container}" sh -lc "cat > '${live_dest_file}'" <"${normalized_auth_file}" || {
                bridge_log_error "Failed to copy live auth cache into ${target_container}"
                exit "${ACP_EXIT_RUNTIME}"
            }
            docker exec "${target_container}" sh -lc "chmod 600 '${live_dest_file}'" >/dev/null 2>&1 || {
                bridge_log_error "Failed to set live auth cache permissions in ${target_container}"
                exit "${ACP_EXIT_RUNTIME}"
            }
            bridge_log_info "Live sync complete for ${target_container}:${live_dest_file}"
        fi
    fi
fi

printf 'Next steps:\n'
printf '  1) make chatgpt-login\n'
printf '  2) make health\n'
printf '  3) make onboard-codex\n'

exit "${ACP_EXIT_SUCCESS}"
