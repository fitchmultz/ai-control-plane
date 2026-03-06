#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Host Install Script - Dry Run and Idempotency Test
#
# Purpose: Verify dry-run behavior and idempotency for host_install_impl.sh using
#          deterministic stubs instead of real systemd operations.
#
# Responsibilities:
#   - Test that install --dry-run performs no writes and no systemctl mutations
#   - Test that render-unit outputs deterministic content with resolved paths/user
#   - Test that repeated install with unchanged template does not rewrite unit file
#   - Test that uninstall --dry-run is non-destructive
#
# Non-scope:
#   - Does NOT test real systemd operations
#   - Does NOT require root privileges
#
# Invariants:
#   - Tests are deterministic and isolated
#   - Uses PATH isolation and stub binaries
#   - Cleans up temp directories after tests
#
# Usage: host_install_dry_run_test.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Examples:
#   bash scripts/tests/host_install_dry_run_test.sh
#   bash scripts/tests/host_install_dry_run_test.sh --help
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
Host Install Dry Run Test

Purpose: Verify dry-run behavior and idempotency for host_install_impl.sh

Usage: host_install_dry_run_test.sh [OPTIONS]

Options:
  --help    Show this help message

Examples:
  bash scripts/tests/host_install_dry_run_test.sh
  bash scripts/tests/host_install_dry_run_test.sh --help

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
HOST_INSTALL_SCRIPT="$REPO_ROOT/scripts/libexec/host_install_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0
TMP_DIR=""
CALL_LOG=""

# -----------------------------------------------------------------------------
# Setup and Teardown
# -----------------------------------------------------------------------------

setup() {
    TMP_DIR=$(mktemp -d)
    CALL_LOG="$TMP_DIR/systemctl_calls.log"
    export CALL_LOG
    mkdir -p "$TMP_DIR/bin"
    mkdir -p "$TMP_DIR/unitdir"
    mkdir -p "$TMP_DIR/reporoot/deploy/systemd"
    mkdir -p "$TMP_DIR/reporoot/demo"

    # Create a stub systemctl command that logs calls
    cat >"$TMP_DIR/bin/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${CALL_LOG:?CALL_LOG must be set by host_install_dry_run_test.sh}"
echo "systemctl:$*" >> "$CALL_LOG"
exit 0
EOF
    chmod +x "$TMP_DIR/bin/systemctl"

    # Create a stub docker compose for detection
    cat >"$TMP_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "compose" && "${2:-}" == "version" ]]; then
    echo "Docker Compose version 2.20.0"
    exit 0
fi
exit 0
EOF
    chmod +x "$TMP_DIR/bin/docker"

    # Create a minimal template file (copy from real template if available)
    if [[ -f "$REPO_ROOT/deploy/systemd/ai-control-plane.service.tmpl" ]]; then
        cp "$REPO_ROOT/deploy/systemd/ai-control-plane.service.tmpl" "$TMP_DIR/reporoot/deploy/systemd/ai-control-plane.service.tmpl"
    else
        # Fallback template
        cat >"$TMP_DIR/reporoot/deploy/systemd/ai-control-plane.service.tmpl" <<'EOF'
[Unit]
Description=AI Control Plane Docker Compose Stack
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
User={{SERVICE_USER}}
Group={{SERVICE_GROUP}}
WorkingDirectory={{WORKING_DIR}}
EnvironmentFile=-{{ENV_FILE}}
ExecStart={{COMPOSE_BIN}} --project-directory {{WORKING_DIR}} -f {{COMPOSE_FILE}} up -d --remove-orphans
ExecStop={{COMPOSE_BIN}} --project-directory {{WORKING_DIR}} -f {{COMPOSE_FILE}} stop
ExecReload={{COMPOSE_BIN}} --project-directory {{WORKING_DIR}} -f {{COMPOSE_FILE}} restart
TimeoutStartSec=180
TimeoutStopSec=120

[Install]
WantedBy=multi-user.target
EOF
    fi

    # Create secrets file with secure permissions
    mkdir -p "$TMP_DIR/etc/ai-control-plane"
    touch "$TMP_DIR/etc/ai-control-plane/secrets.env"
    chmod 600 "$TMP_DIR/etc/ai-control-plane/secrets.env"

    # Create env file and compose file with secure permissions
    touch "$TMP_DIR/reporoot/demo/.env"
    chmod 600 "$TMP_DIR/reporoot/demo/.env"
    touch "$TMP_DIR/reporoot/demo/docker-compose.yml"
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

reset_call_log() {
    true >"$CALL_LOG"
}

get_call_count() {
    if [[ -f "$CALL_LOG" ]]; then
        wc -l <"$CALL_LOG" | tr -d ' '
    else
        echo "0"
    fi
}

# -----------------------------------------------------------------------------
# Test Cases
# -----------------------------------------------------------------------------

test_install_dry_run_performs_no_writes() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local unit_file="$unit_dir/ai-control-plane.service"

    # Ensure unit file doesn't exist initially
    rm -f "$unit_file"

    # Run install with --dry-run
    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"
    local exit_code=0
    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --dry-run \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "install --dry-run exits 0"
    else
        fail "install --dry-run should exit 0, got $exit_code"
    fi

    # Verify no unit file was created
    if [[ ! -f "$unit_file" ]]; then
        pass "install --dry-run does not create unit file"
    else
        fail "install --dry-run should not create unit file"
    fi

    # Verify no systemctl calls were made
    local call_count
    call_count=$(get_call_count)
    if [[ "$call_count" -eq 0 ]]; then
        pass "install --dry-run makes no systemctl calls"
    else
        fail "install --dry-run should make no systemctl calls, found $call_count"
    fi
}

test_install_dry_run_shows_what_would_happen() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local output

    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"
    output=$(CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --dry-run \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        2>&1) || true

    # Should mention what it would do
    if echo "$output" | grep -qi "dry-run\|would\|DRY"; then
        pass "install --dry-run indicates dry-run mode in output"
    else
        fail "install --dry-run should indicate dry-run mode in output"
    fi
}

test_render_unit_outputs_deterministic_content() {
    local repo_root="$TMP_DIR/reporoot"
    local output1

    output1=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --service-user testuser \
        --service-group testgroup \
        2>&1) || true

    # Verify deterministic output by running again and comparing
    local output2
    output2=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --service-user testuser \
        --service-group testgroup \
        2>&1) || true

    if [[ "$output1" == "$output2" ]]; then
        pass "render-unit produces deterministic output"
    else
        fail "render-unit output should be deterministic"
    fi

    # Check output contains expected sections
    if echo "$output1" | grep -q "\[Unit\]"; then
        pass "render-unit output contains [Unit] section"
    else
        fail "render-unit output missing [Unit] section"
    fi

    if echo "$output1" | grep -q "\[Service\]"; then
        pass "render-unit output contains [Service] section"
    else
        fail "render-unit output missing [Service] section"
    fi

    if echo "$output1" | grep -q "\[Install\]"; then
        pass "render-unit output contains [Install] section"
    else
        fail "render-unit output missing [Install] section"
    fi

    # Check user and group are resolved
    if echo "$output1" | grep -q "User=testuser"; then
        pass "render-unit resolves {{SERVICE_USER}} to testuser"
    else
        fail "render-unit should resolve {{SERVICE_USER}}"
    fi

    if echo "$output1" | grep -q "Group=testgroup"; then
        pass "render-unit resolves {{SERVICE_GROUP}} to testgroup"
    else
        fail "render-unit should resolve {{SERVICE_GROUP}}"
    fi

    # Check paths are resolved
    if echo "$output1" | grep -q "$repo_root"; then
        pass "render-unit resolves {{WORKING_DIR}} to repo root"
    else
        fail "render-unit should resolve {{WORKING_DIR}}"
    fi
}

test_install_idempotent_no_rewrite_when_unchanged() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local unit_file="$unit_dir/ai-control-plane.service"

    # First install (no dry-run, but with stubbed systemctl)
    local install_exit=0
    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$TMP_DIR/etc/ai-control-plane/secrets.env" --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        --service-user "$(whoami)" \
        2>/dev/null || install_exit=$?

    if [[ $install_exit -eq 0 ]]; then
        pass "First install exits 0"
    else
        fail "First install should exit 0, got $install_exit"
        return
    fi

    if [[ -f "$unit_file" ]]; then
        pass "First install creates unit file"
    else
        fail "First install should create unit file"
        return
    fi

    # Capture modification time
    local mtime_before
    mtime_before=$(stat -c %Y "$unit_file" 2>/dev/null || stat -f %m "$unit_file")

    # Reset call log
    reset_call_log

    # Second install with same parameters (should be idempotent)
    local second_install_exit=0
    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$TMP_DIR/etc/ai-control-plane/secrets.env" --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        --service-user "$(whoami)" \
        2>/dev/null || second_install_exit=$?

    if [[ $second_install_exit -eq 0 ]]; then
        pass "Second install exits 0"
    else
        fail "Second install should exit 0, got $second_install_exit"
        return
    fi

    # Check modification time hasn't changed
    local mtime_after
    mtime_after=$(stat -c %Y "$unit_file" 2>/dev/null || stat -f %m "$unit_file")

    if [[ "$mtime_before" -eq "$mtime_after" ]]; then
        pass "Second install with unchanged template does not rewrite unit file"
    else
        fail "Second install rewrote unit file unexpectedly"
    fi

    # daemon-reload should still be called on second install (to ensure systemd is aware)
    if grep -q "daemon-reload" "$CALL_LOG"; then
        pass "Second install still triggers daemon-reload"
    else
        fail "Second install should trigger daemon-reload"
    fi
}

test_install_honors_service_name_for_unit_path() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local custom_service="acp-custom"
    local expected_unit_file="$unit_dir/$custom_service.service"
    local exit_code=0

    rm -f "$expected_unit_file"

    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$TMP_DIR/etc/ai-control-plane/secrets.env" --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        --service-name "$custom_service" \
        --service-user "$(whoami)" \
        2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "install with --service-name exits 0"
    else
        fail "install with --service-name should exit 0, got $exit_code"
        return
    fi

    if [[ -f "$expected_unit_file" ]]; then
        pass "--service-name writes custom unit path"
    else
        fail "--service-name should create $expected_unit_file"
    fi
}

test_env_dry_run_is_honored() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local unit_file="$unit_dir/ai-control-plane.service"
    local exit_code=0

    rm -f "$unit_file"
    reset_call_log

    DRY_RUN=1 \
        CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$TMP_DIR/etc/ai-control-plane/secrets.env" --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "DRY_RUN=1 install exits 0"
    else
        fail "DRY_RUN=1 install should exit 0, got $exit_code"
    fi

    if [[ ! -f "$unit_file" ]]; then
        pass "DRY_RUN=1 prevents unit file writes"
    else
        fail "DRY_RUN=1 should prevent unit file writes"
    fi

    if [[ "$(get_call_count)" -eq 0 ]]; then
        pass "DRY_RUN=1 prevents systemctl mutation calls"
    else
        fail "DRY_RUN=1 should prevent systemctl mutation calls"
    fi
}

test_render_unit_works_without_compose_binary() {
    local repo_root="$TMP_DIR/reporoot"
    local no_compose_path="$TMP_DIR/no-compose-bin"
    local exit_code=0
    local output=""

    mkdir -p "$no_compose_path"

    output=$(PATH="$no_compose_path:/usr/bin:/bin" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --service-user testuser \
        --service-group testgroup \
        2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "render-unit works without compose binary installed"
    else
        fail "render-unit should not require compose binary (exit $exit_code)"
        return
    fi

    if echo "$output" | grep -q "ExecStart=docker compose"; then
        pass "render-unit falls back to docker compose placeholder"
    else
        fail "render-unit should use docker compose placeholder when compose binary is unavailable"
    fi
}

test_uninstall_dry_run_is_non_destructive() {
    local unit_dir="$TMP_DIR/unitdir"
    local repo_root="$TMP_DIR/reporoot"
    local unit_file="$unit_dir/ai-control-plane.service"

    # Create a unit file
    echo "[Unit]" >"$unit_file"
    echo "Description=Test" >>"$unit_file"

    reset_call_log

    # Run uninstall with --dry-run
    local exit_code=0
    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" uninstall \
        --dry-run \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        pass "uninstall --dry-run exits 0"
    else
        fail "uninstall --dry-run should exit 0, got $exit_code"
    fi

    # Verify unit file still exists
    if [[ -f "$unit_file" ]]; then
        pass "uninstall --dry-run does not remove unit file"
    else
        fail "uninstall --dry-run should not remove unit file"
    fi

    # Verify no systemctl calls were made
    local call_count
    call_count=$(get_call_count)
    if [[ "$call_count" -eq 0 ]]; then
        pass "uninstall --dry-run makes no systemctl calls"
    else
        fail "uninstall --dry-run should make no systemctl calls, found $call_count"
    fi
}

test_status_command_available() {
    local repo_root="$TMP_DIR/reporoot"
    local exit_code=0

    # Test that status command is recognized (doesn't exit 64 for usage error)
    CALL_LOG="$CALL_LOG" \
        PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" status \
        --repo-root "$repo_root" \
        2>/dev/null || exit_code=$?

    # Should NOT be 64 (usage error)
    if [[ $exit_code -ne 64 ]]; then
        pass "status command is recognized (exit code $exit_code)"
    else
        fail "status command should not exit 64 (usage error)"
    fi
}

test_start_stop_restart_commands_available() {
    local repo_root="$TMP_DIR/reporoot"
    local commands=("start" "stop" "restart")

    for cmd in "${commands[@]}"; do
        local exit_code=0
        CALL_LOG="$CALL_LOG" \
            PATH="$TMP_DIR/bin:$PATH" \
            "$HOST_INSTALL_SCRIPT" "$cmd" \
            --repo-root "$repo_root" \
            2>/dev/null || exit_code=$?

        # Should NOT be 64 (usage error) - may fail for other reasons (no systemd)
        if [[ $exit_code -ne 64 ]]; then
            pass "$cmd command is recognized (exit code $exit_code)"
        else
            fail "$cmd command should not exit 64 (usage error)"
        fi
    done
}

test_render_unit_includes_execstartpre() {
    local repo_root="$TMP_DIR/reporoot"
    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"
    local output

    # Create a proper test template that includes the new placeholders
    cat >"$repo_root/deploy/systemd/ai-control-plane.service.tmpl" <<'EOF'
[Unit]
Description=AI Control Plane

[Service]
Type=oneshot
User={{SERVICE_USER}}
Group={{SERVICE_GROUP}}
WorkingDirectory={{WORKING_DIR}}
EnvironmentFile={{ENV_FILE}}
ExecStartPre={{WORKING_DIR}}/scripts/libexec/prepare_secrets_env_impl.sh --secrets-file {{ENV_FILE}} --compose-env-file {{COMPOSE_ENV_FILE}} {{SECRETS_FETCH_HOOK_ARG}} --service-user {{SERVICE_USER}}
ExecStart={{COMPOSE_BIN}} up -d

[Install]
WantedBy=multi-user.target
EOF

    output=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --service-user testuser \
        --service-group testgroup \
        2>&1) || true

    # Check that ExecStartPre includes canonical prepare_secrets_env implementation invocation
    if echo "$output" | grep -q "scripts/libexec/prepare_secrets_env_impl.sh"; then
        pass "render-unit includes ExecStartPre with canonical prepare_secrets_env impl command"
    else
        fail "render-unit missing ExecStartPre with canonical prepare_secrets_env impl command"
    fi

    # Check that secrets file path is resolved
    if echo "$output" | grep -q "$secrets_file"; then
        pass "render-unit resolves {{ENV_FILE}} to secrets file path"
    else
        fail "render-unit should resolve {{ENV_FILE}} to secrets file path"
    fi

    # Check that compose env file is resolved
    if echo "$output" | grep -q "demo/.env"; then
        pass "render-unit resolves {{COMPOSE_ENV_FILE}} to compose env file"
    else
        fail "render-unit should resolve {{COMPOSE_ENV_FILE}}"
    fi
}

test_render_unit_with_fetch_hook_placeholder() {
    local repo_root="$TMP_DIR/reporoot"
    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"
    local hook_script="$repo_root/scripts/fetch-secrets.sh"
    local output

    # Create fetch hook file
    mkdir -p "$repo_root/scripts"
    echo '#!/bin/bash' >"$hook_script"
    chmod +x "$hook_script"

    cat >"$repo_root/deploy/systemd/ai-control-plane.service.tmpl" <<'EOF'
[Unit]
Description=AI Control Plane

[Service]
ExecStartPre={{WORKING_DIR}}/scripts/libexec/prepare_secrets_env_impl.sh --secrets-file {{ENV_FILE}} --compose-env-file {{COMPOSE_ENV_FILE}} {{SECRETS_FETCH_HOOK_ARG}} --service-user {{SERVICE_USER}}
ExecStart={{COMPOSE_BIN}} up -d
EOF

    output=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --secrets-fetch-hook "$hook_script" \
        2>&1) || true

    # Check that fetch hook is included in the ExecStartPre
    if echo "$output" | grep -q -- "--fetch-hook"; then
        if echo "$output" | grep -q "fetch-secrets.sh"; then
            pass "render-unit includes fetch hook in ExecStartPre"
        else
            fail "render-unit missing fetch hook script name in ExecStartPre"
        fi
    else
        fail "render-unit missing --fetch-hook in ExecStartPre"
    fi
}

test_render_unit_escapes_fetch_hook_with_spaces() {
    local repo_root="$TMP_DIR/reporoot"
    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"
    local hook_script="$repo_root/scripts/fetch secrets.sh"
    local output

    mkdir -p "$repo_root/scripts"
    echo '#!/bin/bash' >"$hook_script"
    chmod +x "$hook_script"

    cat >"$repo_root/deploy/systemd/ai-control-plane.service.tmpl" <<'EOF'
[Unit]
Description=AI Control Plane

[Service]
ExecStartPre={{WORKING_DIR}}/scripts/libexec/prepare_secrets_env_impl.sh --secrets-file {{ENV_FILE}} --compose-env-file {{COMPOSE_ENV_FILE}} {{SECRETS_FETCH_HOOK_ARG}} --service-user {{SERVICE_USER}}
ExecStart={{COMPOSE_BIN}} up -d
EOF

    output=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" render-unit \
        --repo-root "$repo_root" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --secrets-fetch-hook "$hook_script" \
        2>&1) || true

    if echo "$output" | grep -q -- "--fetch-hook"; then
        if echo "$output" | grep -q "fetch\\\\ secrets.sh"; then
            pass "render-unit escapes fetch hook path containing spaces"
        else
            fail "render-unit should escape fetch hook path containing spaces"
        fi
    else
        fail "render-unit missing --fetch-hook option for spaced path"
    fi
}

test_install_fails_when_service_user_cannot_read_secrets() {
    local repo_root="$TMP_DIR/reporoot"
    local unit_dir="$TMP_DIR/unitdir"
    local secrets_file="$TMP_DIR/etc/ai-control-plane/secrets.env"

    if ! id nobody >/dev/null 2>&1; then
        pass "Service-user readability failure test skipped (user 'nobody' not available)"
        return
    fi

    chmod 600 "$secrets_file"

    local exit_code=0
    local output
    output=$(PATH="$TMP_DIR/bin:$PATH" \
        "$HOST_INSTALL_SCRIPT" install \
        --dry-run \
        --repo-root "$repo_root" \
        --unit-dir "$unit_dir" \
        --env-file "$secrets_file" \
        --compose-env-file "$repo_root/demo/.env" \
        --compose-file "$repo_root/demo/docker-compose.yml" \
        --service-user nobody \
        2>&1) || exit_code=$?

    if [[ $exit_code -eq 2 ]] && echo "$output" | grep -q "not readable by service user"; then
        pass "install fails when configured service user cannot read secrets file"
    else
        fail "install should fail with unreadable secrets for non-owner service user"
    fi
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------

echo "=== Host Install Dry Run Test ==="
echo ""

setup

test_install_dry_run_performs_no_writes
test_install_dry_run_shows_what_would_happen
test_render_unit_outputs_deterministic_content
test_render_unit_works_without_compose_binary
test_install_idempotent_no_rewrite_when_unchanged
test_install_honors_service_name_for_unit_path
test_env_dry_run_is_honored
test_uninstall_dry_run_is_non_destructive
test_status_command_available
test_start_stop_restart_commands_available
test_render_unit_includes_execstartpre
test_render_unit_with_fetch_hook_placeholder
test_render_unit_escapes_fetch_hook_with_spaces
test_install_fails_when_service_user_cannot_read_secrets

teardown

echo ""
echo "=== Test Summary ==="
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo ""
    echo "✓ All dry run tests passed"
    exit 0
fi

echo ""
echo "✗ Some dry run tests failed"
exit 1
