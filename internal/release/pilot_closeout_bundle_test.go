// pilot_closeout_bundle_test.go - Tests for pilot closeout bundle workflows.
//
// Purpose:
//
//	Verify pilot closeout bundles assemble the expected documents and remain
//	verifiable after generation.
//
// Responsibilities:
//   - Exercise bundle generation with a fake readiness run
//   - Validate bundle verification and inventory checks
//
// Scope:
//   - Covers internal/release pilot closeout behavior only
//
// Usage:
//   - Run via `go test ./internal/release`
//
// Invariants/Assumptions:
//   - Tests generate local readiness evidence in temporary directories
package release

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPilotCloseoutBundleAndVerify(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)
	charterPath := writeTextFixture(t, repoRoot, "docs/charter.md", "# Charter\n")
	acceptancePath := writeTextFixture(t, repoRoot, "docs/acceptance.md", "# Acceptance\n")
	checklistPath := writeTextFixture(t, repoRoot, "docs/checklist.md", "# Checklist\n")
	operatorPath := writeTextFixture(t, repoRoot, "docs/operator.md", "# Operator\n")

	summary, err := BuildPilotCloseoutBundle(PilotCloseoutOptions{
		RepoRoot:            repoRoot,
		Customer:            "Falcon Insurance Group",
		PilotName:           "Claims Governance Pilot",
		Decision:            "EXPAND",
		CharterPath:         charterPath,
		AcceptanceMemoPath:  acceptancePath,
		ValidationChecklist: checklistPath,
		OperatorChecklist:   operatorPath,
		ReadinessRunDir:     readinessRunDir,
	})
	if err != nil {
		t.Fatalf("BuildPilotCloseoutBundle() error = %v", err)
	}
	if !fileExists(filepath.Join(summary.RunDirectory, "documents", "pilot-acceptance-memo.md")) {
		t.Fatal("expected copied acceptance memo in bundle")
	}
	if !fileExists(filepath.Join(summary.RunDirectory, "evidence", readinessSummaryMarkdown)) {
		t.Fatal("expected copied readiness summary in bundle")
	}
	verified, err := NewPilotCloseoutVerifier().VerifyPilotCloseoutBundle(summary.RunDirectory)
	if err != nil {
		t.Fatalf("VerifyPilotCloseoutBundle() error = %v", err)
	}
	if verified.Decision != "EXPAND" {
		t.Fatalf("decision = %q, want EXPAND", verified.Decision)
	}
}

func TestVerifyPilotCloseoutBundleDetectsInventoryMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)
	summary, err := BuildPilotCloseoutBundle(PilotCloseoutOptions{
		RepoRoot:            repoRoot,
		Customer:            "Falcon Insurance Group",
		PilotName:           "Claims Governance Pilot",
		CharterPath:         writeTextFixture(t, repoRoot, "docs/charter.md", "# Charter\n"),
		AcceptanceMemoPath:  writeTextFixture(t, repoRoot, "docs/acceptance.md", "# Acceptance\n"),
		ValidationChecklist: writeTextFixture(t, repoRoot, "docs/checklist.md", "# Checklist\n"),
		ReadinessRunDir:     readinessRunDir,
	})
	if err != nil {
		t.Fatalf("BuildPilotCloseoutBundle() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(summary.RunDirectory, pilotCloseoutInventory), []byte("wrong-file.txt\n"), 0o644); err != nil {
		t.Fatalf("overwrite inventory: %v", err)
	}
	_, err = NewPilotCloseoutVerifier().VerifyPilotCloseoutBundle(summary.RunDirectory)
	if err == nil {
		t.Fatal("expected inventory mismatch error")
	}
	if !strings.Contains(err.Error(), "inventory mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeReadinessRunFixture(t *testing.T, repoRoot string) string {
	t.Helper()
	writeReadinessPlanFixture(t, repoRoot)
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "closeout-fixture"
	bundlePath := filepath.Join(bundleDir, GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}
	summary, err := RunReadinessEvidence(ReadinessOptions{
		RepoRoot:      repoRoot,
		OutputRoot:    outputRoot,
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunReadinessEvidence() error = %v", err)
	}
	return summary.RunDirectory
}

func writeTextFixture(t *testing.T, repoRoot string, relPath string, contents string) string {
	t.Helper()
	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	return fullPath
}
