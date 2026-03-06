#!/usr/bin/env bash
#
# AI Control Plane - Tool Onboarding Script
#
# Purpose: Configure local CLI/IDE tools to route through the LiteLLM gateway.
# Responsibilities:
#   - Generate/reuse LiteLLM virtual keys for routed modes
#   - Print exact environment exports for each supported tool
#   - Optionally verify gateway connectivity and write Codex config
#
# Non-scope:
#   - Does not start/stop Docker services
#   - Does not perform vendor account login/OAuth itself
#
# Invariants/Assumptions:
#   - Reads secrets from demo/.env when not already exported
#   - Never prints full keys unless --show-key is explicitly provided
#   - Exit codes follow ACP contract (0,1,2,3,64)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${REPO_ROOT}/demo/.env"

ACP_EXIT_SUCCESS=0
ACP_EXIT_DOMAIN=1
ACP_EXIT_PREREQ=2
ACP_EXIT_RUNTIME=3
ACP_EXIT_USAGE=64

DEFAULT_HOST="${GATEWAY_HOST:-127.0.0.1}"
DEFAULT_PORT="${LITELLM_PORT:-4000}"
DEFAULT_BUDGET="10.00"
GENERATED_ALIAS=""
GENERATED_KEY=""

log_info() { printf 'INFO: %s\n' "$*"; }
log_warn() { printf 'WARN: %s\n' "$*" >&2; }
log_error() { printf 'ERROR: %s\n' "$*" >&2; }

show_main_help() {
    cat <<'EOF'
Usage: onboard_impl.sh <tool> [options]

Tools:
  codex
  claude
  opencode
  cursor
  copilot

Options:
  --mode <mode>          auth mode (tool-dependent)
  --alias <alias>        virtual key alias (default: <tool>-cli)
  --budget <usd>         key budget in USD (default: 10.00)
  --model <model>        model alias override
  --host <host>          gateway host (default: 127.0.0.1)
  --port <port>          gateway port (default: 4000)
  --tls                  use https for base URL
  --verify               run authorized gateway checks
  --write-config         write ~/.codex/config.toml (Codex only)
  --show-key             print full key value
  --help, -h             show help

Codex modes:
  subscription           routed through gateway; upstream via ChatGPT provider (default)
  api-key                routed through gateway; upstream via API-key providers
  direct                 no gateway routing; OTEL visibility only

Examples:
  onboard_impl.sh codex --mode subscription --verify
  onboard_impl.sh codex --mode api-key --write-config
  onboard_impl.sh claude --mode api-key --verify
EOF
}

show_tool_help() {
    local tool="$1"
    case "$tool" in
    codex)
        cat <<'EOF'
Codex notes:
  - For subscription mode, run `make chatgpt-login` on the gateway host first.
  - Codex uses OPENAI_BASE_URL without /v1.
  - --write-config writes ~/.codex/config.toml for a LiteLLM provider profile.
EOF
        ;;
    claude)
        cat <<'EOF'
Claude notes:
  - Exports ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY for gateway routing.
  - Keep mode=api-key unless you have a separate subscription OAuth flow configured.
EOF
        ;;
    opencode | cursor | copilot)
        cat <<'EOF'
OpenAI-compatible tool notes:
  - Exports OPENAI_BASE_URL and OPENAI_API_KEY for gateway routing.
EOF
        ;;
    esac
}

ensure_prereqs() {
    if [ ! -f "${ENV_FILE}" ]; then
        log_error "Missing ${ENV_FILE}. Run: make install-env"
        exit "${ACP_EXIT_PREREQ}"
    fi

    if [ -x "${REPO_ROOT}/.bin/acpctl" ]; then
        ACPCTL_BIN="${REPO_ROOT}/.bin/acpctl"
    elif [ -x "${REPO_ROOT}/acpctl" ]; then
        ACPCTL_BIN="${REPO_ROOT}/acpctl"
    elif command -v acpctl >/dev/null 2>&1; then
        ACPCTL_BIN="acpctl"
    else
        log_error "acpctl binary not found. Run: make install-binary"
        exit "${ACP_EXIT_PREREQ}"
    fi

    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required"
        exit "${ACP_EXIT_PREREQ}"
    fi
}

load_env() {
    set -a
    # shellcheck source=/dev/null
    source "${ENV_FILE}"
    set +a
}

redact_key() {
    local key="$1"
    if [ "${#key}" -le 12 ]; then
        printf '***\n'
        return
    fi
    printf '%s...%s\n' "${key:0:8}" "${key: -4}"
}

build_base_url() {
    local host="$1"
    local port="$2"
    local tls_enabled="$3"
    if [ "${tls_enabled}" = "true" ]; then
        printf 'https://%s:%s\n' "${host}" "${port}"
    else
        printf 'http://%s:%s\n' "${host}" "${port}"
    fi
}

extract_key_from_output() {
    awk '/^sk-/{k=$0} END {print k}'
}

generate_virtual_key() {
    local alias="$1"
    local budget="$2"
    GENERATED_ALIAS="${alias}"
    GENERATED_KEY=""

    if [ -z "${LITELLM_MASTER_KEY:-}" ]; then
        log_error "LITELLM_MASTER_KEY is not set (demo/.env)"
        return "${ACP_EXIT_PREREQ}"
    fi

    local output
    if ! output="$("${ACPCTL_BIN}" key gen "${alias}" --budget "${budget}" 2>&1)"; then
        if printf '%s' "${output}" | grep -q "already exists"; then
            local retry_alias
            retry_alias="${alias}-$(date +%Y%m%d%H%M%S)"
            log_warn "Key alias '${alias}' already exists, retrying with '${retry_alias}'"
            GENERATED_ALIAS="${retry_alias}"
            if ! output="$("${ACPCTL_BIN}" key gen "${retry_alias}" --budget "${budget}" 2>&1)"; then
                printf '%s\n' "${output}" >&2
                return "${ACP_EXIT_DOMAIN}"
            fi
        else
            printf '%s\n' "${output}" >&2
            return "${ACP_EXIT_DOMAIN}"
        fi
    fi

    local key
    key="$(printf '%s\n' "${output}" | extract_key_from_output)"
    if [ -z "${key}" ]; then
        log_error "acpctl key generation did not return a key"
        printf '%s\n' "${output}" >&2
        return "${ACP_EXIT_RUNTIME}"
    fi

    GENERATED_KEY="${key}"
}

verify_gateway() {
    local host="$1"
    local port="$2"
    local tls_enabled="$3"
    local auth_key="$4"
    local proto="http"
    if [ "${tls_enabled}" = "true" ]; then
        proto="https"
    fi
    local base="${proto}://${host}:${port}"

    local health_code
    health_code="$(curl -sS -o /dev/null -w '%{http_code}' "${base}/health")"
    if [ "${health_code}" != "200" ] && [ "${health_code}" != "401" ]; then
        log_error "Gateway /health returned HTTP ${health_code}"
        return "${ACP_EXIT_DOMAIN}"
    fi

    local model_code
    model_code="$(curl -sS -o /dev/null -w '%{http_code}' \
        -H "Authorization: Bearer ${auth_key}" \
        "${base}/v1/models")"
    if [ "${model_code}" != "200" ]; then
        log_error "Authorized /v1/models check returned HTTP ${model_code}"
        return "${ACP_EXIT_DOMAIN}"
    fi

    log_info "Gateway health and authorized model checks passed"
}

verify_otel() {
    local host="$1"
    local otel_code
    otel_code="$(curl -sS -o /dev/null -w '%{http_code}' "http://${host}:4318/health" || true)"
    if [ "${otel_code}" != "200" ]; then
        log_warn "OTEL collector check returned HTTP ${otel_code:-n/a} at http://${host}:4318/health"
        return "${ACP_EXIT_DOMAIN}"
    fi
    log_info "OTEL collector health check passed"
}

write_codex_config() {
    local base_url="$1"
    local model="$2"
    local config_dir="${HOME}/.codex"
    local config_path="${config_dir}/config.toml"

    mkdir -p "${config_dir}"
    if [ -f "${config_path}" ]; then
        cp "${config_path}" "${config_path}.bak.$(date +%Y%m%d%H%M%S)"
    fi

    cat >"${config_path}" <<EOF
model = "${model}"
model_provider = "litellm"

[model_providers.litellm]
name = "LiteLLM"
base_url = "${base_url}/v1"
wire_api = "responses"
env_key = "OPENAI_API_KEY"
EOF

    log_info "Wrote ${config_path}"
}

print_exports() {
    local mode="$1"
    local tool="$2"
    local base_url="$3"
    local key="$4"
    local model="$5"
    local show_key="$6"
    local printed_key
    if [ "${show_key}" = "true" ]; then
        printed_key="${key}"
    else
        printed_key="$(redact_key "${key}")"
    fi

    if [ "${tool}" = "claude" ]; then
        printf 'export ANTHROPIC_BASE_URL="%s"\n' "${base_url}"
        printf 'export ANTHROPIC_API_KEY="%s"\n' "${printed_key}"
        printf 'export ANTHROPIC_MODEL="%s"\n' "${model}"
        return
    fi

    if [ "${mode}" = "direct" ]; then
        printf 'export OTEL_EXPORTER_OTLP_ENDPOINT="http://%s:4317"\n' "${HOST}"
        printf 'export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"\n'
        printf 'export OTEL_SERVICE_NAME="codex-cli"\n'
        return
    fi

    printf 'export OPENAI_BASE_URL="%s"\n' "${base_url}"
    printf 'export OPENAI_API_KEY="%s"\n' "${printed_key}"
    printf 'export OPENAI_MODEL="%s"\n' "${model}"
}

tool="${1:-}"
if [ -z "${tool}" ] || [ "${tool}" = "--help" ] || [ "${tool}" = "-h" ]; then
    show_main_help
    exit "${ACP_EXIT_SUCCESS}"
fi
shift || true

MODE=""
ALIAS="${tool}-cli"
BUDGET="${DEFAULT_BUDGET}"
MODEL=""
HOST="${DEFAULT_HOST}"
PORT="${DEFAULT_PORT}"
USE_TLS="false"
VERIFY="false"
WRITE_CONFIG="false"
SHOW_KEY="false"

while [ $# -gt 0 ]; do
    case "$1" in
    --mode)
        MODE="${2:-}"
        shift 2
        ;;
    --alias)
        ALIAS="${2:-}"
        shift 2
        ;;
    --budget)
        BUDGET="${2:-}"
        shift 2
        ;;
    --model)
        MODEL="${2:-}"
        shift 2
        ;;
    --host)
        HOST="${2:-}"
        shift 2
        ;;
    --port)
        PORT="${2:-}"
        shift 2
        ;;
    --tls)
        USE_TLS="true"
        shift
        ;;
    --verify)
        VERIFY="true"
        shift
        ;;
    --write-config)
        WRITE_CONFIG="true"
        shift
        ;;
    --show-key)
        SHOW_KEY="true"
        shift
        ;;
    --help | -h)
        show_main_help
        show_tool_help "${tool}"
        exit "${ACP_EXIT_SUCCESS}"
        ;;
    *)
        log_error "Unknown option: $1"
        exit "${ACP_EXIT_USAGE}"
        ;;
    esac
done

ensure_prereqs
load_env

case "${tool}" in
codex)
    MODE="${MODE:-subscription}"
    MODEL="${MODEL:-$([ "${MODE}" = "subscription" ] && printf 'chatgpt-gpt5.3-codex' || printf 'openai-gpt5.2')}"
    ;;
claude)
    MODE="${MODE:-api-key}"
    MODEL="${MODEL:-claude-haiku-4-5}"
    ;;
opencode | cursor | copilot)
    MODE="${MODE:-api-key}"
    MODEL="${MODEL:-openai-gpt5.2}"
    ;;
*)
    log_error "Unsupported tool: ${tool}"
    exit "${ACP_EXIT_USAGE}"
    ;;
esac

if [ "${MODE}" = "direct" ] && [ "${tool}" != "codex" ]; then
    log_error "Mode 'direct' is only supported for codex"
    exit "${ACP_EXIT_USAGE}"
fi

BASE_URL="$(build_base_url "${HOST}" "${PORT}" "${USE_TLS}")"
KEY_VALUE=""

if [ "${tool}" = "codex" ] && [ "${MODE}" = "subscription" ]; then
    proto="http"
    if [ "${USE_TLS}" = "true" ]; then
        proto="https"
    fi
    health_probe="$(curl -s -o /dev/null -w '%{http_code}' "${proto}://${HOST}:${PORT}/health" 2>/dev/null || true)"
    if [ "${health_probe}" != "200" ] && [ "${health_probe}" != "401" ]; then
        log_warn "Gateway health is not ready for subscription mode (HTTP ${health_probe})."
        log_warn "Complete ChatGPT device login first: make chatgpt-login"
        exit "${ACP_EXIT_DOMAIN}"
    fi
fi

if [ "${MODE}" != "direct" ]; then
    if ! generate_virtual_key "${ALIAS}" "${BUDGET}"; then
        exit "$?"
    fi
    KEY_VALUE="${GENERATED_KEY}"
fi

printf '\n'
printf 'Tool: %s\n' "${tool}"
printf 'Mode: %s\n' "${MODE}"
printf 'Gateway: %s\n' "${BASE_URL}"
printf 'Model: %s\n' "${MODEL}"
if [ -n "${KEY_VALUE}" ]; then
    printf 'Key alias: %s\n' "${GENERATED_ALIAS:-$ALIAS}"
fi
printf '\n'

if [ "${MODE}" = "subscription" ] && [ "${tool}" = "codex" ]; then
    log_info "Run 'make chatgpt-login' on this gateway host before launching Codex."
    printf '\n'
fi

print_exports "${MODE}" "${tool}" "${BASE_URL}" "${KEY_VALUE}" "${MODEL}" "${SHOW_KEY}"
printf '\n'

if [ "${WRITE_CONFIG}" = "true" ] && [ "${tool}" = "codex" ] && [ "${MODE}" != "direct" ]; then
    write_codex_config "${BASE_URL}" "${MODEL}"
    printf '\n'
fi

if [ "${VERIFY}" = "true" ]; then
    if [ "${MODE}" = "direct" ]; then
        verify_otel "${HOST}" || exit "$?"
    else
        verify_gateway "${HOST}" "${PORT}" "${USE_TLS}" "${KEY_VALUE}" || exit "$?"
    fi
fi

printf 'Onboarding complete.\n'
exit "${ACP_EXIT_SUCCESS}"
