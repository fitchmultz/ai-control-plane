#!/usr/bin/env bash
#
# AI Control Plane - Tool Onboarding Bridge
#
# Purpose:
#   Route legacy onboarding script entrypoints to the typed acpctl implementation.
#
# Responsibilities:
#   - Resolve the canonical acpctl binary.
#   - Forward all onboarding arguments without reimplementing workflow logic.
#   - Preserve ACP exit codes from the typed onboarding command.
#
# Non-scope:
#   - Does not own onboarding business logic.
#   - Does not parse demo/.env or call gateway APIs directly.
#
# Invariants/Assumptions:
#   - acpctl onboarding is the single source of truth.
#   - The wrapper remains compatibility-only and intentionally thin.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

resolve_acpctl_bin() {
    if [ -x "${REPO_ROOT}/.bin/acpctl" ]; then
        printf '%s\n' "${REPO_ROOT}/.bin/acpctl"
        return 0
    fi
    if [ -x "${REPO_ROOT}/acpctl" ]; then
        printf '%s\n' "${REPO_ROOT}/acpctl"
        return 0
    fi
    if command -v acpctl >/dev/null 2>&1; then
        command -v acpctl
        return 0
    fi
    printf 'ERROR: acpctl binary not found. Run: make install-binary\n' >&2
    return 2
}

ACPCTL_BIN="$(resolve_acpctl_bin)" || exit $?
exec "${ACPCTL_BIN}" onboard "$@"
