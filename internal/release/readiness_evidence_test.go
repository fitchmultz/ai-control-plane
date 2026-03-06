// readiness_evidence_test.go - Tests for readiness evidence generation and verification.
//
// Purpose:
//
//	Verify readiness evidence artifact generation, summary rendering, and verifier checks.
//
// Responsibilities:
//   - Exercise successful readiness runs with a fake make binary.
//   - Validate production-gate skip behavior when secrets are unavailable.
//   - Confirm verifier catches inventory drift.
//
// Scope:
//   - Focuses on internal/release readiness evidence behavior only.
//
// Usage:
//   - Run via `go test ./internal/release`
//
// Invariants/Assumptions:
//   - Tests use a temporary fake make executable instead of the real build tool.
//   - Evidence artifacts remain fully local to the test temp directory.
package release

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunReadinessEvidence_GeneratesArtifactsAndVerifierPasses(t *testing.T) {
	repoRoot := t.TempDir()
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
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
	if summary.OverallStatus != "PASS" {
		t.Fatalf("overall status = %s, want PASS", summary.OverallStatus)
	}
	if len(summary.GateResults) != 5 {
		t.Fatalf("gate count = %d, want 5", len(summary.GateResults))
	}
	for _, name := range []string{readinessSummaryJSONName, readinessSummaryMarkdown, readinessTrackerMarkdown, readinessDecisionMarkdown, readinessInventoryText} {
		if _, err := os.Stat(filepath.Join(summary.RunDirectory, name)); err != nil {
			t.Fatalf("missing generated file %s: %v", name, err)
		}
	}
	verifier := NewReadinessVerifier()
	verified, err := verifier.VerifyReadinessRun(summary.RunDirectory)
	if err != nil {
		t.Fatalf("VerifyReadinessRun() error = %v", err)
	}
	if verified.RunID != summary.RunID {
		t.Fatalf("verified run id = %s, want %s", verified.RunID, summary.RunID)
	}
}

func TestRunReadinessEvidence_SkipsProductionWithoutSecrets(t *testing.T) {
	repoRoot := t.TempDir()
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
	bundlePath := filepath.Join(bundleDir, GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}

	summary, err := RunReadinessEvidence(ReadinessOptions{
		RepoRoot:          repoRoot,
		OutputRoot:        outputRoot,
		MakeBin:           makeBin,
		BundleVersion:     bundleVersion,
		IncludeProduction: true,
		SecretsEnvFile:    filepath.Join(repoRoot, "missing.env"),
	}, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("RunReadinessEvidence() error = %v", err)
	}
	if !summary.IncludeProduction {
		t.Fatal("IncludeProduction = false, want true")
	}
	if summary.ProductionEnabled {
		t.Fatal("ProductionEnabled = true, want false")
	}
	var sawSkipped bool
	for _, gate := range summary.GateResults {
		if gate.ID == "production_ci" {
			sawSkipped = true
			if gate.Status != "SKIPPED" {
				t.Fatalf("production gate status = %s, want SKIPPED", gate.Status)
			}
		}
	}
	if !sawSkipped {
		t.Fatal("expected production gate result")
	}
}

func TestVerifyReadinessRun_DetectsInventoryMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
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

	inventoryPath := filepath.Join(summary.RunDirectory, readinessInventoryText)
	if err := os.WriteFile(inventoryPath, []byte("wrong-file.txt\n"), 0o644); err != nil {
		t.Fatalf("overwrite inventory: %v", err)
	}
	_, err = NewReadinessVerifier().VerifyReadinessRun(summary.RunDirectory)
	if err == nil {
		t.Fatal("expected verifier to detect inventory mismatch")
	}
	if !strings.Contains(err.Error(), "inventory mismatch") {
		t.Fatalf("unexpected verifier error: %v", err)
	}
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
  ci|supply-chain-gate|supply-chain-allowlist-expiry-check|release-bundle|release-bundle-verify|ci-nightly)
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
