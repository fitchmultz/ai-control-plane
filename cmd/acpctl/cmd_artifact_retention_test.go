// cmd_artifact_retention_test.go - Artifact retention command tests.
//
// Purpose:
//   - Verify stale generated artifact detection and cleanup behavior.
//
// Responsibilities:
//   - Cover check/apply mode semantics for document artifact retention.
//   - Ensure cleanup failures are reported instead of silently ignored.
//
// Scope:
//   - Test-only coverage for cmd_artifact_retention.go.
//
// Usage:
//   - Run with go test ./cmd/acpctl -run TestArtifactRetention.
//
// Invariants/Assumptions:
//   - Tests use temp directories and do not touch repository artifacts.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestArtifactRetentionCheckDetectsStaleArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	writeArtifactRetentionFixture(t, repoRoot)
	stdout, stderr := tempOutputFiles(t)

	code := runArtifactRetention(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, artifactRetentionConfig{
		Mode:         "check",
		KeepEvidence: 1,
		KeepBundles:  1,
		RepoRoot:     repoRoot,
	})
	if code != exitcodes.ACPExitDomain {
		t.Fatalf("runArtifactRetention check exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}
	output := readFile(t, stdout)
	for _, expected := range []string{"20260101-000000", "bundle-20260101.tar.gz", "Retention check failed"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected stdout to contain %q, got %s", expected, output)
		}
	}
}

func TestArtifactRetentionApplyDeletesStaleArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	writeArtifactRetentionFixture(t, repoRoot)
	stdout, stderr := tempOutputFiles(t)

	code := runArtifactRetention(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, artifactRetentionConfig{
		Mode:         "apply",
		KeepEvidence: 1,
		KeepBundles:  1,
		RepoRoot:     repoRoot,
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runArtifactRetention apply exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}

	assertPathMissing(t, filepath.Join(repoRoot, "handoff-packet", "evidence", "20260101-000000"))
	assertPathExists(t, filepath.Join(repoRoot, "handoff-packet", "evidence", "20260201-000000"))
	assertPathMissing(t, filepath.Join(repoRoot, "demo", "logs", "release-bundles", "bundle-20260101.tar.gz"))
	assertPathMissing(t, filepath.Join(repoRoot, "demo", "logs", "release-bundles", "bundle-20260101.tar.gz.sha256"))
	assertPathExists(t, filepath.Join(repoRoot, "demo", "logs", "release-bundles", "bundle-20260201.tar.gz"))
}

func TestArtifactRetentionApplyReportsDeleteFailure(t *testing.T) {
	repoRoot := t.TempDir()
	writeArtifactRetentionFixture(t, repoRoot)
	evidenceRoot := filepath.Join(repoRoot, "handoff-packet", "evidence")
	if err := os.Chmod(evidenceRoot, 0o500); err != nil {
		t.Fatalf("chmod evidence root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(evidenceRoot, 0o700) })
	stdout, stderr := tempOutputFiles(t)

	code := runArtifactRetention(context.Background(), commandRunContext{RepoRoot: repoRoot, Stdout: stdout, Stderr: stderr}, artifactRetentionConfig{
		Mode:         "apply",
		KeepEvidence: 1,
		KeepBundles:  1,
		RepoRoot:     repoRoot,
	})
	if code != exitcodes.ACPExitRuntime {
		t.Fatalf("runArtifactRetention apply exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Retention cleanup failed") || !strings.Contains(got, "20260101-000000") {
		t.Fatalf("expected delete failure on stderr, got %s", got)
	}
}

func writeArtifactRetentionFixture(t *testing.T, repoRoot string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Join(repoRoot, "handoff-packet", "evidence", "20260101-000000", "evidence.txt"), "old")
	testutil.WriteFile(t, filepath.Join(repoRoot, "handoff-packet", "evidence", "20260201-000000", "evidence.txt"), "new")
	bundleRoot := filepath.Join(repoRoot, "demo", "logs", "release-bundles")
	oldBundle := filepath.Join(bundleRoot, "bundle-20260101.tar.gz")
	oldBundleSum := filepath.Join(bundleRoot, "bundle-20260101.tar.gz.sha256")
	newBundle := filepath.Join(bundleRoot, "bundle-20260201.tar.gz")
	testutil.WriteFile(t, oldBundle, "old")
	testutil.WriteFile(t, oldBundleSum, "old-sum")
	testutil.WriteFile(t, newBundle, "new")
	oldTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, path := range []string{oldBundle, oldBundleSum} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}
	if err := os.Chtimes(newBundle, newTime, newTime); err != nil {
		t.Fatalf("chtimes %s: %v", newBundle, err)
	}
}

func tempOutputFiles(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	stdout, err := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if err != nil {
		t.Fatalf("create stdout: %v", err)
	}
	stderr, err := os.CreateTemp(t.TempDir(), "stderr-*.txt")
	if err != nil {
		t.Fatalf("create stderr: %v", err)
	}
	return stdout, stderr
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat err=%v", path, err)
	}
}
