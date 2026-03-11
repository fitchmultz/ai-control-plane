// cmd_generate_docs.go - Hidden reference-doc generation helpers.
//
// Purpose:
//   - Generate tracked reference artifacts from typed source-of-truth inputs.
//
// Responsibilities:
//   - Render CLI, support-matrix, approved-model, and detection references.
//   - Write deterministic tracked artifacts under docs/reference/.
//
// Scope:
//   - Local repository artifact generation only.
//
// Usage:
//   - Invoked internally by `make generate` via `acpctl __generate-docs`.
//
// Invariants/Assumptions:
//   - Generated output is deterministic for equivalent repo state.
//   - Hidden command is not part of the public operator surface.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/contracts"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

func hiddenGenerateDocsCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    "__generate-docs",
		Summary: "Hidden reference artifact generator",
		Hidden:  true,
		Run:     runGenerateDocs,
	})
}

func runGenerateDocs(_ context.Context, runCtx commandRunContext, _ any) int {
	artifacts, err := generatedReferenceArtifacts(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	for path, content := range artifacts {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: mkdir %s: %v\n", filepath.Dir(path), err)
			return exitcodes.ACPExitRuntime
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: write %s: %v\n", path, err)
			return exitcodes.ACPExitRuntime
		}
	}
	return exitcodes.ACPExitSuccess
}

func generatedReferenceArtifacts(repoRoot string) (map[string]string, error) {
	cliReference, err := renderACPCTLReference()
	if err != nil {
		return nil, err
	}
	supportReference, err := renderSupportMatrixReference(repoRoot)
	if err != nil {
		return nil, err
	}
	modelReference, err := renderApprovedModelsReference(repoRoot)
	if err != nil {
		return nil, err
	}
	detectionReference, err := renderDetectionReference(repoRoot)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		repopath.FromRepoRoot(repoRoot, "docs", "reference", "acpctl.md"):          cliReference,
		repopath.FromRepoRoot(repoRoot, "docs", "reference", "support-matrix.md"):  supportReference,
		repopath.FromRepoRoot(repoRoot, "docs", "reference", "approved-models.md"): modelReference,
		repopath.FromRepoRoot(repoRoot, "docs", "reference", "detections.md"):      detectionReference,
	}, nil
}

func renderACPCTLReference() (string, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("# ACPCTL Reference\n\n")
	buf.WriteString("> Generated from the typed command registry. Do not edit manually.\n\n")
	buf.WriteString("`acpctl` is the typed implementation engine for supported host-first workflows. `make` remains the primary human operator UX.\n\n")
	buf.WriteString("## Top-Level Commands\n\n")
	for _, root := range spec.VisibleRoots {
		buf.WriteString("### `" + root.Name + "`\n\n")
		if root.Summary != "" {
			buf.WriteString(root.Summary + ".\n\n")
		}
		if len(root.Children) > 0 {
			buf.WriteString("| Subcommand | Summary |\n| --- | --- |\n")
			for _, child := range root.Children {
				if child.Hidden {
					continue
				}
				buf.WriteString("| `" + child.Name + "` | " + escapeTable(child.Summary) + " |\n")
			}
			buf.WriteString("\n")
		}
		if len(root.Examples) > 0 {
			buf.WriteString("Examples:\n\n```bash\n")
			for _, example := range root.Examples {
				buf.WriteString(strings.ReplaceAll(example, "acpctl ", "./scripts/acpctl.sh ") + "\n")
			}
			buf.WriteString("```\n\n")
		}
	}
	return buf.String(), nil
}

func renderSupportMatrixReference(repoRoot string) (string, error) {
	matrix, err := catalog.LoadSupportMatrix(repopath.FromRepoRoot(repoRoot, "docs", "support-matrix.yaml"))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("# Support Matrix\n\n")
	buf.WriteString("> Generated from `docs/support-matrix.yaml`. Do not edit manually.\n\n")
	buf.WriteString("## Supported Surfaces\n\n")
	buf.WriteString("| Surface | Summary | Validation |\n| --- | --- | --- |\n")
	for _, surface := range matrix.SupportedSurfaces() {
		buf.WriteString("| " + escapeTable(surface.Label) + " | " + escapeTable(surface.Summary) + " | " + escapeTable(strings.Join(surface.Validation, ", ")) + " |\n")
	}
	buf.WriteString("\n## Incubating Surfaces\n\n")
	buf.WriteString("| Surface | Summary | Validation |\n| --- | --- | --- |\n")
	for _, surface := range matrix.IncubatingSurfaces() {
		buf.WriteString("| " + escapeTable(surface.Label) + " | " + escapeTable(surface.Summary) + " | " + escapeTable(strings.Join(surface.Validation, ", ")) + " |\n")
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

func renderApprovedModelsReference(repoRoot string) (string, error) {
	models, err := catalog.LoadModelCatalog(repopath.DemoConfigPath(repoRoot, "model_catalog.yaml"))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("# Approved Models\n\n")
	buf.WriteString("> Generated from `demo/config/model_catalog.yaml`. Do not edit manually.\n\n")
	buf.WriteString("## Online Aliases\n\n")
	buf.WriteString("| Alias | Upstream Model | Credential Env | Managed UI |\n| --- | --- | --- | --- |\n")
	for _, model := range models.OnlineModels {
		buf.WriteString("| `" + model.Alias + "` | `" + model.UpstreamModel + "` | `" + model.CredentialEnv + "` | " + yesNo(model.ManagedUIDefault) + " |\n")
	}
	buf.WriteString("\n## Offline Aliases\n\n")
	buf.WriteString("| Alias | Upstream Model |\n| --- | --- |\n")
	for _, model := range models.OfflineModels {
		buf.WriteString("| `" + model.Alias + "` | `" + model.UpstreamModel + "` |\n")
	}
	buf.WriteString("\n## Managed Browser Defaults\n\n")
	for _, alias := range models.ManagedUIDefaultAliases() {
		buf.WriteString("- `" + alias + "`\n")
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

func renderDetectionReference(repoRoot string) (string, error) {
	detections, err := contracts.LoadDetectionRulesFile(repopath.DemoConfigPath(repoRoot, "detection_rules.yaml"))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("# Detection Rules Reference\n\n")
	buf.WriteString("> Generated from `demo/config/detection_rules.yaml`. Do not edit manually.\n\n")
	buf.WriteString("| Rule ID | Name | Severity | Enabled | Status | Coverage | Expected Signal |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, rule := range detections.DetectionRules {
		buf.WriteString(
			"| `" + rule.RuleID + "` | " +
				escapeTable(rule.Name) + " | " +
				escapeTable(rule.Severity) + " | " +
				yesNo(rule.Enabled) + " | " +
				escapeTable(rule.OperationalStatus) + " | " +
				escapeTable(rule.CoverageTier) + " | " +
				escapeTable(singleLine(rule.ExpectedSignal)) + " |\n",
		)
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

func escapeTable(value string) string {
	return strings.ReplaceAll(singleLine(value), "|", "\\|")
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
