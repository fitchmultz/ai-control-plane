// cmd_db_backup.go - Database backup command implementation.
//
// Purpose:
//   - Own the typed database backup workflow.
//
// Responsibilities:
//   - Execute runtime prerequisite checks for backups.
//   - Create private compressed backup artifacts.
//   - Preserve test-only wrapper helpers until backup tests move to direct CLI entry.
//
// Scope:
//   - Database backup command flow and file output only.
//
// Usage:
//   - Invoked through `acpctl db backup`.
//
// Invariants/Assumptions:
//   - Backup artifacts remain private and deterministic.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

func runDBBackup(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(dbBackupOptions)
	customName := opts.BackupName

	out := output.New()
	logger := workflowLogger(runCtx, "db_backup")
	workflowStart(logger)

	repoRoot := runCtx.RepoRoot
	backupDir := resolveBackupDir(repoRoot)

	backupFile, err := resolveBackupOutputPath(backupDir, customName)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitUsage
	}

	if err := ensureBackupDir(backupDir); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to create backup directory: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if err := requireDBWorkflowPrereqs(repoRoot); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	services, err := openDBServices(repoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	defer services.Close()
	logger = workflowLogger(runCtx, "db_backup", "mode", services.Mode, "backup_file", backupFile)
	if services.Admin == nil {
		err := fmt.Errorf("backup and restore are not supported for external database mode")
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	printDBWorkflowHeader(runCtx.Stdout, out, "=== Database Backup ===", map[string]string{
		"Database":    services.Mode,
		"Backup file": backupFile,
	})

	if code := requireAccessibleDatabase(ctx, runCtx, logger, out, services.Runtime); code != exitcodes.ACPExitSuccess {
		return code
	}

	sql, err := services.Admin.Backup(ctx)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Backup failed: %v\n"), err)
		return exitcodes.ACPExitDomain
	}

	if err := writeCompressedBackupFile(backupFile, sql); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to write backup file: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		fileInfo = nil
	}

	details := map[string]any{
		"Location": backupFile,
	}
	if fileInfo != nil {
		details["Size"] = fmt.Sprintf("%d bytes", fileInfo.Size())
	}
	printDBWorkflowSuccess(runCtx.Stdout, out, "Backup completed successfully!", details)
	fmt.Fprintln(runCtx.Stdout)
	fmt.Fprintln(runCtx.Stdout, "Next step")
	printCommandNextStep(runCtx.Stdout, "Make", "make db-restore")
	printCommandNextStep(runCtx.Stdout, "CLI", "acpctl db restore "+backupFile)
	if fileInfo != nil {
		workflowComplete(logger, "bytes", fileInfo.Size())
	} else {
		workflowComplete(logger)
	}

	return exitcodes.ACPExitSuccess
}

func runDBBackupCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "backup"}, args, stdout, stderr)
}

func resolveBackupOutputPath(backupDir string, customName string) (string, error) {
	if textutil.IsBlank(customName) {
		timestamp := time.Now().Format("20060102-150405")
		return filepath.Join(backupDir, fmt.Sprintf("litellm-backup-%s.sql.gz", timestamp)), nil
	}

	name := textutil.Trim(customName)
	normalized := filepath.Clean(strings.ReplaceAll(name, "\\", "/"))
	if filepath.IsAbs(name) || filepath.IsAbs(normalized) {
		return "", fmt.Errorf("backup name must be a simple filename, not a path")
	}
	if name == "." || name == ".." || normalized != name || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("backup name must be a simple filename without path traversal")
	}

	target := filepath.Join(backupDir, name+".sql.gz")
	baseAbs, err := filepath.Abs(backupDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve backup directory: %w", err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("failed to resolve backup path: %w", err)
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return "", fmt.Errorf("failed to validate backup path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("backup name must stay within the backup directory")
	}

	return target, nil
}

func writeCompressedBackupFile(path string, sql string) error {
	if err := fsutil.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}

	var payload bytes.Buffer
	gzipWriter := gzip.NewWriter(&payload)
	if _, err := gzipWriter.Write([]byte(sql)); err != nil {
		_ = gzipWriter.Close()
		return fmt.Errorf("compress backup: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	if err := fsutil.AtomicWritePrivateFile(path, payload.Bytes()); err != nil {
		return fmt.Errorf("persist backup atomically: %w", err)
	}
	return nil
}
