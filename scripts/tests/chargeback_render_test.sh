#!/usr/bin/env bash
#
# AI Control Plane - Chargeback Render Script Contract Tests
#
# Purpose:
#   Validate the chargeback render shell wrapper delegates to the typed renderer safely.
#
# Responsibilities:
#   - Exercise JSON rendering with hostile field content.
#   - Exercise CSV rendering with standards-compliant escaping and spreadsheet safety.
#   - Verify wrapper integration through demo/scripts/lib/chargeback_render.sh.
#
# Non-scope:
#   - Does not query a real database.
#   - Does not validate markdown rendering paths.
#
# Invariants/Assumptions:
#   - Builds a temporary acpctl binary for the typed renderer surface.
#   - Sources the canonical render library from demo/scripts/lib.

set -euo pipefail

show_help() {
    cat <<'EOF'
Usage: chargeback_render_test.sh [OPTIONS]

Run contract tests for demo/scripts/lib/chargeback_render.sh.

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/chargeback_render_test.sh
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LIB_UNDER_TEST="${PROJECT_ROOT}/demo/scripts/lib/chargeback_render.sh"

TESTS_PASSED=0
TESTS_FAILED=0

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

ACPCTL_BIN_TEST="${TMP_DIR}/acpctl-bin"
go build -trimpath -o "${ACPCTL_BIN_TEST}" "${PROJECT_ROOT}/cmd/acpctl"

# shellcheck source=/dev/null
source "${LIB_UNDER_TEST}"

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

get_timestamp() {
    echo "2026-03-07T17:00:00Z"
}

SCHEMA_VERSION="1.0.0"
VARIANCE_THRESHOLD="15"
ANOMALY_THRESHOLD="200"
FORECAST_ENABLED="true"
ACPCTL_BIN="${ACPCTL_BIN_TEST}"
export PROJECT_ROOT ACPCTL_BIN SCHEMA_VERSION VARIANCE_THRESHOLD ANOMALY_THRESHOLD FORECAST_ENABLED

test_json_wrapper_preserves_content() {
    echo "Test: output_json preserves hostile content via typed renderer..."
    local cost_center_json='[{"cost_center":"cc,\"quoted\"\nnext","team":"ops\\blue","request_count":2,"token_count":5,"spend_amount":12.5,"percent_of_total":25},{"cost_center":"unknown-cc","team":"none","request_count":1,"token_count":1,"spend_amount":1.5,"percent_of_total":3}]'
    local model_json='[{"model":"gpt-4o,\"mini\"","request_count":2,"token_count":5,"spend_amount":12.5}]'
    local anomalies_json='[{"cost_center":"=@finance","team":"team,\nquoted","current_spend":9.5,"previous_spend":3.1,"spike_percent":206.45,"type":"spike"}]'

    local rendered
    rendered="$(output_json "2026-02" "2026-02-01" "2026-02-28" "50.5" "9" "42" "${cost_center_json}" "${model_json}" "N/A" "40.0" "${anomalies_json}" "10.0,11.0,12.0" "1.2" "15" "2026-03-15" "100" "medium" "75.5" "false")"

    if echo "${rendered}" | grep -Fq '"cost_center": "cc,\"quoted\"\nnext"'; then
        pass "JSON keeps escaped newline/quote content"
    else
        fail "JSON should preserve hostile cost center content"
    fi

    if echo "${rendered}" | grep -Fq '"variance_percent": "N/A"'; then
        pass "JSON keeps N/A variance as string"
    else
        fail "JSON should preserve N/A variance"
    fi

    if echo "${rendered}" | grep -Fq '"risk_level": "medium"'; then
        pass "JSON includes budget risk fields from typed renderer"
    else
        fail "JSON should include budget risk fields"
    fi
}

test_csv_wrapper_escapes_and_protects() {
    echo "Test: output_csv escapes CSV fields and protects formula-leading cells..."
    local cost_center_json='[{"cost_center":"=SUM(1,2)","team":"\"quoted\",team\nnext","request_count":12,"token_count":34,"spend_amount":99.01,"percent_of_total":45.6},{"cost_center":"normal","team":"@cmd","request_count":1,"token_count":2,"spend_amount":1.2,"percent_of_total":3.4}]'
    local rendered
    rendered="$(output_csv "2026-02" "${cost_center_json}")"

    if echo "${rendered}" | grep -Fq "'=SUM(1,2)"; then
        pass "CSV prefixes formula-leading cells with apostrophe"
    else
        fail "CSV should protect formula-leading cells"
    fi

    if echo "${rendered}" | grep -Fq "\"\"quoted\"\",team"; then
        pass "CSV escapes embedded double quotes"
    else
        fail "CSV should escape embedded double quotes"
    fi

    if echo "${rendered}" | grep -Fq "'@cmd"; then
        pass "CSV protects @-prefixed cells"
    else
        fail "CSV should protect @-prefixed cells"
    fi
}

main() {
    echo "Chargeback Render Script Contract Tests"
    echo "======================================"
    echo ""

    test_json_wrapper_preserves_content
    test_csv_wrapper_escapes_and_protects

    echo ""
    echo "Results"
    echo "-------"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"

    if [[ "${TESTS_FAILED}" -eq 0 ]]; then
        echo "All chargeback render tests passed."
        exit 0
    fi

    echo "One or more chargeback render tests failed."
    exit 1
}

main "$@"
