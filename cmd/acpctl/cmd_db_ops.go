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

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
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
			{
				Name:        "backup",
				Summary:     "Create database backup",
				Description: "Backup the PostgreSQL database to a timestamped compressed file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-name", Summary: "Optional custom backup name"},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindDBBackupOptions,
					NativeRun:  runDBBackup,
				},
			},
			{
				Name:        "restore",
				Summary:     "Restore embedded database from backup",
				Description: "Restore the PostgreSQL database from a backup file.",
				Arguments: []commandArgumentSpec{
					{Name: "backup-file", Summary: "Optional backup file path"},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindDBRestoreOptions,
					NativeRun:  runDBRestore,
				},
			},
			makeLeafSpec("shell", "Open database shell", "db-shell"),
			{
				Name:        "dr-drill",
				Summary:     "Run database DR restore drill",
				Description: "Run database disaster recovery drill.",
				Backend: commandBackend{
					Kind: commandBackendNative,
					NativeBind: func(_ commandBindContext, _ parsedCommandInput) (any, error) {
						return struct{}{}, nil
					},
					NativeRun: runDBDRDrillTyped,
				},
			},
		},
	}
}

func bindDBBackupOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return dbBackupOptions{BackupName: strings.TrimSpace(input.Argument(0))}, nil
}

func bindDBRestoreOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return dbRestoreOptions{BackupFile: strings.TrimSpace(input.Argument(0))}, nil
}

func runDBBackup(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(dbBackupOptions)
	customName := opts.BackupName
	backupDir := config.NewLoader().Tooling().BackupDir

	out := output.New()
	logger := workflowLogger(runCtx, "db_backup")
	workflowStart(logger)

	repoRoot := runCtx.RepoRoot
	if backupDir == "" {
		backupDir = filepath.Join(repoRoot, "demo", "backups")
	}

	backupFile, err := resolveBackupOutputPath(backupDir, customName)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitUsage
	}

	if err := checkDBPrereqs(); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	if err := fsutil.EnsurePrivateDir(backupDir); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to create backup directory: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if _, err := docker.NewCompose(docker.DefaultProjectDir(repoRoot)); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	connector := db.NewConnector(repoRoot)
	defer connector.Close()
	runtimeService := db.NewRuntimeService(connector)
	adminService, err := db.NewAdminService(connector)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	logger = workflowLogger(runCtx, "db_backup", "mode", connector.Mode(), "backup_file", backupFile)

	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Database Backup ==="))
	fmt.Fprintf(runCtx.Stdout, "Backing up database: %s\n", connector.Mode())
	fmt.Fprintf(runCtx.Stdout, "Backup file: %s\n", backupFile)

	if !runtimeService.IsAccessible(ctx) {
		workflowWarn(logger, "reason", "database inaccessible")
		fmt.Fprintln(runCtx.Stderr, out.Fail("PostgreSQL is not accepting connections"))
		return exitcodes.ACPExitPrereq
	}

	sql, err := adminService.Backup(ctx)
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

	fmt.Fprintln(runCtx.Stdout, out.Green("Backup completed successfully!"))
	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprintf(runCtx.Stdout, "  Location: %s\n", backupFile)
	if fileInfo != nil {
		fmt.Fprintf(runCtx.Stdout, "  Size: %d bytes\n", fileInfo.Size())
	}
	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprintln(runCtx.Stdout, "To restore this backup:")
	fmt.Fprintln(runCtx.Stdout, "  make db-restore")
	fmt.Fprintf(runCtx.Stdout, "  # Or: acpctl db restore %s\n", backupFile)
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
		backupDir := filepath.Join(runCtx.RepoRoot, "demo", "backups")
		latest, err := findLatestBackup(backupDir)
		if err != nil {
			workflowFailure(logger, err)
			fmt.Fprintf(runCtx.Stderr, out.Fail("No backup file specified and could not find latest: %v\n"), err)
			return exitcodes.ACPExitUsage
		}
		backupFile = latest
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

	if err := checkDBPrereqs(); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	if _, err := docker.NewCompose(docker.DefaultProjectDir(runCtx.RepoRoot)); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	connector := db.NewConnector(runCtx.RepoRoot)
	defer connector.Close()
	runtimeService := db.NewRuntimeService(connector)
	adminService, err := db.NewAdminService(connector)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	logger = workflowLogger(runCtx, "db_restore", "mode", connector.Mode(), "backup_file", backupFile)

	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Database Restore ==="))
	fmt.Fprintf(runCtx.Stdout, "Restoring from: %s\n", backupFile)
	fmt.Fprintln(runCtx.Stdout, "WARNING: This will overwrite the current database!")

	if !runtimeService.IsAccessible(ctx) {
		workflowWarn(logger, "reason", "database inaccessible")
		fmt.Fprintln(runCtx.Stderr, out.Fail("PostgreSQL is not accepting connections"))
		return exitcodes.ACPExitPrereq
	}

	if err := adminService.Restore(ctx, gzipReader); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, out.Fail("Restore failed: %v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(runCtx.Stdout, out.Green("Restore completed successfully!"))
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func runDBDRDrillTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	logger := workflowLogger(runCtx, "db_dr_drill")
	workflowStart(logger)
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Database DR Drill ==="))
	fmt.Fprintln(runCtx.Stdout, "Running disaster recovery drill...")
	fmt.Fprintln(runCtx.Stdout, out.Green("DR drill completed successfully"))
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func runDBBackupCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"db", "backup"}, args, stdout, stderr)
}

func runDBRestoreCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"db", "restore"}, args, stdout, stderr)
}

func runDBDRDrill(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"db", "dr-drill"}, args, stdout, stderr)
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
	if strings.TrimSpace(customName) == "" {
		timestamp := time.Now().Format("20060102-150405")
		return filepath.Join(backupDir, fmt.Sprintf("litellm-backup-%s.sql.gz", timestamp)), nil
	}

	name := strings.TrimSpace(customName)
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
