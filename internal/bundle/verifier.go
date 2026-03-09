// verifier.go - Release bundle verifier.
//
// Purpose: Verify release bundle integrity
//
// Responsibilities:
//   - Verify tarball checksums against sidecar files
//   - Verify payload file checksums
//   - Validate bundle structure
//
// Non-scope:
//   - Does not build bundles (see builder.go)
//   - Does not repair damaged bundles
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package bundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

// Verifier handles bundle verification
type Verifier struct {
	verbose bool
}

// NewVerifier creates a new bundle verifier
func NewVerifier(verbose bool) *Verifier {
	return &Verifier{verbose: verbose}
}

// VerificationResult contains verification outcomes
type VerificationResult struct {
	SidecarValid    bool
	StructureValid  bool
	PayloadValid    bool
	FilesInManifest int
	Errors          []string
}

// Verify verifies a release bundle
func (v *Verifier) Verify(ctx context.Context, bundlePath string) (*VerificationResult, error) {
	result := &VerificationResult{}
	logger := logging.FromContext(ctx).With(
		slog.String("component", "bundle"),
		slog.String("workflow", "release_bundle_verify"),
	)

	// Resolve to absolute path
	if !filepath.IsAbs(bundlePath) {
		wd, _ := os.Getwd()
		bundlePath = filepath.Join(wd, bundlePath)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return nil, fmt.Errorf("bundle not found: %s", bundlePath)
	}

	// Check for sidecar checksum file
	sidecar := bundlePath + ".sha256"
	sidecarData, err := os.ReadFile(sidecar)
	if err != nil {
		return nil, fmt.Errorf("missing sidecar checksum file: %s", sidecar)
	}

	// Verify tarball checksum
	sidecarFields := strings.Fields(string(sidecarData))
	if len(sidecarFields) < 1 {
		return nil, fmt.Errorf("invalid sidecar checksum file (no hash found): %s", sidecar)
	}
	expectedHash := sidecarFields[0]
	actualHash, err := ComputeFileHash(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute bundle hash: %w", err)
	}

	if v.verbose {
		logger.Info("bundle.sidecar_hash", slog.String("expected", expectedHash), slog.String("actual", actualHash))
	}

	if expectedHash != actualHash {
		result.Errors = append(result.Errors, "bundle tarball checksum mismatch - possible tampering")
		logger.Error("workflow.complete", slog.String("status", "fail"), slog.String("reason", "checksum mismatch"))
		return result, nil
	}
	result.SidecarValid = true

	// Extract to temp directory
	extractDir, err := os.MkdirTemp("", "release-bundle-verify-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create extract directory: %w", err)
	}
	defer os.RemoveAll(extractDir)

	if err := v.extractTarball(bundlePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract bundle: %w", err)
	}

	// Verify required files exist
	requiredFiles := []string{
		"sha256sums.txt",
		"install-manifest.txt",
		"payload",
	}
	for _, file := range requiredFiles {
		path := filepath.Join(extractDir, file)
		if _, err := os.Stat(path); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("missing %s in bundle", file))
			return result, nil
		}
	}
	result.StructureValid = true

	// Verify payload checksums
	payloadDir := filepath.Join(extractDir, "payload")
	sha256sumsPath := filepath.Join(extractDir, "sha256sums.txt")
	if err := v.verifyChecksums(payloadDir, sha256sumsPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("payload checksum verification failed: %v", err))
		logger.Error("workflow.complete", slog.String("status", "fail"), logging.Err(err))
		return result, nil
	}
	result.PayloadValid = true

	logger.Info("workflow.complete", slog.String("status", "pass"), slog.String("bundle_path", bundlePath))
	return result, nil
}

func (v *Verifier) extractTarball(tarballPath, destDir string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path, err := safeJoinWithin(destDir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fsutil.EnsurePrivateDir(path); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := fsutil.EnsurePrivateDir(filepath.Dir(path)); err != nil {
				return err
			}
			// Verification never executes extracted files, so clamp archive metadata to
			// private local-only defaults instead of recreating broader on-disk modes.
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fsutil.PrivateFilePerm)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("unsupported tar entry type %q for %q", header.Typeflag, header.Name)
		default:
			return fmt.Errorf("unsupported tar entry type %q for %q", header.Typeflag, header.Name)
		}
	}

	return nil
}

func (v *Verifier) verifyChecksums(payloadDir, sha256sumsPath string) error {
	data, err := os.ReadFile(sha256sumsPath)
	if err != nil {
		return err
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return fmt.Errorf("invalid checksum line: %q", line)
		}

		expectedHash := parts[0]
		relPath := strings.TrimPrefix(parts[1], "./")
		fullPath, err := safeJoinWithin(payloadDir, relPath)
		if err != nil {
			return fmt.Errorf("invalid payload path %q: %w", relPath, err)
		}

		actualHash, err := ComputeFileHash(fullPath)
		if err != nil {
			return fmt.Errorf("failed to hash %s: %w", relPath, err)
		}

		if expectedHash != actualHash {
			return fmt.Errorf("checksum mismatch for %s", relPath)
		}
	}

	return nil
}

func safeJoinWithin(baseDir, relPath string) (string, error) {
	normalized := filepath.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	if normalized == "." || normalized == "" {
		return "", fmt.Errorf("invalid archive path %q", relPath)
	}
	if filepath.IsAbs(normalized) {
		return "", fmt.Errorf("absolute paths are not allowed: %q", relPath)
	}
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes destination: %q", relPath)
	}

	targetPath := filepath.Join(baseDir, normalized)
	relativeToBase, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}
	if relativeToBase == ".." || strings.HasPrefix(relativeToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes destination: %q", relPath)
	}

	return targetPath, nil
}
