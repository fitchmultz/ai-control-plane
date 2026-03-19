// cmd_db_off_host_drill.go - Off-host recovery drill command implementation.
//
// Purpose:
//   - Own the typed replacement-host recovery evidence workflow.
//
// Responsibilities:
//   - Define the `acpctl db off-host-drill` command surface.
//   - Validate staged off-host recovery manifests through the DB admin service.
//   - Persist replacement-host recovery evidence artifacts locally.
//
// Scope:
//   - Operator-facing off-host recovery drill execution only.
//
// Usage:
//   - Invoked through `acpctl db off-host-drill` or `make db-off-host-drill`.
//
// Invariants/Assumptions:
//   - ACP validates staged off-host inputs but does not automate off-host replication transport.
//   - Generated evidence artifacts stay local-only under `demo/logs/evidence/`.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

const (
	recoveryEvidenceSummaryJSON      = "summary.json"
	recoveryEvidenceSummaryMarkdown  = "replacement-host-recovery-summary.md"
	recoveryEvidenceContractArtifact = "off-host-recovery-contract.yaml"
	recoveryEvidenceInventory        = "recovery-evidence-inventory.txt"
	recoveryEvidenceLatestRun        = "latest-run.txt"
	recoveryEvidenceLatestSuccess    = "latest-success.txt"
)

type dbOffHostDrillOptions struct {
	ManifestPath string
	OutputRoot   string
}

type replacementHostRecoverySummary struct {
	RunID            string                      `json:"run_id"`
	GeneratedAtUTC   string                      `json:"generated_at_utc"`
	RepoRoot         string                      `json:"repo_root"`
	RunDirectory     string                      `json:"run_directory"`
	ContractPath     string                      `json:"contract_path"`
	DrillMode        db.OffHostRecoveryDrillMode `json:"drill_mode"`
	DrillHost        string                      `json:"drill_host"`
	EvidenceBoundary string                      `json:"evidence_boundary"`
	Result           db.OffHostRecoveryResult    `json:"result"`
	RecoveryCommands []string                    `json:"recovery_commands"`
	GeneratedFiles   []string                    `json:"generated_files"`
}

func dbOffHostDrillCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "off-host-drill",
		Summary:     "Validate a staged off-host backup copy and emit staged-local or separate-host recovery evidence",
		Description: "Load an off-host recovery manifest, verify digest and provenance, restore into a scratch database, verify the core schema, and write replacement-host evidence with explicit drill mode and host labeling.",
		Examples: []string{
			"acpctl db off-host-drill --manifest demo/logs/recovery-inputs/off_host_recovery.yaml",
			"acpctl db off-host-drill --manifest demo/config/off_host_recovery.separate_host.yaml",
		},
		Options: []commandOptionSpec{
			{
				Name:        "manifest",
				ValueName:   "PATH",
				Summary:     "YAML contract describing the staged off-host backup copy, drill mode, and drill host",
				Type:        optionValueString,
				DefaultText: "demo/logs/recovery-inputs/off_host_recovery.yaml",
			},
			{
				Name:        "output-root",
				ValueName:   "PATH",
				Summary:     "Output directory for generated evidence runs",
				Type:        optionValueString,
				DefaultText: "demo/logs/evidence/replacement-host-recovery",
			},
		},
		Bind: bindRepoParsed(bindDBOffHostDrillOptions),
		Run:  runDBOffHostDrill,
	})
}

func bindDBOffHostDrillOptions(bindCtx commandBindContext, input parsedCommandInput) (dbOffHostDrillOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return dbOffHostDrillOptions{}, err
	}
	return dbOffHostDrillOptions{
		ManifestPath: resolveRepoInput(repoRoot, input.StringDefault("manifest", "demo/logs/recovery-inputs/off_host_recovery.yaml")),
		OutputRoot:   resolveRepoInput(repoRoot, input.StringDefault("output-root", "demo/logs/evidence/replacement-host-recovery")),
	}, nil
}

func runDBOffHostDrill(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(dbOffHostDrillOptions)
	out := output.New()
	logger := workflowLogger(runCtx, "db_off_host_drill", "manifest", opts.ManifestPath, "output_root", opts.OutputRoot)
	workflowStart(logger)

	if err := requireDBWorkflowPrereqs(runCtx.RepoRoot); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	contract, rawContract, err := db.LoadOffHostRecoveryContract(opts.ManifestPath)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	contract = db.NormalizeOffHostRecoveryContract(runCtx.RepoRoot, contract)
	if err := db.ValidateOffHostRecoveryContract(runCtx.RepoRoot, contract); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitDomain
	}

	services, err := openDBServices(runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	defer services.Close()
	if services.Admin == nil {
		err := fmt.Errorf("off-host recovery drill is not supported for external database mode")
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	printDBWorkflowHeader(runCtx.Stdout, out, "=== Off-Host Recovery Drill ===", map[string]string{
		"Manifest":        opts.ManifestPath,
		"Drill mode":      string(contract.DrillMode),
		"Drill host":      displayOffHostRecoveryDrillHost(contract),
		"Backup file":     contract.BackupFile,
		"Backup source":   contract.BackupSourceURI,
		"Inventory":       contract.InventoryPath,
		"Secrets env":     contract.SecretsEnvFile,
		"Evidence output": opts.OutputRoot,
	})

	if code := requireAccessibleDatabase(ctx, runCtx, logger, out, services.Runtime); code != exitcodes.ACPExitSuccess {
		return code
	}

	now := time.Now().UTC()
	result, err := services.Admin.RunOffHostRecoveryDrill(ctx, runCtx.RepoRoot, contract, now)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitDomain
	}

	summary, err := persistReplacementHostRecoveryEvidence(runCtx.RepoRoot, opts.OutputRoot, opts.ManifestPath, rawContract, result, now)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(fmt.Sprintf("write recovery evidence: %v", err)))
		return exitcodes.ACPExitRuntime
	}

	printDBWorkflowSuccess(runCtx.Stdout, out, "Replacement-host recovery evidence completed successfully!", map[string]any{
		"Run directory":         summary.RunDirectory,
		"Drill mode":            string(result.DrillMode),
		"Drill host":            result.DrillHost,
		"Backup file":           result.BackupFile,
		"Backup SHA256":         result.BackupSHA256,
		"Verified core tables":  fmt.Sprintf("%d/%d", result.Verification.FoundTables, result.Verification.ExpectedTables),
		"Scratch database":      result.ScratchDatabase,
		"Repo version verified": result.RepoVersion,
	})
	workflowComplete(logger, "run_dir", summary.RunDirectory)
	return exitcodes.ACPExitSuccess
}

func persistReplacementHostRecoveryEvidence(repoRoot string, outputRoot string, manifestPath string, rawContract []byte, result db.OffHostRecoveryResult, now time.Time) (*replacementHostRecoverySummary, error) {
	run, err := artifactrun.Create(outputRoot, "replacement-host-recovery", now)
	if err != nil {
		return nil, err
	}

	summary := &replacementHostRecoverySummary{
		RunID:            run.ID,
		GeneratedAtUTC:   now.Format(time.RFC3339),
		RepoRoot:         repoRoot,
		RunDirectory:     run.Directory,
		ContractPath:     manifestPath,
		DrillMode:        result.DrillMode,
		DrillHost:        result.DrillHost,
		EvidenceBoundary: result.DrillMode.EvidenceBoundary(),
		Result:           result,
		RecoveryCommands: replacementHostRecoveryCommands(result),
	}

	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, recoveryEvidenceSummaryJSON), summary); err != nil {
		return nil, err
	}
	if err := artifactrun.WriteArtifacts(run.Directory, []artifactrun.Artifact{
		{Path: recoveryEvidenceSummaryMarkdown, Body: []byte(renderReplacementHostRecoverySummary(summary)), Perm: fsutil.PrivateFilePerm},
		{Path: recoveryEvidenceContractArtifact, Body: rawContract, Perm: fsutil.PrivateFilePerm},
	}); err != nil {
		return nil, err
	}

	files, err := artifactrun.Finalize(run.Directory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  recoveryEvidenceInventory,
		LatestPointers: []string{recoveryEvidenceLatestRun, recoveryEvidenceLatestSuccess},
	})
	if err != nil {
		return nil, err
	}
	summary.GeneratedFiles = files

	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, recoveryEvidenceSummaryJSON), summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func replacementHostRecoveryCommands(result db.OffHostRecoveryResult) []string {
	return []string{
		fmt.Sprintf("./scripts/acpctl.sh host apply --inventory %s --env-file %s --skip-smoke-tests", result.InventoryPath, result.SecretsEnvFile),
		fmt.Sprintf("./scripts/acpctl.sh db restore %s", result.BackupFile),
		fmt.Sprintf("./scripts/acpctl.sh host apply --inventory %s --env-file %s", result.InventoryPath, result.SecretsEnvFile),
		fmt.Sprintf("make health COMPOSE_ENV_FILE=%s", result.SecretsEnvFile),
		fmt.Sprintf("make prod-smoke COMPOSE_ENV_FILE=%s", result.SecretsEnvFile),
		"make host-service-status",
	}
}

func renderReplacementHostRecoverySummary(summary *replacementHostRecoverySummary) string {
	var builder strings.Builder
	builder.WriteString("# Replacement Host Recovery Evidence\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Contract path: `%s`\n", summary.ContractPath))
	builder.WriteString(fmt.Sprintf("- Drill mode: `%s`\n", summary.DrillMode))
	builder.WriteString(fmt.Sprintf("- Drill host: `%s`\n", summary.DrillHost))
	builder.WriteString(fmt.Sprintf("- Off-host source URI: `%s`\n", summary.Result.BackupSourceURI))
	builder.WriteString(fmt.Sprintf("- Staged backup file: `%s`\n", summary.Result.BackupFile))
	builder.WriteString(fmt.Sprintf("- Backup SHA256: `%s`\n", summary.Result.BackupSHA256))
	builder.WriteString(fmt.Sprintf("- Backup size: `%d` bytes\n", summary.Result.BackupSizeBytes))
	builder.WriteString(fmt.Sprintf("- Canonical local backup dir: `%s`\n", summary.Result.LocalBackupDir))
	builder.WriteString(fmt.Sprintf("- Used off-host input: `%t`\n", summary.Result.UsedOffHostInput))
	builder.WriteString(fmt.Sprintf("- Inventory path: `%s`\n", summary.Result.InventoryPath))
	builder.WriteString(fmt.Sprintf("- Secrets env file: `%s`\n", summary.Result.SecretsEnvFile))
	builder.WriteString(fmt.Sprintf("- Repo version: `%s`\n", summary.Result.RepoVersion))
	builder.WriteString(fmt.Sprintf("- Scratch database: `%s`\n", summary.Result.ScratchDatabase))
	builder.WriteString(fmt.Sprintf("- Verified core tables: `%d/%d`\n", summary.Result.Verification.FoundTables, summary.Result.Verification.ExpectedTables))
	builder.WriteString(fmt.Sprintf("- PostgreSQL: `%s`\n", summary.Result.Verification.Version))

	builder.WriteString("\n## Claim Boundary\n\n")
	builder.WriteString(summary.EvidenceBoundary + "\n")

	builder.WriteString("\n## Replacement-Host Workflow\n\n")
	for index, command := range summary.RecoveryCommands {
		builder.WriteString(fmt.Sprintf("%d. `%s`\n", index+1, command))
	}

	builder.WriteString("\n## Generated Files\n\n")
	for _, file := range summary.GeneratedFiles {
		builder.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	return builder.String()
}

func displayOffHostRecoveryDrillHost(contract db.OffHostRecoveryContract) string {
	if strings.TrimSpace(contract.DrillHost) != "" {
		return contract.DrillHost
	}
	return "auto-detect at runtime"
}

func runDBOffHostDrillCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "off-host-drill"}, args, stdout, stderr)
}
