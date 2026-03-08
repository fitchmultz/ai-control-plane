// atomic_test.go - Tests for shared atomic and permission-aware filesystem helpers.
//
// Purpose:
//
//	Verify atomic writes preserve the repository's explicit public/private mode
//	contracts for generated local files.
//
// Responsibilities:
//   - Assert private directory helpers create 0700 directories.
//   - Assert private file helpers create 0600 files.
//   - Assert public file helpers remain available for non-sensitive artifacts.
//
// Scope:
//   - Covers internal/fsutil helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/fsutil`.
//
// Invariants/Assumptions:
//   - Tests run on POSIX-like filesystems that report owner/group/other mode bits.
package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePrivateDirUsesPrivateMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "private")
	if err := EnsurePrivateDir(target); err != nil {
		t.Fatalf("EnsurePrivateDir() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateDirPerm {
		t.Fatalf("directory mode = %04o, want %04o", got, PrivateDirPerm)
	}
}

func TestAtomicWritePrivateFileUsesPrivateMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "secret.txt")
	if err := AtomicWritePrivateFile(target, []byte("secret\n")); err != nil {
		t.Fatalf("AtomicWritePrivateFile() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateFilePerm {
		t.Fatalf("file mode = %04o, want %04o", got, PrivateFilePerm)
	}
}

func TestAtomicWritePublicFileUsesPublicMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact.txt")
	if err := AtomicWritePublicFile(target, []byte("artifact\n")); err != nil {
		t.Fatalf("AtomicWritePublicFile() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PublicFilePerm {
		t.Fatalf("file mode = %04o, want %04o", got, PublicFilePerm)
	}
}
