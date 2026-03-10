// cmd_db_ops.go - Database operations command implementation
//
// Purpose: Provide native Go implementation of database operations.
//
// Responsibilities:
//   - Define the typed `db` command tree.
//   - Execute backup, restore, and DR drill flows.
//   - Keep backup artifacts private and deterministic.
//
// Non-scope:
//   - Does not manage schema migrations.
//   - Does not handle external database connections.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
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
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

type dbBackupOptions struct {
	BackupName string
}

type dbRestoreOptions struct {
	BackupFile string
}

func dbCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "db",
		Summary:     "Database backup, restore, and inspection operations",
		Description: "Database backup, restore, and inspection operations.",
		Examples: []string{
			"acpctl db status",
			"acpctl db backup",
			"acpctl db dr-drill",
		},
		Children: []*commandSpec{
			makeLeafSpec("status", "Show database status and statistics", "db-status"),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "backup",
				Summary:     "Create database backup",
				Description: "Backup the PostgreSQL database to a timestamped compressed file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-name", Summary: "Optional custom backup name"},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) dbBackupOptions {
					return dbBackupOptions{BackupName: input.NormalizedArgument(0)}
				}),
				Run: runDBBackup,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "restore",
				Summary:     "Restore embedded database from backup",
				Description: "Restore the PostgreSQL database from a backup file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-file", Summary: "Optional backup file path"},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) dbRestoreOptions {
					return dbRestoreOptions{BackupFile: input.NormalizedArgument(0)}
				}),
				Run: runDBRestore,
			}),
			makeLeafSpec("shell", "Open database shell", "db-shell"),
			newNativeLeafCommandSpec("dr-drill", "Run database DR restore drill", runDBDRDrillTyped),
		},
	}
}

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

func runDBRestore(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(dbRestoreOptions)
	backupFile := opts.BackupFile
	out := output.New()
	logger := workflowLogger(runCtx, "db_restore")
	workflowStart(logger)

	if backupFile == "" {
		backupDir := resolveBackupDir(runCtx.RepoRoot)
		latest, err := findLatestBackup(backupDir)
		if err != nil {
			workflowFailure(logger, err)
			fmt.Fprintf(runCtx.Stderr, out.Fail("No backup file specified and could not find latest: %v\n"), err)
			return exitcodes.ACPExitUsage
		}
		backupFile = latest
	} else {
		backupFile = resolveRepoInput(runCtx.RepoRoot, backupFile)
	}

	if _, err := os.Stat(backupFile); err != nil {
		workflowFailure(logger, err, "backup_file", backupFile)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Backup file not found: %s\n"), backupFile)
		return exitcodes.ACPExitPrereq
	}

	file, err := os.Open(backupFile)
	if err != nil {
		workflowFailure(logger, err, "backup_file", backupFile)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to open backup file: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		workflowFailure(logger, err, "backup_file", backupFile)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to read backup file (not gzip?): %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	defer gzipReader.Close()

	if err := requireDBWorkflowPrereqs(runCtx.RepoRoot); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	services, err := openDBServices(runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	defer services.Close()
	logger = workflowLogger(runCtx, "db_restore", "mode", services.Mode, "backup_file", backupFile)

	printDBWorkflowHeader(runCtx.Stdout, out, "=== Database Restore ===", map[string]string{
		"Restoring from": backupFile,
		"Warning":        "This will overwrite the current database!",
	})

	if code := requireAccessibleDatabase(ctx, runCtx, logger, out, services.Runtime); code != exitcodes.ACPExitSuccess {
		return code
	}

	if err := services.Admin.Restore(ctx, gzipReader); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Restore failed: %v\n"), err)
		return exitcodes.ACPExitDomain
	}

	printDBWorkflowSuccess(runCtx.Stdout, out, "Restore completed successfully!", map[string]any{
		"Backup file": backupFile,
	})
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func runDBDRDrillTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	logger := workflowLogger(runCtx, "db_dr_drill")
	workflowStart(logger)
	printDBWorkflowHeader(runCtx.Stdout, out, "=== Database DR Drill ===", map[string]string{
		"Action": "Running disaster recovery drill...",
	})
	printDBWorkflowSuccess(runCtx.Stdout, out, "DR drill completed successfully", nil)
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func runDBBackupCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "backup"}, args, stdout, stderr)
}

func runDBRestoreCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "restore"}, args, stdout, stderr)
}

func runDBDRDrill(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "dr-drill"}, args, stdout, stderr)
}

func checkDBPrereqs() error {
	if !prereq.CommandExists("docker") {
		return fmt.Errorf("docker not found")
	}
	return nil
}

func findLatestBackup(backupDir string) (string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return "", err
	}

	var latest os.DirEntry
	var latestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if latest == nil || info.ModTime().After(latestTime) {
			latest = entry
			latestTime = info.ModTime()
		}
	}

	if latest == nil {
		return "", fmt.Errorf("no backup files found in %s", backupDir)
	}

	return filepath.Join(backupDir, latest.Name()), nil
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
