// Package filesync implements typed file synchronization workflows.
//
// Purpose:
//
//	Provide deterministic file sync behavior for control-plane workflows that
//	are migrating from shell scripts to typed Go commands.
//
// Responsibilities:
//   - Define canonical source-to-destination file mappings.
//   - Copy mapped files from repository sources into Helm chart file paths.
//   - Create destination directories when needed.
//
// Non-scope:
//   - Does not delete extraneous destination files.
//   - Does not perform Git operations.
//
// Invariants/Assumptions:
//   - Mapping paths are repository-relative POSIX-style paths.
//   - Caller provides a valid repository root path.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package filesync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Mapping defines one repository-relative source to destination copy pair.
type Mapping struct {
	Source      string
	Destination string
}

// HelmMappings mirrors the legacy sync_helm_files.sh source-to-destination map.
var HelmMappings = []Mapping{
	{Source: "demo/scripts/chargeback_report.sh", Destination: "deploy/helm/ai-control-plane/files/chargeback_report.sh"},
	{Source: "demo/scripts/lib/chargeback_db.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_db.sh"},
	{Source: "demo/scripts/lib/chargeback_analysis.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_analysis.sh"},
	{Source: "demo/scripts/lib/chargeback_render.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_render.sh"},
	{Source: "demo/scripts/lib/chargeback_io.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_io.sh"},
}

// SyncOptions controls Helm file synchronization behavior.
type SyncOptions struct {
	RepoRoot string
	Writer   io.Writer
}

// SyncHelmFiles copies canonical source files into Helm chart files paths.
func SyncHelmFiles(options SyncOptions) error {
	repoRoot := strings.TrimSpace(options.RepoRoot)
	if repoRoot == "" {
		return fmt.Errorf("repository root is required")
	}

	out := options.Writer
	if out == nil {
		out = io.Discard
	}

	fmt.Fprintln(out, "Synchronizing Helm chart files...")
	for _, mapping := range HelmMappings {
		fmt.Fprintf(out, "  Syncing %s -> %s\n", mapping.Source, mapping.Destination)
		src := filepath.Join(repoRoot, filepath.FromSlash(mapping.Source))
		dst := filepath.Join(repoRoot, filepath.FromSlash(mapping.Destination))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("sync %s -> %s: %w", mapping.Source, mapping.Destination, err)
		}
	}
	fmt.Fprintln(out, "✓ Synchronization complete.")
	return nil
}

func copyFile(src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", src)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return fmt.Errorf("copy bytes: %w", err)
	}
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("close destination: %w", err)
	}
	if err := os.Chmod(dst, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("set destination mode: %w", err)
	}

	return nil
}
