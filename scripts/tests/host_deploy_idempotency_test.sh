#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Deploy Script - Idempotency Contract Test
#
# Purpose: Verify idempotency-related behavior for host_deploy_impl.sh using
#          deterministic stubs instead of real Ansible hosts.
#
# Responsibilities:
#   - Verify check mode passes --check/--diff to ansible-playbook
#   - Verify apply mode omits check flags
#   - Verify CLI extra vars are forwarded to ansible-playbook
#   - Verify two consecutive applies are no-op safe (simulated contract)
#   - Verify container fallback uses container-safe path mapping
#
# Non-scope:
#   - Does NOT test real host SSH connectivity
#   - Does NOT test actual Docker/Ansible convergence behavior
#
# Invariants:
#   - Tests are deterministic and isolated
#   - Uses PATH isolation and stub binaries
#
# Usage: host_deploy_idempotency_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_deploy_idempotency_test.sh
#   bash scripts/tests/host_deploy_idempotency_test.sh --help
#
# Exit Codes:
#   0   - All tests passed
#   1   - One or more tests failed
# =============================================================================

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

show_help() {
    cat <<'EOF'
Host Deploy Idempotency Contract Test

Purpose: Verify idempotency-related behavior for host_deploy_impl.sh

Usage: host_deploy_idempotency_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_deploy_idempotency_test.sh
  bash scripts/tests/host_deploy_idempotency_test.sh --help

Exit Codes:
  0   - All tests passed
  1   - One or more tests failed
EOF
}

# Parse arguments
if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# -----------------------------------------------------------------------------
# Test Configuration
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HOST_DEPLOY_SCRIPT="$REPO_ROOT/scripts/libexec/host_deploy_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0
TMP_DIR=""

# -----------------------------------------------------------------------------
# Setup and Teardown
# -----------------------------------------------------------------------------

setup() {
    TMP_DIR=$(mktemp -d)
    mkdir -p "$TMP_DIR/bin"
}

teardown() {
    if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
        rm -rf "$TMP_DIR"
    fi
}

# -----------------------------------------------------------------------------
# Test Helpers
# -----------------------------------------------------------------------------

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

create_inventory() {
    local path="$1"
    cat >"$path" <<'EOF'
all:
  children:
    gateway:
      hosts:
        localhost:
          ansible_host: 127.0.0.1
          ansible_connection: local
EOF
}

create_arg_echo_ansible_stub() {
    local stub_path="$1"
    cat >"$stub_path" <<'EOF'
#!/bin/bash
echo "STUB_ANSIBLE_PLAYBOOK_ARGS: $*"
exit 0
EOF
    chmod +x "$stub_path"
}

# -----------------------------------------------------------------------------
# Test Cases
# -----------------------------------------------------------------------------

test_check_mode_calls_ansible_check() {
    local inventory="$TMP_DIR/check_inventory.yml"
    local output
    local exit_code=0

    create_inventory "$inventory"
    create_arg_echo_ansible_stub "$TMP_DIR/bin/ansible-playbook"

    output=$(PATH="$TMP_DIR/bin:$PATH" "$HOST_DEPLOY_SCRIPT" check --inventory "$inventory" 2>&1) || exit_code=$?

    if echo "$output" | grep -q "STUB_ANSIBLE_PLAYBOOK_ARGS"; then
        pass "Check mode invokes ansible-playbook"
    else
        fail "Check mode should invoke ansible-playbook"
    fi

    if echo "$output" | grep -q -- "--check"; then
        pass "Check mode passes --check"
    else
        fail "Check mode should pass --check"
    fi

    if echo "$output" | grep -q -- "--diff"; then
        pass "Check mode passes --diff"
    else
        fail "Check mode should pass --diff"
    fi

    if [[ $exit_code -eq 0 ]]; then
        pass "Check mode exits 0 with stubbed ansible success"
    else
        fail "Check mode should exit 0 with stubbed ansible success, got $exit_code"
    fi
}

test_apply_mode_calls_ansible_without_check() {
    local inventory="$TMP_DIR/apply_inventory.yml"
    local output
    local exit_code=0

    create_inventory "$inventory"
    create_arg_echo_ansible_stub "$TMP_DIR/bin/ansible-playbook"

    output=$(PATH="$TMP_DIR/bin:$PATH" "$HOST_DEPLOY_SCRIPT" apply --inventory "$inventory" 2>&1) || exit_code=$?

    if echo "$output" | grep -q "STUB_ANSIBLE_PLAYBOOK_ARGS"; then
        pass "Apply mode invokes ansible-playbook"
    else
        fail "Apply mode should invoke ansible-playbook"
    fi

    if echo "$output" | grep -q -- "--check"; then
        fail "Apply mode should not pass --check"
    else
        pass "Apply mode omits --check"
    fi

    if [[ $exit_code -eq 0 ]]; then
        pass "Apply mode exits 0 with stubbed ansible success"
    else
        fail "Apply mode should exit 0 with stubbed ansible success, got $exit_code"
    fi
}

test_extra_vars_passed_to_ansible() {
    local inventory="$TMP_DIR/vars_inventory.yml"
    local output

    create_inventory "$inventory"
    create_arg_echo_ansible_stub "$TMP_DIR/bin/ansible-playbook"

    output=$(PATH="$TMP_DIR/bin:$PATH" "$HOST_DEPLOY_SCRIPT" check \
        --inventory "$inventory" \
        --repo-path /custom/path \
        --env-file /custom/.env \
        --tls-mode tls \
        --public-url https://example.com 2>&1)

    if echo "$output" | grep -q "acp_repo_path=/custom/path"; then
        pass "acp_repo_path forwarded to ansible"
    else
        fail "acp_repo_path should be forwarded to ansible"
    fi

    if echo "$output" | grep -q "acp_env_file=/custom/.env"; then
        pass "acp_env_file forwarded to ansible"
    else
        fail "acp_env_file should be forwarded to ansible"
    fi

    if echo "$output" | grep -q "acp_tls_mode=tls"; then
        pass "acp_tls_mode forwarded to ansible"
    else
        fail "acp_tls_mode should be forwarded to ansible"
    fi

    if echo "$output" | grep -q "acp_public_url=https://example.com"; then
        pass "acp_public_url forwarded to ansible"
    else
        fail "acp_public_url should be forwarded to ansible"
    fi
}

test_apply_idempotent_contract() {
    local inventory="$TMP_DIR/idempotent_inventory.yml"
    local output_first
    local output_second
    local exit_code=0
    local state_file="$TMP_DIR/idempotent_state"

    create_inventory "$inventory"

    cat >"$TMP_DIR/bin/ansible-playbook" <<'EOF'
#!/bin/bash
set -euo pipefail
state_file="${ACP_HOST_DEPLOY_TEST_STATE:?missing ACP_HOST_DEPLOY_TEST_STATE}"
if [[ ! -f "$state_file" ]]; then
    echo "SIMULATED_CONVERGENCE_RESULT=changed"
    touch "$state_file"
else
    echo "SIMULATED_CONVERGENCE_RESULT=noop"
fi
exit 0
EOF
    chmod +x "$TMP_DIR/bin/ansible-playbook"

    output_first=$(ACP_HOST_DEPLOY_TEST_STATE="$state_file" PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_DEPLOY_SCRIPT" apply --inventory "$inventory" 2>&1) || exit_code=$?
    if [[ $exit_code -eq 0 ]] && echo "$output_first" | grep -q "SIMULATED_CONVERGENCE_RESULT=changed"; then
        pass "First apply reports simulated change and exits 0"
    else
        fail "First apply should report change and exit 0"
    fi

    exit_code=0
    output_second=$(ACP_HOST_DEPLOY_TEST_STATE="$state_file" PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_DEPLOY_SCRIPT" apply --inventory "$inventory" 2>&1) || exit_code=$?
    if [[ $exit_code -eq 0 ]] && echo "$output_second" | grep -q "SIMULATED_CONVERGENCE_RESULT=noop"; then
        pass "Second apply is simulated no-op and exits 0"
    else
        fail "Second apply should be a no-op and exit 0"
    fi
}

test_container_fallback_uses_container_safe_paths() {
    local inventory="$TMP_DIR/container_inventory.yml"
    local output
    local exit_code=0

    create_inventory "$inventory"

    cat >"$TMP_DIR/bin/docker" <<'EOF'
#!/bin/bash
echo "DOCKER_STUB_ARGS: $*"
exit 0
EOF
    chmod +x "$TMP_DIR/bin/docker"
    rm -f "$TMP_DIR/bin/ansible-playbook"

    output=$(PATH="$TMP_DIR/bin:/usr/bin:/bin" bash "$HOST_DEPLOY_SCRIPT" check --inventory "$inventory" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "Container fallback path exits 0 with docker stub"
    else
        fail "Container fallback path should exit 0 with docker stub, got $exit_code"
    fi

    if echo "$output" | grep -q "/workspace/deploy/ansible/playbooks/gateway_host.yml"; then
        pass "Container fallback uses /workspace playbook path"
    else
        fail "Container fallback should use /workspace playbook path"
    fi

    if echo "$output" | grep -q -- "-i /inventory/$(basename "$inventory")"; then
        pass "Container fallback maps inventory path to /inventory"
    else
        fail "Container fallback should map inventory to /inventory mount"
    fi
}

test_script_uses_standard_exit_codes() {
    if grep -q "source.*scripts/lib/prereq.sh" "$HOST_DEPLOY_SCRIPT"; then
        pass "Script sources prereq.sh for standardized exit codes"
    else
        fail "Script should source prereq.sh"
    fi

    if grep -q "ACP_EXIT_SUCCESS\|ACP_EXIT_DOMAIN\|ACP_EXIT_PREREQ\|ACP_EXIT_RUNTIME\|ACP_EXIT_USAGE" "$HOST_DEPLOY_SCRIPT"; then
        pass "Script references standardized exit code constants"
    else
        fail "Script should reference standardized exit code constants"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Deploy Idempotency Contract Test ==="
echo ""

setup

test_check_mode_calls_ansible_check
test_apply_mode_calls_ansible_without_check
test_extra_vars_passed_to_ansible
test_apply_idempotent_contract
test_container_fallback_uses_container_safe_paths
test_script_uses_standard_exit_codes

teardown

echo ""
echo "=== Test Summary ==="
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All idempotency contract tests passed"
    exit 0
fi

echo ""
echo "✗ Some idempotency contract tests failed"
exit 1
