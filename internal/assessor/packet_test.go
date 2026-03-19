// packet_test.go - Tests for assessor packet workflows.
//
// Purpose:
//   - Verify assessor packets assemble canonical reviewer docs and current
//     verified evidence without overstating external validation status.
//
// Responsibilities:
//   - Exercise packet generation with a fake readiness run.
//   - Validate packet verification and inventory checks.
//   - Confirm the packet preserves the preparation-only truth boundary.
//
// Scope:
//   - Covers internal/assessor behavior only.
//
// Usage:
//   - Run via `go test ./internal/assessor`.
//
// Invariants/Assumptions:
//   - Tests generate local readiness evidence in temporary directories.
package assessor

import (
	"context"
	"encoding/json"
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
	writeCanonicalReviewerDocFixtures(t, repoRoot)
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)

	summary, err := Build(context.Background(), Options{RepoRoot: repoRoot, ReadinessRunDir: readinessRunDir})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "docs", "security", "SECURITY_WHITEPAPER_AND_THREAT_MODEL.md")) {
		t.Fatal("expected canonical reviewer document in packet")
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "evidence", "readiness", readiness.SummaryMarkdownName)) {
		t.Fatal("expected copied readiness summary in packet")
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, filepath.FromSlash(summary.ReleaseBundlePacketPath))) {
		t.Fatal("expected copied release bundle in packet")
	}
	if !summary.PreparationOnly {
		t.Fatal("expected preparation_only=true")
	}
	if summary.ExternalAssessmentCompleted {
		t.Fatal("expected external_assessment_completed=false")
	}
	if summary.RoadmapItemStatus != roadmapItemStatusOpen {
		t.Fatalf("roadmap item status = %q, want %q", summary.RoadmapItemStatus, roadmapItemStatusOpen)
	}

	verified, err := NewVerifier().VerifyRun(context.Background(), summary.RunDirectory)
	if err != nil {
		t.Fatalf("VerifyRun() error = %v", err)
	}
	if verified.RoadmapItemID != roadmapItemID {
		t.Fatalf("roadmap item id = %d, want %d", verified.RoadmapItemID, roadmapItemID)
	}
}

func TestBuildCopiesFullReadinessInventory(t *testing.T) {
	repoRoot := t.TempDir()
	writeCanonicalReviewerDocFixtures(t, repoRoot)
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)

	extraPath := filepath.Join(readinessRunDir, "logs", "custom-extra.log")
	if err := os.MkdirAll(filepath.Dir(extraPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(extraPath) error = %v", err)
	}
	if err := os.WriteFile(extraPath, []byte("extra\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(extraPath) error = %v", err)
	}
	if _, err := artifactrun.Finalize(readinessRunDir, filepath.Join(repoRoot, "demo", "logs", "evidence"), artifactrun.FinalizeOptions{InventoryName: readiness.InventoryFileName}); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	summary, err := Build(context.Background(), Options{RepoRoot: repoRoot, ReadinessRunDir: readinessRunDir})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !artifactrun.FileExists(filepath.Join(summary.RunDirectory, "evidence", "readiness", "logs", "custom-extra.log")) {
		t.Fatal("expected assessor packet to include full readiness inventory")
	}
}

func TestBuildUsesLatestSuccessfulReadinessRunByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	writeCanonicalReviewerDocFixtures(t, repoRoot)
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)

	summary, err := Build(context.Background(), Options{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if summary.ReadinessRunDir != readinessRunDir {
		t.Fatalf("ReadinessRunDir = %q, want %q", summary.ReadinessRunDir, readinessRunDir)
	}
}

func TestVerifyRunRejectsFalseExternalAssessmentClaim(t *testing.T) {
	repoRoot := t.TempDir()
	writeCanonicalReviewerDocFixtures(t, repoRoot)
	readinessRunDir := writeReadinessRunFixture(t, repoRoot)

	summary, err := Build(context.Background(), Options{RepoRoot: repoRoot, ReadinessRunDir: readinessRunDir})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	summaryPath := filepath.Join(summary.RunDirectory, SummaryJSONName)
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("ReadFile(summary.json) error = %v", err)
	}
	var mutated Summary
	if err := json.Unmarshal(data, &mutated); err != nil {
		t.Fatalf("Unmarshal(summary.json) error = %v", err)
	}
	mutated.ExternalAssessmentCompleted = true
	if err := artifactrun.WriteJSON(summaryPath, &mutated); err != nil {
		t.Fatalf("WriteJSON(summary.json) error = %v", err)
	}

	_, err = NewVerifier().VerifyRun(context.Background(), summary.RunDirectory)
	if err == nil {
		t.Fatal("VerifyRun() error = nil, want truth-boundary rejection")
	}
	if !strings.Contains(err.Error(), "external_assessment_completed=false") {
		t.Fatalf("unexpected verifier error: %v", err)
	}
}

func writeCanonicalReviewerDocFixtures(t *testing.T, repoRoot string) {
	t.Helper()
	for _, spec := range canonicalReviewerDocuments {
		writeTextFixture(t, repoRoot, spec.RepoRelativePath, "# "+spec.Title+"\n")
	}
}

func writeReadinessRunFixture(t *testing.T, repoRoot string) string {
	t.Helper()
	writeReadinessPlanFixture(t, repoRoot)
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bundleDir) error = %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "assessor-fixture"
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(bundlePath) error = %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(bundleChecksum) error = %v", err)
	}
	summary, err := readiness.RunContext(context.Background(), readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    outputRoot,
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	})
	if err != nil {
		t.Fatalf("RunContext() error = %v", err)
	}
	return summary.RunDirectory
}

func writeTextFixture(t *testing.T, repoRoot string, relPath string, contents string) string {
	t.Helper()
	fullPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", fullPath, err)
	}
	return fullPath
}

func writeFakeMake(t *testing.T, repoRoot string) string {
	t.Helper()
	binDir := filepath.Join(repoRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(binDir) error = %v", err)
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
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
	return path
}

func writeReadinessPlanFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "demo", "config", "readiness_evidence.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
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
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
