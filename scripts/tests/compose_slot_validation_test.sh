#!/usr/bin/env bash
set -euo pipefail

# Compose Slot Validation Test
#
# Purpose:
#   - Verify compose files generate valid configs or satisfy static fallback checks.
#
# Responsibilities:
#   - Prefer docker compose validation when available.
#   - Fall back to static assertions when compose tooling is unavailable.
#
# Scope:
#   - Compose config validation behavior only.
#
# Usage:
#   - bash scripts/tests/compose_slot_validation_test.sh
#
# Invariants/Assumptions:
#   - Tests remain deterministic regardless of local tooling availability.

show_help() {
    cat <<'EOF'
Usage: compose_slot_validation_test.sh [OPTIONS]

Validate compose configuration generation or static syntax markers.

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

printf 'Compose Slot Validation Test\n'
printf '============================\n'

if docker compose version >/dev/null 2>&1; then
    docker compose -f "${COMPOSE_DIR}/docker-compose.yml" config >/dev/null
    docker compose -f "${COMPOSE_DIR}/docker-compose.offline.yml" config >/dev/null
    printf '  ✓ docker compose validates main and offline configs\n'
elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f "${COMPOSE_DIR}/docker-compose.yml" config >/dev/null
    docker-compose -f "${COMPOSE_DIR}/docker-compose.offline.yml" config >/dev/null
    printf '  ✓ docker-compose validates main and offline configs\n'
else
    if ! grep -q "^services:" "${COMPOSE_DIR}/docker-compose.yml"; then
        printf '  ✗ main compose file missing services section\n'
        exit 1
    fi
    printf '  ✓ main compose file includes services section\n'
fi

if grep -Eq 'short-key|:-invalid' "${COMPOSE_DIR}/docker-compose.yml"; then
    printf '  ✗ main compose file contains insecure secret fallbacks\n'
    exit 1
fi
printf '  ✓ main compose file has no insecure secret fallbacks\n'

if ! grep -q '127.0.0.1:${POSTGRES_HOST_PORT:-5432}:5432' "${COMPOSE_DIR}/docker-compose.offline.yml"; then
    printf '  ✗ offline postgres should stay localhost-bound\n'
    exit 1
fi
printf '  ✓ offline postgres stays localhost-bound\n'
