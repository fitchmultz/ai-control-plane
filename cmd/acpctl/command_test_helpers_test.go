// command_test_helpers_test.go - Shared command entry helpers for unit tests.
//
// Purpose:
//   - Keep command-level tests on the canonical CLI execution path instead of
//     file-local wrapper helpers.
//
// Responsibilities:
//   - Provide a small helper for invoking `run(...)` with explicit args and
//     captured output files.
//
// Scope:
//   - Test-only helpers for `cmd/acpctl`.
//
// Usage:
//   - Used by command tests that previously called local wrapper helpers.
//
// Invariants/Assumptions:
//   - Tests pass explicit command-path args rather than bypassing parsing.
package main

import (
	"context"
	"os"
	"testing"
)

func runTestCommand(t *testing.T, ctx context.Context, stdout *os.File, stderr *os.File, args ...string) int {
	t.Helper()
	return run(ctx, args, stdout, stderr)
}
