// builder.go - Release bundle builder
//
// Purpose: Build release bundles with checksums and manifests
//
// Responsibilities:
//   - Copy files to staging area
//   - Create install manifest and checksums
//   - Create reproducible tarballs
//   - Generate checksum sidecars
//
// Non-scope:
//   - Does not verify bundles (see verifier.go)
//   - Does not parse arguments (see parser.go)
package release

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Builder handles bundle construction
type Builder struct {
	repoRoot string
	verbose  bool
}

// NewBuilder creates a new bundle builder
func NewBuilder(repoRoot string, verbose bool) *Builder {
	return &Builder{
		repoRoot: repoRoot,
		verbose:  verbose,
	}
}

// Build creates a release bundle from the plan
func (b *Builder) Build(plan *Plan, stdout io.Writer) error {
	// Create output directory
	if err := os.MkdirAll(plan.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create staging area
	stageDir, err := os.MkdirTemp("", "release-bundle-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	payloadDir := filepath.Join(stageDir, "payload")
	if err := os.MkdirAll(payloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create payload directory: %w", err)
	}

	if b.verbose {
		fmt.Fprintf(stdout, "Staging directory: %s\n", stageDir)
	}

	// Copy canonical files to payload
	if err := b.copyPayloadFiles(payloadDir, stdout); err != nil {
		return err
	}

	// Build install manifest
	installManifest := filepath.Join(stageDir, "install-manifest.txt")
	manifestPaths := append([]string(nil), CanonicalPaths...)
	sort.Strings(manifestPaths)
	manifestContent := strings.Join(manifestPaths, "\n") + "\n"
	if err := os.WriteFile(installManifest, []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("failed to create install manifest: %w", err)
	}

	// Build checksums for payload files
	sha256sumsPath := filepath.Join(stageDir, "sha256sums.txt")
	if err := b.buildChecksums(payloadDir, sha256sumsPath); err != nil {
		return fmt.Errorf("failed to build checksums: %w", err)
	}

	// Create tarball
	if err := b.createReproducibleTarball(stageDir, plan.BundlePath); err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	// Create checksum sidecar
	if err := b.createChecksumSidecar(plan.BundlePath); err != nil {
		return fmt.Errorf("failed to create checksum: %w", err)
	}

	return nil
}

func (b *Builder) copyPayloadFiles(payloadDir string, stdout io.Writer) error {
	for _, relPath := range CanonicalPaths {
		src := filepath.Join(b.repoRoot, relPath)
		dst := filepath.Join(payloadDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", relPath, err)
		}
		if b.verbose {
			fmt.Fprintf(stdout, "  Copying: %s\n", relPath)
		}
	}
	return nil
}

func (b *Builder) buildChecksums(payloadDir, outputPath string) error {
	var checksums []string

	err := filepath.Walk(payloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(payloadDir, path)
		relPath = filepath.ToSlash(relPath)
		hash, err := ComputeFileHash(path)
		if err != nil {
			return err
		}
		checksums = append(checksums, fmt.Sprintf("%s  ./%s", hash, relPath))
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(checksums)
	content := strings.Join(checksums, "\n") + "\n"
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func (b *Builder) createReproducibleTarball(stageDir, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	archiveTime := reproducibleArchiveTime()
	gzipWriter.Header.ModTime = archiveTime
	gzipWriter.Header.OS = 255

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Files to include (in order)
	files := []string{
		"install-manifest.txt",
		"sha256sums.txt",
	}

	// Add payload files
	payloadDir := filepath.Join(stageDir, "payload")
	var payloadFiles []string
	filepath.Walk(payloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(stageDir, path)
		relPath = filepath.ToSlash(relPath)
		payloadFiles = append(payloadFiles, relPath)
		return nil
	})
	sort.Strings(payloadFiles)
	files = append(files, payloadFiles...)

	for _, relPath := range files {
		fullPath := filepath.Join(stageDir, filepath.FromSlash(relPath))
		info, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)
		// Normalize for reproducibility
		header.ModTime = archiveTime
		header.AccessTime = archiveTime
		header.ChangeTime = archiveTime
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return err
			}
			if _, err := tarWriter.Write(data); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *Builder) createChecksumSidecar(bundlePath string) error {
	hash, err := ComputeFileHash(bundlePath)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("%s  %s\n", hash, filepath.Base(bundlePath))
	return os.WriteFile(bundlePath+".sha256", []byte(content), 0644)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// ComputeFileHash computes SHA256 hash of a file
func ComputeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// HumanReadableSize converts bytes to human readable format
func HumanReadableSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func reproducibleArchiveTime() time.Time {
	const fallbackEpoch = int64(0)

	sourceDateEpoch := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH"))
	if sourceDateEpoch == "" {
		return time.Unix(fallbackEpoch, 0).UTC()
	}

	seconds, err := strconv.ParseInt(sourceDateEpoch, 10, 64)
	if err != nil || seconds < 0 {
		return time.Unix(fallbackEpoch, 0).UTC()
	}
	return time.Unix(seconds, 0).UTC()
}
