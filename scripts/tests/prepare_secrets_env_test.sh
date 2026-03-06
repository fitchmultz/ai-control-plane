#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Prepare Secrets Env Script - Regression Test Suite
#
# Purpose: Validate secrets preparation contract behavior for host deployments.
#
# Responsibilities:
#   - Verify fetch-hook failures are surfaced with redacted output only
#   - Verify successful prepare flow syncs canonical secrets into compose env path
#   - Verify symlinked destination paths are rejected
#
# Non-scope:
#   - Does NOT validate deployment key/value semantics (handled elsewhere)
#   - Does NOT exercise systemd unit execution
#
# Invariants:
#   - Tests must never print raw secret fixture values
#   - Tests run in isolated temp directories only
#
# Usage: prepare_secrets_env_test.sh [--help]
#
# Exit Codes:
#   0   - All tests passed
#   1   - One or more tests failed
# =============================================================================

show_help() {
    cat <<'HELP'
Usage: prepare_secrets_env_test.sh [OPTIONS]

Regression tests for prepare-secrets environment implementation flow.

Options:
  --help, -h    Show this help message

Examples:
  bash scripts/tests/prepare_secrets_env_test.sh

Exit Codes:
  0   - All tests passed
  1   - One or more tests failed
HELP
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PREPARE_SCRIPT="$REPO_ROOT/scripts/libexec/prepare_secrets_env_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0
TMP_ROOT=""

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

setup() {
    TMP_ROOT="$(mktemp -d)"
}

teardown() {
    if [[ -n "$TMP_ROOT" && -d "$TMP_ROOT" ]]; then
        rm -rf "$TMP_ROOT"
    fi
}

create_base_secrets_file() {
    local path="$1"
    cat >"$path" <<'ENVEOF'
LITELLM_MASTER_KEY=skmasterkey1234567890abcdefghijklmnop
LITELLM_SALT_KEY=sksaltkey1234567890abcdefghijklmnopqr
DATABASE_URL=postgresql://litellm:password@postgres:5432/litellm
POSTGRES_USER=litellm
POSTGRES_PASSWORD=password
POSTGRES_DB=litellm
ENVEOF
    chmod 600 "$path"
}

test_fetch_hook_failure_is_redacted() {
    local case_dir="$TMP_ROOT/case_hook_failure"
    mkdir -p "$case_dir"

    local secrets_file="$case_dir/secrets.env"
    local compose_env_file="$case_dir/demo/.env"
    local hook="$case_dir/failing_hook.sh"
    local raw_secret="demo-secret-token-1234567890abcdef1234567890"

    create_base_secrets_file "$secrets_file"

    cat >"$hook" <<EOFHOOK
#!/usr/bin/env bash
set -euo pipefail
echo "Authorization: Bearer $raw_secret"
echo "Bearer $raw_secret" >&2
exit 42
EOFHOOK
    chmod +x "$hook"

    local output
    local exit_code=0
    output="$($PREPARE_SCRIPT --secrets-file "$secrets_file" --compose-env-file "$compose_env_file" --fetch-hook "$hook" 2>&1)" || exit_code=$?

    if [[ $exit_code -ne 3 ]]; then
        fail "Fetch-hook failure should exit 3, got $exit_code"
        return
    fi
    if ! echo "$output" | grep -q "Fetch hook failed (exit 42)"; then
        fail "Fetch-hook failure output should preserve real hook exit code"
        return
    fi
    if echo "$output" | grep -q "$raw_secret"; then
        fail "Fetch-hook failure output leaked raw secret"
        return
    fi
    if ! echo "$output" | grep -q "\[REDACTED\]"; then
        fail "Fetch-hook failure output should include redacted marker"
        return
    fi

    pass "Fetch-hook failure path redacts output and preserves exit code"
}

test_successful_prepare_syncs_compose_env() {
    local case_dir="$TMP_ROOT/case_success"
    mkdir -p "$case_dir/demo"

    local secrets_file="$case_dir/secrets.env"
    local compose_env_file="$case_dir/demo/.env"

    create_base_secrets_file "$secrets_file"

    local exit_code=0
    "$PREPARE_SCRIPT" \
        --secrets-file "$secrets_file" \
        --compose-env-file "$compose_env_file" \
        --service-user "$(id -un)" \
        >/dev/null 2>&1 || exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        fail "Successful prepare flow should exit 0, got $exit_code"
        return
    fi

    if [[ ! -f "$compose_env_file" ]]; then
        fail "Compose env file not created"
        return
    fi

    local mode
    mode="$(stat -c '%a' "$compose_env_file" 2>/dev/null || stat -f '%Lp' "$compose_env_file")"
    if [[ "$mode" != "600" ]]; then
        fail "Compose env file mode should be 600, got $mode"
        return
    fi

    if ! cmp -s "$secrets_file" "$compose_env_file"; then
        fail "Compose env file should match canonical secrets file"
        return
    fi

    pass "Successful prepare flow syncs canonical secrets to compose env path"
}

test_symlink_destination_is_rejected() {
    local case_dir="$TMP_ROOT/case_symlink"
    mkdir -p "$case_dir/real"

    local secrets_file="$case_dir/secrets.env"
    local symlink_dest="$case_dir/demo.env"
    local real_dest="$case_dir/real/target.env"

    create_base_secrets_file "$secrets_file"
    touch "$real_dest"
    ln -s "$real_dest" "$symlink_dest"

    local output
    local exit_code=0
    output="$($PREPARE_SCRIPT --secrets-file "$secrets_file" --compose-env-file "$symlink_dest" 2>&1)" || exit_code=$?

    if [[ $exit_code -ne 3 ]]; then
        fail "Symlink destination should exit 3, got $exit_code"
        return
    fi

    if ! echo "$output" | grep -qi "must not contain symlinks"; then
        fail "Symlink destination failure should mention symlink restriction"
        return
    fi

    pass "Symlink destination path is rejected"
}

main() {
    echo "=== prepare_secrets_env_impl.sh regression tests ==="

    setup
    trap teardown EXIT

    test_fetch_hook_failure_is_redacted
    test_successful_prepare_syncs_compose_env
    test_symlink_destination_is_rejected

    echo ""
    echo "=== Test Summary ==="
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"

    if [[ $TESTS_FAILED -ne 0 ]]; then
        exit 1
    fi

    exit 0
}

main
