#!/usr/bin/env bash
#
# AI Control Plane - Docker Compose OK Stub
#
# Purpose:
#   Provide deterministic docker/compose responses for shell contract tests.
#
# Responsibilities:
#   - Report docker compose availability.
#   - Emit a mounted overlay path for inspect checks.
#   - Quietly succeed for log probes.
#
# Scope:
#   - Test-only stub under scripts/tests/stubs.
#
# Usage:
#   - Install into a temp bin dir as `docker`.
#
# Invariants/Assumptions:
#   - Satisfies the happy-path behavior expected by ChatGPT login tests.

set -euo pipefail

case "${1:-}" in
compose)
    [[ "${2:-}" == "version" ]] && exit 0
    ;;
inspect)
    printf '/tmp/litellm-chatgpt.yaml -> /app/config.yaml\n'
    exit 0
    ;;
logs)
    exit 0
    ;;
esac

exit 0
