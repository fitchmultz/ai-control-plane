// cmd_db_backup_retention.go - Database backup retention command implementation.
//
// Purpose:
//   - Enforce deterministic retention for local database backup artifacts.
//
// Responsibilities:
//   - Define the `acpctl db backup-retention` command surface.
//   - Detect stale `.sql.gz` backup files in the canonical backup directory.
//   - Support check/apply modes with standard ACP exit codes.
//
// Scope:
//   - Database backup retention flow only.
//
// Usage:
//   - Invoked through `acpctl db backup-retention`.
//
// Invariants/Assumptions:
//   - Only canonical backup artifacts inside the resolved backup directory are managed.
//   - Retention ordering is newest-first by modtime, then name for deterministic ties.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

type dbBackupRetentionConfig struct {
	Mode     string
	Keep     int
	RepoRoot string
}

type backupArtifact struct {
	Path    string
	Name    string
	ModTime time.Time
}

func dbBackupRetentionCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "backup-retention",
		Summary:     "Enforce backup retention policy",
		Description: "Check or apply the database backup retention policy for canonical `.sql.gz` snapshots.",
		Options: []commandOptionSpec{
			{Name: "check", Summary: "Check only; fail if stale backups exist", Type: optionValueBool},
			{Name: "apply", Summary: "Delete stale backups", Type: optionValueBool},
			{Name: "keep", ValueName: "N", Summary: "Number of newest backups to retain", Type: optionValueInt, DefaultText: "7"},
		},
		Bind: bindRepoParsed(bindDBBackupRetentionOptions),
		Run:  runDBBackupRetention,
	})
}

func bindDBBackupRetentionOptions(bindCtx commandBindContext, input parsedCommandInput) (dbBackupRetentionConfig, error) {
	config := dbBackupRetentionConfig{
		Mode:     "check",
		Keep:     7,
		RepoRoot: bindCtx.RepoRoot,
	}
	if input.Bool("check") && input.Bool("apply") {
		return dbBackupRetentionConfig{}, fmt.Errorf("--check and --apply cannot be used together")
	}
	if input.Bool("apply") {
		config.Mode = "apply"
	}
	keep, err := input.IntDefault("keep", config.Keep)
	if err != nil || keep < 1 {
		return dbBackupRetentionConfig{}, fmt.Errorf("--keep requires a positive integer")
	}
	config.Keep = keep
	return config, nil
}

func runDBBackupRetention(_ context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(dbBackupRetentionConfig)
	out := output.New()
	logger := workflowLogger(runCtx, "db_backup_retention", "mode", config.Mode, "keep", config.Keep)
	workflowStart(logger)

	backupDir := resolveBackupDir(config.RepoRoot)
	artifacts, err := collectBackupArtifacts(backupDir)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitRuntime
	}
	stale := computeStaleBackups(artifacts, config.Keep)

	fmt.Fprintln(runCtx.Stdout, out.Bold("Backup retention results"))
	fmt.Fprintf(runCtx.Stdout, "  Backup dir: %s\n", backupDir)
	fmt.Fprintf(runCtx.Stdout, "  Keep newest: %d\n", config.Keep)
	fmt.Fprintf(runCtx.Stdout, "  Total backups: %d\n", len(artifacts))
	fmt.Fprintln(runCtx.Stdout)

	if len(stale) == 0 {
		fmt.Fprintln(runCtx.Stdout, out.Green("No stale backup files found."))
		workflowComplete(logger, "stale_backups", 0)
		return exitcodes.ACPExitSuccess
	}

	fmt.Fprintln(runCtx.Stdout, out.Yellow("Stale backup files:"))
	for _, artifact := range stale {
		fmt.Fprintf(runCtx.Stdout, "  - %s\n", artifact.Path)
	}

	if config.Mode == "check" {
		workflowWarn(logger, "status", "stale_backups_detected", "count", len(stale))
		fmt.Fprintln(runCtx.Stdout)
		fmt.Fprintln(runCtx.Stdout, out.Fail("Retention check failed."))
		fmt.Fprintln(runCtx.Stdout, "Run: acpctl db backup-retention --apply")
		return exitcodes.ACPExitDomain
	}

	for _, artifact := range stale {
		if err := os.Remove(artifact.Path); err != nil {
			workflowFailure(logger, err, "backup_file", artifact.Path)
			fmt.Fprintln(runCtx.Stderr, out.Fail(fmt.Sprintf("Failed to delete %s: %v", artifact.Path, err)))
			return exitcodes.ACPExitRuntime
		}
	}

	fmt.Fprintln(runCtx.Stdout)
	fmt.Fprintln(runCtx.Stdout, out.Green("Backup retention cleanup applied successfully."))
	workflowComplete(logger, "deleted_backups", len(stale))
	return exitcodes.ACPExitSuccess
}

func runDBBackupRetentionCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "backup-retention"}, args, stdout, stderr)
}

func collectBackupArtifacts(backupDir string) ([]backupArtifact, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	artifacts := make([]backupArtifact, 0, len(entries))
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
		artifacts = append(artifacts, backupArtifact{
			Path:    filepath.Join(backupDir, entry.Name()),
			Name:    entry.Name(),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].ModTime.Equal(artifacts[j].ModTime) {
			return artifacts[i].Name > artifacts[j].Name
		}
		return artifacts[i].ModTime.After(artifacts[j].ModTime)
	})
	return artifacts, nil
}

func computeStaleBackups(ordered []backupArtifact, keep int) []backupArtifact {
	if keep >= len(ordered) {
		return nil
	}
	return append([]backupArtifact(nil), ordered[keep:]...)
}
