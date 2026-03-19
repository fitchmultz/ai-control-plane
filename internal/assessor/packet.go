// packet.go - Assessor packet generation and verification.
//
// Purpose:
//
//	Assemble a single ACP-native assessor handoff packet that packages the
//	canonical external-review briefing documents with current verified evidence.
//
// Responsibilities:
//   - Copy the canonical reviewer documents into a timestamped packet.
//   - Include a verified readiness evidence run plus its referenced release bundle.
//   - Render machine-readable and human-readable packet summaries.
//   - Enforce preparation-only truth: roadmap item #22 remains open.
//
// Scope:
//   - Local packet assembly and verification only.
//   - Does not perform or claim a completed external assessment.
//
// Usage:
//   - Called from `acpctl deploy assessor-packet build`.
//   - Called from `acpctl deploy assessor-packet verify`.
//
// Invariants/Assumptions:
//   - Packets live under `demo/logs/assessor-packet/`.
//   - Roadmap item #22 stays open until a real outside assessment exists.
package assessor

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
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

const (
	SummaryJSONName       = "summary.json"
	SummaryMarkdownName   = "assessor-summary.md"
	InventoryFileName     = "packet-inventory.txt"
	LatestRunPointerName  = "latest-run.txt"
	roadmapItemID         = 22
	roadmapItemStatusOpen = "OPEN"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

// Options describes the source inputs for an assessor packet.
type Options struct {
	RepoRoot        string
	OutputRoot      string
	ReadinessRunDir string
}

// ReviewerDocument captures one canonical tracked document included in the packet.
type ReviewerDocument struct {
	Title      string `json:"title"`
	Why        string `json:"why"`
	SourcePath string `json:"source_path"`
	PacketPath string `json:"packet_path"`
}

// Summary describes one generated assessor packet.
type Summary struct {
	RunID                           string             `json:"run_id"`
	GeneratedAtUTC                  string             `json:"generated_at_utc"`
	RepoRoot                        string             `json:"repo_root"`
	RunDirectory                    string             `json:"run_directory"`
	RoadmapItemID                   int                `json:"roadmap_item_id"`
	RoadmapItemStatus               string             `json:"roadmap_item_status"`
	PreparationOnly                 bool               `json:"preparation_only"`
	ExternalAssessmentCompleted     bool               `json:"external_assessment_completed"`
	BundleVersion                   string             `json:"bundle_version"`
	ReadinessRunDir                 string             `json:"readiness_run_dir"`
	ReadinessOverallStatus          string             `json:"readiness_overall_status"`
	ReadinessArtifacts              []string           `json:"readiness_artifacts"`
	ReleaseBundleSourcePath         string             `json:"release_bundle_source_path"`
	ReleaseBundleChecksumSourcePath string             `json:"release_bundle_checksum_source_path"`
	ReleaseBundlePacketPath         string             `json:"release_bundle_packet_path"`
	ReleaseBundleChecksumPacketPath string             `json:"release_bundle_checksum_packet_path"`
	ReviewerDocuments               []ReviewerDocument `json:"reviewer_documents"`
	GeneratedFiles                  []string           `json:"generated_files"`
}

// Verifier validates the generated assessor packet directory.
type Verifier struct{}

type reviewerDocumentSpec struct {
	Title            string
	Why              string
	RepoRelativePath string
}

var canonicalReviewerDocuments = []reviewerDocumentSpec{
	{
		Title:            "Threat model and security whitepaper",
		Why:              "Explains architecture, trust boundaries, abuse paths, mitigations, and residual risks.",
		RepoRelativePath: "docs/security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md",
	},
	{
		Title:            "Compliance crosswalk",
		Why:              "Maps ACP evidence and ownership to SOC 2, ISO 27001, and NIST-style controls.",
		RepoRelativePath: "docs/COMPLIANCE_CROSSWALK.md",
	},
	{
		Title:            "Go-to-market scope",
		Why:              "Defines validated, conditionally ready, and not-yet-validated claim boundaries.",
		RepoRelativePath: "docs/GO_TO_MARKET_SCOPE.md",
	},
	{
		Title:            "Known limitations register",
		Why:              "Shows current material gaps and open findings.",
		RepoRelativePath: "docs/KNOWN_LIMITATIONS.md",
	},
	{
		Title:            "CVE governance policy",
		Why:              "Shows how vulnerabilities are triaged, accepted temporarily, reviewed, and communicated.",
		RepoRelativePath: "docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md",
	},
	{
		Title:            "Shared responsibility model",
		Why:              "Makes customer vs ACP control ownership explicit.",
		RepoRelativePath: "docs/SHARED_RESPONSIBILITY_MODEL.md",
	},
	{
		Title:            "Pilot/control ownership matrix",
		Why:              "Clarifies implementation and operating ownership lines.",
		RepoRelativePath: "docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md",
	},
	{
		Title:            "Evidence workflow",
		Why:              "Shows how current local readiness proof is regenerated.",
		RepoRelativePath: "docs/release/READINESS_EVIDENCE_WORKFLOW.md",
	},
	{
		Title:            "Evidence map",
		Why:              "Connects repo claims to commands and source artifacts.",
		RepoRelativePath: "docs/evidence/EVIDENCE_MAP.md",
	},
	{
		Title:            "Go/No-Go criteria",
		Why:              "Shows release decision rules and the still-open independent-review expectation.",
		RepoRelativePath: "docs/release/GO_NO_GO.md",
	},
}

// NewVerifier creates an assessor packet verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// Build assembles a timestamped assessor packet.
func Build(ctx context.Context, opts Options) (summary *Summary, err error) {
	if err = textutil.RequireNonBlank("repo root", opts.RepoRoot); err != nil {
		return nil, err
	}
	if textutil.IsBlank(opts.OutputRoot) {
		opts.OutputRoot = repopath.DemoLogsPath(opts.RepoRoot, "assessor-packet")
	}

	logger := logging.WorkflowLogger(ctx,
		slog.String("component", "assessor"),
		slog.String("workflow", "assessor_packet_build"),
	)
	logging.WorkflowStart(logger,
		slog.String("output_root", opts.OutputRoot),
		slog.String("readiness_run_dir", opts.ReadinessRunDir),
	)
	defer func() {
		if err != nil {
			logging.WorkflowFailure(logger, err)
		}
	}()

	if textutil.IsBlank(opts.ReadinessRunDir) {
		opts.ReadinessRunDir, err = readiness.ResolveLatestSuccessRun(repopath.DemoLogsPath(opts.RepoRoot, "evidence"))
		if err != nil {
			return nil, fmt.Errorf("resolve latest successful readiness run: %w", err)
		}
	}

	readinessSummary, err := readiness.NewVerifier().VerifyRun(ctx, opts.ReadinessRunDir)
	if err != nil {
		return nil, fmt.Errorf("verify readiness run for assessor packet: %w", err)
	}
	if readinessSummary.OverallStatus != "PASS" {
		return nil, fmt.Errorf("assessor packet requires a passing readiness run; got %s", readinessSummary.OverallStatus)
	}

	run, err := artifactrun.Create(opts.OutputRoot, "assessor-packet", nowUTC())
	if err != nil {
		return nil, err
	}
	reviewerDocuments, err := copyReviewerDocuments(run.Directory, opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	readinessArtifacts, err := copyReadinessArtifacts(run.Directory, opts.ReadinessRunDir)
	if err != nil {
		return nil, err
	}
	releaseBundlePacketPath, releaseChecksumPacketPath, err := copyReleaseBundleArtifacts(run.Directory, readinessSummary)
	if err != nil {
		return nil, err
	}

	summary = &Summary{
		RunID:                           run.ID,
		GeneratedAtUTC:                  nowUTC().Format(time.RFC3339),
		RepoRoot:                        opts.RepoRoot,
		RunDirectory:                    run.Directory,
		RoadmapItemID:                   roadmapItemID,
		RoadmapItemStatus:               roadmapItemStatusOpen,
		PreparationOnly:                 true,
		ExternalAssessmentCompleted:     false,
		BundleVersion:                   readinessSummary.BundleVersion,
		ReadinessRunDir:                 opts.ReadinessRunDir,
		ReadinessOverallStatus:          readinessSummary.OverallStatus,
		ReadinessArtifacts:              readinessArtifacts,
		ReleaseBundleSourcePath:         readinessSummary.BundlePath,
		ReleaseBundleChecksumSourcePath: readinessSummary.BundleChecksumPath,
		ReleaseBundlePacketPath:         releaseBundlePacketPath,
		ReleaseBundleChecksumPacketPath: releaseChecksumPacketPath,
		ReviewerDocuments:               reviewerDocuments,
	}
	if err = persistRun(opts.OutputRoot, summary); err != nil {
		return nil, err
	}
	logging.WorkflowComplete(logger,
		slog.String("run_id", summary.RunID),
		slog.String("run_directory", summary.RunDirectory),
		slog.String("readiness_run_dir", summary.ReadinessRunDir),
	)
	return summary, nil
}

// ResolveLatestRun resolves the most recent assessor packet pointer.
func ResolveLatestRun(outputRoot string) (string, error) {
	return artifactrun.ResolveLatest(outputRoot, LatestRunPointerName)
}

func persistRun(outputRoot string, summary *Summary) error {
	if err := writeArtifacts(summary.RunDirectory, summary); err != nil {
		return err
	}
	files, err := artifactrun.Finalize(summary.RunDirectory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryFileName,
		LatestPointers: []string{LatestRunPointerName},
	})
	if err != nil {
		return err
	}
	summary.GeneratedFiles = files
	return writeArtifacts(summary.RunDirectory, summary)
}

func writeArtifacts(runDir string, summary *Summary) error {
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		return fmt.Errorf("write assessor summary json: %w", err)
	}
	return artifactrun.WriteArtifacts(runDir, []artifactrun.Artifact{{
		Path: SummaryMarkdownName,
		Body: []byte(renderSummary(summary)),
		Perm: fsutil.PrivateFilePerm,
	}})
}

// VerifyRun validates the generated assessor packet.
func (v *Verifier) VerifyRun(ctx context.Context, runDir string) (summary *Summary, err error) {
	logger := logging.WorkflowLogger(ctx,
		slog.String("component", "assessor"),
		slog.String("workflow", "assessor_packet_verify"),
	)
	logging.WorkflowStart(logger, slog.String("run_directory", runDir))
	defer func() {
		if err != nil {
			logging.WorkflowFailure(logger, err, slog.String("run_directory", runDir))
		}
	}()

	data, err := os.ReadFile(filepath.Join(runDir, SummaryJSONName))
	if err != nil {
		return nil, fmt.Errorf("read assessor summary: %w", err)
	}
	if err = json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse assessor summary: %w", err)
	}
	if textutil.IsBlank(summary.RunDirectory) {
		return nil, fmt.Errorf("summary missing run_directory")
	}
	if summary.RunDirectory != runDir {
		return nil, fmt.Errorf("summary run_directory %q does not match requested directory %q", summary.RunDirectory, runDir)
	}
	if summary.RoadmapItemID != roadmapItemID {
		return nil, fmt.Errorf("summary roadmap_item_id = %d, want %d", summary.RoadmapItemID, roadmapItemID)
	}
	if summary.RoadmapItemStatus != roadmapItemStatusOpen {
		return nil, fmt.Errorf("summary roadmap_item_status must remain %q", roadmapItemStatusOpen)
	}
	if !summary.PreparationOnly {
		return nil, fmt.Errorf("summary must record preparation_only=true")
	}
	if summary.ExternalAssessmentCompleted {
		return nil, fmt.Errorf("summary must record external_assessment_completed=false")
	}
	if summary.ReadinessOverallStatus != "PASS" {
		return nil, fmt.Errorf("summary readiness_overall_status must be PASS")
	}
	if len(summary.ReviewerDocuments) != len(canonicalReviewerDocuments) {
		return nil, fmt.Errorf("summary reviewer_documents count = %d, want %d", len(summary.ReviewerDocuments), len(canonicalReviewerDocuments))
	}
	if err = artifactrun.Verify(runDir, artifactrun.VerifyOptions{
		InventoryName: InventoryFileName,
		RequiredFiles: []string{SummaryJSONName, SummaryMarkdownName, InventoryFileName},
	}); err != nil {
		return nil, err
	}
	for _, doc := range summary.ReviewerDocuments {
		if textutil.IsBlank(doc.PacketPath) {
			return nil, fmt.Errorf("reviewer document missing packet_path")
		}
		if !artifactrun.FileExists(filepath.Join(runDir, filepath.FromSlash(doc.PacketPath))) {
			return nil, fmt.Errorf("missing reviewer document: %s", doc.PacketPath)
		}
	}
	for _, artifact := range summary.ReadinessArtifacts {
		if !artifactrun.FileExists(filepath.Join(runDir, filepath.FromSlash(artifact))) {
			return nil, fmt.Errorf("missing readiness artifact: %s", artifact)
		}
	}
	if !artifactrun.FileExists(filepath.Join(runDir, filepath.FromSlash(summary.ReleaseBundlePacketPath))) {
		return nil, fmt.Errorf("missing release bundle artifact: %s", summary.ReleaseBundlePacketPath)
	}
	if !artifactrun.FileExists(filepath.Join(runDir, filepath.FromSlash(summary.ReleaseBundleChecksumPacketPath))) {
		return nil, fmt.Errorf("missing release bundle checksum artifact: %s", summary.ReleaseBundleChecksumPacketPath)
	}

	logging.WorkflowComplete(logger,
		slog.String("run_id", summary.RunID),
		slog.String("run_directory", summary.RunDirectory),
		slog.String("roadmap_item_status", summary.RoadmapItemStatus),
	)
	return summary, nil
}

func copyReviewerDocuments(runDir string, repoRoot string) ([]ReviewerDocument, error) {
	documents := make([]ReviewerDocument, 0, len(canonicalReviewerDocuments))
	for _, spec := range canonicalReviewerDocuments {
		sourcePath := filepath.Join(repoRoot, filepath.FromSlash(spec.RepoRelativePath))
		if err := copyPacketInput(runDir, spec.RepoRelativePath, sourcePath); err != nil {
			return nil, err
		}
		documents = append(documents, ReviewerDocument{
			Title:      spec.Title,
			Why:        spec.Why,
			SourcePath: sourcePath,
			PacketPath: spec.RepoRelativePath,
		})
	}
	return documents, nil
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
		relativeDestination := filepath.ToSlash(filepath.Join("evidence", "readiness", name))
		if err := copyPacketInput(runDir, relativeDestination, sourcePath); err != nil {
			return nil, err
		}
		copied = append(copied, relativeDestination)
	}
	return copied, nil
}

func copyReleaseBundleArtifacts(runDir string, readinessSummary *readiness.Summary) (string, string, error) {
	if textutil.IsBlank(readinessSummary.BundlePath) {
		return "", "", fmt.Errorf("readiness summary missing bundle_path")
	}
	if textutil.IsBlank(readinessSummary.BundleChecksumPath) {
		return "", "", fmt.Errorf("readiness summary missing bundle_checksum_path")
	}
	bundlePacketPath := filepath.ToSlash(filepath.Join("evidence", "release-bundle", filepath.Base(readinessSummary.BundlePath)))
	checksumPacketPath := filepath.ToSlash(filepath.Join("evidence", "release-bundle", filepath.Base(readinessSummary.BundleChecksumPath)))
	if err := copyPacketInput(runDir, bundlePacketPath, readinessSummary.BundlePath); err != nil {
		return "", "", err
	}
	if err := copyPacketInput(runDir, checksumPacketPath, readinessSummary.BundleChecksumPath); err != nil {
		return "", "", err
	}
	return bundlePacketPath, checksumPacketPath, nil
}

func copyPacketInput(runDir string, relativeDestination string, sourcePath string) error {
	if !artifactrun.FileExists(sourcePath) {
		return fmt.Errorf("packet source file does not exist: %s", sourcePath)
	}
	destination := filepath.Join(runDir, filepath.FromSlash(relativeDestination))
	if err := fsutil.EnsurePrivateDir(filepath.Dir(destination)); err != nil {
		return fmt.Errorf("create packet destination dir: %w", err)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read packet source %s: %w", sourcePath, err)
	}
	if err := fsutil.AtomicWritePrivateFile(destination, data); err != nil {
		return fmt.Errorf("write packet destination %s: %w", destination, err)
	}
	return nil
}

func renderSummary(summary *Summary) string {
	var builder strings.Builder
	builder.WriteString("# Assessor Packet Summary\n\n")
	builder.WriteString("> Preparation packet only. No external security review, architecture review, audit, certification, or equivalent outside assessment has been completed from this repository. Roadmap item `#22` remains open.\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Packet directory: `%s`\n", summary.RunDirectory))
	builder.WriteString(fmt.Sprintf("- Roadmap item: `#%d`\n", summary.RoadmapItemID))
	builder.WriteString(fmt.Sprintf("- Roadmap item status: `%s`\n", summary.RoadmapItemStatus))
	builder.WriteString(fmt.Sprintf("- Preparation only: `%t`\n", summary.PreparationOnly))
	builder.WriteString(fmt.Sprintf("- External assessment completed: `%t`\n", summary.ExternalAssessmentCompleted))
	builder.WriteString(fmt.Sprintf("- Included readiness run: `%s`\n", summary.ReadinessRunDir))
	builder.WriteString(fmt.Sprintf("- Included bundle version: `%s`\n", summary.BundleVersion))
	builder.WriteString(fmt.Sprintf("- Included release bundle: `%s`\n", summary.ReleaseBundlePacketPath))
	builder.WriteString(fmt.Sprintf("- Included release checksum: `%s`\n", summary.ReleaseBundleChecksumPacketPath))
	builder.WriteString("\n## Canonical Reviewer Documents\n\n")
	for _, doc := range summary.ReviewerDocuments {
		builder.WriteString(fmt.Sprintf("- `%s` — %s\n", doc.PacketPath, doc.Why))
	}
	builder.WriteString("\n## Included Readiness Evidence\n\n")
	for _, artifact := range summary.ReadinessArtifacts {
		builder.WriteString(fmt.Sprintf("- `%s`\n", artifact))
	}
	builder.WriteString("\n## Generated Files\n\n")
	for _, file := range summary.GeneratedFiles {
		builder.WriteString(fmt.Sprintf("- `%s`\n", file))
	}
	builder.WriteString("\n## Reviewer Handling Note\n\n")
	builder.WriteString("Use this packet as ACP-native assessor handoff material only. It packages current tracked documentation plus local generated evidence into one verifiable directory, but it does not itself satisfy roadmap item `#22`.\n")
	return builder.String()
}
