// atomic_write_test.go - Behavior-focused coverage for atomic filesystem helpers.
//
// Purpose:
//   - Verify atomic writes preserve content and fail clearly when preconditions are unmet.
//
// Responsibilities:
//   - Cover overwrite behavior and missing-parent failures.
//
// Scope:
//   - Atomic write behavior only.
//
// Usage:
//   - Run via `go test ./internal/fsutil`.
//
// Invariants/Assumptions:
//   - Tests operate entirely within temp directories.
package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteFileOverwritesContentAtomically(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := AtomicWriteFile(target, []byte("new\n"), PublicFilePerm); err != nil {
		t.Fatalf("AtomicWriteFile() error = %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "new\n" {
		t.Fatalf("content = %q", content)
	}
}

func TestAtomicWriteFileRequiresExistingParentDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing", "artifact.txt")
	err := AtomicWriteFile(target, []byte("new\n"), PublicFilePerm)
	if err == nil || !strings.Contains(err.Error(), "create temp file") {
		t.Fatalf("expected create temp file error, got %v", err)
	}
}
