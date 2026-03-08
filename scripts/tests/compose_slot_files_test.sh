#!/usr/bin/env bash
set -euo pipefail

# Compose Slot Files Test
#
# Purpose:
#   - Verify required compose files and core service declarations exist.
#
# Responsibilities:
#   - Assert tracked compose files are present.
#   - Assert core services and key port/volume identifiers remain declared.
#
# Scope:
#   - Static compose file presence and content checks only.
#
# Usage:
#   - bash scripts/tests/compose_slot_files_test.sh
#
# Invariants/Assumptions:
#   - Tests do not require Docker Compose to be installed.

show_help() {
    cat <<'EOF'
Usage: compose_slot_files_test.sh [OPTIONS]

Validate compose file presence and core declarations.

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
COMPOSE_DIR="${REPO_ROOT}/demo"

printf 'Compose Slot Files Test\n'
printf '=======================\n'

for file in docker-compose.yml docker-compose.offline.yml docker-compose.tls.yml; do
    if [[ ! -f "${COMPOSE_DIR}/${file}" ]]; then
        printf '  ✗ missing %s\n' "${file}"
        exit 1
    fi
    printf '  ✓ found %s\n' "${file}"
done

if ! grep -q "litellm:" "${COMPOSE_DIR}/docker-compose.yml"; then
    printf '  ✗ docker-compose.yml missing litellm service\n'
    exit 1
fi
printf '  ✓ docker-compose.yml includes litellm service\n'

if ! grep -q "postgres:" "${COMPOSE_DIR}/docker-compose.yml"; then
    printf '  ✗ docker-compose.yml missing postgres service\n'
    exit 1
fi
printf '  ✓ docker-compose.yml includes postgres service\n'

if ! grep -q "4000" "${COMPOSE_DIR}/docker-compose.yml"; then
    printf '  ✗ compose file should reference port 4000\n'
    exit 1
fi
printf '  ✓ compose file references port 4000\n'

if ! grep -q "pgdata" "${COMPOSE_DIR}/docker-compose.yml"; then
    printf '  ✗ compose file should reference pgdata volume\n'
    exit 1
fi
printf '  ✓ compose file references pgdata volume\n'
