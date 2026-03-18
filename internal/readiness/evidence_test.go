// evidence_test.go - Tests for readiness evidence generation and verification.
//
// Purpose:
//
//	Verify readiness evidence artifact generation, summary rendering, and
//	verifier checks.
//
// Responsibilities:
//   - Exercise successful readiness runs with a fake make binary.
//   - Validate production-gate skip behavior when secrets are unavailable.
//   - Confirm embedded placeholder expansion and inventory drift detection.
//
// Scope:
//   - Focuses on internal/readiness behavior only.
//
// Usage:
//   - Run via `go test ./internal/readiness`.
//
// Invariants/Assumptions:
//   - Tests use a temporary fake make executable instead of the real build tool.
//   - Evidence artifacts remain fully local to the test temp directory.
package readiness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

func TestRunContextGeneratesArtifactsAndVerifierPasses(t *testing.T) {
	repoRoot := t.TempDir()
	writePlanFixture(t, repoRoot)
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}

	summary, err := RunContext(context.Background(), Options{
		RepoRoot:      repoRoot,
		OutputRoot:    outputRoot,
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	})
	if err != nil {
		t.Fatalf("RunContext() error = %v", err)
	}
	if summary.OverallStatus != "PASS" {
		t.Fatalf("overall status = %s, want PASS", summary.OverallStatus)
	}
	if len(summary.GateResults) != 3 {
		t.Fatalf("gate count = %d, want 3", len(summary.GateResults))
	}
	for _, name := range []string{SummaryJSONName, SummaryMarkdownName, TrackerMarkdownName, DecisionMarkdownName, InventoryFileName} {
		if _, err := os.Stat(filepath.Join(summary.RunDirectory, name)); err != nil {
			t.Fatalf("missing generated file %s: %v", name, err)
		}
	}
	verified, err := NewVerifier().VerifyRun(summary.RunDirectory)
	if err != nil {
		t.Fatalf("VerifyRun() error = %v", err)
	}
	if verified.RunID != summary.RunID {
		t.Fatalf("verified run id = %s, want %s", verified.RunID, summary.RunID)
	}
	for _, gate := range summary.GateResults {
		if gate.Status == "SKIPPED" || strings.TrimSpace(gate.LogPath) == "" {
			continue
		}
		if got := statMode(t, gate.LogPath); got != fsutil.PrivateFilePerm {
			t.Fatalf("gate log mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
		}
	}
	if got := statMode(t, summary.RunDirectory); got != fsutil.PrivateDirPerm {
		t.Fatalf("run directory mode = %04o, want %04o", got, fsutil.PrivateDirPerm)
	}
	if got := statMode(t, filepath.Join(outputRoot, LatestRunPointerName)); got != fsutil.PrivateFilePerm {
		t.Fatalf("latest run pointer mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
	}
}

func TestRunContextSkipsProductionWithoutSecrets(t *testing.T) {
	repoRoot := t.TempDir()
	writePlanFixture(t, repoRoot)
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}

	summary, err := RunContext(context.Background(), Options{
		RepoRoot:          repoRoot,
		OutputRoot:        outputRoot,
		MakeBin:           makeBin,
		BundleVersion:     bundleVersion,
		IncludeProduction: true,
		SecretsEnvFile:    filepath.Join(repoRoot, "missing.env"),
	})
	if err != nil {
		t.Fatalf("RunContext() error = %v", err)
	}
	if !summary.IncludeProduction {
		t.Fatal("IncludeProduction = false, want true")
	}
	if summary.ProductionEnabled {
		t.Fatal("ProductionEnabled = true, want false")
	}
	skipped := map[string]bool{}
	for _, gate := range summary.GateResults {
		if gate.ID == "production_ci" || gate.ID == "replacement_host_recovery_evidence" {
			skipped[gate.ID] = true
			if gate.Status != "SKIPPED" {
				t.Fatalf("production gate %s status = %s, want SKIPPED", gate.ID, gate.Status)
			}
		}
	}
	for _, gateID := range []string{"production_ci", "replacement_host_recovery_evidence"} {
		if !skipped[gateID] {
			t.Fatalf("expected production gate result for %s", gateID)
		}
	}
}

func TestVerifyRunDetectsInventoryMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	writePlanFixture(t, repoRoot)
	outputRoot := filepath.Join(repoRoot, "demo", "logs", "evidence")
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	makeBin := writeFakeMake(t, repoRoot)
	bundleVersion := "testrun"
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte("checksum\n"), 0o644); err != nil {
		t.Fatalf("write bundle checksum: %v", err)
	}

	summary, err := RunContext(context.Background(), Options{
		RepoRoot:      repoRoot,
		OutputRoot:    outputRoot,
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	})
	if err != nil {
		t.Fatalf("RunContext() error = %v", err)
	}

	inventoryPath := filepath.Join(summary.RunDirectory, InventoryFileName)
	if err := os.WriteFile(inventoryPath, []byte("wrong-file.txt\n"), 0o644); err != nil {
		t.Fatalf("overwrite inventory: %v", err)
	}
	_, err = NewVerifier().VerifyRun(summary.RunDirectory)
	if err == nil {
		t.Fatal("expected verifier to detect inventory mismatch")
	}
	if !strings.Contains(err.Error(), "inventory mismatch") {
		t.Fatalf("unexpected verifier error: %v", err)
	}
}

func TestExpandCommandArgsReplacesEmbeddedPlaceholders(t *testing.T) {
	args := []string{
		"release-bundle",
		"RELEASE_BUNDLE_VERSION=${BUNDLE_VERSION}",
		"SECRETS_ENV_FILE=${SECRETS_ENV_FILE}",
	}
	got := expandCommandArgs(args, Options{
		BundleVersion:  "v2026.03.08",
		SecretsEnvFile: "/tmp/secrets.env",
	})
	if got[1] != "RELEASE_BUNDLE_VERSION=v2026.03.08" {
		t.Fatalf("bundle version expansion = %q", got[1])
	}
	if got[2] != "SECRETS_ENV_FILE=/tmp/secrets.env" {
		t.Fatalf("secrets env expansion = %q", got[2])
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
  ci|release-bundle|release-bundle-verify|ci-nightly|db-off-host-drill)
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

func writePlanFixture(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, planRelativePath)
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
  - id: production_ci
    title: Production CI Gate
    required: true
    production_only: true
    log_name: make-ci-nightly.log
    command:
      - ci-nightly
      - SECRETS_ENV_FILE=${SECRETS_ENV_FILE}
    notes: Optional customer-like gate; requires a real secrets file.
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
  - id: replacement_host_recovery_evidence
    title: Replacement Host Recovery Evidence
    required: true
    production_only: true
    log_name: make-db-off-host-drill.log
    command:
      - db-off-host-drill
      - OFF_HOST_RECOVERY_MANIFEST=demo/logs/recovery-inputs/off_host_recovery.yaml
    notes: Validates staged off-host recovery inputs.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write readiness plan fixture: %v", err)
	}
}

func statMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%s) error = %v", path, err)
	}
	return info.Mode().Perm()
}
