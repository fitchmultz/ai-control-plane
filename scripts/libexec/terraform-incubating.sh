#!/usr/bin/env bash
#
# AI Control Plane - Incubating Terraform Validation Runner
#
# Purpose:
#   Run explicit internal-only Terraform validation workflows for the incubating
#   cloud deployment assets without dirtying the repository checkout.
#
# Responsibilities:
#   - Check Terraform formatting under deploy/incubating/terraform.
#   - Validate incubating Terraform examples from a staged temporary tree.
#   - Run an AWS dry-run plan in validation-only or live-account mode.
#   - Run an optional tfsec scan when the tool is installed.
#
# Non-scope:
#   - Does not promote Terraform into the public operator UX or default CI.
#   - Does not perform Terraform apply or cloud runtime smoke tests.
#
# Invariants/Assumptions:
#   - Terraform assets remain under deploy/incubating/.
#   - Validation must not leave .terraform or lock files in the repository tree.
#   - AWS dry-run planning defaults to validation-only mode with infra targets.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

REPO_ROOT="$(bridge_repo_root)"
INCUBATING_ROOT="${REPO_ROOT}/deploy/incubating"
TERRAFORM_ROOT="${INCUBATING_ROOT}/terraform"
AWS_EXAMPLE_DIR="${TERRAFORM_ROOT}/examples/aws-complete"
TF_MIN_VERSION="1.5.0"
TF_CONTAINER_IMAGE="${TF_CONTAINER_IMAGE:-hashicorp/terraform:1.5.7}"
TFSEC_CONTAINER_IMAGE="${TFSEC_CONTAINER_IMAGE:-aquasec/tfsec:latest}"
TF_AWS_PLAN_MODE="${TF_AWS_PLAN_MODE:-validation-only}"
TF_AWS_PLAN_TARGETS_DEFAULT="terraform_data.deployment_guardrails,module.vpc,module.eks,module.rds,module.irsa,random_password.rds,aws_s3_bucket.backups,aws_s3_bucket_versioning.backups,aws_s3_bucket_public_access_block.backups,aws_s3_bucket_server_side_encryption_configuration.backups,aws_s3_bucket_lifecycle_configuration.backups,aws_iam_role_policy.backup_replication"
TF_AWS_VARS_FILE_DEFAULT="${AWS_EXAMPLE_DIR}/terraform.tfvars.example"

if [ -t 1 ]; then
    COLOR_RED='\033[31m'
    COLOR_GREEN='\033[32m'
    COLOR_YELLOW='\033[33m'
    COLOR_BOLD='\033[1m'
    COLOR_RESET='\033[0m'
else
    COLOR_RED=''
    COLOR_GREEN=''
    COLOR_YELLOW=''
    COLOR_BOLD=''
    COLOR_RESET=''
fi

show_help() {
    cat <<'EOF'
Usage: terraform-incubating.sh <fmt-check|validate|plan-aws|security-check> [OPTIONS]

Run explicit internal-only Terraform validation for deploy/incubating/terraform.

Commands:
  fmt-check       Run terraform fmt -check -recursive against deploy/incubating/terraform
  validate        Run fmt-check, then terraform init -backend=false and validate for all examples
  plan-aws        Run an AWS dry-run plan for examples/aws-complete
  security-check  Run tfsec against deploy/incubating/terraform when tfsec is available
  help            Show this help text

Environment:
  TF_CONTAINER_IMAGE        Terraform Docker image fallback when host terraform is missing or too old
                            Default: hashicorp/terraform:1.5.7
  TFSEC_CONTAINER_IMAGE     Optional tfsec Docker image fallback
                            Default: aquasec/tfsec:latest
  TF_AWS_PLAN_MODE          validation-only (default) or live
                            validation-only avoids live AWS lookups and scopes the plan to infra targets
  TF_AWS_VARS_FILE          Vars file to use for plan-aws
                            Default: deploy/incubating/terraform/examples/aws-complete/terraform.tfvars.example
  TF_AWS_PLAN_TARGETS       Comma-separated terraform -target list for plan-aws
                            Default: internal infra-target set
                            Use "none" to disable explicit targeting

Examples:
  ./scripts/libexec/terraform-incubating.sh fmt-check
  ./scripts/libexec/terraform-incubating.sh validate
  ./scripts/libexec/terraform-incubating.sh plan-aws
  TF_AWS_PLAN_MODE=live ./scripts/libexec/terraform-incubating.sh plan-aws
  TF_AWS_PLAN_TARGETS=none TF_AWS_PLAN_MODE=live ./scripts/libexec/terraform-incubating.sh plan-aws
  ./scripts/libexec/terraform-incubating.sh security-check

Exit codes:
  0   Success
  2   Prerequisite failure
  3   Runtime/internal failure
  64  Usage error
EOF
}

version_gte() {
    local lhs="$1"
    local rhs="$2"
    local lhs_major lhs_minor lhs_patch rhs_major rhs_minor rhs_patch
    IFS=. read -r lhs_major lhs_minor lhs_patch <<< "${lhs}"
    IFS=. read -r rhs_major rhs_minor rhs_patch <<< "${rhs}"
    lhs_patch="${lhs_patch:-0}"
    rhs_patch="${rhs_patch:-0}"

    if (( lhs_major > rhs_major )); then
        return 0
    fi
    if (( lhs_major < rhs_major )); then
        return 1
    fi
    if (( lhs_minor > rhs_minor )); then
        return 0
    fi
    if (( lhs_minor < rhs_minor )); then
        return 1
    fi
    (( lhs_patch >= rhs_patch ))
}

terraform_host_version() {
    if ! command -v terraform >/dev/null 2>&1; then
        return 1
    fi
    terraform version 2>/dev/null | awk 'NR==1 {gsub(/^v/, "", $2); print $2}'
}

terraform_runner() {
    local host_version
    if host_version="$(terraform_host_version)"; then
        if version_gte "${host_version}" "${TF_MIN_VERSION}"; then
            printf 'host\n'
            return 0
        fi
        printf 'INFO: host terraform %s is older than %s; using Docker fallback %s\n' "${host_version}" "${TF_MIN_VERSION}" "${TF_CONTAINER_IMAGE}" >&2
    else
        printf 'INFO: host terraform not found; using Docker fallback %s\n' "${TF_CONTAINER_IMAGE}" >&2
    fi
    bridge_require_command docker >/dev/null
    printf 'docker\n'
}

run_terraform() {
    local mount_root="$1"
    local workdir="$2"
    shift 2

    case "$(terraform_runner)" in
    host)
        (
            cd "${workdir}"
            terraform "$@"
        )
        ;;
    docker)
        docker run --rm \
            --user "$(id -u):$(id -g)" \
            -e HOME=/tmp \
            -e TF_IN_AUTOMATION=1 \
            -v "${mount_root}:${mount_root}" \
            -w "${workdir}" \
            "${TF_CONTAINER_IMAGE}" "$@"
        ;;
    *)
        bridge_log_error "unable to determine terraform runner"
        return "${ACP_EXIT_RUNTIME}"
        ;;
    esac
}

stage_incubating_tree() {
    local stage_dir
    stage_dir="$(mktemp -d)"
    mkdir -p "${stage_dir}/deploy"
    cp -R "${INCUBATING_ROOT}" "${stage_dir}/deploy/"
    printf '%s\n' "${stage_dir}"
}

run_fmt_check() {
    printf '%bRunning terraform fmt check...%b\n' "${COLOR_BOLD}" "${COLOR_RESET}"
    run_terraform "${REPO_ROOT}" "${REPO_ROOT}" fmt -check -recursive "${TERRAFORM_ROOT}"
    printf '%b✓ Terraform formatting is clean%b\n' "${COLOR_GREEN}" "${COLOR_RESET}"
}

run_validate() {
    run_fmt_check

    printf '%bRunning terraform validate across incubating examples...%b\n' "${COLOR_BOLD}" "${COLOR_RESET}"
    local stage_dir example_dir staged_examples_root
    stage_dir="$(stage_incubating_tree)"
    trap 'rm -rf "${stage_dir:-}"' RETURN

    staged_examples_root="${stage_dir}/deploy/incubating/terraform/examples"
    for example_dir in "${staged_examples_root}"/*; do
        [ -d "${example_dir}" ] || continue
        printf '  -> %s\n' "${example_dir#${stage_dir}/}"
        run_terraform "${stage_dir}" "${example_dir}" init -backend=false -input=false -no-color >/dev/null
        run_terraform "${stage_dir}" "${example_dir}" validate -no-color >/dev/null
    done

    printf '%b✓ Terraform validation passed for all examples%b\n' "${COLOR_GREEN}" "${COLOR_RESET}"
}

apply_plan_targets() {
    local -n args_ref=$1
    local targets_csv="$2"
    local target trimmed

    if [ "${targets_csv}" = "none" ]; then
        return 0
    fi

    IFS=, read -r -a target_list <<< "${targets_csv}"
    for target in "${target_list[@]}"; do
        trimmed="$(printf '%s' "${target}" | awk '{$1=$1; print}')"
        [ -n "${trimmed}" ] || continue
        args_ref+=("-target=${trimmed}")
    done
}

run_plan_aws() {
    case "${TF_AWS_PLAN_MODE}" in
    validation-only|live) ;;
    *)
        bridge_log_error "TF_AWS_PLAN_MODE must be validation-only or live (got: ${TF_AWS_PLAN_MODE})"
        return "${ACP_EXIT_USAGE}"
        ;;
    esac

    local vars_file stage_dir staged_aws_dir staged_vars_file targets_csv
    vars_file="${TF_AWS_VARS_FILE:-${TF_AWS_VARS_FILE_DEFAULT}}"
    if [ ! -f "${vars_file}" ]; then
        bridge_log_error "vars file not found: ${vars_file}"
        return "${ACP_EXIT_USAGE}"
    fi

    stage_dir="$(stage_incubating_tree)"
    trap 'rm -rf "${stage_dir:-}"' RETURN
    staged_aws_dir="${stage_dir}/deploy/incubating/terraform/examples/aws-complete"
    staged_vars_file="${staged_aws_dir}/terraform.tfvars.plan"
    cp "${vars_file}" "${staged_vars_file}"

    printf '%bRunning AWS dry-run plan (%s mode)...%b\n' "${COLOR_BOLD}" "${TF_AWS_PLAN_MODE}" "${COLOR_RESET}"
    run_terraform "${stage_dir}" "${staged_aws_dir}" init -backend=false -input=false -no-color >/dev/null

    local -a plan_args
    plan_args=(plan -input=false -lock=false -refresh=false -no-color -var-file="${staged_vars_file}" -out=tfplan.bin)
    if [ "${TF_AWS_PLAN_MODE}" = "validation-only" ]; then
        plan_args+=("-var=validation_only=true")
    fi

    targets_csv="${TF_AWS_PLAN_TARGETS:-${TF_AWS_PLAN_TARGETS_DEFAULT}}"
    apply_plan_targets plan_args "${targets_csv}"

    run_terraform "${stage_dir}" "${staged_aws_dir}" "${plan_args[@]}" >/dev/null
    printf '%b✓ AWS dry-run plan completed (%s mode)%b\n' "${COLOR_GREEN}" "${TF_AWS_PLAN_MODE}" "${COLOR_RESET}"
}

run_security_check() {
    if command -v tfsec >/dev/null 2>&1; then
        printf '%bRunning tfsec against incubating Terraform assets...%b\n' "${COLOR_BOLD}" "${COLOR_RESET}"
        tfsec "${TERRAFORM_ROOT}"
        printf '%b✓ tfsec completed%b\n' "${COLOR_GREEN}" "${COLOR_RESET}"
        return 0
    fi

    if command -v docker >/dev/null 2>&1; then
        printf '%bRunning tfsec Docker fallback against incubating Terraform assets...%b\n' "${COLOR_BOLD}" "${COLOR_RESET}"
        docker run --rm \
            --user "$(id -u):$(id -g)" \
            -v "${REPO_ROOT}:${REPO_ROOT}" \
            -w "${REPO_ROOT}" \
            "${TFSEC_CONTAINER_IMAGE}" "${TERRAFORM_ROOT}"
        printf '%b✓ tfsec completed%b\n' "${COLOR_GREEN}" "${COLOR_RESET}"
        return 0
    fi

    printf '%b! tfsec not installed and Docker unavailable; skipping optional Terraform security scan%b\n' "${COLOR_YELLOW}" "${COLOR_RESET}"
}

command="${1:-help}"
case "${command}" in
fmt-check)
    run_fmt_check
    ;;
validate)
    run_validate
    ;;
plan-aws)
    run_plan_aws
    ;;
security-check)
    run_security_check
    ;;
help|-h|--help)
    show_help
    ;;
*)
    bridge_log_error "unknown terraform-incubating command: ${command}"
    show_help >&2
    exit "${ACP_EXIT_USAGE}"
    ;;
esac
