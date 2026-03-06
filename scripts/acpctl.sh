#!/usr/bin/env bash
#
# AI Control Plane - acpctl Wrapper
#
# Purpose: Provide a stable script entrypoint for acpctl invocations in docs/scripts.
# Responsibilities:
#   - Honor ACPCTL_BIN override when explicitly provided
#   - Prefer repo-local .bin/acpctl when present
#   - Fall back to repo-root ./acpctl binary
#   - Export ACP_REPO_ROOT so the typed CLI resolves repo-local config consistently
#   - Pass through all arguments unchanged
#
# Non-scope:
#   - Does not implement command logic
#   - Does not mutate operator-config values beyond ACP_REPO_ROOT export
#
# Invariants/Assumptions:
#   - Script is executed from anywhere and resolves repo root from its own path

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
export ACP_REPO_ROOT="${ACP_REPO_ROOT:-${REPO_ROOT}}"

if [ -n "${ACPCTL_BIN:-}" ]; then
    if [ -x "${ACPCTL_BIN}" ]; then
        exec "${ACPCTL_BIN}" "$@"
    fi

    printf 'ERROR: ACPCTL_BIN is set but not executable: %s\n' "${ACPCTL_BIN}" >&2
    exit 2
fi

if [ -x "${REPO_ROOT}/.bin/acpctl" ]; then
    exec "${REPO_ROOT}/.bin/acpctl" "$@"
fi

if [ -x "${REPO_ROOT}/acpctl" ]; then
    exec "${REPO_ROOT}/acpctl" "$@"
fi

if command -v acpctl >/dev/null 2>&1; then
    exec acpctl "$@"
fi

printf 'ERROR: acpctl binary not found. Run: make install-binary\n' >&2
exit 2
