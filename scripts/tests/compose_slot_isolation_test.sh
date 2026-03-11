#!/usr/bin/env bash
set -euo pipefail

# Compose Slot Isolation Test
#
# Purpose:
#   - Verify compose configuration keeps slot-specific ports and volume names isolated.
#
# Responsibilities:
#   - Assert slot-scoped host port overrides and persistent volume naming.
#   - Assert slot-aware volume contracts stay intact.
#
# Scope:
#   - Slot isolation and naming contracts only.
#
# Usage:
#   - bash scripts/tests/compose_slot_isolation_test.sh
#
# Invariants/Assumptions:
#   - Tests operate on tracked compose files only.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: compose_slot_isolation_test.sh [OPTIONS]

Validate compose slot isolation contracts.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

REPO_ROOT="$(test_repo_root)"
COMPOSE_FILE="${REPO_ROOT}/demo/docker-compose.yml"
UI_OVERLAY_FILE="${REPO_ROOT}/demo/docker-compose.ui.yml"

printf 'Compose Slot Isolation Test\n'
printf '===========================\n'

if ! grep -Fq '${LITELLM_HOST_PORT:-4000}:4000' "${COMPOSE_FILE}"; then
    printf '  ✗ compose should expose slot-specific LiteLLM host port overrides\n'
    exit 1
fi
printf '  ✓ compose exposes slot-specific LiteLLM host port overrides\n'

if ! grep -q 'name: ai_control_plane_pgdata_${ACP_SLOT:-active}' "${COMPOSE_FILE}"; then
    printf '  ✗ compose should use ACP_SLOT-scoped pgdata names\n'
    exit 1
fi
printf '  ✓ compose uses ACP_SLOT-scoped pgdata names\n'

if ! grep -q 'librechat_mongodb_data_active' "${UI_OVERLAY_FILE}" || ! grep -q 'librechat_mongodb_data_standby' "${UI_OVERLAY_FILE}"; then
    printf '  ✗ UI overlay should declare distinct active and standby librechat volumes\n'
    exit 1
fi
printf '  ✓ UI overlay declares distinct active and standby librechat volumes\n'
