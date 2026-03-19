// cmd_host_failover_drill.go - Host-first HA failover drill command implementation.
//
// Purpose:
//   - Own the typed active-passive failover drill evidence workflow.
//
// Responsibilities:
//   - Define the `acpctl host failover-drill` command surface.
//   - Validate customer-operated failover drill manifests through the HA package.
//   - Persist failover drill evidence artifacts locally.
//
// Scope:
//   - Operator-facing failover drill validation and evidence archiving only.
//
// Usage:
//   - Invoked through `acpctl host failover-drill` or `make ha-failover-drill`.
//
// Invariants/Assumptions:
//   - ACP does not automate PostgreSQL promotion, fencing, or traffic cutover.
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
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/ha"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

const (
	haFailoverSummaryJSON      = "summary.json"
	haFailoverSummaryMarkdown  = "ha-failover-summary.md"
	haFailoverContractArtifact = "ha-failover-contract.yaml"
	haFailoverInventory        = "evidence-inventory.txt"
	haFailoverLatestRun        = "latest-run.txt"
	haFailoverLatestSuccess    = "latest-success.txt"
)

type hostFailoverDrillOptions struct {
	ManifestPath string
	OutputRoot   string
}

type hostFailoverCopiedEvidence struct {
	Kind       string `json:"kind"`
	SourcePath string `json:"source_path"`
	BundlePath string `json:"bundle_path"`
}

type hostFailoverDrillSummary struct {
	RunID            string                       `json:"run_id"`
	GeneratedAtUTC   string                       `json:"generated_at_utc"`
	RepoRoot         string                       `json:"repo_root"`
	RunDirectory     string                       `json:"run_directory"`
	ContractPath     string                       `json:"contract_path"`
	EvidenceBoundary string                       `json:"evidence_boundary"`
	Contract         ha.FailoverDrillContract     `json:"contract"`
	RunbookSteps     []string                     `json:"runbook_steps"`
	CopiedEvidence   []hostFailoverCopiedEvidence `json:"copied_evidence"`
	GeneratedFiles   []string                     `json:"generated_files"`
}

func hostFailoverDrillCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "failover-drill",
		Summary:     "Validate a customer-operated active-passive failover drill manifest and archive evidence",
		Description: "Load an HA failover drill manifest, verify the required replication, fencing, promotion, traffic-cutover, and postcheck evidence files, then write a local evidence bundle with explicit claim-boundary language.",
		Examples: []string{
			"acpctl host failover-drill --manifest demo/logs/recovery-inputs/ha_failover_drill.yaml",
			"acpctl host failover-drill --manifest demo/config/ha_failover_drill.example.yaml --output-root demo/logs/evidence/ha-failover-drill",
		},
		Options: []commandOptionSpec{
			{
				Name:        "manifest",
				ValueName:   "PATH",
				Summary:     "YAML contract describing the customer-operated active-passive failover drill",
				Type:        optionValueString,
				DefaultText: "demo/logs/recovery-inputs/ha_failover_drill.yaml",
			},
			{
				Name:        "output-root",
				ValueName:   "PATH",
				Summary:     "Output directory for generated failover drill evidence runs",
				Type:        optionValueString,
				DefaultText: "demo/logs/evidence/ha-failover-drill",
			},
		},
		Bind: bindRepoParsed(bindHostFailoverDrillOptions),
		Run:  runHostFailoverDrill,
	})
}

func bindHostFailoverDrillOptions(bindCtx commandBindContext, input parsedCommandInput) (hostFailoverDrillOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return hostFailoverDrillOptions{}, err
	}
	return hostFailoverDrillOptions{
		ManifestPath: resolveRepoInput(repoRoot, input.StringDefault("manifest", "demo/logs/recovery-inputs/ha_failover_drill.yaml")),
		OutputRoot:   resolveRepoInput(repoRoot, input.StringDefault("output-root", "demo/logs/evidence/ha-failover-drill")),
	}, nil
}

func runHostFailoverDrill(ctx context.Context, runCtx commandRunContext, raw any) int {
	_ = ctx
	opts := raw.(hostFailoverDrillOptions)
	out := output.New()
	logger := workflowLogger(runCtx, "host_failover_drill", "manifest", opts.ManifestPath, "output_root", opts.OutputRoot)
	workflowStart(logger)

	contract, rawContract, err := ha.LoadFailoverDrillContract(opts.ManifestPath)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	contract = ha.NormalizeFailoverDrillContract(runCtx.RepoRoot, contract)
	if err := ha.ValidateFailoverDrillContract(runCtx.RepoRoot, contract); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitDomain
	}

	printCommandSection(runCtx.Stdout, out, "Active-passive failover drill")
	printCommandDetail(runCtx.Stdout, "Manifest", opts.ManifestPath)
	printCommandDetail(runCtx.Stdout, "Active host", contract.ActiveHost)
	printCommandDetail(runCtx.Stdout, "Passive host", contract.PassiveHost)
	printCommandDetail(runCtx.Stdout, "Traffic cutover", string(contract.TrafficCutover.Method))
	printCommandDetail(runCtx.Stdout, "Evidence output", opts.OutputRoot)

	now := time.Now().UTC()
	summary, err := persistHostFailoverDrillEvidence(runCtx.RepoRoot, opts.OutputRoot, opts.ManifestPath, rawContract, contract, now)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(fmt.Sprintf("write failover drill evidence: %v", err)))
		return exitcodes.ACPExitRuntime
	}

	printCommandSuccess(runCtx.Stdout, out, "Active-passive failover drill evidence archived")
	printCommandDetail(runCtx.Stdout, "Run directory", summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Summary", filepath.Join(summary.RunDirectory, haFailoverSummaryMarkdown))
	printCommandDetail(runCtx.Stdout, "Inventory", filepath.Join(summary.RunDirectory, haFailoverInventory))
	workflowComplete(logger, "run_dir", summary.RunDirectory)
	return exitcodes.ACPExitSuccess
}

func persistHostFailoverDrillEvidence(repoRoot string, outputRoot string, manifestPath string, rawContract []byte, contract ha.FailoverDrillContract, now time.Time) (*hostFailoverDrillSummary, error) {
	run, err := artifactrun.Create(outputRoot, "ha-failover-drill", now)
	if err != nil {
		return nil, err
	}

	copiedEvidence, evidenceArtifacts, err := failoverEvidenceArtifacts(contract)
	if err != nil {
		return nil, err
	}
	artifacts := append([]artifactrun.Artifact{
		{Path: haFailoverContractArtifact, Body: rawContract, Perm: fsutil.PrivateFilePerm},
	}, evidenceArtifacts...)

	summary := &hostFailoverDrillSummary{
		RunID:            run.ID,
		GeneratedAtUTC:   now.Format(time.RFC3339),
		RepoRoot:         repoRoot,
		RunDirectory:     run.Directory,
		ContractPath:     manifestPath,
		EvidenceBoundary: ha.EvidenceBoundary(),
		Contract:         contract,
		RunbookSteps:     ha.CanonicalRunbookSteps(contract),
		CopiedEvidence:   copiedEvidence,
	}

	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, haFailoverSummaryJSON), summary); err != nil {
		return nil, err
	}
	artifacts = append(artifacts, artifactrun.Artifact{Path: haFailoverSummaryMarkdown, Body: []byte(renderHostFailoverDrillSummary(summary)), Perm: fsutil.PrivateFilePerm})
	if err := artifactrun.WriteArtifacts(run.Directory, artifacts); err != nil {
		return nil, err
	}

	files, err := artifactrun.Finalize(run.Directory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  haFailoverInventory,
		LatestPointers: []string{haFailoverLatestRun, haFailoverLatestSuccess},
	})
	if err != nil {
		return nil, err
	}
	summary.GeneratedFiles = files
	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, haFailoverSummaryJSON), summary); err != nil {
		return nil, err
	}
	if err := artifactrun.WriteArtifacts(run.Directory, []artifactrun.Artifact{{
		Path: haFailoverSummaryMarkdown,
		Body: []byte(renderHostFailoverDrillSummary(summary)),
		Perm: fsutil.PrivateFilePerm,
	}}); err != nil {
		return nil, err
	}
	return summary, nil
}

func failoverEvidenceArtifacts(contract ha.FailoverDrillContract) ([]hostFailoverCopiedEvidence, []artifactrun.Artifact, error) {
	sources := []struct {
		kind string
		path string
	}{
		{kind: "replication", path: contract.ReplicationEvidencePath},
		{kind: "fencing", path: contract.Fencing.EvidencePath},
		{kind: "promotion", path: contract.Promotion.EvidencePath},
		{kind: "traffic-cutover", path: contract.TrafficCutover.EvidencePath},
		{kind: "postcheck", path: contract.PostcheckEvidencePath},
	}

	copied := make([]hostFailoverCopiedEvidence, 0, len(sources))
	artifacts := make([]artifactrun.Artifact, 0, len(sources))
	for _, source := range sources {
		body, err := os.ReadFile(source.path)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s evidence: %w", source.kind, err)
		}
		bundlePath := filepath.ToSlash(filepath.Join("evidence", source.kind+copyEvidenceExtension(source.path)))
		copied = append(copied, hostFailoverCopiedEvidence{
			Kind:       source.kind,
			SourcePath: source.path,
			BundlePath: bundlePath,
		})
		artifacts = append(artifacts, artifactrun.Artifact{Path: bundlePath, Body: body, Perm: fsutil.PrivateFilePerm})
	}
	return copied, artifacts, nil
}

func copyEvidenceExtension(path string) string {
	ext := strings.TrimSpace(filepath.Ext(path))
	if ext == "" {
		return ".txt"
	}
	return ext
}

func renderHostFailoverDrillSummary(summary *hostFailoverDrillSummary) string {
	var builder strings.Builder
	builder.WriteString("# Active-Passive Failover Drill Evidence\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Contract path: `%s`\n", summary.ContractPath))
	builder.WriteString(fmt.Sprintf("- Drill host: `%s`\n", displayFailoverDrillHost(summary.Contract)))
	builder.WriteString(fmt.Sprintf("- Active host: `%s`\n", summary.Contract.ActiveHost))
	builder.WriteString(fmt.Sprintf("- Passive host: `%s`\n", summary.Contract.PassiveHost))
	builder.WriteString(fmt.Sprintf("- Promoted host: `%s`\n", summary.Contract.Promotion.PromotedHost))
	builder.WriteString(fmt.Sprintf("- Inventory path: `%s`\n", summary.Contract.InventoryPath))
	builder.WriteString(fmt.Sprintf("- Secrets env file: `%s`\n", summary.Contract.SecretsEnvFile))
	builder.WriteString(fmt.Sprintf("- Traffic cutover method: `%s`\n", summary.Contract.TrafficCutover.Method))

	builder.WriteString("\n## Claim Boundary\n\n")
	builder.WriteString(summary.EvidenceBoundary + "\n")

	builder.WriteString("\n## Canonical Operator Sequence\n\n")
	for index, step := range summary.RunbookSteps {
		builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, step))
	}

	builder.WriteString("\n## Archived Evidence\n\n")
	for _, file := range summary.CopiedEvidence {
		builder.WriteString(fmt.Sprintf("- `%s` → `%s`\n", file.SourcePath, file.BundlePath))
	}

	if strings.TrimSpace(summary.Contract.Notes) != "" {
		builder.WriteString("\n## Notes\n\n")
		builder.WriteString(summary.Contract.Notes + "\n")
	}

	builder.WriteString("\n## Generated Files\n\n")
	for _, file := range summary.GeneratedFiles {
		builder.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	return builder.String()
}

func displayFailoverDrillHost(contract ha.FailoverDrillContract) string {
	if strings.TrimSpace(contract.DrillHost) != "" {
		return contract.DrillHost
	}
	return "operator-specified runtime host not recorded"
}
