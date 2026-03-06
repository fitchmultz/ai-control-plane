#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Preflight Script - Checks Test
#
# Purpose: Verify host_preflight_impl.sh check behavior and exit code contracts.
#
# Responsibilities:
#   - Validate deterministic pass/fail behavior using stubbed dependencies
#   - Assert standardized exit-code mapping (0/1/2/64)
#   - Verify known failure modes are detected with actionable outcomes
#
# Non-scope:
#   - Does NOT validate real host readiness
#   - Does NOT require Docker daemon or external network access
#
# Invariants:
#   - Stub commands take precedence over host commands
#   - Tests are deterministic and machine-independent
#   - Stubbed command outputs fully control check outcomes
#
# Usage: host_preflight_checks_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_preflight_checks_test.sh
#   bash scripts/tests/host_preflight_checks_test.sh --help
#
# Exit Codes:
#   0   - All tests passed
#   1   - One or more tests failed
# =============================================================================

show_help() {
    cat <<'HELP'
Host Preflight Checks Test

Purpose: Verify host_preflight_impl.sh check behavior and exit code contracts.

Usage: host_preflight_checks_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_preflight_checks_test.sh
  bash scripts/tests/host_preflight_checks_test.sh --help

Exit Codes:
  0   - All tests passed
  1   - One or more tests failed
HELP
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PREFLIGHT_SCRIPT="$REPO_ROOT/scripts/libexec/host_preflight_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0
TEST_TMP=""
STUB_BIN=""

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

setup_test_env() {
    TEST_TMP="$(mktemp -d)"
    STUB_BIN="$TEST_TMP/bin"
    mkdir -p "$STUB_BIN"
}

teardown_test_env() {
    if [[ -n "$TEST_TMP" && -d "$TEST_TMP" ]]; then
        rm -rf "$TEST_TMP"
    fi
}

reset_stubs() {
    rm -rf "$STUB_BIN"
    mkdir -p "$STUB_BIN"
}

create_stub_bin() {
    local cmd="$1"
    local script="$2"

    cat >"$STUB_BIN/$cmd" <<STUB
#!/usr/bin/env bash
set -euo pipefail
$script
STUB
    chmod +x "$STUB_BIN/$cmd"
}

create_common_success_stubs() {
    create_stub_bin sort 'cat'
    create_stub_bin awk '/usr/bin/awk "$@"'
    create_stub_bin id 'if [[ "${1:-}" == "-un" ]]; then echo tester; elif [[ "${1:-}" == "-nG" ]]; then echo "docker"; else exit 0; fi'
    create_stub_bin docker '
if [[ "${1:-}" == "compose" && "${2:-}" == "version" && "${3:-}" == "--short" ]]; then
    echo "v2.30.0"
    exit 0
fi
if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    echo "Docker Compose version v2.30.0"
    exit 0
fi
if [[ "${1:-}" == "version" && "${2:-}" == "--format" ]]; then
    echo "25.0.0"
    exit 0
fi
exit 0'
    create_stub_bin df 'echo "Filesystem 1K-blocks Used Available Use% Mounted on"; echo "/dev/sda1 104857600 1 20971520 1% /"'
    create_stub_bin timedatectl '
if [[ "${1:-}" == "show" ]]; then
    echo "yes"
    exit 0
fi
if [[ "${1:-}" == "status" ]]; then
    echo "NTP enabled: yes"
    exit 0
fi
exit 0'
    create_stub_bin sudo 'if [[ "${1:-}" == "-u" ]]; then shift 2; "$@"; else "$@"; fi'
    create_stub_bin ss 'echo "State Recv-Q Send-Q Local Address:Port Peer Address:Port"; exit 0'
}

run_preflight() {
    PATH="$STUB_BIN:/usr/bin:/bin" USER="tester" /usr/bin/bash "$PREFLIGHT_SCRIPT" "$@"
}

assert_exit_code() {
    local test_name="$1"
    local expected_rc="$2"
    shift 2

    local rc=0
    run_preflight "$@" >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == "$expected_rc" ]]; then
        pass "$test_name"
    else
        fail "$test_name (exit $rc, expected $expected_rc)"
    fi
}

assert_exit_code_any() {
    local test_name="$1"
    shift
    local rc=0
    run_preflight "$@" >/dev/null 2>&1 || rc=$?
    if [[ "$rc" == 0 || "$rc" == 1 ]]; then
        pass "$test_name"
    else
        fail "$test_name (exit $rc, expected 0 or 1)"
    fi
}

test_skip_all_checks_exits_0() {
    reset_stubs
    create_common_success_stubs
    assert_exit_code "skipping all checks exits 0" 0 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check service-user-permissions \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_unknown_check_exits_64() {
    reset_stubs
    create_common_success_stubs
    assert_exit_code "unknown check ID exits 64" 64 --skip-check unknown-check
}

test_disk_probe_error_exits_2() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin df 'echo "Filesystem 1K-blocks Used Available Use% Mounted on"'
    assert_exit_code "disk probe failure exits 2" 2 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check time-sync \
        --skip-check service-user-permissions \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_missing_time_tools_exits_2() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin timedatectl 'exit 1'
    create_stub_bin chronyc 'exit 1'
    create_stub_bin ntpq 'exit 1'
    assert_exit_code "missing time sync tools exits 2" 2 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check service-user-permissions \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_old_docker_version_exits_1() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin docker '
if [[ "${1:-}" == "version" && "${2:-}" == "--format" ]]; then
    echo "23.0.0"
    exit 0
fi
if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    echo "v2.30.0"
    exit 0
fi
exit 0'
    assert_exit_code "docker version below minimum exits 1" 1 \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check service-user-permissions \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_time_sync_disabled_exits_1() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin timedatectl '
if [[ "${1:-}" == "show" ]]; then
    echo "no"
    exit 0
fi
if [[ "${1:-}" == "status" ]]; then
    echo "NTP enabled: no"
    exit 0
fi
exit 0'
    assert_exit_code "time sync disabled exits 1" 1 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check service-user-permissions \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_missing_service_user_exits_1() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin id '
if [[ "${1:-}" == "missing-user" ]]; then
    exit 1
fi
if [[ "${1:-}" == "-un" ]]; then
    echo tester
    exit 0
fi
if [[ "${1:-}" == "-nG" ]]; then
    echo docker
    exit 0
fi
exit 0'
    assert_exit_code "missing service user exits 1" 1 \
        --service-user missing-user \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check required-open-ports \
        --skip-check required-blocked-ports
}

test_required_open_port_in_use_exits_1() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin ss 'echo "LISTEN 0 128 127.0.0.1:4000 0.0.0.0:*"'
    assert_exit_code "required open port in use exits 1" 1 \
        --require-port-open 4000 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check service-user-permissions \
        --skip-check required-blocked-ports
}

test_required_blocked_port_listening_exits_1() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin ss 'echo "LISTEN 0 128 127.0.0.1:5432 0.0.0.0:*"'
    assert_exit_code "required blocked port listening exits 1" 1 \
        --require-port-blocked 5432 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check service-user-permissions \
        --skip-check required-open-ports
}

test_valid_port_arguments_not_usage_error() {
    reset_stubs
    create_common_success_stubs
    create_stub_bin ss 'echo "State Recv-Q Send-Q Local Address:Port Peer Address:Port"'
    assert_exit_code_any "valid port arguments accepted" \
        --require-port-open 4000 \
        --require-port-blocked 5432 \
        --skip-check docker-version \
        --skip-check compose-version \
        --skip-check disk-headroom \
        --skip-check time-sync \
        --skip-check service-user-permissions
}

echo "Host Preflight Checks Test"
echo "========================="
echo ""

setup_test_env

test_skip_all_checks_exits_0
test_unknown_check_exits_64
test_disk_probe_error_exits_2
test_missing_time_tools_exits_2
test_old_docker_version_exits_1
test_time_sync_disabled_exits_1
test_missing_service_user_exits_1
test_required_open_port_in_use_exits_1
test_required_blocked_port_listening_exits_1
test_valid_port_arguments_not_usage_error

teardown_test_env

echo ""
echo "========================="
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi
exit 0
