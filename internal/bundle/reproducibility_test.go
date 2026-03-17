// reproducibility_test.go - Tests for release reproducibility and extraction safety.
//
// Purpose: Verify deterministic release bundle output and secure extraction behavior.
// Responsibilities:
//   - Validate SOURCE_DATE_EPOCH handling for archive timestamps
//   - Ensure repeated bundle builds produce identical tarballs
//   - Prevent tar path traversal during extraction
//
// Scope:
//   - Focuses on builder/verifier behavior only
//
// Usage:
//   - Run via `go test ./internal/bundle`
//
// Invariants/Assumptions:
//   - CanonicalPaths can be temporarily overridden in tests and restored with defer
package bundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReproducibleArchiveTime(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")
	got := reproducibleArchiveTime()
	want := time.Unix(1700000000, 0).UTC()
	if !got.Equal(want) {
		t.Fatalf("reproducibleArchiveTime() = %v, want %v", got, want)
	}

	t.Setenv("SOURCE_DATE_EPOCH", "not-a-number")
	got = reproducibleArchiveTime()
	want = time.Unix(0, 0).UTC()
	if !got.Equal(want) {
		t.Fatalf("reproducibleArchiveTime() fallback = %v, want %v", got, want)
	}
}

func TestBuilderBuild_DeterministicTarball(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "Makefile"), []byte("all:\n\t@echo ok\n"), 0o644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	original := CanonicalPaths
	CanonicalPaths = []string{"Makefile", "README.md"}
	defer func() { CanonicalPaths = original }()

	outputDir := filepath.Join(repoRoot, "output")
	builder := NewBuilder(repoRoot, false)

	plan1 := &Plan{
		Version:    "1.0.0",
		RepoRoot:   repoRoot,
		OutputDir:  outputDir,
		BundlePath: filepath.Join(outputDir, "first.tar.gz"),
	}
	if err := builder.Build(context.Background(), plan1); err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	firstHash, err := ComputeFileHash(plan1.BundlePath)
	if err != nil {
		t.Fatalf("failed to hash first bundle: %v", err)
	}

	now := time.Now().Add(2 * time.Hour)
	for _, rel := range []string{"Makefile", "README.md"} {
		full := filepath.Join(repoRoot, rel)
		if err := os.Chtimes(full, now, now); err != nil {
			t.Fatalf("failed to change file time for %s: %v", rel, err)
		}
	}

	plan2 := &Plan{
		Version:    "1.0.0",
		RepoRoot:   repoRoot,
		OutputDir:  outputDir,
		BundlePath: filepath.Join(outputDir, "second.tar.gz"),
	}
	if err := builder.Build(context.Background(), plan2); err != nil {
		t.Fatalf("second build failed: %v", err)
	}
	secondHash, err := ComputeFileHash(plan2.BundlePath)
	if err != nil {
		t.Fatalf("failed to hash second bundle: %v", err)
	}

	if firstHash != secondHash {
		t.Fatalf("bundle hash mismatch; expected deterministic output\nfirst:  %s\nsecond: %s", firstHash, secondHash)
	}
}

func TestVerifierExtractTarballRejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "malicious.tar.gz")

	bundleFile, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tarball: %v", err)
	}
	gzipWriter := gzip.NewWriter(bundleFile)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name: "../outside.txt",
		Mode: 0o644,
		Size: int64(len("bad")),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("bad")); err != nil {
		t.Fatalf("failed to write payload: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	if err := bundleFile.Close(); err != nil {
		t.Fatalf("failed to close tarball: %v", err)
	}

	verifier := NewVerifier(false)
	err = verifier.extractTarball(tarPath, filepath.Join(tmpDir, "extract"))
	if err == nil {
		t.Fatal("expected extraction error for path traversal entry")
	}
	if !strings.Contains(err.Error(), "path escapes destination") {
		t.Fatalf("expected path traversal error, got: %v", err)
	}
}

func TestVerifierVerify_InvalidSidecarFormat(t *testing.T) {
	tmpDir := t.TempDir()
	bundlePath := filepath.Join(tmpDir, "bundle.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("not-a-real-bundle"), 0o644); err != nil {
		t.Fatalf("failed to write bundle: %v", err)
	}
	if err := os.WriteFile(bundlePath+".sha256", []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write sidecar: %v", err)
	}

	verifier := NewVerifier(false)
	_, err := verifier.Verify(context.Background(), bundlePath)
	if err == nil {
		t.Fatal("expected verifier to reject malformed sidecar")
	}
	if !strings.Contains(err.Error(), "invalid sidecar checksum file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuilderBuild_InstallManifestSorted(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "z-file.txt"), []byte("z"), 0o644); err != nil {
		t.Fatalf("failed to write z-file.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "a-file.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("failed to write a-file.txt: %v", err)
	}

	original := CanonicalPaths
	CanonicalPaths = []string{"z-file.txt", "a-file.txt"}
	defer func() { CanonicalPaths = original }()

	outputDir := filepath.Join(repoRoot, "output")
	plan := &Plan{
		Version:    "1.0.0",
		RepoRoot:   repoRoot,
		OutputDir:  outputDir,
		BundlePath: filepath.Join(outputDir, "bundle.tar.gz"),
	}

	builder := NewBuilder(repoRoot, false)
	if err := builder.Build(context.Background(), plan); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	extractDir := t.TempDir()
	verifier := NewVerifier(false)
	if err := verifier.extractTarball(plan.BundlePath, extractDir); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	manifestData, err := os.ReadFile(filepath.Join(extractDir, "install-manifest.txt"))
	if err != nil {
		t.Fatalf("failed to read install-manifest.txt: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(manifestData)), "\n")
	want := []string{"a-file.txt", "z-file.txt"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("manifest order = %v, want %v", lines, want)
	}
}

func TestCollectRegularFiles_PropagatesWalkErrors(t *testing.T) {
	_, err := collectRegularFiles(filepath.Join(t.TempDir(), "missing-root"))
	if err == nil {
		t.Fatal("expected collectRegularFiles to return walk error")
	}
	if !strings.Contains(err.Error(), "collect files under") {
		t.Fatalf("unexpected error: %v", err)
	}
}
