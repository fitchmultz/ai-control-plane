// cmd_pilot_closeout_bundle_test.go - Tests for the pilot closeout bundle CLI.
//
// Purpose:
//
//	Verify the closeout-bundle command parses arguments and invokes the
//	internal release workflow with stable exit behavior.
//
// Responsibilities:
//   - Verify usage/help behavior
//   - Verify bundle build success path
//
// Scope:
//   - Covers command-layer behavior only
//   - Does not exercise the real filesystem bundle assembly
//
// Usage:
//   - Run via `go test ./cmd/acpctl`
//
// Invariants/Assumptions:
//   - Tests use a temporary repository root and stubbed release workflows
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunPilotCloseoutBundleCommand_NoArgsReturnsUsage(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := runPilotCloseoutBundleCommand(context.Background(), nil, stdout, stderr)
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit code, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stdout), "Usage: acpctl deploy pilot-closeout-bundle") {
		t.Fatalf("expected help output, got %s", readFile(t, stdout))
	}
}

func TestRunPilotCloseoutBundleBuild_Succeeds(t *testing.T) {
	repoRoot := t.TempDir()
	charter := writeFileForPath(t, repoRoot, "docs/charter.md", "# Charter\n")
	acceptance := writeFileForPath(t, repoRoot, "docs/acceptance.md", "# Acceptance\n")
	checklist := writeFileForPath(t, repoRoot, "docs/checklist.md", "# Checklist\n")
	readinessRunDir := filepath.Join(repoRoot, "demo", "logs", "evidence", "readiness-fixture")
	if err := os.MkdirAll(readinessRunDir, 0o755); err != nil {
		t.Fatalf("mkdir readiness run dir: %v", err)
	}
	for _, name := range []string{"summary.json", "readiness-summary.md", "presentation-readiness-tracker.md", "go-no-go-decision.md", "evidence-inventory.txt"} {
		writeFile(t, filepath.Join(readinessRunDir, name), "fixture\n")
	}
	writeFile(t, filepath.Join(readinessRunDir, "evidence-inventory.txt"), "evidence-inventory.txt\ngo-no-go-decision.md\npresentation-readiness-tracker.md\nreadiness-summary.md\nsummary.json\n")
	writeFile(t, filepath.Join(readinessRunDir, "summary.json"), "{\n  \"run_id\": \"fixture\",\n  \"run_directory\": \""+readinessRunDir+"\",\n  \"overall_status\": \"PASS\",\n  \"bundle_path\": \""+filepath.Join(repoRoot, "bundle.tar.gz")+"\",\n  \"bundle_checksum_path\": \""+filepath.Join(repoRoot, "bundle.tar.gz.sha256")+"\",\n  \"gate_results\": []\n}\n")
	writeFile(t, filepath.Join(repoRoot, "bundle.tar.gz"), "bundle\n")
	writeFile(t, filepath.Join(repoRoot, "bundle.tar.gz.sha256"), "checksum\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runPilotCloseoutBundleCommand(context.Background(), []string{
			"build",
			"--customer", "Falcon Insurance Group",
			"--pilot-name", "Claims Governance Pilot",
			"--charter", charter,
			"--acceptance-memo", acceptance,
			"--validation-checklist", checklist,
			"--readiness-run-dir", readinessRunDir,
		}, stdout, stderr)
	})
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "Pilot closeout bundle complete") {
		t.Fatalf("expected completion output, got %s", readFile(t, stdout))
	}
}

func writeFileForPath(t *testing.T, repoRoot string, relPath string, contents string) string {
	t.Helper()
	path := filepath.Join(repoRoot, relPath)
	writeFile(t, path, contents)
	return path
}
