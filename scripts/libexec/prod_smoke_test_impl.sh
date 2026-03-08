#!/usr/bin/env bash
#
# AI Control Plane - Runtime Smoke Bridge
#
# Purpose:
#   Preserve the legacy bridge entrypoint for runtime production smoke checks
#   while delegating to the typed acpctl command surface.
#
# Responsibilities:
#   - Resolve the canonical acpctl binary.
#   - Forward all arguments to `acpctl smoke`.
#
# Non-scope:
#   - Does not implement smoke test logic directly.
#
# Invariants/Assumptions:
#   - `acpctl smoke` is the authoritative implementation.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

ACPCTL_BIN="$(bridge_acpctl_bin)" || exit $?
exec "${ACPCTL_BIN}" smoke "$@"
