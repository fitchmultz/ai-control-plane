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

const (
	PrivateDirPerm  os.FileMode = 0o700
	PrivateFilePerm os.FileMode = 0o600
	PublicDirPerm   os.FileMode = 0o755
	PublicFilePerm  os.FileMode = 0o644
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

func EnsureDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("chmod directory %s: %w", path, err)
	}
	return nil
}

func EnsurePrivateDir(path string) error {
	return EnsureDir(path, PrivateDirPerm)
}

func EnsurePublicDir(path string) error {
	return EnsureDir(path, PublicDirPerm)
}

func AtomicWritePrivateFile(path string, data []byte) error {
	return AtomicWriteFile(path, data, PrivateFilePerm)
}

func AtomicWritePublicFile(path string, data []byte) error {
	return AtomicWriteFile(path, data, PublicFilePerm)
}
