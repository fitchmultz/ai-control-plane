#!/usr/bin/env bash
set -euo pipefail

# Onboard Make Contract Test
#
# Purpose:
#   - Verify make-driven onboarding targets use the wizard-only surface.
#
# Responsibilities:
#   - Assert supported onboarding make targets remain present.
#   - Ensure make targets only preselect tools and do not pass removed flags.
#   - Keep the onboarding make contract aligned with the live CLI cutover.
#
# Scope:
#   - Onboarding makefile wiring validation only.
#
# Usage:
#   - bash scripts/tests/onboard_verify_mode_test.sh
#
# Invariants/Assumptions:
#   - The wizard is the only supported onboarding interface.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/tests/test_helpers.sh
source "${SCRIPT_DIR}/test_helpers.sh"

show_help() {
    cat <<'EOF'
Usage: onboard_verify_mode_test.sh [OPTIONS]

Validate onboarding makefile wiring after the wizard cutover.

Options:
  --help    Show this help message
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

PROJECT_ROOT="$(test_repo_root)"
mk_file="${PROJECT_ROOT}/mk/onboard.mk"
mk_contents="$(<"${mk_file}")"
onboarding_section="$(awk '
    /^\.PHONY: onboard$/ {capture=1}
    capture {print}
    /^\.PHONY: chatgpt-login$/ {exit}
' "${mk_file}")"

printf 'Onboard Make Contract Test\n'
printf '==========================\n'

for target in onboard: onboard-help: onboard-codex: onboard-claude: onboard-opencode: onboard-cursor: chatgpt-login: chatgpt-auth-copy:; do
    if ! grep -q "^${target}" "${mk_file}"; then
        printf '  ✗ missing make target %s\n' "${target}"
        exit 1
    fi
done
printf '  ✓ onboarding make targets remain present\n'

test_assert_contains "${onboarding_section}" '@$(ACPCTL_BIN) onboard' "generic onboarding target launches the wizard" || exit 1
test_assert_contains "${onboarding_section}" '@$(ACPCTL_BIN) onboard codex' "codex target only preselects the tool" || exit 1
test_assert_contains "${onboarding_section}" '@$(ACPCTL_BIN) onboard claude' "claude target only preselects the tool" || exit 1
test_assert_contains "${onboarding_section}" '@$(ACPCTL_BIN) onboard opencode' "opencode target only preselects the tool" || exit 1
test_assert_contains "${onboarding_section}" '@$(ACPCTL_BIN) onboard cursor' "cursor target only preselects the tool" || exit 1

for legacy_flag in --mode --verify --write-config --show-key --host --port --tls --budget --alias --model; do
    if grep -Fq -- "${legacy_flag}" <<<"${onboarding_section}"; then
        printf '  ✗ make onboarding targets must not pass legacy flag %s\n' "${legacy_flag}"
        exit 1
    fi
done
printf '  ✓ onboarding make targets use the wizard surface only\n'
