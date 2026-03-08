// atomic_modes_test.go - Permission-focused coverage for filesystem helpers.
//
// Purpose:
//   - Verify public/private helpers enforce explicit permission contracts.
//
// Responsibilities:
//   - Assert directory helpers set expected modes.
//   - Assert file helpers set expected modes.
//
// Scope:
//   - Permission and visibility contracts only.
//
// Usage:
//   - Run via `go test ./internal/fsutil`.
//
// Invariants/Assumptions:
//   - Tests run on filesystems that report owner/group/other mode bits.
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
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PrivateDirPerm {
		t.Fatalf("directory mode = %04o, want %04o", got, PrivateDirPerm)
	}
}

func TestEnsurePublicDirUsesPublicMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "public")
	if err := EnsurePublicDir(target); err != nil {
		t.Fatalf("EnsurePublicDir() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PublicDirPerm {
		t.Fatalf("directory mode = %04o, want %04o", got, PublicDirPerm)
	}
}

func TestAtomicWritePrivateFileUsesPrivateMode(t *testing.T) {
	target := filepath.Join(t.TempDir(), "secret.txt")
	if err := AtomicWritePrivateFile(target, []byte("secret\n")); err != nil {
		t.Fatalf("AtomicWritePrivateFile() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
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
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != PublicFilePerm {
		t.Fatalf("file mode = %04o, want %04o", got, PublicFilePerm)
	}
}
