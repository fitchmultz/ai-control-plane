#!/usr/bin/env bash
#
# AI Control Plane - Host Deploy Bridge
#
# Purpose:
#   Execute the repository's declarative Ansible host deployment workflow via a
#   stable bridge entrypoint.
#
# Responsibilities:
#   - Validate local Ansible prerequisites and repository surfaces.
#   - Run syntax-checked Ansible check/apply operations against the gateway playbook.
#   - Forward explicit host deployment overrides as Ansible extra vars.
#
# Non-scope:
#   - Does not reimplement playbook logic.
#   - Does not perform local systemd lifecycle actions directly.
#
# Invariants/Assumptions:
#   - `deploy/ansible/playbooks/gateway_host.yml` is the source of truth for
#     remote host convergence.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

show_help() {
    cat <<'EOF'
Usage: host_deploy_impl.sh <check|apply> [OPTIONS]

Run the declarative host deployment playbook in dry-run (`check`) or apply mode.

Options:
  --inventory PATH           Inventory file (default: deploy/ansible/inventory/hosts.yml)
  --limit TARGET             Optional Ansible --limit selector
  --repo-path PATH           Override acp_repo_path
  --env-file PATH            Override acp_env_file
  --tls-mode plain|tls       Override acp_tls_mode
  --public-url URL           Override acp_public_url
  --no-wait                  Set acp_wait_for_stabilization=false
  --skip-smoke-tests         Set acp_run_smoke_tests=false
  --stabilization-seconds N  Override acp_stabilization_seconds
  --extra-var KEY=VALUE      Additional Ansible extra var (repeatable)
  --help                     Show this help message

Examples:
  host_deploy_impl.sh check --inventory deploy/ansible/inventory/hosts.yml
  host_deploy_impl.sh apply --inventory deploy/ansible/inventory/hosts.yml --limit gateway
  host_deploy_impl.sh apply --repo-path /opt/ai-control-plane --tls-mode tls --public-url https://gateway.example.com

Exit Codes:
  0   Success
  2   Usage or prerequisite failure
  3   Runtime or playbook failure
EOF
}

[[ $# -ge 1 ]] || {
    show_help >&2
    exit 2
}
subcommand="$1"
shift

case "${subcommand}" in
check | apply) ;;
--help | -h)
    show_help
    exit 0
    ;;
*)
    printf 'ERROR: unknown host deploy command: %s\n' "${subcommand}" >&2
    show_help >&2
    exit 2
    ;;
esac

inventory="deploy/ansible/inventory/hosts.yml"
limit_target=""
repo_path=""
env_file=""
tls_mode=""
public_url=""
wait_for_stabilization="true"
run_smoke_tests="true"
stabilization_seconds=""
declare -a extra_vars=()

while [[ $# -gt 0 ]]; do
    case "$1" in
    --inventory)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --inventory\n' >&2
            exit 2
        }
        inventory="$2"
        shift 2
        ;;
    --limit)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --limit\n' >&2
            exit 2
        }
        limit_target="$2"
        shift 2
        ;;
    --repo-path)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --repo-path\n' >&2
            exit 2
        }
        repo_path="$2"
        shift 2
        ;;
    --env-file)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --env-file\n' >&2
            exit 2
        }
        env_file="$2"
        shift 2
        ;;
    --tls-mode)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --tls-mode\n' >&2
            exit 2
        }
        tls_mode="$2"
        shift 2
        ;;
    --public-url)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --public-url\n' >&2
            exit 2
        }
        public_url="$2"
        shift 2
        ;;
    --no-wait)
        wait_for_stabilization="false"
        shift
        ;;
    --skip-smoke-tests)
        run_smoke_tests="false"
        shift
        ;;
    --stabilization-seconds)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --stabilization-seconds\n' >&2
            exit 2
        }
        stabilization_seconds="$2"
        shift 2
        ;;
    --extra-var)
        [[ $# -ge 2 ]] || {
            printf 'ERROR: missing value for --extra-var\n' >&2
            exit 2
        }
        extra_vars+=("$2")
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

if [[ -n "${tls_mode}" && "${tls_mode}" != "plain" && "${tls_mode}" != "tls" ]]; then
    printf 'ERROR: --tls-mode must be plain or tls\n' >&2
    exit 2
fi

repo_root="$(bridge_repo_root)"
bridge_require_command ansible-playbook
inventory="$(bridge_abspath "${inventory}")"
playbook_path="${repo_root}/deploy/ansible/playbooks/gateway_host.yml"
ansible_cfg="${repo_root}/deploy/ansible/ansible.cfg"

[[ -f "${inventory}" ]] || {
    printf 'ERROR: inventory file not found: %s\n' "${inventory}" >&2
    exit 2
}
[[ -f "${playbook_path}" ]] || {
    printf 'ERROR: playbook not found: %s\n' "${playbook_path}" >&2
    exit 3
}
[[ -f "${ansible_cfg}" ]] || {
    printf 'ERROR: ansible config not found: %s\n' "${ansible_cfg}" >&2
    exit 3
}

declare -a ansible_args=("-i" "${inventory}" "${playbook_path}")
if [[ -n "${limit_target}" ]]; then
    ansible_args+=("--limit" "${limit_target}")
fi
if [[ "${subcommand}" == "check" ]]; then
    ansible_args+=("--check")
fi

[[ -n "${repo_path}" ]] && extra_vars+=("acp_repo_path=${repo_path}")
[[ -n "${env_file}" ]] && extra_vars+=("acp_env_file=${env_file}")
[[ -n "${tls_mode}" ]] && extra_vars+=("acp_tls_mode=${tls_mode}")
[[ -n "${public_url}" ]] && extra_vars+=("acp_public_url=${public_url}")
extra_vars+=("acp_wait_for_stabilization=${wait_for_stabilization}")
extra_vars+=("acp_run_smoke_tests=${run_smoke_tests}")
[[ -n "${stabilization_seconds}" ]] && extra_vars+=("acp_stabilization_seconds=${stabilization_seconds}")

for item in "${extra_vars[@]}"; do
    ansible_args+=("--extra-vars" "${item}")
done

ANSIBLE_CONFIG="${ansible_cfg}" ansible-playbook --syntax-check "${ansible_args[@]}"
ANSIBLE_CONFIG="${ansible_cfg}" ansible-playbook "${ansible_args[@]}"
