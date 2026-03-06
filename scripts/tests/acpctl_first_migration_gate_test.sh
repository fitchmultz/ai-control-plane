#!/bin/bash
set -euo pipefail

# ACPCTL-First Migration Gate Test
#
# Purpose:
#   Enforce acpctl-first policy for top-level operator shell entrypoints and
#   prevent introduction of newly large shell scripts during migration.
#
# Responsibilities:
#   - Fail when changed top-level scripts bypass acpctl delegation.
#   - Fail when a changed shell script is newly introduced above complexity
#     threshold or crosses the threshold from below.
#   - Provide deterministic policy feedback suitable for local CI use.
#
# Non-scope:
#   - Does NOT migrate legacy scripts automatically.
#   - Does NOT execute runtime workflows or Docker services.
#
# Invariants/Assumptions:
#   - `acpctl` is the canonical operator command surface for migrated flows.
#   - Exit code 0 means policy checks passed; 1 means policy violation.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

MAX_NEW_SCRIPT_LINES=250
MAX_WRAPPER_LINES=180

show_help() {
    cat <<'EOF'
Usage: acpctl_first_migration_gate_test.sh [OPTIONS]

Enforce acpctl-first migration policy for shell scripts.

Options:
  --max-new-lines N      Max lines for newly introduced shell scripts (default: 250)
  --max-wrapper-lines N  Max lines for top-level scripts in scripts/ (default: 180)
  --help, -h             Show this help message

Policy checks:
  1) Changed top-level scripts (scripts/*.sh) must delegate to acpctl.
  2) Changed shell scripts fail when they are:
     - new and exceed max-new-lines, or
     - crossing max-new-lines from below to above.

Examples:
  bash scripts/tests/acpctl_first_migration_gate_test.sh
  bash scripts/tests/acpctl_first_migration_gate_test.sh --max-new-lines 220
  make script-tests

Exit codes:
  0   Policy checks passed
  1   One or more policy checks failed
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
EOF
}

die_usage() {
    echo "Usage error: $*" >&2
    echo "Run with --help for usage." >&2
    exit 64
}

is_non_test_shell_path() {
    local path="$1"
    [[ "$path" == *.sh ]] || return 1
    [[ "$path" == */tests/* ]] && return 1
    [[ "$path" == */fixtures/* ]] && return 1
    [[ "$path" == */lib/* ]] && return 1
    [[ "$path" == scripts/libexec/* ]] && return 1
    [[ "$path" == deploy/helm/ai-control-plane/files/* ]] && return 1
    return 0
}

is_changed_operator_script() {
    local path="$1"
    [[ "$path" =~ ^scripts/[^/]+\.sh$ ]] || return 1
    return 0
}

has_acpctl_delegation() {
    local path="$1"
    awk '
        BEGIN { heredoc="" }
        {
            if (heredoc != "") {
                if ($0 == heredoc) {
                    heredoc=""
                }
                next
            }

            if (match($0, /<<[-]?[[:space:]]*["'\''"]?([A-Za-z_][A-Za-z0-9_]*)["'\''"]?/, m)) {
                heredoc=m[1]
                next
            }

            if ($0 ~ /^[[:space:]]*#/) {
                next
            }

            if ($0 ~ /(^|[[:space:];|&()])exec[[:space:]].*(scripts\/acpctl\.sh|acpctl)([[:space:]]|$)/) {
                found=1
                exit
            }

            if ($0 ~ /(^|[[:space:];|&()])scripts\/acpctl\.sh([[:space:]]|$)/) {
                found=1
                exit
            }

            if ($0 ~ /(^|[[:space:];|&()])acpctl([[:space:]]|$)/ && $0 ~ /\$@/) {
                found=1
                exit
            }
        }
        END { exit(found ? 0 : 1) }
    ' "$path"
}

collect_changed_files() {
    if ! command -v git >/dev/null 2>&1; then
        return 1
    fi
    if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        return 1
    fi

    declare -A changed=()

    while IFS= read -r -d '' file; do
        changed["$file"]=1
    done < <(git diff --name-only -z)

    while IFS= read -r -d '' file; do
        changed["$file"]=1
    done < <(git diff --cached --name-only -z)

    while IFS= read -r -d '' file; do
        changed["$file"]=1
    done < <(git ls-files --others --exclude-standard -z)

    if [[ "${#changed[@]}" -gt 0 ]]; then
        CHANGE_BASE_REF="HEAD"
        printf '%s\n' "${!changed[@]}" | sort
        return 0
    fi

    if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
        CHANGE_BASE_REF="HEAD~1"
        git diff --name-only HEAD~1..HEAD | sort
        return 0
    fi

    return 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
    --max-new-lines)
        shift
        [[ $# -gt 0 ]] || die_usage "Missing value for --max-new-lines"
        [[ "$1" =~ ^[0-9]+$ ]] || die_usage "Invalid integer for --max-new-lines: $1"
        MAX_NEW_SCRIPT_LINES="$1"
        ;;
    --max-wrapper-lines)
        shift
        [[ $# -gt 0 ]] || die_usage "Missing value for --max-wrapper-lines"
        [[ "$1" =~ ^[0-9]+$ ]] || die_usage "Invalid integer for --max-wrapper-lines: $1"
        MAX_WRAPPER_LINES="$1"
        ;;
    --help | -h)
        show_help
        exit 0
        ;;
    *)
        die_usage "Unknown option: $1"
        ;;
    esac
    shift
done

echo "ACPCTL-First Migration Gate"
echo "==========================="

declare -a CHANGED_FILES=()
CHANGE_BASE_REF="HEAD"
if ! mapfile -t CHANGED_FILES < <(collect_changed_files); then
    echo "No comparable change set detected; skipping policy checks."
    exit 0
fi

if [[ "${#CHANGED_FILES[@]}" -eq 0 ]]; then
    echo "No changed files detected; policy checks passed."
    exit 0
fi

declare -a FAILURES=()

for path in "${CHANGED_FILES[@]}"; do
    [[ -f "$path" ]] || continue
    is_changed_operator_script "$path" || continue

    if ! has_acpctl_delegation "$path"; then
        FAILURES+=("Top-level script must delegate to acpctl: $path")
        continue
    fi

    line_count="$(wc -l <"$path" | tr -d ' ')"
    if [[ "$line_count" -gt "$MAX_WRAPPER_LINES" ]]; then
        FAILURES+=("Top-level acpctl-delegating script exceeds wrapper limit ($MAX_WRAPPER_LINES lines): $path has $line_count")
    fi
done

for path in "${CHANGED_FILES[@]}"; do
    [[ -f "$path" ]] || continue
    is_non_test_shell_path "$path" || continue

    current_lines="$(wc -l <"$path" | tr -d ' ')"

    if ! git cat-file -e "$CHANGE_BASE_REF:$path" >/dev/null 2>&1; then
        if [[ "$current_lines" -gt "$MAX_NEW_SCRIPT_LINES" ]]; then
            FAILURES+=("New shell script exceeds complexity threshold ($MAX_NEW_SCRIPT_LINES lines): $path has $current_lines")
        fi
        continue
    fi

    previous_lines="$(git show "$CHANGE_BASE_REF:$path" | wc -l | tr -d ' ')"
    if [[ "$previous_lines" -le "$MAX_NEW_SCRIPT_LINES" && "$current_lines" -gt "$MAX_NEW_SCRIPT_LINES" ]]; then
        FAILURES+=("Shell script crossed complexity threshold ($MAX_NEW_SCRIPT_LINES lines): $path grew from $previous_lines to $current_lines")
    fi
done

if [[ "${#FAILURES[@]}" -gt 0 ]]; then
    echo ""
    echo "Policy violations:"
    printf '  - %s\n' "${FAILURES[@]}"
    echo ""
    echo "Remediation:"
    echo "  1) Move new operator behavior into acpctl (typed core)."
    echo "  2) Keep shell entrypoints thin wrappers."
    echo "  3) Move oversized wrappers into scripts/libexec or typed modules."
    exit 1
fi

echo "All acpctl-first migration policy checks passed."
exit 0
