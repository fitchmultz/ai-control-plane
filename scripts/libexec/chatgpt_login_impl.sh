#!/usr/bin/env bash
#
# AI Control Plane - ChatGPT Device Login Trigger
#
# Purpose: Trigger LiteLLM ChatGPT provider OAuth device flow for gateway-routed Codex.
# Responsibilities:
#   - Validate gateway reachability and authorized model visibility
#   - Send a starter request to the ChatGPT-backed model to prompt device login
#   - Print next steps for completing browser OAuth
#
# Non-scope:
#   - Does not complete OAuth browser interaction
#   - Does not provision Codex client config/env vars
#
# Invariants/Assumptions:
#   - Reads LITELLM_MASTER_KEY from demo/.env when not exported
#   - Uses authorized checks (Bearer token) for /v1/models and /v1/responses

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${REPO_ROOT}/demo/.env"

ACP_EXIT_SUCCESS=0
ACP_EXIT_DOMAIN=1
ACP_EXIT_PREREQ=2
ACP_EXIT_RUNTIME=3
ACP_EXIT_USAGE=64

HOST="${GATEWAY_HOST:-127.0.0.1}"
PORT="${LITELLM_PORT:-4000}"
MODEL="chatgpt-gpt5.3-codex"
USE_TLS="false"

show_help() {
    cat <<'EOF'
Usage: chatgpt_login_impl.sh [options]

Options:
  --host <host>      Gateway host (default: 127.0.0.1)
  --port <port>      Gateway port (default: 4000)
  --model <alias>    ChatGPT model alias (default: chatgpt-gpt5.3-codex)
  --tls              Use https
  --help, -h         Show help

Examples:
  chatgpt_login_impl.sh
  chatgpt_login_impl.sh --model chatgpt-gpt5.3-codex

Fallback (org blocks remote device login):
  codex login
  make chatgpt-auth-copy
  make chatgpt-login
EOF
}

log_info() { printf 'INFO: %s\n' "$*"; }
log_error() { printf 'ERROR: %s\n' "$*" >&2; }

resolve_compose_cmd() {
    if docker compose version >/dev/null 2>&1; then
        printf 'docker compose'
        return
    fi
    if command -v docker-compose >/dev/null 2>&1; then
        printf 'docker-compose'
        return
    fi
    printf ''
}

ensure_chatgpt_overlay_running() {
    if ! command -v docker >/dev/null 2>&1; then
        log_error "docker is required"
        return 1
    fi

    local compose_cmd
    compose_cmd="$(resolve_compose_cmd)"
    if [ -z "${compose_cmd}" ]; then
        log_error "docker compose (or docker-compose) is required"
        return 1
    fi

    mkdir -p "${REPO_ROOT}/demo/auth/chatgpt"

    local mounts
    mounts="$(docker inspect demo-litellm-1 --format '{{range .Mounts}}{{println .Source "->" .Destination}}{{end}}' 2>/dev/null || true)"
    if printf '%s' "${mounts}" | grep -q 'litellm-chatgpt.yaml'; then
        log_info "ChatGPT compose overlay already active"
        return 0
    fi

    log_info "Switching LiteLLM to ChatGPT overlay config"
    (
        cd "${REPO_ROOT}/demo" &&
            ${compose_cmd} -f docker-compose.yml -f docker-compose.chatgpt.yml up -d litellm
    )
}

print_device_prompt_from_logs() {
    if ! command -v docker >/dev/null 2>&1; then
        return 1
    fi

    local logs
    logs="$(docker logs --tail 400 demo-litellm-1 2>&1 || true)"
    if ! printf '%s' "${logs}" | grep -q "Sign in with ChatGPT using device code"; then
        return 1
    fi

    local device_url
    local device_code
    device_url="$(printf '%s\n' "${logs}" | grep -Eo 'https://auth\.openai\.com/codex/device' | tail -n 1)"
    device_code="$(printf '%s\n' "${logs}" | grep -Eo '[A-Z0-9]{4}-[A-Z0-9]{5}' | tail -n 1)"

    printf '\nChatGPT device login required before gateway health checks can pass.\n'
    if [ -n "${device_url}" ]; then
        printf 'Visit: %s\n' "${device_url}"
    fi
    if [ -n "${device_code}" ]; then
        printf 'Code:  %s\n' "${device_code}"
    fi
    printf '\nIf your org disables remote device login, use fallback auth cache copy:\n'
    printf '  codex login\n'
    printf '  make chatgpt-auth-copy\n'
    printf '\nAfter completing device login, run:\n'
    printf '  make health\n'
    printf '  make onboard-codex VERIFY=1\n\n'
    return 0
}

while [ $# -gt 0 ]; do
    case "$1" in
    --host)
        HOST="${2:-}"
        shift 2
        ;;
    --port)
        PORT="${2:-}"
        shift 2
        ;;
    --model)
        MODEL="${2:-}"
        shift 2
        ;;
    --tls)
        USE_TLS="true"
        shift
        ;;
    --help | -h)
        show_help
        exit "${ACP_EXIT_SUCCESS}"
        ;;
    *)
        log_error "Unknown option: $1"
        exit "${ACP_EXIT_USAGE}"
        ;;
    esac
done

if [ ! -f "${ENV_FILE}" ]; then
    log_error "Missing ${ENV_FILE}. Run: make install-env"
    exit "${ACP_EXIT_PREREQ}"
fi

if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required"
    exit "${ACP_EXIT_PREREQ}"
fi

set -a
# shellcheck source=/dev/null
source "${ENV_FILE}"
set +a

if [ -z "${LITELLM_MASTER_KEY:-}" ]; then
    log_error "LITELLM_MASTER_KEY is required in demo/.env"
    exit "${ACP_EXIT_PREREQ}"
fi

ensure_chatgpt_overlay_running || exit "${ACP_EXIT_PREREQ}"

PROTO="http"
if [ "${USE_TLS}" = "true" ]; then
    PROTO="https"
fi
BASE_URL="${PROTO}://${HOST}:${PORT}"

health_code="000"
for _ in $(seq 1 60); do
    health_code="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/health" 2>/dev/null || true)"
    if [ "${health_code}" = "200" ] || [ "${health_code}" = "401" ]; then
        break
    fi
    if print_device_prompt_from_logs; then
        exit "${ACP_EXIT_SUCCESS}"
    fi
    sleep 2
done

if [ "${health_code}" != "200" ] && [ "${health_code}" != "401" ]; then
    log_error "Gateway health check failed: ${BASE_URL}/health -> HTTP ${health_code}"
    exit "${ACP_EXIT_DOMAIN}"
fi
log_info "Gateway health endpoint reachable (HTTP ${health_code})"

models_response="$(curl -sS \
    -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
    "${BASE_URL}/v1/models")"
if ! printf '%s' "${models_response}" | grep -q "\"${MODEL}\""; then
    log_error "Model alias '${MODEL}' not found in /v1/models"
    log_error "Confirm overlay file is present: demo/config/litellm-chatgpt.yaml"
    exit "${ACP_EXIT_DOMAIN}"
fi
log_info "Model alias '${MODEL}' is present"

request_payload="$(
    cat <<EOF
{"model":"${MODEL}","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}
EOF
)"

response="$(
    curl -sS -X POST "${BASE_URL}/v1/responses" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
        -d "${request_payload}"
)"

printf '\nResponse snippet:\n'
printf '%s\n' "${response}" | cut -c1-600
printf '\n'

if printf '%s' "${response}" | grep -qi "visit"; then
    log_info "Device-flow login prompt detected. Complete OAuth in your browser, then rerun this target."
    exit "${ACP_EXIT_SUCCESS}"
fi

if printf '%s' "${response}" | grep -q "\"output\""; then
    log_info "ChatGPT provider request succeeded. Gateway-side ChatGPT auth appears active."
    exit "${ACP_EXIT_SUCCESS}"
fi

log_error "Unexpected response; inspect LiteLLM logs: make logs-litellm"
exit "${ACP_EXIT_DOMAIN}"
