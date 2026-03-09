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
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

ACPCTL_BIN="$(bridge_acpctl_bin)" || exit $?
exec "${ACPCTL_BIN}" onboard "$@"
