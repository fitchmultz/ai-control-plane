#!/usr/bin/env bash
#
# AI Control Plane - Claude Mode Switcher Tests
#
# Responsibility: Validate switch-claude-mode behavior across success
# and failure modes using an isolated HOME directory.
#
# Non-scope: Does NOT validate Claude Code installation or network connectivity.
#            Does NOT generate keys (onboarding/gateway responsibilities).
#
# Invariants:
#   - Never touches the caller's real ~/.claude directory
#   - Does not print secrets or token-like values
#   - Covers help output, status detection, switching, and error handling
#

set -euo pipefail

#------------------------------------------------------------------------------
# Paths
#------------------------------------------------------------------------------

if [ -n "${BASH_SOURCE[0]:-}" ]; then
    readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
    readonly SCRIPT_DIR="$(pwd)/scripts/tests"
fi

readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly SWITCH_SCRIPT="${PROJECT_ROOT}/scripts/libexec/switch_claude_mode_impl.sh"

# Source terminal UI library for COLOR_* and SYMBOL_* definitions
# shellcheck source=../lib/terminal_ui.sh
source "${PROJECT_ROOT}/scripts/lib/terminal_ui.sh"

# Source common.sh for verbose_log and _is_verbose helpers
# shellcheck source=../../local/scripts/demo_scenarios/lib/common.sh
source "${PROJECT_ROOT}/local/scripts/demo_scenarios/lib/common.sh"

#------------------------------------------------------------------------------
# Output helpers
#------------------------------------------------------------------------------

TESTS_PASSED=0
TESTS_FAILED=0
VERBOSE="${VERBOSE:-0}"

show_help() {
    cat <<'EOF'
Usage: switch_claude_mode_test.sh [OPTIONS]

Test suite for switch-claude-mode command behavior.

OPTIONS:
  --verbose, -v     Enable verbose output
  --help, -h        Show this help message

EXAMPLES:
  scripts/tests/switch_claude_mode_test.sh
  scripts/tests/switch_claude_mode_test.sh --verbose
EOF
}

pass() {
    echo -e "${SYMBOL_PASS} PASS: $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${SYMBOL_FAIL} FAIL: $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

section() {
    echo
    echo -e "${COLOR_BLUE}▶ $1${COLOR_RESET}"
    echo "────────────────────────────────────────"
}

#------------------------------------------------------------------------------
# Arg parsing
#------------------------------------------------------------------------------

while [[ $# -gt 0 ]]; do
    case "$1" in
    --verbose | -v)
        VERBOSE=1
        shift
        ;;
    --help | -h)
        show_help
        exit 0
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done

#------------------------------------------------------------------------------
# Test helpers
#------------------------------------------------------------------------------

make_temp_home() {
    local temp_home
    temp_home="$(mktemp -d)"
    mkdir -p "${temp_home}/.claude"
    echo "$temp_home"
}

write_local_settings() {
    local claude_dir="$1"
    cat >"${claude_dir}/settings.local.json" <<'EOF'
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:4000",
    "ANTHROPIC_AUTH_TOKEN": "<LITELLM_VIRTUAL_KEY>"
  }
}
EOF
}

write_max_settings() {
    local claude_dir="$1"
    cat >"${claude_dir}/settings.max.json" <<'EOF'
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:4000",
    "ANTHROPIC_CUSTOM_HEADERS": "x-litellm-api-key: Bearer <LITELLM_VIRTUAL_KEY>"
  }
}
EOF
}

assert_no_token_like_output() {
    local output="$1"
    if echo "$output" | grep -qiE 'sk-[A-Za-z0-9]{20,}'; then
        fail "Output contains token-like value (sk-...)"
        verbose_log "Output: $output"
        return 1
    fi
    return 0
}

#------------------------------------------------------------------------------
# Test cases
#------------------------------------------------------------------------------

test_script_exists_and_help() {
    section "Script Existence + Help"

    if [ -f "$SWITCH_SCRIPT" ]; then
        pass "switch_claude_mode_impl.sh exists"
    else
        fail "switch_claude_mode_impl.sh missing at $SWITCH_SCRIPT"
        return 0
    fi

    if [ -x "$SWITCH_SCRIPT" ]; then
        pass "switch_claude_mode_impl.sh is executable"
    else
        fail "switch_claude_mode_impl.sh is not executable"
    fi

    local output
    if output=$("$SWITCH_SCRIPT" --help 2>&1); then
        pass "--help exits 0"
    else
        fail "--help returned non-zero exit code"
        return 0
    fi

    if echo "$output" | grep -q "Usage:"; then
        pass "Help contains Usage"
    else
        fail "Help missing Usage"
    fi

    if echo "$output" | grep -q "Examples:"; then
        pass "Help contains Examples"
    else
        fail "Help missing Examples"
    fi
}

test_status_when_unconfigured() {
    section "Status (Unconfigured)"

    (
        local temp_home
        temp_home="$(make_temp_home)"
        trap 'rm -rf "$temp_home"' EXIT

        local output
        local exit_code=0
        output=$(HOME="$temp_home" "$SWITCH_SCRIPT" status 2>&1) || exit_code=$?

        if [ "$exit_code" -eq 0 ]; then
            pass "status exits 0"
        else
            fail "status should exit 0 (exit $exit_code)"
        fi

        if echo "$output" | grep -qi "no active configuration"; then
            pass "status reports no active configuration"
        else
            fail "status should report no active configuration"
            verbose_log "Output: $output"
        fi

        assert_no_token_like_output "$output" || true
    )
}

test_switching_and_backups() {
    section "Switching + Backups"

    (
        local temp_home
        temp_home="$(make_temp_home)"
        trap 'rm -rf "$temp_home"' EXIT

        write_local_settings "${temp_home}/.claude"
        write_max_settings "${temp_home}/.claude"

        # Switch to api-key
        local output
        if output=$(HOME="$temp_home" "$SWITCH_SCRIPT" api-key 2>&1); then
            pass "api-key switch exits 0"
        else
            fail "api-key switch returned non-zero"
            verbose_log "Output: $output"
            return 0
        fi

        if [ -f "${temp_home}/.claude/settings.json" ] && [ ! -L "${temp_home}/.claude/settings.json" ]; then
            pass "settings.json created as a real file"
        else
            fail "settings.json missing or is a symlink"
        fi

        if cmp -s "${temp_home}/.claude/settings.local.json" "${temp_home}/.claude/settings.json"; then
            pass "settings.json matches settings.local.json"
        else
            fail "settings.json does not match settings.local.json"
        fi

        # Trigger backup creation
        echo "old-config" >"${temp_home}/.claude/settings.json"
        if output=$(HOME="$temp_home" "$SWITCH_SCRIPT" api-key 2>&1); then
            pass "api-key switch overwrites with backup"
        else
            fail "api-key switch with existing settings.json should succeed"
            verbose_log "Output: $output"
        fi

        if ls "${temp_home}/.claude/settings.json.backup."* >/dev/null 2>&1; then
            pass "backup file created"
        else
            fail "backup file not created"
        fi

        # Switch to subscription
        if output=$(HOME="$temp_home" "$SWITCH_SCRIPT" subscription 2>&1); then
            pass "subscription switch exits 0"
        else
            fail "subscription switch returned non-zero"
            verbose_log "Output: $output"
            return 0
        fi

        if cmp -s "${temp_home}/.claude/settings.max.json" "${temp_home}/.claude/settings.json"; then
            pass "settings.json matches settings.max.json"
        else
            fail "settings.json does not match settings.max.json"
        fi

        if output=$(HOME="$temp_home" "$SWITCH_SCRIPT" status 2>&1); then
            if echo "$output" | grep -qi "MAX Subscription Mode"; then
                pass "status detects subscription mode"
            else
                fail "status should detect subscription mode"
                verbose_log "Output: $output"
            fi
        else
            fail "status should succeed after switching"
        fi

        assert_no_token_like_output "$output" || true
    )
}

test_missing_source_file_errors() {
    section "Missing Config Error Handling"

    (
        local temp_home
        temp_home="$(make_temp_home)"
        trap 'rm -rf "$temp_home"' EXIT

        # Only provide max settings; api-key should fail.
        write_max_settings "${temp_home}/.claude"

        local output
        local exit_code=0
        output=$(HOME="$temp_home" "$SWITCH_SCRIPT" api-key 2>&1) || exit_code=$?

        if [ "${exit_code:-0}" -ne 0 ]; then
            pass "api-key fails when settings.local.json missing"
        else
            fail "api-key should fail when settings.local.json missing"
        fi

        if echo "$output" | grep -q "claude --mode api-key --write-config"; then
            pass "error message includes onboarding guidance"
        else
            fail "error message missing onboarding guidance"
            verbose_log "Output: $output"
        fi

        assert_no_token_like_output "$output" || true
    )
}

#------------------------------------------------------------------------------
# Main
#------------------------------------------------------------------------------

echo "Claude Mode Switcher Test Suite"

test_script_exists_and_help
test_status_when_unconfigured
test_switching_and_backups
test_missing_source_file_errors

echo
echo "Results: passed=$TESTS_PASSED failed=$TESTS_FAILED"

if [ "$TESTS_FAILED" -eq 0 ]; then
    exit 0
fi

exit 1
