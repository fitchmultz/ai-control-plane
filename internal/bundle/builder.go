// builder.go - Release bundle builder.
//
// Purpose:
//
//	Build reproducible release bundles with checksums and manifests.
//
// Responsibilities:
//   - Copy canonical files to a staging payload.
//   - Create install manifests and payload checksums.
//   - Create reproducible tarballs and checksum sidecars.
//
// Scope:
//   - Release tarball construction only.
//
// Usage:
//   - Called by `acpctl deploy release-bundle build`.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package bundle

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

type releaseMetadata struct {
	SchemaVersion string   `json:"schema_version"`
	Product       string   `json:"product"`
	Version       string   `json:"version"`
	BundleName    string   `json:"bundle_name"`
	PayloadRoot   string   `json:"payload_root"`
	Canonical     []string `json:"canonical_files"`
}

// Builder handles bundle construction.
type Builder struct {
	repoRoot string
	verbose  bool
}

// NewBuilder creates a new bundle builder.
func NewBuilder(repoRoot string, verbose bool) *Builder {
	return &Builder{
		repoRoot: repoRoot,
		verbose:  verbose,
	}
}

// Build creates a release bundle from the plan.
func (b *Builder) Build(ctx context.Context, plan *Plan) (err error) {
	logger := logging.WorkflowLogger(ctx,
		slog.String("component", "bundle"),
		slog.String("workflow", "release_bundle_build"),
		slog.String("bundle_path", plan.BundlePath),
	)
	logging.WorkflowStart(logger, slog.String("output_dir", plan.OutputDir))
	defer func() {
		if err != nil {
			logging.WorkflowFailure(logger, err)
		}
	}()

	if err = fsutil.EnsurePrivateDir(plan.OutputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	stageDir, err := os.MkdirTemp("", "release-bundle-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	payloadDir := filepath.Join(stageDir, "payload")
	if err = fsutil.EnsurePrivateDir(payloadDir); err != nil {
		return fmt.Errorf("failed to create payload directory: %w", err)
	}

	if b.verbose {
		logger.Info("stage.directory", slog.String("path", stageDir))
	}

	if err = b.copyPayloadFiles(ctx, payloadDir); err != nil {
		return err
	}

	installManifest := filepath.Join(stageDir, "install-manifest.txt")
	manifestPaths := append([]string(nil), CanonicalPaths...)
	sort.Strings(manifestPaths)
	manifestContent := strings.Join(manifestPaths, "\n") + "\n"
	if err = fsutil.AtomicWritePrivateFile(installManifest, []byte(manifestContent)); err != nil {
		return fmt.Errorf("failed to create install manifest: %w", err)
	}

	sha256sumsPath := filepath.Join(stageDir, "sha256sums.txt")
	if err = b.buildChecksums(payloadDir, sha256sumsPath); err != nil {
		return fmt.Errorf("failed to build checksums: %w", err)
	}

	releaseMetadataPath := filepath.Join(stageDir, "release-metadata.json")
	metadata := releaseMetadata{
		SchemaVersion: "1",
		Product:       "ai-control-plane",
		Version:       plan.Version,
		BundleName:    GetBundleName(plan.Version),
		PayloadRoot:   "payload",
		Canonical:     append([]string(nil), CanonicalPaths...),
	}
	sort.Strings(metadata.Canonical)
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal release metadata: %w", err)
	}
	metadataBytes = append(metadataBytes, '\n')
	if err = fsutil.AtomicWritePrivateFile(releaseMetadataPath, metadataBytes); err != nil {
		return fmt.Errorf("failed to create release metadata: %w", err)
	}

	if err = b.createReproducibleTarball(stageDir, plan.BundlePath); err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	if err = b.createChecksumSidecar(plan.BundlePath); err != nil {
		return fmt.Errorf("failed to create checksum: %w", err)
	}

	logging.WorkflowComplete(logger, slog.String("bundle_path", plan.BundlePath))
	return nil
}

func (b *Builder) copyPayloadFiles(ctx context.Context, payloadDir string) error {
	logger := logging.FromContext(ctx).With(
		slog.String("component", "bundle"),
		slog.String("step", "copy_payload_files"),
	)
	for _, relPath := range CanonicalPaths {
		src := filepath.Join(b.repoRoot, relPath)
		dst := filepath.Join(payloadDir, relPath)
		if err := fsutil.EnsurePrivateDir(filepath.Dir(dst)); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", relPath, err)
		}
		if b.verbose {
			logger.Info("payload.file_copied", slog.String("path", relPath))
		}
	}
	return nil
}

func (b *Builder) buildChecksums(payloadDir string, outputPath string) error {
	payloadFiles, err := collectRegularFiles(payloadDir)
	if err != nil {
		return err
	}

	checksums := make([]string, 0, len(payloadFiles))
	for _, relPath := range payloadFiles {
		fullPath := filepath.Join(payloadDir, filepath.FromSlash(relPath))
		hash, err := ComputeFileHash(fullPath)
		if err != nil {
			return fmt.Errorf("hash payload file %s: %w", relPath, err)
		}
		checksums = append(checksums, fmt.Sprintf("%s  ./%s", hash, relPath))
	}

	sort.Strings(checksums)
	content := strings.Join(checksums, "\n") + "\n"
	return fsutil.AtomicWritePrivateFile(outputPath, []byte(content))
}

func (b *Builder) createReproducibleTarball(stageDir string, outputPath string) error {
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fsutil.PrivateFilePerm)
	if err != nil {
		return err
	}
	if err := file.Chmod(fsutil.PrivateFilePerm); err != nil {
		_ = file.Close()
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

	files := []string{
		"install-manifest.txt",
		"release-metadata.json",
		"sha256sums.txt",
	}

	payloadDir := filepath.Join(stageDir, "payload")
	payloadFiles, err := collectRegularFiles(payloadDir)
	if err != nil {
		return err
	}
	for _, relPath := range payloadFiles {
		files = append(files, filepath.ToSlash(filepath.Join("payload", relPath)))
	}

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

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		if _, err := tarWriter.Write(data); err != nil {
			return err
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
	return fsutil.AtomicWritePrivateFile(bundlePath+".sha256", []byte(content))
}

func collectRegularFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect files under %s: %w", root, err)
	}
	sort.Strings(files)
	return files, nil
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return fsutil.AtomicWritePrivateFile(dst, data)
}

// ComputeFileHash computes the SHA256 hash of a file.
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

// HumanReadableSize converts bytes to human readable format.
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

	sourceDateEpoch := config.NewLoader().Tooling().SourceDateEpoch
	if sourceDateEpoch == "" {
		return time.Unix(fallbackEpoch, 0).UTC()
	}

	seconds, err := strconv.ParseInt(sourceDateEpoch, 10, 64)
	if err != nil || seconds < 0 {
		return time.Unix(fallbackEpoch, 0).UTC()
	}
	return time.Unix(seconds, 0).UTC()
}
