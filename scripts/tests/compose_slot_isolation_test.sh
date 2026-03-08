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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/demo/docker-compose.yml"

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

if ! grep -q 'librechat_mongodb_data_active' "${COMPOSE_FILE}" || ! grep -q 'librechat_mongodb_data_standby' "${COMPOSE_FILE}"; then
	printf '  ✗ compose should declare distinct active and standby librechat volumes\n'
	exit 1
fi
printf '  ✓ compose declares distinct active and standby librechat volumes\n'
