#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Deploy Script - Prerequisites Test
#
# Purpose: Verify prerequisite checking behavior for host_deploy_impl.sh
#
# Responsibilities:
#   - Test missing inventory file exits 2 (prereq failure)
#   - Test invalid YAML inventory exits 2
#   - Verify meaningful error messages are displayed
#
# Non-scope:
#   - Does NOT test actual Ansible execution
#   - Does NOT test host connectivity
#
# Invariants:
#   - Tests are deterministic and isolated
#   - Tests use temporary files that are cleaned up
#
# Usage: host_deploy_prereqs_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_deploy_prereqs_test.sh
#   bash scripts/tests/host_deploy_prereqs_test.sh --help
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
Host Deploy Prerequisites Test

Purpose: Verify prerequisite checking behavior for host_deploy_impl.sh

Usage: host_deploy_prereqs_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_deploy_prereqs_test.sh
  bash scripts/tests/host_deploy_prereqs_test.sh --help

Test Coverage:
  - Missing inventory file exits 2 (prereq failure)
  - Invalid YAML inventory exits 2
  - Meaningful error messages are displayed

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

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary directory
TMP_DIR=""

# -----------------------------------------------------------------------------
# Setup and Teardown
# -----------------------------------------------------------------------------

setup() {
    TMP_DIR=$(mktemp -d)
    mkdir -p "$TMP_DIR/bin"
    mkdir -p "$TMP_DIR/minpath"

    # Keep only minimal utilities for targeted PATH isolation checks.
    ln -s /usr/bin/dirname "$TMP_DIR/minpath/dirname"

    # Deterministic python3 stub for inventory validation path.
    cat >"$TMP_DIR/bin/python3" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "-c" ]]; then
    # "import yaml" probe from host_deploy_impl.sh
    exit 0
fi

if [[ "${1:-}" == "-" ]]; then
    inventory_path="${2:-}"
    if grep -Fq "invalid: yaml: content: [" "$inventory_path"; then
        exit 1
    fi
    exit 0
fi

exit 0
EOF
    chmod +x "$TMP_DIR/bin/python3"
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

# -----------------------------------------------------------------------------
# Test Cases
# -----------------------------------------------------------------------------

test_missing_inventory_file_exits_2() {
    local exit_code=0
    local output

    output=$("$HOST_DEPLOY_SCRIPT" check --inventory /nonexistent/path/hosts.yml 2>&1) || exit_code=$?

    if [[ $exit_code -eq 2 ]]; then
        pass "Missing inventory file exits with code 2"
    else
        fail "Missing inventory file should exit 2, got $exit_code"
    fi

    if echo "$output" | grep -qi "inventory.*not found\|not found.*inventory"; then
        pass "Missing inventory produces meaningful error message"
    else
        fail "Missing inventory should produce 'not found' error message"
    fi
}

test_invalid_yaml_inventory_exits_2() {
    local exit_code=0
    local output
    local invalid_inventory="$TMP_DIR/invalid_inventory.yml"

    # Create invalid YAML
    cat >"$invalid_inventory" <<'EOF'
all:
  children:
    gateway:
      hosts:
        invalid: yaml: content: [
EOF

    output=$(PATH="$TMP_DIR/bin:$PATH" "$HOST_DEPLOY_SCRIPT" check --inventory "$invalid_inventory" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 2 ]]; then
        pass "Invalid YAML inventory exits with code 2"
    else
        fail "Invalid YAML inventory should exit 2, got $exit_code"
    fi
}

test_valid_yaml_inventory_passes_prereq() {
    local exit_code=0
    local output
    local valid_inventory="$TMP_DIR/valid_inventory.yml"
    local stub_dir="$TMP_DIR/bin"

    # Create valid YAML inventory (but ansible will fail due to no hosts)
    cat >"$valid_inventory" <<'EOF'
all:
  children:
    gateway:
      hosts:
        test-host:
          ansible_host: 127.0.0.1
EOF

    # Create a stub ansible-playbook so this test does not require Docker/network.
    cat >"$stub_dir/ansible-playbook" <<'EOF'
#!/usr/bin/env bash
echo "STUB_ANSIBLE_OK"
exit 0
EOF
    chmod +x "$stub_dir/ansible-playbook"

    output=$(PATH="$stub_dir:$PATH" "$HOST_DEPLOY_SCRIPT" check --inventory "$valid_inventory" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "Valid YAML inventory passes prereq checks and runs ansible"
    else
        fail "Valid YAML inventory should exit 0 with ansible stub, got $exit_code"
    fi

    if echo "$output" | grep -q "STUB_ANSIBLE_OK"; then
        pass "Valid YAML inventory reaches ansible execution path"
    else
        fail "Valid YAML inventory should invoke ansible-playbook stub"
    fi
}

test_missing_required_binaries_exits_2() {
    local exit_code=0
    local output
    local valid_inventory="$TMP_DIR/valid_inventory_no_runtime.yml"

    cat >"$valid_inventory" <<'EOF'
all:
  children:
    gateway:
      hosts:
        test-host:
          ansible_host: 127.0.0.1
EOF

    output=$(PATH="$TMP_DIR/minpath" /bin/bash "$HOST_DEPLOY_SCRIPT" check --inventory "$valid_inventory" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 2 ]]; then
        pass "Missing required binaries exits with code 2"
    else
        fail "Missing required binaries should exit 2, got $exit_code"
    fi

    if echo "$output" | grep -Eqi "command not found|required binary not found"; then
        pass "Missing binary failure reports actionable message"
    else
        fail "Missing binary failure should report actionable message"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Deploy Prerequisites Test ==="
echo ""

setup

# Run all tests
test_missing_inventory_file_exits_2
test_invalid_yaml_inventory_exits_2
test_valid_yaml_inventory_passes_prereq
test_missing_required_binaries_exits_2

teardown

# Summary
echo ""
echo "=== Test Summary ==="
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All prerequisites tests passed"
    exit 0
else
    echo ""
    echo "✗ Some prerequisites tests failed"
    exit 1
fi
