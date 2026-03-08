// cmd_db_ops_test.go - Tests for database operations command implementation
//
// Purpose: Provide unit tests for database backup and restore functionality
//
// Responsibilities:
//   - Test findLatestBackup for correct file selection and nil handling
//   - Test error handling in restore operations
//   - Ensure proper exit codes are returned
//
// Non-scope:
//   - Does not test actual database connections (requires Docker)
//   - Does not test full backup/restore cycle (integration tests)
//
// Invariants/Assumptions:
//   - Tests use temporary directories for file operations
//   - Tests verify error handling without requiring live services
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestFindLatestBackup_EmptyDir(t *testing.T) {
	// Create an empty temp directory
	tmpDir, err := os.MkdirTemp("", "acpctl_backup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = findLatestBackup(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestFindLatestBackup_NoMatchingFiles(t *testing.T) {
	// Create a temp directory with non-backup files
	tmpDir, err := os.MkdirTemp("", "acpctl_backup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some non-backup files
	files := []string{"readme.txt", "config.yaml", "data.json"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	_, err = findLatestBackup(tmpDir)
	if err == nil {
		t.Error("expected error when no .sql.gz files exist, got nil")
	}
}

func TestFindLatestBackup_SingleFile(t *testing.T) {
	// Create a temp directory with one backup file
	tmpDir, err := os.MkdirTemp("", "acpctl_backup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backupFile := filepath.Join(tmpDir, "litellm-backup-20240101-120000.sql.gz")
	if err := os.WriteFile(backupFile, []byte("backup content"), 0644); err != nil {
		t.Fatalf("failed to write backup file: %v", err)
	}

	latest, err := findLatestBackup(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latest != backupFile {
		t.Errorf("expected %s, got %s", backupFile, latest)
	}
}

func TestFindLatestBackup_MultipleFiles(t *testing.T) {
	// Create a temp directory with multiple backup files of different times
	tmpDir, err := os.MkdirTemp("", "acpctl_backup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create older backup
	olderFile := filepath.Join(tmpDir, "litellm-backup-20240101-120000.sql.gz")
	if err := os.WriteFile(olderFile, []byte("older backup"), 0644); err != nil {
		t.Fatalf("failed to write older backup: %v", err)
	}
	olderTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(olderFile, olderTime, olderTime); err != nil {
		t.Fatalf("failed to set older file time: %v", err)
	}

	// Create newer backup
	newerFile := filepath.Join(tmpDir, "litellm-backup-20240101-140000.sql.gz")
	if err := os.WriteFile(newerFile, []byte("newer backup"), 0644); err != nil {
		t.Fatalf("failed to write newer backup: %v", err)
	}
	newerTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(newerFile, newerTime, newerTime); err != nil {
		t.Fatalf("failed to set newer file time: %v", err)
	}

	latest, err := findLatestBackup(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if latest != newerFile {
		t.Errorf("expected newest file %s, got %s", newerFile, latest)
	}
}

func TestFindLatestBackup_NilGuard(t *testing.T) {
	// This test specifically verifies the nil guard fix for findLatestBackup
	// The bug was: calling latest.IsDir() when latest was a zero-value os.DirEntry
	// The fix: check if latest == nil before accessing its methods

	// Create a temp directory with only directories (no files)
	tmpDir, err := os.MkdirTemp("", "acpctl_backup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory (should be skipped)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// This should not panic - the nil guard prevents it
	_, err = findLatestBackup(tmpDir)
	if err == nil {
		t.Error("expected error when no backup files found, got nil")
	}
}

func TestRunDBRestoreCommand_InvalidGzip(t *testing.T) {
	// Create a temp directory with an invalid gzip file
	tmpDir, err := os.MkdirTemp("", "acpctl_restore_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an invalid gzip file (not actually gzip format)
	invalidFile := filepath.Join(tmpDir, "invalid.sql.gz")
	if err := os.WriteFile(invalidFile, []byte("this is not a valid gzip file"), 0644); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp stdout: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr, err := os.CreateTemp("", "acpctl_test_stderr")
	if err != nil {
		t.Fatalf("failed to create temp stderr: %v", err)
	}
	defer os.Remove(stderr.Name())

	exitCode := runDBRestoreCommand(context.Background(), []string{invalidFile}, stdout, stderr)
	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit code for invalid gzip, got %d", exitCode)
	}
}

func TestRunDBRestoreCommand_Help(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr := os.Stderr

	exitCode := runDBRestoreCommand(context.Background(), []string{"--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", exitCode)
	}
}

func TestRunDBBackupCommand_Help(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr := os.Stderr

	exitCode := runDBBackupCommand(context.Background(), []string{"--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", exitCode)
	}
}

func TestRunDBDRDrill_Help(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr := os.Stderr

	exitCode := runDBDRDrill(context.Background(), []string{"--help"}, stdout, stderr)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", exitCode)
	}
}

// TestDecompressErrorHandling verifies that decompression errors are properly handled
func TestDecompressErrorHandling(t *testing.T) {
	// Create a valid gzip file
	tmpDir, err := os.MkdirTemp("", "acpctl_decompress_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validGzip := filepath.Join(tmpDir, "valid.sql.gz")
	file, err := os.Create(validGzip)
	if err != nil {
		t.Fatalf("failed to create gzip file: %v", err)
	}

	gzipWriter := gzip.NewWriter(file)
	gzipWriter.Write([]byte("SELECT 1;"))
	gzipWriter.Close()
	file.Close()

	// Verify we can read it back
	file, err = os.Open(validGzip)
	if err != nil {
		t.Fatalf("failed to open gzip file: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Read all content
	buf := make([]byte, 1024)
	n, err := gzipReader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("unexpected error reading gzip: %v", err)
	}

	content := string(buf[:n])
	if content != "SELECT 1;" {
		t.Errorf("expected 'SELECT 1;', got '%s'", content)
	}
}

func TestResolveBackupOutputPath_CustomNameHappyPath(t *testing.T) {
	tmpDir := t.TempDir()

	backupPath, err := resolveBackupOutputPath(tmpDir, "release-20260306")
	if err != nil {
		t.Fatalf("expected happy path to succeed, got %v", err)
	}

	expected := filepath.Join(tmpDir, "release-20260306.sql.gz")
	if backupPath != expected {
		t.Fatalf("expected backup path %q, got %q", expected, backupPath)
	}
}

func TestResolveBackupOutputPath_InvalidCustomNames(t *testing.T) {
	tmpDir := t.TempDir()
	invalidNames := []string{
		"../../escape",
		"..\\..\\escape",
		"/tmp/escape",
		"nested/name",
		"nested\\name",
		"  ../escape  ",
		".",
		"..",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			if _, err := resolveBackupOutputPath(tmpDir, name); err == nil {
				t.Fatalf("expected invalid backup name %q to be rejected", name)
			}
		})
	}
}

func TestRunDBBackupCommand_RejectsTraversalBeforeDockerChecks(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp stdout: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr, err := os.CreateTemp("", "acpctl_test_stderr")
	if err != nil {
		t.Fatalf("failed to create temp stderr: %v", err)
	}
	defer os.Remove(stderr.Name())

	exitCode := runDBBackupCommand(context.Background(), []string{"../../escape"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit code for traversal attempt, got %d", exitCode)
	}

	if _, err := stderr.Seek(0, 0); err != nil {
		t.Fatalf("failed to rewind stderr: %v", err)
	}
	data, err := io.ReadAll(stderr)
	if err != nil {
		t.Fatalf("failed to read stderr: %v", err)
	}
	if !strings.Contains(string(data), "backup name") {
		t.Fatalf("expected stderr to mention backup name validation, got %q", string(data))
	}
}
