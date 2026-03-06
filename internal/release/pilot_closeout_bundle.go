// pilot_closeout_bundle.go - Pilot closeout bundle generation and verification.
//
// Purpose:
//
//	Assemble a single local evidence bundle for pilot closeout review from the
//	customer memo, validation records, and latest readiness evidence.
//
// Responsibilities:
//   - Copy closeout source documents into a timestamped bundle directory
//   - Include the referenced readiness evidence set in the bundle
//   - Render machine-readable and operator-readable bundle summaries
//   - Verify bundle inventory consistency
//
// Scope:
//   - Covers local artifact assembly only
//   - Does not mutate tracked pilot documents
//
// Usage:
//   - Called from `acpctl deploy pilot-closeout-bundle build`
//   - Called from `acpctl deploy pilot-closeout-bundle verify`
//
// Invariants/Assumptions:
//   - Bundles live under `demo/logs/pilot-closeout/`
//   - Input documents already exist and remain the source of truth
package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	pilotCloseoutSummaryJSON = "summary.json"
	pilotCloseoutSummaryMD   = "closeout-summary.md"
	pilotCloseoutInventory   = "bundle-inventory.txt"
	pilotCloseoutLatestRun   = "latest-run.txt"
)

// PilotCloseoutOptions describes the source inputs for a pilot closeout bundle.
type PilotCloseoutOptions struct {
	RepoRoot            string
	OutputRoot          string
	Customer            string
	PilotName           string
	Decision            string
	CharterPath         string
	AcceptanceMemoPath  string
	ValidationChecklist string
	OperatorChecklist   string
	ReadinessRunDir     string
}

// PilotCloseoutSummary describes one generated closeout bundle.
type PilotCloseoutSummary struct {
	RunID               string   `json:"run_id"`
	GeneratedAtUTC      string   `json:"generated_at_utc"`
	RepoRoot            string   `json:"repo_root"`
	RunDirectory        string   `json:"run_directory"`
	Customer            string   `json:"customer"`
	PilotName           string   `json:"pilot_name"`
	Decision            string   `json:"decision"`
	ReadinessRunDir     string   `json:"readiness_run_dir"`
	CharterPath         string   `json:"charter_path"`
	AcceptanceMemoPath  string   `json:"acceptance_memo_path"`
	ValidationChecklist string   `json:"validation_checklist_path"`
	OperatorChecklist   string   `json:"operator_checklist_path,omitempty"`
	GeneratedFiles      []string `json:"generated_files"`
}

// PilotCloseoutVerifier validates the generated closeout bundle directory.
type PilotCloseoutVerifier struct{}

// NewPilotCloseoutVerifier creates a bundle verifier.
func NewPilotCloseoutVerifier() *PilotCloseoutVerifier {
	return &PilotCloseoutVerifier{}
}

// BuildPilotCloseoutBundle assembles a timestamped pilot closeout bundle.
func BuildPilotCloseoutBundle(opts PilotCloseoutOptions) (*PilotCloseoutSummary, error) {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if strings.TrimSpace(opts.OutputRoot) == "" {
		opts.OutputRoot = filepath.Join(opts.RepoRoot, "demo", "logs", "pilot-closeout")
	}
	if strings.TrimSpace(opts.Customer) == "" {
		return nil, fmt.Errorf("customer is required")
	}
	if strings.TrimSpace(opts.PilotName) == "" {
		return nil, fmt.Errorf("pilot name is required")
	}
	if strings.TrimSpace(opts.Decision) == "" {
		opts.Decision = "PENDING_REVIEW"
	}
	if strings.TrimSpace(opts.CharterPath) == "" {
		return nil, fmt.Errorf("charter path is required")
	}
	if strings.TrimSpace(opts.AcceptanceMemoPath) == "" {
		return nil, fmt.Errorf("acceptance memo path is required")
	}
	if strings.TrimSpace(opts.ValidationChecklist) == "" {
		return nil, fmt.Errorf("validation checklist path is required")
	}
	if strings.TrimSpace(opts.ReadinessRunDir) == "" {
		readinessRunDir, err := ResolveLatestReadinessRun(filepath.Join(opts.RepoRoot, "demo", "logs", "evidence"))
		if err != nil {
			return nil, fmt.Errorf("resolve latest readiness run: %w", err)
		}
		opts.ReadinessRunDir = readinessRunDir
	}

	if _, err := NewReadinessVerifier().VerifyReadinessRun(opts.ReadinessRunDir); err != nil {
		return nil, fmt.Errorf("verify readiness run for bundle: %w", err)
	}

	nowUTC := readinessNow()
	runID := fmt.Sprintf("pilot-closeout-%s", nowUTC.Format("20060102T150405Z"))
	runDir := filepath.Join(opts.OutputRoot, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create pilot closeout directory: %w", err)
	}

	summary := &PilotCloseoutSummary{
		RunID:               runID,
		GeneratedAtUTC:      nowUTC.Format(time.RFC3339),
		RepoRoot:            opts.RepoRoot,
		RunDirectory:        runDir,
		Customer:            opts.Customer,
		PilotName:           opts.PilotName,
		Decision:            opts.Decision,
		ReadinessRunDir:     opts.ReadinessRunDir,
		CharterPath:         opts.CharterPath,
		AcceptanceMemoPath:  opts.AcceptanceMemoPath,
		ValidationChecklist: opts.ValidationChecklist,
		OperatorChecklist:   opts.OperatorChecklist,
	}

	if err := copyBundleInput(runDir, "documents/pilot-charter.md", opts.CharterPath); err != nil {
		return nil, err
	}
	if err := copyBundleInput(runDir, "documents/pilot-acceptance-memo.md", opts.AcceptanceMemoPath); err != nil {
		return nil, err
	}
	if err := copyBundleInput(runDir, "documents/pilot-customer-validation-checklist.md", opts.ValidationChecklist); err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.OperatorChecklist) != "" {
		if err := copyBundleInput(runDir, "documents/pilot-operator-handoff-checklist.md", opts.OperatorChecklist); err != nil {
			return nil, err
		}
	}
	if err := copyReadinessArtifacts(runDir, opts.ReadinessRunDir); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(runDir, pilotCloseoutSummaryMD), []byte(renderPilotCloseoutSummary(summary)), 0o644); err != nil {
		return nil, fmt.Errorf("write closeout summary markdown: %w", err)
	}
	files, err := readinessGeneratedFiles(runDir)
	if err != nil {
		return nil, fmt.Errorf("walk pilot closeout bundle: %w", err)
	}
	files = append(files, pilotCloseoutSummaryJSON)
	files = append(files, pilotCloseoutInventory)
	sort.Strings(files)
	summary.GeneratedFiles = files
	if err := writeJSON(filepath.Join(runDir, pilotCloseoutSummaryJSON), summary); err != nil {
		return nil, fmt.Errorf("write closeout summary json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, pilotCloseoutInventory), []byte(strings.Join(files, "\n")+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write closeout inventory: %w", err)
	}
	if err := writeLatestPilotCloseoutPointer(opts.OutputRoot, summary.RunDirectory); err != nil {
		return nil, err
	}
	return summary, nil
}

// VerifyPilotCloseoutBundle validates the generated bundle.
func (v *PilotCloseoutVerifier) VerifyPilotCloseoutBundle(runDir string) (*PilotCloseoutSummary, error) {
	data, err := os.ReadFile(filepath.Join(runDir, pilotCloseoutSummaryJSON))
	if err != nil {
		return nil, fmt.Errorf("read pilot closeout summary: %w", err)
	}
	var summary PilotCloseoutSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse pilot closeout summary: %w", err)
	}
	if strings.TrimSpace(summary.RunDirectory) == "" {
		return nil, fmt.Errorf("summary missing run_directory")
	}
	if summary.RunDirectory != runDir {
		return nil, fmt.Errorf("summary run_directory %q does not match requested directory %q", summary.RunDirectory, runDir)
	}
	for _, name := range []string{pilotCloseoutSummaryMD, pilotCloseoutInventory} {
		if !fileExists(filepath.Join(runDir, name)) {
			return nil, fmt.Errorf("missing closeout artifact: %s", name)
		}
	}
	inventoryData, err := os.ReadFile(filepath.Join(runDir, pilotCloseoutInventory))
	if err != nil {
		return nil, fmt.Errorf("read closeout inventory: %w", err)
	}
	expected := filterNonEmpty(strings.Split(strings.ReplaceAll(string(inventoryData), "\r\n", "\n"), "\n"))
	actual, err := readinessGeneratedFiles(runDir)
	if err != nil {
		return nil, fmt.Errorf("walk closeout inventory: %w", err)
	}
	if !stringSlicesEqual(expected, actual) {
		return nil, fmt.Errorf("inventory mismatch between %s and filesystem", pilotCloseoutInventory)
	}
	return &summary, nil
}

func copyBundleInput(runDir string, relativeDestination string, sourcePath string) error {
	if !fileExists(sourcePath) {
		return fmt.Errorf("bundle source file does not exist: %s", sourcePath)
	}
	destination := filepath.Join(runDir, filepath.FromSlash(relativeDestination))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create bundle destination dir: %w", err)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read bundle source %s: %w", sourcePath, err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		return fmt.Errorf("write bundle destination %s: %w", destination, err)
	}
	return nil
}

func copyReadinessArtifacts(runDir string, readinessRunDir string) error {
	artifacts := []string{
		readinessSummaryJSONName,
		readinessSummaryMarkdown,
		readinessTrackerMarkdown,
		readinessDecisionMarkdown,
		readinessInventoryText,
	}
	for _, name := range artifacts {
		if err := copyBundleInput(runDir, filepath.ToSlash(filepath.Join("evidence", name)), filepath.Join(readinessRunDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func writeLatestPilotCloseoutPointer(outputRoot string, runDir string) error {
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return fmt.Errorf("create pilot closeout output root: %w", err)
	}
	return os.WriteFile(filepath.Join(outputRoot, pilotCloseoutLatestRun), []byte(runDir+"\n"), 0o644)
}

func renderPilotCloseoutSummary(summary *PilotCloseoutSummary) string {
	var builder strings.Builder
	builder.WriteString("# Pilot Closeout Bundle Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Customer: `%s`\n", summary.Customer))
	builder.WriteString(fmt.Sprintf("- Pilot: `%s`\n", summary.PilotName))
	builder.WriteString(fmt.Sprintf("- Decision: `%s`\n", summary.Decision))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Bundle directory: `%s`\n", summary.RunDirectory))
	builder.WriteString(fmt.Sprintf("- Readiness run: `%s`\n", summary.ReadinessRunDir))
	builder.WriteString("\n## Included Documents\n\n")
	builder.WriteString("- `documents/pilot-charter.md`\n")
	builder.WriteString("- `documents/pilot-acceptance-memo.md`\n")
	builder.WriteString("- `documents/pilot-customer-validation-checklist.md`\n")
	if strings.TrimSpace(summary.OperatorChecklist) != "" {
		builder.WriteString("- `documents/pilot-operator-handoff-checklist.md`\n")
	}
	builder.WriteString("- `evidence/readiness-summary.md`\n")
	builder.WriteString("- `evidence/presentation-readiness-tracker.md`\n")
	builder.WriteString("- `evidence/go-no-go-decision.md`\n")
	builder.WriteString("- `evidence/evidence-inventory.txt`\n")
	return builder.String()
}
