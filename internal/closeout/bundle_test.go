// bundle_test.go - Tests for pilot closeout bundle workflows.
//
// Purpose:
//
//	Verify pilot closeout bundles assemble the expected documents and remain
//	verifiable after generation.
//
// Responsibilities:
//   - Exercise bundle generation with a fake readiness run.
//   - Validate bundle verification and inventory checks.
//   - Confirm the full readiness inventory is copied into the closeout bundle.
//
// Scope:
//   - Covers internal/closeout behavior only.
//
// Usage:
//   - Run via `go test ./internal/closeout`.
//
// Invariants/Assumptions:
//   - Tests generate local readiness evidence in temporary directories.
package closeout

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
)

func TestBuildAndVerify(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)
	charterPath := writeTextFixture(t, repoRoot, "docs/charter.md", "# Charter\n")
	acceptancePath := writeTextFixture(t, repoRoot, "docs/acceptance.md", "# Acceptance\n")
	checklistPath := writeTextFixture(t, repoRoot, "docs/checklist.md", "# Checklist\n")
	operatorPath := writeTextFixture(t, repoRoot, "docs/operator.md", "# Operator\n")

	summary, err := Build(context.Background(), Options{
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
		t.Fatalf("Build() error = %v", err)
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "documents", "pilot-acceptance-memo.md")) {
		t.Fatal("expected copied acceptance memo in bundle")
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "evidence", readiness.SummaryMarkdownName)) {
		t.Fatal("expected copied readiness summary in bundle")
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "evidence", "make-ci.log")) {
		t.Fatal("expected copied readiness log in bundle")
	}
	verified, err := NewVerifier().VerifyRun(summary.RunDirectory)
	if err != nil {
		t.Fatalf("VerifyRun() error = %v", err)
	}
	if verified.Decision != "EXPAND" {
		t.Fatalf("decision = %q, want EXPAND", verified.Decision)
	}
}

func TestBuildCopiesFullReadinessInventory(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)
	extraPath := filepath.Join(readinessRunDir, "logs", "custom-extra.log")
	if err := os.MkdirAll(filepath.Dir(extraPath), 0o755); err != nil {
		t.Fatalf("mkdir extra log dir: %v", err)
	}
	if err := os.WriteFile(extraPath, []byte("extra\n"), 0o644); err != nil {
		t.Fatalf("write extra log: %v", err)
	}
	if _, err := artifactrun.Finalize(readinessRunDir, filepath.Join(repoRoot, "demo", "logs", "evidence"), artifactrun.FinalizeOptions{
		InventoryName: readiness.InventoryFileName,
	}); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	summary, err := Build(context.Background(), Options{
		RepoRoot:            repoRoot,
		Customer:            "Falcon Insurance Group",
		PilotName:           "Claims Governance Pilot",
		CharterPath:         writeTextFixture(t, repoRoot, "docs/charter.md", "# Charter\n"),
		AcceptanceMemoPath:  writeTextFixture(t, repoRoot, "docs/acceptance.md", "# Acceptance\n"),
		ValidationChecklist: writeTextFixture(t, repoRoot, "docs/checklist.md", "# Checklist\n"),
		ReadinessRunDir:     readinessRunDir,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "evidence", "logs", "custom-extra.log")) {
		t.Fatal("expected closeout bundle to include full readiness inventory, including extra log")
	}
}

func TestVerifyRunDetectsInventoryMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)
	summary, err := Build(context.Background(), Options{
		RepoRoot:            repoRoot,
		Customer:            "Falcon Insurance Group",
		PilotName:           "Claims Governance Pilot",
		CharterPath:         writeTextFixture(t, repoRoot, "docs/charter.md", "# Charter\n"),
		AcceptanceMemoPath:  writeTextFixture(t, repoRoot, "docs/acceptance.md", "# Acceptance\n"),
		ValidationChecklist: writeTextFixture(t, repoRoot, "docs/checklist.md", "# Checklist\n"),
		ReadinessRunDir:     readinessRunDir,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(summary.RunDirectory, InventoryFileName), []byte("wrong-file.txt\n"), 0o644); err != nil {
		t.Fatalf("overwrite inventory: %v", err)
	}
	_, err = NewVerifier().VerifyRun(summary.RunDirectory)
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
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}
	summary, err := readiness.RunContext(context.Background(), readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    outputRoot,
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunContext() error = %v", err)
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

func writeFakeMake(t *testing.T, repoRoot string) string {
	t.Helper()
	binDir := filepath.Join(repoRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	path := filepath.Join(binDir, "make")
	script := `#!/bin/sh
set -eu
printf 'fake make %s\n' "$*"
case "$1" in
  ci|release-bundle|release-bundle-verify|ci-nightly)
    exit 0
    ;;
  *)
    echo "unexpected target: $1" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake make: %v", err)
	}
	return path
}

func writeReadinessPlanFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "demo", "config", "readiness_evidence.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create readiness plan dir: %v", err)
	}
	const content = `gates:
  - id: local_ci
    title: Local CI Gate
    required: true
    log_name: make-ci.log
    command:
      - ci
    notes: Validates the host-first baseline command surface.
  - id: release_bundle
    title: Release Bundle Gate
    required: true
    log_name: make-release-bundle.log
    command:
      - release-bundle
      - RELEASE_BUNDLE_VERSION=${BUNDLE_VERSION}
    notes: Builds the canonical deployment bundle for the run.
  - id: release_bundle_verify
    title: Release Bundle Verify Gate
    required: true
    log_name: make-release-bundle-verify.log
    command:
      - release-bundle-verify
      - RELEASE_BUNDLE_VERSION=${BUNDLE_VERSION}
    notes: Confirms bundle integrity using the current checksum sidecar.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write readiness plan fixture: %v", err)
	}
}
