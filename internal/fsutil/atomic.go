// Package fsutil provides shared filesystem helpers for typed workflows.
//
// Purpose:
//
//	Centralize safe, reusable file-writing primitives for local workflow state
//	and generated artifacts.
//
// Responsibilities:
//   - Persist files atomically within a directory.
//   - Preserve caller-selected file permissions.
//   - Keep ad hoc temp-file-and-rename logic out of business workflows.
//
// Scope:
//   - Local filesystem write helpers only.
//
// Usage:
//   - Called by onboarding and release workflows when writing generated state.
//
// Invariants/Assumptions:
//   - Writes occur on the same filesystem as the destination path.
//   - Parent directories are created by callers when needed.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	cleanup := func() {
		_ = os.Remove(tempPath)
	}
	defer cleanup()

	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp file into place: %w", err)
	}
	return nil
}
