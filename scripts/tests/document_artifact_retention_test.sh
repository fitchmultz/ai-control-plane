#!/usr/bin/env bash
set -euo pipefail

# Document Artifact Retention Test
#
# Purpose:
#   Verify retention guard behavior for handoff evidence and release-bundle
#   document artifacts.
#
# Responsibilities:
#   - Validate --help contract for the retention implementation
#   - Validate --check fails when stale artifacts exist
#   - Validate --apply prunes stale artifacts deterministically
#   - Validate post-cleanup --check succeeds
#
# Non-scope:
#   - Does NOT mutate real repository artifact directories
#   - Does NOT run Docker or runtime health checks
#   - Does NOT verify release bundle contents/checksums
#
# Invariants:
#   - Tests run in a temporary fake repo tree
#   - Exit 0 means all assertions passed
#   - Exit 1 means one or more assertions failed

show_help() {
    cat <<'EOF'
Usage: document_artifact_retention_test.sh [--help]

Run contract tests for scripts/libexec/document_artifact_retention_impl.sh.

Examples:
  bash scripts/tests/document_artifact_retention_test.sh
  bash scripts/tests/document_artifact_retention_test.sh --help

Exit codes:
  0   All tests passed
  1   One or more tests failed
EOF
}

if [[ "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCRIPT_UNDER_TEST="$REPO_ROOT/scripts/libexec/document_artifact_retention_impl.sh"

TESTS_PASSED=0
TESTS_FAILED=0

pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++)) || true
}

fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++)) || true
}

assert_file_exists() {
    local path="$1"
    local message="$2"
    if [[ -f "$path" ]]; then
        pass "$message"
    else
        fail "$message"
    fi
}

assert_file_missing() {
    local path="$1"
    local message="$2"
    if [[ ! -e "$path" ]]; then
        pass "$message"
    else
        fail "$message"
    fi
}

echo "Document Artifact Retention Contract Tests"
echo ""

if [[ -x "$SCRIPT_UNDER_TEST" ]]; then
    pass "script under test is executable"
else
    fail "script under test must be executable"
fi

HELP_OUTPUT="$("$SCRIPT_UNDER_TEST" --help 2>&1 || true)"
if echo "$HELP_OUTPUT" | grep -q "keep-evidence"; then
    pass "--help documents --keep-evidence"
else
    fail "--help missing --keep-evidence"
fi
if echo "$HELP_OUTPUT" | grep -q "keep-bundles"; then
    pass "--help documents --keep-bundles"
else
    fail "--help missing --keep-bundles"
fi

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/acp-retention-test.XXXXXX")"
trap 'rm -rf "$TMP_ROOT"' EXIT

FAKE_REPO="$TMP_ROOT/repo"
mkdir -p "$FAKE_REPO/handoff-packet/evidence/20260210-120000"
mkdir -p "$FAKE_REPO/handoff-packet/evidence/20260211-120000"
mkdir -p "$FAKE_REPO/demo/logs/release-bundles"

echo "old evidence" >"$FAKE_REPO/handoff-packet/evidence/20260210-120000/old.log"
echo "new evidence" >"$FAKE_REPO/handoff-packet/evidence/20260211-120000/new.log"

OLD_BASE="$FAKE_REPO/demo/logs/release-bundles/ai-control-plane-deploy-old"
NEW_BASE="$FAKE_REPO/demo/logs/release-bundles/ai-control-plane-deploy-new"

echo "old bundle" >"${OLD_BASE}.tar.gz"
echo "old checksum" >"${OLD_BASE}.tar.gz.sha256"
echo "old sig" >"${OLD_BASE}.tar.gz.asc"

echo "new bundle" >"${NEW_BASE}.tar.gz"
echo "new checksum" >"${NEW_BASE}.tar.gz.sha256"
echo "new sig" >"${NEW_BASE}.tar.gz.asc"

# Force deterministic recency ordering
touch -t 202602101200 "${OLD_BASE}.tar.gz"
touch -t 202602111200 "${NEW_BASE}.tar.gz"

set +e
"$SCRIPT_UNDER_TEST" --check --repo-root "$FAKE_REPO" --keep-evidence 1 --keep-bundles 1 >/dev/null 2>&1
CHECK_EXIT=$?
set -e
if [[ "$CHECK_EXIT" -eq 1 ]]; then
    pass "--check fails when stale artifacts exist"
else
    fail "--check should fail with stale artifacts (exit=$CHECK_EXIT)"
fi

"$SCRIPT_UNDER_TEST" --apply --repo-root "$FAKE_REPO" --keep-evidence 1 --keep-bundles 1 >/dev/null

assert_file_missing "$FAKE_REPO/handoff-packet/evidence/20260210-120000/old.log" "old evidence file removed"
if [[ ! -d "$FAKE_REPO/handoff-packet/evidence/20260210-120000" ]]; then
    pass "old evidence directory removed"
else
    fail "old evidence directory should be removed"
fi

assert_file_exists "$FAKE_REPO/handoff-packet/evidence/20260211-120000/new.log" "new evidence retained"
assert_file_missing "${OLD_BASE}.tar.gz" "old bundle tar removed"
assert_file_missing "${OLD_BASE}.tar.gz.sha256" "old bundle checksum removed"
assert_file_missing "${OLD_BASE}.tar.gz.asc" "old bundle signature removed"
assert_file_exists "${NEW_BASE}.tar.gz" "new bundle tar retained"
assert_file_exists "${NEW_BASE}.tar.gz.sha256" "new bundle checksum retained"
assert_file_exists "${NEW_BASE}.tar.gz.asc" "new bundle signature retained"

set +e
"$SCRIPT_UNDER_TEST" --check --repo-root "$FAKE_REPO" --keep-evidence 1 --keep-bundles 1 >/dev/null 2>&1
POST_CHECK_EXIT=$?
set -e
if [[ "$POST_CHECK_EXIT" -eq 0 ]]; then
    pass "--check passes after cleanup"
else
    fail "--check should pass after cleanup (exit=$POST_CHECK_EXIT)"
fi

echo ""
echo "Summary"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
    exit 1
fi

exit 0
