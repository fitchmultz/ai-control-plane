// cmd_assessor_packet_test.go - Tests for the typed assessor packet commands.
//
// Purpose:
//   - Verify `acpctl deploy assessor-packet build|verify` bindings.
//
// Responsibilities:
//   - Cover successful assessor packet generation.
//   - Cover successful assessor packet verification.
//   - Keep command tests independent from the live repository.
//
// Scope:
//   - Command-layer assessor packet behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests use temp repositories and deterministic fixture data.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/assessor"
	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
)

func TestRunAssessorPacketBuildTypedBuildsPacket(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeAssessorCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newCommandOutputFiles(t)
	code := runAssessorPacketBuildTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, assessor.Options{
		RepoRoot:        repoRoot,
		OutputRoot:      filepath.Join(repoRoot, "demo", "logs", "assessor-packet"),
		ReadinessRunDir: readinessRunDir,
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runAssessorPacketBuildTyped() exit = %d, want %d stderr=%s", code, exitcodes.ACPExitSuccess, readDBCommandOutput(t, stderr))
	}
	if got := readDBCommandOutput(t, stdout); !strings.Contains(got, "Assessor packet complete") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunAssessorPacketVerifyTypedVerifiesLatestPacket(t *testing.T) {
	repoRoot := t.TempDir()
	readinessRunDir := writeAssessorCommandFixtureRepo(t, repoRoot)
	summary, err := assessor.Build(context.Background(), assessor.Options{RepoRoot: repoRoot, OutputRoot: filepath.Join(repoRoot, "demo", "logs", "assessor-packet"), ReadinessRunDir: readinessRunDir})
	if err != nil {
		t.Fatalf("assessor.Build() error = %v", err)
	}
	stdout, stderr := newCommandOutputFiles(t)
	code := runAssessorPacketVerifyTyped(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, "")
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runAssessorPacketVerifyTyped() exit = %d, want %d stderr=%s", code, exitcodes.ACPExitSuccess, readDBCommandOutput(t, stderr))
	}
	output := readDBCommandOutput(t, stdout)
	if !strings.Contains(output, summary.ReadinessRunDir) || !strings.Contains(output, "Assessor packet verified") {
		t.Fatalf("unexpected stdout: %s", output)
	}
}

func writeAssessorCommandFixtureRepo(t *testing.T, repoRoot string) string {
	t.Helper()
	for _, spec := range []string{
		"docs/security/SECURITY_WHITEPAPER_AND_THREAT_MODEL.md",
		"docs/COMPLIANCE_CROSSWALK.md",
		"docs/GO_TO_MARKET_SCOPE.md",
		"docs/KNOWN_LIMITATIONS.md",
		"docs/security/CVE_REMEDIATION_AND_RISK_ACCEPTANCE_POLICY.md",
		"docs/SHARED_RESPONSIBILITY_MODEL.md",
		"docs/PILOT_CONTROL_OWNERSHIP_MATRIX.md",
		"docs/release/READINESS_EVIDENCE_WORKFLOW.md",
		"docs/evidence/EVIDENCE_MAP.md",
		"docs/release/GO_NO_GO.md",
	} {
		writeFile(t, filepath.Join(repoRoot, spec), "# fixture\n")
	}
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "readiness_evidence.yaml"), `gates:
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
`)
	makeBin := filepath.Join(repoRoot, "bin", "make")
	if err := os.MkdirAll(filepath.Dir(makeBin), 0o755); err != nil {
		t.Fatalf("MkdirAll(makeBin) error = %v", err)
	}
	if err := os.WriteFile(makeBin, []byte(`#!/bin/sh
set -eu
case "$1" in
  ci|release-bundle|release-bundle-verify|ci-nightly)
    exit 0
    ;;
  *)
    echo "unexpected target: $1" >&2
    exit 1
    ;;
esac
`), 0o755); err != nil {
		t.Fatalf("WriteFile(makeBin) error = %v", err)
	}
	bundleDir := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bundleDir) error = %v", err)
	}
	bundleVersion := "assessor-command-fixture"
	bundlePath := filepath.Join(bundleDir, bundle.GetBundleName(bundleVersion))
	writeFile(t, bundlePath, "bundle\n")
	writeFile(t, bundlePath+".sha256", "checksum\n")
	summary, err := readiness.RunContext(context.Background(), readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    filepath.Join(repoRoot, "demo", "logs", "evidence"),
		MakeBin:       makeBin,
		BundleVersion: bundleVersion,
	})
	if err != nil {
		t.Fatalf("readiness.RunContext() error = %v", err)
	}
	return summary.RunDirectory
}
