#!/usr/bin/env bash
#
# AI Control Plane - Curl Log Stub
#
# Purpose:
#   Capture curl invocations and optionally emit HTTP 200 status markers.
#
# Responsibilities:
#   - Append argv payloads to a caller-provided log file.
#   - Print `200` when the caller requests `%{http_code}` output.
#
# Scope:
#   - Test-only stub under scripts/tests/stubs.
#
# Usage:
#   - Install into a temp bin dir and set ACP_TEST_CURL_LOG.
#
# Invariants/Assumptions:
#   - ACP_TEST_CURL_LOG is provided by the caller.

set -euo pipefail

log_file="${ACP_TEST_CURL_LOG:?missing ACP_TEST_CURL_LOG}"
printf '%s\n' "$*" >>"${log_file}"

if [[ "$*" == *"%{http_code}"* ]]; then
    printf '200'
fi
