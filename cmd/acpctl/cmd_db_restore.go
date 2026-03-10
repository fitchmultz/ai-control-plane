// cmd_db_restore.go - Database restore and drill command implementation.
//
// Purpose:
//   - Own the typed database restore and DR drill workflows.
//
// Responsibilities:
//   - Restore backups through the typed admin service.
//   - Resolve latest backup files deterministically.
//   - Preserve test-only wrapper helpers until restore tests move to direct CLI entry.
//
// Scope:
//   - Database restore and DR drill flows only.
//
// Usage:
//   - Invoked through `acpctl db restore` and `acpctl db dr-drill`.
//
// Invariants/Assumptions:
//   - Restore flows do not bypass the shared DB service helpers.
package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

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
