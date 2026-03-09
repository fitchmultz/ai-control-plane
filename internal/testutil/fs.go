// Package testutil provides shared deterministic helpers for unit tests.
//
// Purpose:
//   - Centralize reusable filesystem helpers for test-only fixture setup.
//
// Responsibilities:
//   - Create parent directories for fixture files.
//   - Write deterministic file contents with caller-selected permissions.
//   - Keep repeated test setup out of package-local helper files.
//
// Scope:
//   - Generic filesystem helpers for unit tests only.
//
// Usage:
//   - Import from `_test.go` files and call `WriteFile`, `WriteFileMode`, or
//   - `WriteRepoFile` to provision fixture inputs.
//
// Invariants/Assumptions:
//   - Helpers fail tests immediately on setup errors.
//   - Helpers do not depend on package-specific domain types.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteFile writes a UTF-8 text fixture using standard test file permissions.
func WriteFile(t testing.TB, path string, content string) string {
	t.Helper()
	return WriteFileMode(t, path, content, 0o644)
}

// WriteFileMode writes a text fixture with explicit permissions.
func WriteFileMode(t testing.TB, path string, content string, mode os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// WriteRepoFile writes a repo-relative fixture underneath the provided root.
func WriteRepoFile(t testing.TB, repoRoot string, relativePath string, content string) string {
	t.Helper()
	return WriteFile(t, filepath.Join(repoRoot, relativePath), content)
}
