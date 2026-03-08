#!/usr/bin/env bash
#
# AI Control Plane - Release Bundle Bridge
#
# Purpose:
#   Preserve the legacy bridge entrypoint for release-bundle operations while
#   delegating to the typed acpctl command surface.
#
# Responsibilities:
#   - Resolve the canonical acpctl binary.
#   - Forward all arguments to `acpctl deploy release-bundle`.
#
# Non-scope:
#   - Does not implement release-bundle logic directly.
#
# Invariants/Assumptions:
#   - `acpctl deploy release-bundle` is the authoritative implementation.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/libexec/common.sh
source "${SCRIPT_DIR}/common.sh"

ACPCTL_BIN="$(bridge_acpctl_bin)" || exit $?
exec "${ACPCTL_BIN}" deploy release-bundle "$@"
