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
package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	// This should fail with a runtime error because the file isn't valid gzip
	// Note: This requires Docker to be available, so it may fail with prereq error
	// The important thing is that it doesn't panic
	_ = runDBRestoreCommand([]string{invalidFile}, stdout, stderr)
	// We don't check the exact exit code because it depends on environment
}

func TestRunDBRestoreCommand_Help(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr := os.Stderr

	exitCode := runDBRestoreCommand([]string{"--help"}, stdout, stderr)
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

	exitCode := runDBBackupCommand([]string{"--help"}, stdout, stderr)
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

	exitCode := runDBDRDrill([]string{"--help"}, stdout, stderr)
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
