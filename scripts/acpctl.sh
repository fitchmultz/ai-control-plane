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
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/libexec/common.sh"

export ACP_REPO_ROOT="${ACP_REPO_ROOT:-$(bridge_repo_root)}"
ACPCTL_RESOLVED_BIN="$(bridge_acpctl_bin)" || exit $?
exec "${ACPCTL_RESOLVED_BIN}" "$@"
