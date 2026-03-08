#!/usr/bin/env bash
#
# Make Env Scope Test
#
# Purpose:
#   Verify Make no longer exports repo secrets globally and that Docker Compose
#   receives the env file explicitly when needed.
#
# Responsibilities:
#   - Assert generic Make recipes do not inherit `demo/.env` secrets.
#   - Assert Compose recipes include a concrete `--env-file` argument.
#
# Non-scope:
#   - Does not start real Docker services.
#   - Does not validate compose runtime behavior beyond CLI argument wiring.
#
# Invariants:
#   - Tests run inside an isolated fixture with a stub docker binary.
#   - The real repository Make variables file is used directly.
#
# Usage:
#   bash scripts/tests/make_env_scope_test.sh
#
# Exit Codes:
#   0 success
#   1 contract failure

set -euo pipefail

show_help() {
    cat <<'EOF'
Usage: make_env_scope_test.sh [OPTIONS]

Verify Make keeps secrets scoped and passes Compose env files explicitly.

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
VARIABLES_MK="${REPO_ROOT}/mk/variables.mk"

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_ROOT}"' EXIT

FIXTURE="${TMP_ROOT}/fixture"
STUB_BIN="${TMP_ROOT}/bin"
DOCKER_LOG="${TMP_ROOT}/docker.log"

mkdir -p "${FIXTURE}/demo" "${STUB_BIN}"
FIXTURE_REALPATH="$(cd "${FIXTURE}" && pwd -P)"

cat >"${FIXTURE}/demo/.env" <<'EOF'
LITELLM_MASTER_KEY=sk-secret-from-demo-env
LITELLM_SALT_KEY=sk-salt-from-demo-env
DATABASE_URL=postgresql://litellm:litellm@postgres:5432/litellm
ACP_DATABASE_MODE=external
EOF

cat >"${STUB_BIN}/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

log_file="${ACP_TEST_DOCKER_LOG:?missing ACP_TEST_DOCKER_LOG}"

if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    exit 0
fi

printf '%s\n' "$*" >>"${log_file}"
exit 0
EOF
chmod +x "${STUB_BIN}/docker"

INLINE_MAKEFILE="${TMP_ROOT}/Makefile"
cat >"${INLINE_MAKEFILE}" <<EOF
include ${VARIABLES_MK}

.PHONY: print-env compose-config print-db-mode
print-env:
	@env

compose-config:
	@\$(DOCKER_COMPOSE_PROJECT) -f \$(COMPOSE_FILE) config >/dev/null

print-db-mode:
	@printf '%s\n' "\$(DB_MODE)"
EOF

run_make() {
    PATH="${STUB_BIN}:${PATH}" \
        ACP_TEST_DOCKER_LOG="${DOCKER_LOG}" \
        make -s -C "${FIXTURE}" -f "${INLINE_MAKEFILE}" "$@"
}

echo "Make Env Scope Test"
echo "==================="

if run_make print-env | grep -q '^LITELLM_MASTER_KEY='; then
    echo "  ✗ generic recipe inherited LITELLM_MASTER_KEY"
    exit 1
fi
echo "  ✓ generic recipe does not inherit secret env"

if [[ "$(run_make print-db-mode)" != "external" ]]; then
    echo "  ✗ DB_MODE did not resolve from demo/.env"
    exit 1
fi
echo "  ✓ non-secret database mode still resolves from demo/.env"

run_make compose-config
expected_env_file="--env-file ${FIXTURE_REALPATH}/demo/.env"
if ! grep -Fq -- "${expected_env_file}" "${DOCKER_LOG}"; then
    echo "  ✗ compose command did not receive explicit env file"
    cat "${DOCKER_LOG}"
    exit 1
fi
echo "  ✓ compose command receives explicit env file"

echo "All make env scope tests passed."
