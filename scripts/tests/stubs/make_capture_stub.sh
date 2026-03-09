#!/usr/bin/env bash
#
# AI Control Plane - Make Capture Stub
#
# Purpose:
#   Capture delegated make arguments for shell contract tests.
#
# Responsibilities:
#   - Record make arguments to a caller-provided capture file.
#   - Exit successfully so delegation tests stay deterministic.
#
# Scope:
#   - Test-only stub under scripts/tests/stubs.
#
# Usage:
#   - Install into a temp bin dir and set ACPCTL_TEST_CAPTURE_FILE.
#
# Invariants/Assumptions:
#   - ACPCTL_TEST_CAPTURE_FILE is provided by the caller.

set -euo pipefail

capture_file="${ACPCTL_TEST_CAPTURE_FILE:?missing ACPCTL_TEST_CAPTURE_FILE}"
printf '%s\n' "$@" >"${capture_file}"
