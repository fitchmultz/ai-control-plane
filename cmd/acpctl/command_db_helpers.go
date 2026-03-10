// command_db_helpers.go - Shared database command helpers.
//
// Purpose:
//   - Keep database workflow commands aligned on one prerequisite, connector,
//     and operator-output contract.
//
// Responsibilities:
//   - Resolve and validate the canonical backup root.
//   - Construct connector-backed runtime/admin services with stable error
//     handling.
//   - Render consistent database workflow sections, success banners, and next
//     steps.
//
// Scope:
//   - Command-layer helpers for `acpctl db` workflows only.
//
// Usage:
//   - Used by backup, restore, and DR drill command handlers.
//
// Invariants/Assumptions:
//   - Database workflow commands remain operator-facing and deterministic.
//   - Connector lifecycle stays owned by the command layer.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

type dbServices struct {
	Connector *db.Connector
	Runtime   *db.RuntimeService
	Admin     *db.AdminService
	Mode      string
}

func resolveBackupDir(repoRoot string) string {
	backupDir := config.NewLoader().Tooling().BackupDir
	if backupDir == "" {
		return repopath.DemoBackupsPath(repoRoot)
	}
	return resolveRepoInput(repoRoot, backupDir)
}

func ensureBackupDir(path string) error {
	return fsutil.EnsurePrivateDir(path)
}

func requireDBWorkflowPrereqs(repoRoot string) error {
	if err := checkDBPrereqs(); err != nil {
		return err
	}
	if _, err := docker.NewCompose(docker.DefaultProjectDir(repoRoot)); err != nil {
		return fmt.Errorf("docker compose not available: %w", err)
	}
	return nil
}

func openDBServices(repoRoot string) (*dbServices, error) {
	connector := db.NewConnector(repoRoot)
	adminService, err := db.NewAdminService(connector)
	if err != nil {
		connector.Close()
		return nil, err
	}
	return &dbServices{
		Connector: connector,
		Runtime:   db.NewRuntimeService(connector),
		Admin:     adminService,
		Mode:      string(connector.Mode()),
	}, nil
}

func (s *dbServices) Close() {
	if s == nil || s.Connector == nil {
		return
	}
	s.Connector.Close()
}

func requireAccessibleDatabase(ctx context.Context, runCtx commandRunContext, logger *slog.Logger, out *output.Output, runtimeService *db.RuntimeService) int {
	if runtimeService.IsAccessible(ctx) {
		return exitcodes.ACPExitSuccess
	}
	workflowWarn(logger, "reason", "database inaccessible")
	fmt.Fprintln(runCtx.Stderr, out.Fail("PostgreSQL is not accepting connections"))
	return exitcodes.ACPExitPrereq
}

func printDBWorkflowHeader(stdout *os.File, out *output.Output, title string, details map[string]string) {
	printCommandSection(stdout, out, title)
	for label, value := range details {
		printCommandDetail(stdout, label, value)
	}
}

func printDBWorkflowSuccess(stdout *os.File, out *output.Output, title string, details map[string]any) {
	printCommandSuccess(stdout, out, title)
	for label, value := range details {
		printCommandDetail(stdout, label, value)
	}
}
