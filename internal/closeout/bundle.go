// bundle.go - Pilot closeout bundle generation and verification.
//
// Purpose:
//
//	Assemble a single local evidence bundle for pilot closeout review from the
//	customer memo, validation records, and readiness evidence.
//
// Responsibilities:
//   - Copy closeout source documents into a timestamped bundle directory.
//   - Include the referenced readiness evidence set in the bundle.
//   - Render machine-readable and operator-readable bundle summaries.
//   - Verify bundle inventory consistency.
//
// Scope:
//   - Covers local artifact assembly only.
//   - Does not mutate tracked pilot documents.
//
// Usage:
//   - Called from `acpctl deploy pilot-closeout-bundle build`.
//   - Called from `acpctl deploy pilot-closeout-bundle verify`.
//
// Invariants/Assumptions:
//   - Bundles live under `demo/logs/pilot-closeout/`.
//   - Input documents already exist and remain the source of truth.
package closeout

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
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

const (
	SummaryJSONName   = "summary.json"
	SummaryMarkdown   = "closeout-summary.md"
	InventoryFileName = "bundle-inventory.txt"
	LatestRunPointer  = "latest-run.txt"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

// Options describes the source inputs for a pilot closeout bundle.
type Options struct {
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

// Summary describes one generated closeout bundle.
type Summary struct {
	RunID               string   `json:"run_id"`
	GeneratedAtUTC      string   `json:"generated_at_utc"`
	RepoRoot            string   `json:"repo_root"`
	RunDirectory        string   `json:"run_directory"`
	Customer            string   `json:"customer"`
	PilotName           string   `json:"pilot_name"`
	Decision            string   `json:"decision"`
	ReadinessRunDir     string   `json:"readiness_run_dir"`
	ReadinessArtifacts  []string `json:"readiness_artifacts"`
	CharterPath         string   `json:"charter_path"`
	AcceptanceMemoPath  string   `json:"acceptance_memo_path"`
	ValidationChecklist string   `json:"validation_checklist_path"`
	OperatorChecklist   string   `json:"operator_checklist_path,omitempty"`
	GeneratedFiles      []string `json:"generated_files"`
}

// Verifier validates the generated closeout bundle directory.
type Verifier struct{}

// NewVerifier creates a closeout bundle verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// Build assembles a timestamped pilot closeout bundle.
func Build(ctx context.Context, opts Options) (summary *Summary, err error) {
	if textutil.IsBlank(opts.OutputRoot) {
		opts.OutputRoot = filepath.Join(opts.RepoRoot, "demo", "logs", "pilot-closeout")
	}
	logger := logging.WorkflowLogger(ctx,
		slog.String("component", "closeout"),
		slog.String("workflow", "pilot_closeout_bundle_build"),
	)
	logging.WorkflowStart(logger,
		slog.String("output_root", opts.OutputRoot),
		slog.String("customer", opts.Customer),
		slog.String("pilot_name", opts.PilotName),
	)
	defer func() {
		if err != nil {
			logging.WorkflowFailure(logger, err)
		}
	}()

	if err = textutil.RequireNonBlank("repo root", opts.RepoRoot); err != nil {
		return nil, err
	}
	if err = textutil.RequireNonBlank("customer", opts.Customer); err != nil {
		return nil, err
	}
	if err = textutil.RequireNonBlank("pilot name", opts.PilotName); err != nil {
		return nil, err
	}
	if textutil.IsBlank(opts.Decision) {
		opts.Decision = "PENDING_REVIEW"
	}
	if err = textutil.RequireNonBlank("charter path", opts.CharterPath); err != nil {
		return nil, err
	}
	if err = textutil.RequireNonBlank("acceptance memo path", opts.AcceptanceMemoPath); err != nil {
		return nil, err
	}
	if err = textutil.RequireNonBlank("validation checklist path", opts.ValidationChecklist); err != nil {
		return nil, err
	}
	if textutil.IsBlank(opts.ReadinessRunDir) {
		var readinessRunDir string
		readinessRunDir, err = readiness.ResolveLatestRun(filepath.Join(opts.RepoRoot, "demo", "logs", "evidence"))
		if err != nil {
			return nil, fmt.Errorf("resolve latest readiness run: %w", err)
		}
		opts.ReadinessRunDir = readinessRunDir
	}

	if _, err = readiness.NewVerifier().VerifyRun(ctx, opts.ReadinessRunDir); err != nil {
		return nil, fmt.Errorf("verify readiness run for bundle: %w", err)
	}

	var run *artifactrun.Run
	run, err = artifactrun.Create(opts.OutputRoot, "pilot-closeout", nowUTC())
	if err != nil {
		return nil, err
	}
	readinessArtifacts, err := copyReadinessArtifacts(run.Directory, opts.ReadinessRunDir)
	if err != nil {
		return nil, err
	}
	summary = &Summary{
		RunID:               run.ID,
		GeneratedAtUTC:      nowUTC().Format(time.RFC3339),
		RepoRoot:            opts.RepoRoot,
		RunDirectory:        run.Directory,
		Customer:            opts.Customer,
		PilotName:           opts.PilotName,
		Decision:            opts.Decision,
		ReadinessRunDir:     opts.ReadinessRunDir,
		ReadinessArtifacts:  readinessArtifacts,
		CharterPath:         opts.CharterPath,
		AcceptanceMemoPath:  opts.AcceptanceMemoPath,
		ValidationChecklist: opts.ValidationChecklist,
		OperatorChecklist:   opts.OperatorChecklist,
	}

	if err = copyBundleInput(run.Directory, "documents/pilot-charter.md", opts.CharterPath); err != nil {
		return nil, err
	}
	if err = copyBundleInput(run.Directory, "documents/pilot-acceptance-memo.md", opts.AcceptanceMemoPath); err != nil {
		return nil, err
	}
	if err = copyBundleInput(run.Directory, "documents/pilot-customer-validation-checklist.md", opts.ValidationChecklist); err != nil {
		return nil, err
	}
	if !textutil.IsBlank(opts.OperatorChecklist) {
		if err = copyBundleInput(run.Directory, "documents/pilot-operator-handoff-checklist.md", opts.OperatorChecklist); err != nil {
			return nil, err
		}
	}
	if err = persistRun(opts.OutputRoot, summary); err != nil {
		return nil, err
	}
	logging.WorkflowComplete(logger,
		slog.String("run_id", summary.RunID),
		slog.String("run_directory", summary.RunDirectory),
		slog.String("decision", summary.Decision),
	)
	return summary, nil
}

// ResolveLatestRun resolves the most recent pilot closeout bundle pointer.
func ResolveLatestRun(outputRoot string) (string, error) {
	return artifactrun.ResolveLatest(outputRoot, LatestRunPointer)
}

func persistRun(outputRoot string, summary *Summary) error {
	if err := writeArtifacts(summary.RunDirectory, summary); err != nil {
		return err
	}
	files, err := artifactrun.Finalize(summary.RunDirectory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryFileName,
		LatestPointers: []string{LatestRunPointer},
	})
	if err != nil {
		return err
	}
	summary.GeneratedFiles = files
	return writeArtifacts(summary.RunDirectory, summary)
}

func writeArtifacts(runDir string, summary *Summary) error {
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		return fmt.Errorf("write closeout summary json: %w", err)
	}
	return artifactrun.WriteArtifacts(runDir, []artifactrun.Artifact{{
		Path: SummaryMarkdown,
		Body: []byte(renderSummary(summary)),
		Perm: fsutil.PrivateFilePerm,
	}})
}

// VerifyRun validates the generated bundle.
func (v *Verifier) VerifyRun(ctx context.Context, runDir string) (summary *Summary, err error) {
	logger := logging.WorkflowLogger(ctx,
		slog.String("component", "closeout"),
		slog.String("workflow", "pilot_closeout_bundle_verify"),
	)
	logging.WorkflowStart(logger, slog.String("run_directory", runDir))
	defer func() {
		if err != nil {
			logging.WorkflowFailure(logger, err, slog.String("run_directory", runDir))
		}
	}()

	data, err := os.ReadFile(filepath.Join(runDir, SummaryJSONName))
	if err != nil {
		return nil, fmt.Errorf("read pilot closeout summary: %w", err)
	}
	if err = json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse pilot closeout summary: %w", err)
	}
	if textutil.IsBlank(summary.RunDirectory) {
		return nil, fmt.Errorf("summary missing run_directory")
	}
	if summary.RunDirectory != runDir {
		return nil, fmt.Errorf("summary run_directory %q does not match requested directory %q", summary.RunDirectory, runDir)
	}
	if err = artifactrun.Verify(runDir, artifactrun.VerifyOptions{
		InventoryName: InventoryFileName,
		RequiredFiles: []string{SummaryJSONName, SummaryMarkdown, InventoryFileName},
	}); err != nil {
		return nil, err
	}
	logging.WorkflowComplete(logger,
		slog.String("run_id", summary.RunID),
		slog.String("run_directory", summary.RunDirectory),
		slog.String("decision", summary.Decision),
	)
	return summary, nil
}

func copyBundleInput(runDir string, relativeDestination string, sourcePath string) error {
	if !artifactrun.FileExists(sourcePath) {
		return fmt.Errorf("bundle source file does not exist: %s", sourcePath)
	}
	destination := filepath.Join(runDir, filepath.FromSlash(relativeDestination))
	if err := fsutil.EnsurePrivateDir(filepath.Dir(destination)); err != nil {
		return fmt.Errorf("create bundle destination dir: %w", err)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read bundle source %s: %w", sourcePath, err)
	}
	if err := fsutil.AtomicWritePrivateFile(destination, data); err != nil {
		return fmt.Errorf("write bundle destination %s: %w", destination, err)
	}
	return nil
}

func copyReadinessArtifacts(runDir string, readinessRunDir string) ([]string, error) {
	artifacts, err := artifactrun.ReadInventory(readinessRunDir, readiness.InventoryFileName)
	if err != nil {
		return nil, fmt.Errorf("read readiness inventory: %w", err)
	}
	copied := make([]string, 0, len(artifacts))
	for _, name := range artifacts {
		sourcePath := filepath.Join(readinessRunDir, filepath.FromSlash(name))
		if !artifactrun.FileExists(sourcePath) {
			return nil, fmt.Errorf("missing readiness artifact: %s", sourcePath)
		}
		relativeDestination := filepath.ToSlash(filepath.Join("evidence", name))
		if err := copyBundleInput(runDir, relativeDestination, sourcePath); err != nil {
			return nil, err
		}
		copied = append(copied, relativeDestination)
	}
	return copied, nil
}

func renderSummary(summary *Summary) string {
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
	if !textutil.IsBlank(summary.OperatorChecklist) {
		builder.WriteString("- `documents/pilot-operator-handoff-checklist.md`\n")
	}
	builder.WriteString("\n## Included Readiness Evidence\n\n")
	for _, name := range summary.ReadinessArtifacts {
		builder.WriteString(fmt.Sprintf("- `%s`\n", name))
	}
	return builder.String()
}
