// verifier_edge_test.go - Focused coverage for bundle parser and verifier edges.
//
// Purpose:
//   - Verify bundle helper edge cases that the main happy-path tests do not hit.
//
// Responsibilities:
//   - Cover detached-HEAD default version detection.
//   - Verify checksum parsing failures and payload mismatches.
//   - Lock down safeJoinWithin path rejection rules.
//
// Scope:
//   - Parser and verifier helper edge cases only.
//
// Usage:
//   - Run via `go test ./internal/bundle`.
//
// Invariants/Assumptions:
//   - Tests use isolated temp directories and do not require a real Git repo.
package bundle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestGetDefaultVersion_CoversDetachedHeadAndShortSHAFallback(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteFile(t, filepath.Join(repoRoot, ".git", "HEAD"), "abcdef1234567890\n")
	if got := GetDefaultVersion(repoRoot); got != "abcdef1" {
		t.Fatalf("GetDefaultVersion detached HEAD = %q", got)
	}

	shortRepo := t.TempDir()
	testutil.WriteFile(t, filepath.Join(shortRepo, ".git", "HEAD"), "abc\n")
	if got := GetDefaultVersion(shortRepo); got != "dev" {
		t.Fatalf("GetDefaultVersion short SHA = %q", got)
	}
}

func TestVerifierVerifyChecksums_CoversMalformedLinesAndMismatches(t *testing.T) {
	payloadDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(payloadDir, "app.txt"), "payload\n")
	shaPath := filepath.Join(t.TempDir(), "sha256sums.txt")

	testutil.WriteFile(t, shaPath, "not-enough-fields\n")
	err := NewVerifier(false).verifyChecksums(payloadDir, shaPath)
	if err == nil || !strings.Contains(err.Error(), "invalid checksum line") {
		t.Fatalf("expected malformed checksum line error, got %v", err)
	}

	testutil.WriteFile(t, shaPath, "deadbeef ./app.txt\n")
	err = NewVerifier(false).verifyChecksums(payloadDir, shaPath)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch for app.txt") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestSafeJoinWithin_RejectsInvalidArchivePaths(t *testing.T) {
	baseDir := t.TempDir()

	for _, relPath := range []string{"", ".", "/tmp/escape", "..", "../escape", `..\escape`} {
		if _, err := safeJoinWithin(baseDir, relPath); err == nil {
			t.Fatalf("expected safeJoinWithin(%q) to fail", relPath)
		}
	}

	joined, err := safeJoinWithin(baseDir, `nested\file.txt`)
	if err != nil {
		t.Fatalf("expected windows-style separators to normalize, got %v", err)
	}
	if !strings.HasSuffix(joined, filepath.Join("nested", "file.txt")) {
		t.Fatalf("unexpected joined path: %s", joined)
	}
}

func TestVerifierVerify_ReportsPayloadChecksumFailure(t *testing.T) {
	repoRoot := t.TempDir()
	original := CanonicalPaths
	CanonicalPaths = []string{"README.md"}
	t.Cleanup(func() { CanonicalPaths = original })
	testutil.WriteRepoFile(t, repoRoot, "README.md", "content\n")

	outputDir := filepath.Join(repoRoot, "output")
	plan := &Plan{
		Version:    "v1.0.0",
		RepoRoot:   repoRoot,
		OutputDir:  outputDir,
		BundlePath: filepath.Join(outputDir, "bundle.tar.gz"),
	}
	if err := NewBuilder(repoRoot, false).Build(plan, io.Discard); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	extractDir := t.TempDir()
	verifier := NewVerifier(false)
	if err := verifier.extractTarball(plan.BundlePath, extractDir); err != nil {
		t.Fatalf("extractTarball returned error: %v", err)
	}
	payloadFile := filepath.Join(extractDir, "payload", "README.md")
	if err := os.WriteFile(payloadFile, []byte("tampered\n"), 0o600); err != nil {
		t.Fatalf("tamper payload: %v", err)
	}
	shaPath := filepath.Join(extractDir, "sha256sums.txt")
	if err := os.WriteFile(shaPath, []byte("deadbeef ./README.md\n"), 0o600); err != nil {
		t.Fatalf("rewrite checksums: %v", err)
	}

	err := verifier.verifyChecksums(filepath.Join(extractDir, "payload"), shaPath)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch for README.md") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}
