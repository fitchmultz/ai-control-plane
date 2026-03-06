// readiness_evidence.go - Readiness evidence generation and verification.
//
// Purpose:
//
//	Generate reproducible readiness evidence runs and verify their artifact shape.
//
// Responsibilities:
//   - Execute readiness gate commands and capture logs in a timestamped run directory.
//   - Render machine-readable and executive-readable summaries for the current run.
//   - Verify that generated evidence inventories and referenced artifacts are consistent.
//
// Scope:
//   - Covers local proof-pack generation for the repository's validated command surface.
//   - Does not mutate tracked documentation or deploy to customer environments.
//
// Usage:
//   - Called from `acpctl deploy readiness-evidence run`
//   - Called from `acpctl deploy readiness-evidence verify`
//
// Invariants/Assumptions:
//   - Evidence runs live under `demo/logs/evidence/readiness-<TIMESTAMP>/`.
//   - Commands are executed from the repository root using the current make binary.
//   - Generated evidence is local-only and intentionally untracked.
package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	readinessSummaryJSONName    = "summary.json"
	readinessSummaryMarkdown    = "readiness-summary.md"
	readinessTrackerMarkdown    = "presentation-readiness-tracker.md"
	readinessDecisionMarkdown   = "go-no-go-decision.md"
	readinessInventoryText      = "evidence-inventory.txt"
	readinessLatestSuccess      = "latest-success.txt"
	readinessLatestRun          = "latest-run.txt"
	productionSecretsEnvDefault = "/etc/ai-control-plane/secrets.env"
)

var (
	readinessNow          = func() time.Time { return time.Now().UTC() }
	readinessGateExecutor = executeReadinessGate
)

// ReadinessOptions describes how to build a readiness evidence run.
type ReadinessOptions struct {
	RepoRoot          string
	OutputRoot        string
	MakeBin           string
	BundleVersion     string
	IncludeProduction bool
	SecretsEnvFile    string
	Verbose           bool
}

// ReadinessGateResult captures one gate execution in a readiness run.
type ReadinessGateResult struct {
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

// ReadinessSummary is the canonical machine-readable result for a run.
type ReadinessSummary struct {
	RunID              string                `json:"run_id"`
	GeneratedAtUTC     string                `json:"generated_at_utc"`
	RepoRoot           string                `json:"repo_root"`
	RunDirectory       string                `json:"run_directory"`
	BundleVersion      string                `json:"bundle_version"`
	BundlePath         string                `json:"bundle_path,omitempty"`
	BundleChecksumPath string                `json:"bundle_checksum_path,omitempty"`
	IncludeProduction  bool                  `json:"include_production"`
	ProductionEnabled  bool                  `json:"production_enabled"`
	SecretsEnvFile     string                `json:"secrets_env_file,omitempty"`
	OverallStatus      string                `json:"overall_status"`
	FailingGateCount   int                   `json:"failing_gate_count"`
	SkippedGateCount   int                   `json:"skipped_gate_count"`
	GateResults        []ReadinessGateResult `json:"gate_results"`
	GeneratedFiles     []string              `json:"generated_files"`
}

type readinessGateSpec struct {
	id       string
	title    string
	required bool
	logName  string
	command  []string
	notes    string
}

// ReadinessVerifier checks consistency of a generated readiness run.
type ReadinessVerifier struct{}

// NewReadinessVerifier creates a readiness verifier.
func NewReadinessVerifier() *ReadinessVerifier {
	return &ReadinessVerifier{}
}

// RunReadinessEvidence executes the configured readiness gates and writes artifacts.
func RunReadinessEvidence(opts ReadinessOptions, stdout io.Writer, stderr io.Writer) (*ReadinessSummary, error) {
	if opts.RepoRoot == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if opts.OutputRoot == "" {
		opts.OutputRoot = filepath.Join(opts.RepoRoot, "demo", "logs", "evidence")
	}
	if opts.MakeBin == "" {
		opts.MakeBin = "make"
	}
	if opts.BundleVersion == "" {
		opts.BundleVersion = GetDefaultVersion(opts.RepoRoot)
	}
	if opts.SecretsEnvFile == "" {
		opts.SecretsEnvFile = productionSecretsEnvDefault
	}

	nowUTC := readinessNow()
	runID := fmt.Sprintf("readiness-%s", nowUTC.Format("20060102T150405Z"))
	runDir := filepath.Join(opts.OutputRoot, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create readiness run directory: %w", err)
	}

	productionEnabled := opts.IncludeProduction && fileExists(opts.SecretsEnvFile)
	gates := buildReadinessGates(opts, productionEnabled)
	summary := &ReadinessSummary{
		RunID:              runID,
		GeneratedAtUTC:     nowUTC.Format(time.RFC3339),
		RepoRoot:           opts.RepoRoot,
		RunDirectory:       runDir,
		BundleVersion:      opts.BundleVersion,
		BundlePath:         filepath.Join(opts.RepoRoot, "demo", "logs", "release-bundles", GetBundleName(opts.BundleVersion)),
		BundleChecksumPath: filepath.Join(opts.RepoRoot, "demo", "logs", "release-bundles", GetBundleName(opts.BundleVersion)+".sha256"),
		IncludeProduction:  opts.IncludeProduction,
		ProductionEnabled:  productionEnabled,
		SecretsEnvFile:     opts.SecretsEnvFile,
		OverallStatus:      "PASS",
	}

	if opts.IncludeProduction && !productionEnabled {
		fmt.Fprintf(stdout, "Production gate requested but secrets file not found: %s\n", opts.SecretsEnvFile)
	}

	for index, gate := range gates {
		result := ReadinessGateResult{
			ID:          gate.id,
			Title:       gate.title,
			Command:     strings.Join(gate.command, " "),
			CommandArgs: append([]string(nil), gate.command...),
			Required:    gate.required,
			Status:      "PENDING",
			LogPath:     filepath.Join(runDir, gate.logName),
			Notes:       gate.notes,
		}

		if summary.OverallStatus == "FAIL" {
			result.Status = "SKIPPED"
			result.Notes = appendReadinessNote(result.Notes, "Earlier required gate failed; this gate was not executed.")
			summary.SkippedGateCount++
			summary.GateResults = append(summary.GateResults, result)
			continue
		}

		if !gate.required && !productionEnabled && gate.id == "production_ci" {
			result.Status = "SKIPPED"
			result.Notes = appendReadinessNote(result.Notes, fmt.Sprintf("Production gate skipped because secrets file is unavailable: %s", opts.SecretsEnvFile))
			summary.SkippedGateCount++
			summary.GateResults = append(summary.GateResults, result)
			continue
		}

		if index > 0 {
			fmt.Fprintln(stdout, "")
		}
		fmt.Fprintf(stdout, "[%s] %s\n", gate.id, gate.title)
		startedAt := readinessNow()
		result.StartedAtUTC = startedAt.Format(time.RFC3339)
		status, finishedAt, runErr := readinessGateExecutor(opts.RepoRoot, opts.MakeBin, gate, result.LogPath, stdout, stderr)
		result.Status = status
		result.FinishedAtUTC = finishedAt.Format(time.RFC3339)
		result.Duration = finishedAt.Sub(startedAt).Round(time.Second).String()
		if runErr != nil {
			result.Notes = appendReadinessNote(result.Notes, runErr.Error())
		}
		if result.Status == "FAIL" {
			summary.FailingGateCount++
			summary.OverallStatus = "FAIL"
		}
		summary.GateResults = append(summary.GateResults, result)
	}

	if err := writeReadinessArtifacts(runDir, summary); err != nil {
		return nil, err
	}
	generatedFiles, err := readinessGeneratedFiles(runDir)
	if err != nil {
		return nil, err
	}
	summary.GeneratedFiles = generatedFiles
	if err := writeReadinessSummary(runDir, summary); err != nil {
		return nil, err
	}
	if err := writeLatestReadinessPointers(opts.OutputRoot, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

// ResolveLatestReadinessRun resolves the most recent readiness run pointer.
func ResolveLatestReadinessRun(outputRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(outputRoot, readinessLatestRun))
	if err != nil {
		return "", fmt.Errorf("read latest readiness run pointer: %w", err)
	}
	runDir := strings.TrimSpace(string(data))
	if runDir == "" {
		return "", fmt.Errorf("latest readiness run pointer is empty")
	}
	return runDir, nil
}

func buildReadinessGates(opts ReadinessOptions, productionEnabled bool) []readinessGateSpec {
	gates := []readinessGateSpec{
		{id: "local_ci", title: "Local CI Gate", required: true, logName: "make-ci.log", command: []string{"ci"}, notes: "Validates the host-first baseline command surface."},
	}
	if opts.IncludeProduction {
		gates = append(gates, readinessGateSpec{
			id:       "production_ci",
			title:    "Production CI Gate",
			required: productionEnabled,
			logName:  "make-ci-nightly.log",
			command:  []string{"ci-nightly", fmt.Sprintf("SECRETS_ENV_FILE=%s", opts.SecretsEnvFile)},
			notes:    "Optional customer-like gate; requires a real secrets file.",
		})
	}
	gates = append(gates,
		readinessGateSpec{id: "supply_chain", title: "Supply Chain Gate", required: true, logName: "make-supply-chain-gate.log", command: []string{"supply-chain-gate"}, notes: "Validates the supply-chain policy contract."},
		readinessGateSpec{id: "supply_chain_allowlist", title: "Allowlist Expiry Gate", required: true, logName: "make-supply-chain-allowlist-expiry-check.log", command: []string{"supply-chain-allowlist-expiry-check"}, notes: "Checks time-bound vulnerability exceptions."},
		readinessGateSpec{id: "release_bundle", title: "Release Bundle Gate", required: true, logName: "make-release-bundle.log", command: []string{"release-bundle", fmt.Sprintf("RELEASE_BUNDLE_VERSION=%s", opts.BundleVersion)}, notes: "Builds the canonical deployment bundle for the run."},
		readinessGateSpec{id: "release_bundle_verify", title: "Release Bundle Verify Gate", required: true, logName: "make-release-bundle-verify.log", command: []string{"release-bundle-verify", fmt.Sprintf("RELEASE_BUNDLE_VERSION=%s", opts.BundleVersion)}, notes: "Confirms bundle integrity using the current checksum sidecar."},
	)
	return gates
}

func executeReadinessGate(repoRoot string, makeBin string, gate readinessGateSpec, logPath string, stdout io.Writer, stderr io.Writer) (string, time.Time, error) {
	logFile, err := os.Create(logPath)
	if err != nil {
		return "FAIL", time.Now().UTC(), fmt.Errorf("create log file: %w", err)
	}
	defer logFile.Close()

	logWriter := io.MultiWriter(stdout, logFile)
	errWriter := io.MultiWriter(stderr, logFile)
	cmd := exec.Command(makeBin, gate.command...)
	cmd.Dir = repoRoot
	cmd.Stdout = logWriter
	cmd.Stderr = errWriter
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), "READINESS_EVIDENCE_ACTIVE=1")
	cmdErr := cmd.Run()
	finishedAt := time.Now().UTC()
	if cmdErr == nil {
		return "PASS", finishedAt, nil
	}

	var exitErr *exec.ExitError
	if errors.As(cmdErr, &exitErr) {
		return "FAIL", finishedAt, fmt.Errorf("command exited with status %d", exitErr.ExitCode())
	}
	return "FAIL", finishedAt, fmt.Errorf("command execution failed: %w", cmdErr)
}

func writeReadinessArtifacts(runDir string, summary *ReadinessSummary) error {
	if err := writeJSON(filepath.Join(runDir, readinessSummaryJSONName), summary); err != nil {
		return fmt.Errorf("write readiness summary json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, readinessSummaryMarkdown), []byte(renderReadinessSummaryMarkdown(summary)), 0o644); err != nil {
		return fmt.Errorf("write readiness summary markdown: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, readinessTrackerMarkdown), []byte(renderReadinessTrackerMarkdown(summary)), 0o644); err != nil {
		return fmt.Errorf("write readiness tracker markdown: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, readinessDecisionMarkdown), []byte(renderReadinessDecisionMarkdown(summary)), 0o644); err != nil {
		return fmt.Errorf("write go/no-go decision markdown: %w", err)
	}
	return nil
}

func writeReadinessSummary(runDir string, summary *ReadinessSummary) error {
	files, err := readinessGeneratedFiles(runDir)
	if err != nil {
		return fmt.Errorf("build readiness inventory: %w", err)
	}
	files = append(files, readinessInventoryText)
	sort.Strings(files)
	summary.GeneratedFiles = files
	if err := writeJSON(filepath.Join(runDir, readinessSummaryJSONName), summary); err != nil {
		return fmt.Errorf("rewrite readiness summary json: %w", err)
	}
	inventoryPath := filepath.Join(runDir, readinessInventoryText)
	if err := os.WriteFile(inventoryPath, []byte(strings.Join(files, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, readinessTrackerMarkdown), []byte(renderReadinessTrackerMarkdown(summary)), 0o644); err != nil {
		return fmt.Errorf("rewrite readiness tracker markdown: %w", err)
	}
	return os.WriteFile(filepath.Join(runDir, readinessSummaryMarkdown), []byte(renderReadinessSummaryMarkdown(summary)), 0o644)
}

func writeLatestReadinessPointers(outputRoot string, summary *ReadinessSummary) error {
	latestRunPath := filepath.Join(outputRoot, readinessLatestRun)
	if err := os.WriteFile(latestRunPath, []byte(summary.RunDirectory+"\n"), 0o644); err != nil {
		return fmt.Errorf("write latest run pointer: %w", err)
	}
	if summary.OverallStatus == "PASS" {
		latestSuccessPath := filepath.Join(outputRoot, readinessLatestSuccess)
		if err := os.WriteFile(latestSuccessPath, []byte(summary.RunDirectory+"\n"), 0o644); err != nil {
			return fmt.Errorf("write latest success pointer: %w", err)
		}
	}
	return nil
}

func readinessGeneratedFiles(runDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, relErr := filepath.Rel(runDir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// VerifyReadinessRun validates the generated readiness evidence directory.
func (v *ReadinessVerifier) VerifyReadinessRun(runDir string) (*ReadinessSummary, error) {
	summaryPath := filepath.Join(runDir, readinessSummaryJSONName)
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("read readiness summary json: %w", err)
	}
	var summary ReadinessSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse readiness summary json: %w", err)
	}
	if summary.RunDirectory == "" {
		return nil, fmt.Errorf("summary missing run_directory")
	}
	if summary.RunDirectory != runDir {
		return nil, fmt.Errorf("summary run_directory %q does not match requested directory %q", summary.RunDirectory, runDir)
	}
	for _, name := range []string{readinessSummaryMarkdown, readinessTrackerMarkdown, readinessDecisionMarkdown, readinessInventoryText} {
		if !fileExists(filepath.Join(runDir, name)) {
			return nil, fmt.Errorf("missing generated artifact: %s", name)
		}
	}
	inventoryPath := filepath.Join(runDir, readinessInventoryText)
	inventoryData, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("read readiness inventory: %w", err)
	}
	inventoryEntries := filterNonEmpty(strings.Split(strings.ReplaceAll(string(inventoryData), "\r\n", "\n"), "\n"))
	actualEntries, err := readinessGeneratedFiles(runDir)
	if err != nil {
		return nil, fmt.Errorf("walk readiness inventory: %w", err)
	}
	if !stringSlicesEqual(inventoryEntries, actualEntries) {
		return nil, fmt.Errorf("inventory mismatch between evidence-inventory.txt and filesystem")
	}
	for _, gate := range summary.GateResults {
		if gate.LogPath == "" || gate.Status == "SKIPPED" {
			continue
		}
		if !fileExists(gate.LogPath) {
			return nil, fmt.Errorf("missing gate log: %s", gate.LogPath)
		}
	}
	if summary.OverallStatus == "PASS" {
		if !fileExists(summary.BundlePath) {
			return nil, fmt.Errorf("missing release bundle referenced by readiness summary: %s", summary.BundlePath)
		}
		if !fileExists(summary.BundleChecksumPath) {
			return nil, fmt.Errorf("missing release bundle checksum referenced by readiness summary: %s", summary.BundleChecksumPath)
		}
	}
	return &summary, nil
}

func renderReadinessSummaryMarkdown(summary *ReadinessSummary) string {
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

func renderReadinessTrackerMarkdown(summary *ReadinessSummary) string {
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

func renderReadinessDecisionMarkdown(summary *ReadinessSummary) string {
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

func writeJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func filterNonEmpty(items []string) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func appendReadinessNote(existing string, note string) string {
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
		return "-"
	}
	replacer := strings.NewReplacer("|", "\\|", "\n", " ")
	return replacer.Replace(value)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
