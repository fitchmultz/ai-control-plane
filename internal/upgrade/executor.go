// Package upgrade provides typed host-first upgrade and rollback workflows.
//
// Purpose:
//   - Execute typed host-first upgrade checks, upgrades, and rollbacks.
//
// Responsibilities:
//   - Validate explicit upgrade paths and checkout/version expectations.
//   - Snapshot canonical config and embedded database state for rollback.
//   - Apply tracked config and database migrations plus host convergence.
//   - Restore tracked snapshots during rollback.
//
// Scope:
//   - Upgrade orchestration only.
//
// Usage:
//   - Called by `acpctl upgrade plan|check|execute|rollback`.
//
// Invariants/Assumptions:
//   - Only explicit release edges are supported.
//   - Execute and rollback are supported only for embedded database mode.
package upgrade

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/hostdeploy"
	"github.com/mitchfultz/ai-control-plane/internal/migration"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

const (
	SummaryJSONName  = "summary.json"
	InventoryName    = "inventory.txt"
	LatestPointer    = "latest-upgrade-run.txt"
	configBackupName = "config.before.env"
	databaseBackup   = "database.before.sql.gz"
)

// Options configures upgrade planning, checking, and execution.
type Options struct {
	RepoRoot             string
	FromVersion          string
	ToVersion            string
	Inventory            string
	Limit                string
	RepoPath             string
	EnvFile              string
	OutputRoot           string
	ExtraVars            []string
	WaitForStabilization bool
	RunSmokeTests        bool
	StabilizationSeconds string
	Stdout               io.Writer
	Stderr               io.Writer
}

// RollbackOptions configures rollback execution.
type RollbackOptions struct {
	RepoRoot             string
	RunDir               string
	Inventory            string
	Limit                string
	RepoPath             string
	EnvFile              string
	ExtraVars            []string
	WaitForStabilization bool
	RunSmokeTests        bool
	StabilizationSeconds string
	Stdout               io.Writer
	Stderr               io.Writer
}

// Summary captures the persisted upgrade run metadata needed for rollback.
type Summary struct {
	RunID              string   `json:"run_id"`
	RunDirectory       string   `json:"run_directory"`
	CreatedAtUTC       string   `json:"created_at_utc"`
	FromVersion        string   `json:"from_version"`
	ToVersion          string   `json:"to_version"`
	ConfigBackupPath   string   `json:"config_backup_path"`
	DatabaseBackupPath string   `json:"database_backup_path"`
	EnvFile            string   `json:"env_file"`
	Inventory          string   `json:"inventory"`
	Path               []string `json:"path"`
	RollbackSteps      []string `json:"rollback_steps"`
}

type serviceBundle struct {
	Runtime  *db.RuntimeService
	Admin    *db.AdminService
	Migrator *db.MigrationService
	Close    func() error
}

var openServices = func(repoRoot string) (serviceBundle, error) {
	connector := db.NewConnector(repoRoot)
	admin, _ := db.NewAdminService(connector)
	return serviceBundle{
		Runtime:  db.NewRuntimeService(connector),
		Admin:    admin,
		Migrator: db.NewMigrationService(connector),
		Close:    connector.Close,
	}, nil
}

var runHostDeploy = hostdeploy.Execute
var nowUTC = func() time.Time { return time.Now().UTC() }

// PlanForOptions builds the explicit upgrade plan for the provided options.
func PlanForOptions(opts Options) (*Plan, error) {
	plan, err := BuildPlan(opts.FromVersion, ResolveTargetVersion(opts.RepoRoot, opts.ToVersion), DefaultCatalog())
	if err != nil {
		return nil, wrap(ErrorKindDomain, err)
	}
	return plan, nil
}

// Check validates that the requested upgrade path and host convergence are supported.
func Check(ctx context.Context, opts Options) (*Plan, error) {
	plan, err := PlanForOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := requireCheckoutVersion(opts.RepoRoot, plan.ToVersion); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "acp-upgrade-check-*")
	if err != nil {
		return nil, wrap(ErrorKindRuntime, fmt.Errorf("create temp check directory: %w", err))
	}
	defer os.RemoveAll(tempDir)

	migratedEnv := filepath.Join(tempDir, filepath.Base(opts.EnvFile))
	if err := snapshotFile(opts.EnvFile, migratedEnv); err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}
	if err := applyConfigMigrations(migratedEnv, plan); err != nil {
		return nil, err
	}
	if err := validateMigratedConfig(opts.RepoRoot, migratedEnv); err != nil {
		return nil, err
	}

	services, err := openServices(opts.RepoRoot)
	if err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}
	defer services.Close()
	if _, err := services.Runtime.Summary(ctx); err != nil {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("database precheck failed: %w", err))
	}
	if services.Admin == nil {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("upgrade check is supported only for embedded database mode because rollback requires typed backup/restore"))
	}

	if err := runHostDeploy(ctx, hostdeploy.Options{
		Mode:                 "check",
		RepoRoot:             opts.RepoRoot,
		Inventory:            opts.Inventory,
		Limit:                opts.Limit,
		RepoPath:             opts.RepoPath,
		EnvFile:              migratedEnv,
		WaitForStabilization: opts.WaitForStabilization,
		RunSmokeTests:        opts.RunSmokeTests,
		StabilizationSeconds: opts.StabilizationSeconds,
		ExtraVars:            appendUpgradeExtraVars(opts.ExtraVars, plan, false),
		Stdout:               opts.Stdout,
		Stderr:               opts.Stderr,
	}); err != nil {
		return nil, mapHostDeployError(err)
	}

	return plan, nil
}

// Execute applies the explicit upgrade plan and persists rollback artifacts.
func Execute(ctx context.Context, opts Options) (*Summary, error) {
	plan, err := PlanForOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := requireCheckoutVersion(opts.RepoRoot, plan.ToVersion); err != nil {
		return nil, err
	}

	services, err := openServices(opts.RepoRoot)
	if err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}
	defer services.Close()
	if services.Admin == nil {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("upgrade execute is supported only for embedded database mode because rollback requires typed backup/restore"))
	}

	runtimeSummary, err := services.Runtime.Summary(ctx)
	if err != nil {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("pre-upgrade database check failed: %w", err))
	}

	outputRoot := strings.TrimSpace(opts.OutputRoot)
	if outputRoot == "" {
		outputRoot = repopath.DemoLogsPath(opts.RepoRoot, "upgrades")
	}
	run, err := artifactrun.Create(outputRoot, "upgrade", nowUTC())
	if err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}

	configBackupPath := filepath.Join(run.Directory, configBackupName)
	if err := snapshotFile(opts.EnvFile, configBackupPath); err != nil {
		return nil, annotateRunDir(wrap(ErrorKindRuntime, err), run.Directory)
	}

	sqlDump, err := services.Admin.Backup(ctx)
	if err != nil {
		return nil, annotateRunDir(wrap(ErrorKindRuntime, fmt.Errorf("pre-upgrade database backup failed: %w", err)), run.Directory)
	}
	databaseBackupPath := filepath.Join(run.Directory, databaseBackup)
	if err := writeCompressedSQLBackup(databaseBackupPath, sqlDump); err != nil {
		return nil, annotateRunDir(wrap(ErrorKindRuntime, err), run.Directory)
	}

	summary := &Summary{
		RunID:              run.ID,
		RunDirectory:       run.Directory,
		CreatedAtUTC:       nowUTC().Format(time.RFC3339),
		FromVersion:        plan.FromVersion,
		ToVersion:          plan.ToVersion,
		ConfigBackupPath:   configBackupPath,
		DatabaseBackupPath: databaseBackupPath,
		EnvFile:            opts.EnvFile,
		Inventory:          opts.Inventory,
		Path:               append([]string(nil), plan.Path...),
		RollbackSteps: append([]string{
			fmt.Sprintf("Check out release %s before running rollback", plan.FromVersion),
			fmt.Sprintf("Run: acpctl upgrade rollback --run-dir %s --inventory %s --env-file %s", run.Directory, opts.Inventory, opts.EnvFile),
		}, plan.Rollback...),
	}
	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, SummaryJSONName), summary); err != nil {
		return nil, annotateRunDir(wrap(ErrorKindRuntime, err), run.Directory)
	}
	if _, err := artifactrun.Finalize(run.Directory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryName,
		LatestPointers: []string{LatestPointer},
	}); err != nil {
		return nil, annotateRunDir(wrap(ErrorKindRuntime, err), run.Directory)
	}

	if err := applyConfigMigrations(opts.EnvFile, plan); err != nil {
		return nil, annotateRunDir(err, run.Directory)
	}
	if err := validateMigratedConfig(opts.RepoRoot, opts.EnvFile); err != nil {
		return nil, annotateRunDir(err, run.Directory)
	}
	if err := applyDatabaseMigrations(ctx, services, runtimeSummary.DatabaseName, plan); err != nil {
		return nil, annotateRunDir(err, run.Directory)
	}

	if err := runHostDeploy(ctx, hostdeploy.Options{
		Mode:                 "apply",
		RepoRoot:             opts.RepoRoot,
		Inventory:            opts.Inventory,
		Limit:                opts.Limit,
		RepoPath:             opts.RepoPath,
		EnvFile:              opts.EnvFile,
		WaitForStabilization: opts.WaitForStabilization,
		RunSmokeTests:        opts.RunSmokeTests,
		StabilizationSeconds: opts.StabilizationSeconds,
		ExtraVars:            appendUpgradeExtraVars(opts.ExtraVars, plan, false),
		Stdout:               opts.Stdout,
		Stderr:               opts.Stderr,
	}); err != nil {
		return nil, annotateRunDir(mapHostDeployError(err), run.Directory)
	}

	return summary, nil
}

// Rollback restores the pre-upgrade config and embedded database snapshots.
func Rollback(ctx context.Context, opts RollbackOptions) (*Summary, error) {
	runDir := strings.TrimSpace(opts.RunDir)
	if runDir == "" {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("upgrade run directory is required"))
	}
	if err := artifactrun.Verify(runDir, artifactrun.VerifyOptions{InventoryName: InventoryName, RequiredFiles: []string{SummaryJSONName}}); err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}

	data, err := os.ReadFile(filepath.Join(runDir, SummaryJSONName))
	if err != nil {
		return nil, wrap(ErrorKindRuntime, fmt.Errorf("read upgrade summary: %w", err))
	}
	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, wrap(ErrorKindRuntime, fmt.Errorf("parse upgrade summary: %w", err))
	}
	if err := requireCheckoutVersion(opts.RepoRoot, summary.FromVersion); err != nil {
		return nil, err
	}

	envFile := strings.TrimSpace(opts.EnvFile)
	if envFile == "" {
		envFile = summary.EnvFile
	}
	inventory := strings.TrimSpace(opts.Inventory)
	if inventory == "" {
		inventory = summary.Inventory
	}
	if err := artifactrun.Verify(runDir, artifactrun.VerifyOptions{
		InventoryName: InventoryName,
		RequiredFiles: []string{
			filepath.Base(summary.ConfigBackupPath),
			filepath.Base(summary.DatabaseBackupPath),
			SummaryJSONName,
		},
	}); err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}

	if err := snapshotFile(summary.ConfigBackupPath, envFile); err != nil {
		return nil, wrap(ErrorKindRuntime, fmt.Errorf("restore config snapshot: %w", err))
	}

	services, err := openServices(opts.RepoRoot)
	if err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}
	defer services.Close()
	if services.Admin == nil {
		return nil, wrap(ErrorKindDomain, fmt.Errorf("upgrade rollback is supported only for embedded database mode"))
	}

	reader, err := openCompressedSQLBackup(summary.DatabaseBackupPath)
	if err != nil {
		return nil, wrap(ErrorKindRuntime, err)
	}
	defer reader.Close()
	if err := services.Admin.Restore(ctx, reader); err != nil {
		return nil, wrap(ErrorKindRuntime, fmt.Errorf("restore database snapshot: %w", err))
	}

	if err := runHostDeploy(ctx, hostdeploy.Options{
		Mode:                 "apply",
		RepoRoot:             opts.RepoRoot,
		Inventory:            inventory,
		Limit:                opts.Limit,
		RepoPath:             opts.RepoPath,
		EnvFile:              envFile,
		WaitForStabilization: opts.WaitForStabilization,
		RunSmokeTests:        opts.RunSmokeTests,
		StabilizationSeconds: opts.StabilizationSeconds,
		ExtraVars: append([]string(nil), appendUpgradeExtraVars(opts.ExtraVars, &Plan{
			FromVersion: summary.ToVersion,
			ToVersion:   summary.FromVersion,
		}, true)...),
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	}); err != nil {
		return nil, mapHostDeployError(err)
	}

	summary.EnvFile = envFile
	summary.Inventory = inventory
	return &summary, nil
}

func requireCheckoutVersion(repoRoot string, targetVersion string) error {
	checkoutVersion := bundle.GetDefaultVersion(repoRoot)
	if checkoutVersion != strings.TrimSpace(targetVersion) {
		return wrap(ErrorKindDomain, fmt.Errorf("current checkout VERSION is %s but target version is %s; run upgrade workflows from the target release checkout", checkoutVersion, targetVersion))
	}
	return nil
}

func applyConfigMigrations(envPath string, plan *Plan) error {
	for _, edge := range plan.Edges {
		for _, mutation := range edge.Config {
			if _, err := migrationApplyEnvMutation(envPath, mutation, true); err != nil {
				return wrap(ErrorKindDomain, err)
			}
		}
	}
	return nil
}

func validateMigratedConfig(repoRoot string, envPath string) error {
	issues, err := validation.ValidateDeploymentConfig(repoRoot, validation.ConfigValidationOptions{
		Profile:        validation.ConfigValidationProfileProduction,
		SecretsEnvFile: envPath,
	})
	if err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("post-migration config validation failed: %w", err))
	}
	if len(issues) > 0 {
		return wrap(ErrorKindDomain, fmt.Errorf("post-migration config validation failed: %s", strings.Join(issues, "; ")))
	}
	return nil
}

func applyDatabaseMigrations(ctx context.Context, services serviceBundle, databaseName string, plan *Plan) error {
	for _, edge := range plan.Edges {
		if len(edge.Database) == 0 {
			continue
		}
		if err := migrationApplySQL(ctx, services.Migrator, databaseName, edge.Database); err != nil {
			return wrap(ErrorKindRuntime, err)
		}
	}
	return nil
}

func appendUpgradeExtraVars(extraVars []string, plan *Plan, rollback bool) []string {
	values := append([]string(nil), extraVars...)
	values = append(values,
		"acp_upgrade_mode=true",
		"acp_upgrade_rollback="+strconv.FormatBool(rollback),
		"acp_upgrade_from_version="+plan.FromVersion,
		"acp_upgrade_to_version="+plan.ToVersion,
	)
	return values
}

func mapHostDeployError(err error) error {
	var deployErr *hostdeploy.Error
	if !errors.As(err, &deployErr) {
		return wrap(ErrorKindRuntime, err)
	}
	switch deployErr.Kind {
	case hostdeploy.ErrorKindPrereq:
		return wrap(ErrorKindPrereq, err)
	case hostdeploy.ErrorKindUsage, hostdeploy.ErrorKindDomain:
		return wrap(ErrorKindDomain, err)
	default:
		return wrap(ErrorKindRuntime, err)
	}
}

func annotateRunDir(err error, runDir string) error {
	if err == nil {
		return nil
	}
	message := fmt.Errorf("upgrade execution failed; rollback artifacts are preserved at %s: %w", runDir, err)
	switch {
	case IsKind(err, ErrorKindDomain):
		return wrap(ErrorKindDomain, message)
	case IsKind(err, ErrorKindPrereq):
		return wrap(ErrorKindPrereq, message)
	default:
		return wrap(ErrorKindRuntime, message)
	}
}

func snapshotFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read snapshot source %s: %w", src, err)
	}
	if err := fsutil.EnsurePrivateDir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("create snapshot parent %s: %w", filepath.Dir(dst), err)
	}
	if err := fsutil.AtomicWritePrivateFile(dst, data); err != nil {
		return fmt.Errorf("write snapshot destination %s: %w", dst, err)
	}
	return nil
}

func writeCompressedSQLBackup(path string, sql string) error {
	var payload bytes.Buffer
	gzipWriter := gzip.NewWriter(&payload)
	if _, err := gzipWriter.Write([]byte(sql)); err != nil {
		_ = gzipWriter.Close()
		return fmt.Errorf("compress database backup: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("close database backup gzip stream: %w", err)
	}
	if err := fsutil.AtomicWritePrivateFile(path, payload.Bytes()); err != nil {
		return fmt.Errorf("persist database backup: %w", err)
	}
	return nil
}

func openCompressedSQLBackup(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open compressed database backup %s: %w", path, err)
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("open gzip reader for %s: %w", path, err)
	}
	return readCloser{
		Reader: reader,
		closers: []io.Closer{
			reader,
			file,
		},
	}, nil
}

type readCloser struct {
	io.Reader
	closers []io.Closer
}

func (r readCloser) Close() error {
	var first error
	for _, closer := range r.closers {
		if err := closer.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

var migrationApplyEnvMutation = func(path string, mutation migration.EnvMutation, write bool) (migration.EnvMutationResult, error) {
	return migration.ApplyEnvMutation(path, mutation, write)
}

var migrationApplySQL = migration.ApplySQL
