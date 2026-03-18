// evidence.go - Readiness evidence generation and verification.
//
// Purpose:
//
//	Generate reproducible readiness evidence runs and verify their artifact
//	shape through the shared artifact-run model.
//
// Responsibilities:
//   - Execute readiness gate commands and capture logs in a timestamped run
//     directory.
//   - Render machine-readable and executive-readable summaries for the current
//     run.
//   - Verify that generated evidence inventories and referenced artifacts are
//     consistent.
//
// Scope:
//   - Covers local proof-pack generation for the repository's validated command
//     surface.
//   - Does not mutate tracked documentation or deploy to customer environments.
//
// Usage:
//   - Called from `acpctl deploy readiness-evidence run`.
//   - Called from `acpctl deploy readiness-evidence verify`.
//
// Invariants/Assumptions:
//   - Evidence runs live under `demo/logs/evidence/readiness-<TIMESTAMP>/`.
//   - Commands are executed from the repository root using the current make
//     binary.
//   - Generated evidence is local-only and intentionally untracked.
package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const (
	SummaryJSONName          = "summary.json"
	SummaryMarkdownName      = "readiness-summary.md"
	TrackerMarkdownName      = "presentation-readiness-tracker.md"
	DecisionMarkdownName     = "go-no-go-decision.md"
	InventoryFileName        = "evidence-inventory.txt"
	LatestSuccessPointerName = "latest-success.txt"
	LatestRunPointerName     = "latest-run.txt"
	productionSecretsDefault = "/etc/ai-control-plane/secrets.env"
)

var (
	nowUTC       = func() time.Time { return time.Now().UTC() }
	gateExecutor = executeGate
)

// Options describes how to build a readiness evidence run.
type Options struct {
	RepoRoot          string
	OutputRoot        string
	MakeBin           string
	BundleVersion     string
	IncludeProduction bool
	SecretsEnvFile    string
	Verbose           bool
}

// GateResult captures one gate execution in a readiness run.
type GateResult struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Command       string   `json:"command"`
	CommandArgs   []string `json:"command_args"`
	Required      bool     `json:"required"`
	Status        string   `json:"status"`
	StartedAtUTC  string   `json:"started_at_utc,omitempty"`
	FinishedAtUTC string   `json:"finished_at_utc,omitempty"`
	Duration      string   `json:"duration,omitempty"`
	LogPath       string   `json:"log_path,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

// Summary is the canonical machine-readable result for a readiness run.
type Summary struct {
	RunID              string       `json:"run_id"`
	GeneratedAtUTC     string       `json:"generated_at_utc"`
	RepoRoot           string       `json:"repo_root"`
	RunDirectory       string       `json:"run_directory"`
	BundleVersion      string       `json:"bundle_version"`
	BundlePath         string       `json:"bundle_path,omitempty"`
	BundleChecksumPath string       `json:"bundle_checksum_path,omitempty"`
	IncludeProduction  bool         `json:"include_production"`
	ProductionEnabled  bool         `json:"production_enabled"`
	SecretsEnvFile     string       `json:"secrets_env_file,omitempty"`
	OverallStatus      string       `json:"overall_status"`
	FailingGateCount   int          `json:"failing_gate_count"`
	SkippedGateCount   int          `json:"skipped_gate_count"`
	GateResults        []GateResult `json:"gate_results"`
	GeneratedFiles     []string     `json:"generated_files"`
}

type gateSpec struct {
	ID             string
	Title          string
	Required       bool
	LogName        string
	Command        []string
	Notes          string
	ProductionOnly bool
}

// Verifier checks consistency of a generated readiness run.
type Verifier struct{}

// NewVerifier creates a readiness verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// RunContext executes the configured readiness gates and writes artifacts.
func RunContext(ctx context.Context, opts Options) (*Summary, error) {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	logger := logging.FromContext(ctx).With(
		slog.String("component", "readiness"),
		slog.String("workflow", "evidence"),
	)
	if strings.TrimSpace(opts.OutputRoot) == "" {
		opts.OutputRoot = filepath.Join(opts.RepoRoot, "demo", "logs", "evidence")
	}
	if strings.TrimSpace(opts.MakeBin) == "" {
		opts.MakeBin = "make"
	}
	if strings.TrimSpace(opts.BundleVersion) == "" {
		opts.BundleVersion = bundle.GetDefaultVersion(opts.RepoRoot)
	}
	if strings.TrimSpace(opts.SecretsEnvFile) == "" {
		opts.SecretsEnvFile = productionSecretsDefault
	}

	run, err := artifactrun.Create(opts.OutputRoot, "readiness", nowUTC())
	if err != nil {
		return nil, err
	}
	productionEnabled := opts.IncludeProduction && artifactrun.FileExists(opts.SecretsEnvFile)
	gates, err := materializeGates(opts, productionEnabled)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		RunID:              run.ID,
		GeneratedAtUTC:     nowUTC().Format(time.RFC3339),
		RepoRoot:           opts.RepoRoot,
		RunDirectory:       run.Directory,
		BundleVersion:      opts.BundleVersion,
		BundlePath:         filepath.Join(opts.RepoRoot, "demo", "logs", "release-bundles", bundle.GetBundleName(opts.BundleVersion)),
		BundleChecksumPath: filepath.Join(opts.RepoRoot, "demo", "logs", "release-bundles", bundle.GetBundleName(opts.BundleVersion)+".sha256"),
		IncludeProduction:  opts.IncludeProduction,
		ProductionEnabled:  productionEnabled,
		SecretsEnvFile:     opts.SecretsEnvFile,
		OverallStatus:      "PASS",
	}
	if opts.IncludeProduction && !productionEnabled {
		logger.Warn("gate.production_skipped", slog.String("secrets_env_file", opts.SecretsEnvFile), slog.String("reason", "secrets file not found"))
	}

	for _, gate := range gates {
		result := GateResult{
			ID:          gate.ID,
			Title:       gate.Title,
			Command:     strings.Join(gate.Command, " "),
			CommandArgs: append([]string(nil), gate.Command...),
			Required:    gate.Required,
			Status:      "PENDING",
			LogPath:     filepath.Join(run.Directory, gate.LogName),
			Notes:       gate.Notes,
		}

		if summary.OverallStatus == "FAIL" {
			result.Status = "SKIPPED"
			result.Notes = appendNote(result.Notes, "Earlier required gate failed; this gate was not executed.")
			logger.Warn("gate.skipped", slog.String("gate_id", gate.ID), slog.String("reason", "earlier required gate failed"))
			summary.SkippedGateCount++
			summary.GateResults = append(summary.GateResults, result)
			continue
		}
		if gate.ProductionOnly && !productionEnabled {
			result.Status = "SKIPPED"
			result.Notes = appendNote(result.Notes, fmt.Sprintf("Production gate skipped because secrets file is unavailable: %s", opts.SecretsEnvFile))
			logger.Warn("gate.skipped", slog.String("gate_id", gate.ID), slog.String("reason", "production secrets unavailable"))
			summary.SkippedGateCount++
			summary.GateResults = append(summary.GateResults, result)
			continue
		}
		startedAt := nowUTC()
		result.StartedAtUTC = startedAt.Format(time.RFC3339)
		logger.Info("gate.start", slog.String("gate_id", gate.ID), slog.String("title", gate.Title), slog.String("log_path", result.LogPath))
		status, finishedAt, runErr := gateExecutor(ctx, opts.RepoRoot, opts.MakeBin, gate, result.LogPath)
		result.Status = status
		result.FinishedAtUTC = finishedAt.Format(time.RFC3339)
		result.Duration = finishedAt.Sub(startedAt).Round(time.Second).String()
		if runErr != nil {
			result.Notes = appendNote(result.Notes, runErr.Error())
		}
		if result.Status == "FAIL" {
			logger.Error("gate.complete", slog.String("gate_id", gate.ID), slog.String("status", result.Status), slog.String("duration", result.Duration), logging.Err(runErr))
			summary.FailingGateCount++
			summary.OverallStatus = "FAIL"
		} else {
			logger.Info("gate.complete", slog.String("gate_id", gate.ID), slog.String("status", result.Status), slog.String("duration", result.Duration))
		}
		summary.GateResults = append(summary.GateResults, result)
	}

	if err := persistRun(opts.OutputRoot, summary); err != nil {
		return nil, err
	}
	logger.Info("workflow.complete", slog.String("run_id", summary.RunID), slog.String("run_directory", summary.RunDirectory), slog.String("overall_status", summary.OverallStatus))
	return summary, nil
}

// ResolveLatestRun resolves the most recent readiness run pointer.
func ResolveLatestRun(outputRoot string) (string, error) {
	return artifactrun.ResolveLatest(outputRoot, LatestRunPointerName)
}

func persistRun(outputRoot string, summary *Summary) error {
	if err := writeArtifacts(summary.RunDirectory, summary); err != nil {
		return err
	}

	latestPointers := []string{LatestRunPointerName}
	if summary.OverallStatus == "PASS" {
		latestPointers = append(latestPointers, LatestSuccessPointerName)
	}
	files, err := artifactrun.Finalize(summary.RunDirectory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryFileName,
		LatestPointers: latestPointers,
	})
	if err != nil {
		return err
	}
	summary.GeneratedFiles = files
	return writeArtifacts(summary.RunDirectory, summary)
}

func executeGate(ctx context.Context, repoRoot string, makeBin string, gate gateSpec, logPath string) (string, time.Time, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fsutil.PrivateFilePerm)
	if err != nil {
		return "FAIL", time.Now().UTC(), fmt.Errorf("create log file: %w", err)
	}
	if err := logFile.Chmod(fsutil.PrivateFilePerm); err != nil {
		_ = logFile.Close()
		return "FAIL", time.Now().UTC(), fmt.Errorf("chmod log file: %w", err)
	}
	defer logFile.Close()

	res := proc.Run(ctx, proc.Request{
		Name:    makeBin,
		Args:    gate.Command,
		Dir:     repoRoot,
		Env:     []string{"READINESS_EVIDENCE_ACTIVE=1"},
		Stdout:  logFile,
		Stderr:  logFile,
		Timeout: 30 * time.Minute,
	})
	finishedAt := time.Now().UTC()
	if res.Err == nil {
		return "PASS", finishedAt, nil
	}
	if proc.IsTimeout(res.Err) {
		return "FAIL", finishedAt, fmt.Errorf("command timed out after %s", 30*time.Minute)
	}
	if proc.IsCanceled(res.Err) {
		return "FAIL", finishedAt, fmt.Errorf("command canceled")
	}
	if exitCode, ok := proc.ExitCode(res.Err); ok {
		return "FAIL", finishedAt, fmt.Errorf("command exited with status %d", exitCode)
	}
	return "FAIL", finishedAt, fmt.Errorf("command execution failed: %w", res.Err)
}

func writeArtifacts(runDir string, summary *Summary) error {
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		return fmt.Errorf("write readiness summary json: %w", err)
	}
	return artifactrun.WriteArtifacts(runDir, []artifactrun.Artifact{
		{Path: SummaryMarkdownName, Body: []byte(renderSummaryMarkdown(summary)), Perm: fsutil.PrivateFilePerm},
		{Path: TrackerMarkdownName, Body: []byte(renderTrackerMarkdown(summary)), Perm: fsutil.PrivateFilePerm},
		{Path: DecisionMarkdownName, Body: []byte(renderDecisionMarkdown(summary)), Perm: fsutil.PrivateFilePerm},
	})
}

// VerifyRun validates the generated readiness evidence directory.
func (v *Verifier) VerifyRun(runDir string) (*Summary, error) {
	summaryPath := filepath.Join(runDir, SummaryJSONName)
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("read readiness summary json: %w", err)
	}
	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse readiness summary json: %w", err)
	}
	if summary.RunDirectory == "" {
		return nil, fmt.Errorf("summary missing run_directory")
	}
	if summary.RunDirectory != runDir {
		return nil, fmt.Errorf("summary run_directory %q does not match requested directory %q", summary.RunDirectory, runDir)
	}
	if err := artifactrun.Verify(runDir, artifactrun.VerifyOptions{
		InventoryName: InventoryFileName,
		RequiredFiles: []string{
			SummaryJSONName,
			SummaryMarkdownName,
			TrackerMarkdownName,
			DecisionMarkdownName,
			InventoryFileName,
		},
	}); err != nil {
		return nil, err
	}
	for _, gate := range summary.GateResults {
		if gate.LogPath == "" || gate.Status == "SKIPPED" {
			continue
		}
		if !artifactrun.FileExists(gate.LogPath) {
			return nil, fmt.Errorf("missing gate log: %s", gate.LogPath)
		}
	}
	if summary.OverallStatus == "PASS" {
		if !artifactrun.FileExists(summary.BundlePath) {
			return nil, fmt.Errorf("missing release bundle referenced by readiness summary: %s", summary.BundlePath)
		}
		if !artifactrun.FileExists(summary.BundleChecksumPath) {
			return nil, fmt.Errorf("missing release bundle checksum referenced by readiness summary: %s", summary.BundleChecksumPath)
		}
	}
	return &summary, nil
}

func renderSummaryMarkdown(summary *Summary) string {
	var builder strings.Builder
	builder.WriteString("# Readiness Evidence Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Overall status: **%s**\n", summary.OverallStatus))
	builder.WriteString(fmt.Sprintf("- Production gate included: `%t`\n", summary.IncludeProduction))
	builder.WriteString(fmt.Sprintf("- Production gate executed: `%t`\n", summary.ProductionEnabled))
	builder.WriteString("\n## Gate Results\n\n")
	builder.WriteString("| Gate | Status | Command | Log | Notes |\n")
	builder.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, gate := range summary.GateResults {
		logPath := gate.LogPath
		if logPath != "" {
			logPath = fmt.Sprintf("`%s`", gate.LogPath)
		}
		builder.WriteString(fmt.Sprintf("| %s | %s | `%s` | %s | %s |\n", gate.Title, gate.Status, gate.Command, logPath, normalizeMarkdownCell(gate.Notes)))
	}
	return builder.String()
}

func renderTrackerMarkdown(summary *Summary) string {
	var builder strings.Builder
	builder.WriteString("# Presentation Readiness Tracker\n\n")
	builder.WriteString("> Generated from `make readiness-evidence`. This file is local evidence, not a committed certification snapshot.\n\n")
	builder.WriteString(fmt.Sprintf("Current run: `%s` (%s)\n\n", summary.RunID, summary.GeneratedAtUTC))
	builder.WriteString("| Gate | Latest Status | Command | Evidence | Notes |\n")
	builder.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, gate := range summary.GateResults {
		builder.WriteString(fmt.Sprintf("| %s | %s | `%s` | `%s` | %s |\n", gate.Title, gate.Status, gate.Command, gate.LogPath, normalizeMarkdownCell(gate.Notes)))
	}
	builder.WriteString("\n## Generated Artifacts\n\n")
	for _, file := range summary.GeneratedFiles {
		builder.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	return builder.String()
}

func renderDecisionMarkdown(summary *Summary) string {
	decision := "GO"
	confidence := "HIGH"
	if summary.OverallStatus != "PASS" {
		decision = "NO_GO"
		confidence = "LOW"
	}
	var builder strings.Builder
	builder.WriteString("# Go/No-Go Decision\n\n")
	builder.WriteString(fmt.Sprintf("- Decision: **%s**\n", decision))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Evidence run: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Confidence: **%s**\n", confidence))
	builder.WriteString("\n## Decision Basis\n\n")
	if summary.OverallStatus == "PASS" {
		builder.WriteString("All required gates in this run passed. This is a current local proof pack for the validated repository baseline. Customer-environment controls still require customer-side validation.\n")
	} else {
		builder.WriteString("At least one required gate failed in this run. Do not present this run as a current readiness certification set.\n")
	}
	builder.WriteString("\n## Gate Outcomes\n\n")
	for _, gate := range summary.GateResults {
		builder.WriteString(fmt.Sprintf("- **%s:** %s (`%s`)\n", gate.Title, gate.Status, gate.LogPath))
	}
	return builder.String()
}

func appendNote(existing string, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return note
	}
	return existing + " " + note
}

func normalizeMarkdownCell(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", " "), "\n", " ")
}
