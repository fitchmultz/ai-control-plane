// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Persist rendered report outputs using repo-standard filesystem safety.
//
// Responsibilities:
//   - Resolve archive destinations relative to the repo root when needed.
//   - Create private archive directories.
//   - Write report artifacts atomically with private default modes.
//
// Non-scope:
//   - Does not decide what content to render.
//   - Does not send notifications.
//
// Invariants/Assumptions:
//   - Parent directories are local and writable.
//   - Atomic writes stay on the same filesystem as the destination path.
//
// Scope:
//   - Chargeback artifact archival only.
//
// Usage:
//   - Used by the chargeback workflow and archival-focused tests.
package chargeback

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

type FileArchiver struct{}

func (FileArchiver) Archive(repoRoot string, archiveDir string, reportMonth string, fileBase string, outputs ReportOutputs) (map[string]string, error) {
	archiveBase := resolveArchiveBase(repoRoot, archiveDir)
	if archiveBase == "" {
		return map[string]string{}, nil
	}
	targetDir := filepath.Join(archiveBase, reportMonth)
	if err := fsutil.EnsurePrivateDir(targetDir); err != nil {
		return nil, fmt.Errorf("create archive directory: %w", err)
	}

	files := map[string]string{
		"md":   outputs.Markdown,
		"json": outputs.JSON,
		"csv":  outputs.CSV,
	}
	base := strings.TrimSpace(fileBase)
	if base == "" {
		base = archiveFileBase(reportMonth, ReportScope{Kind: ReportScopeGlobal})
	}
	paths := make(map[string]string, len(files))
	for extension, content := range files {
		path := filepath.Join(targetDir, fmt.Sprintf("%s.%s", base, extension))
		if err := fsutil.AtomicWritePrivateFile(path, []byte(content)); err != nil {
			return nil, fmt.Errorf("write archive %s: %w", path, err)
		}
		paths[extension] = path
	}
	return paths, nil
}

func resolveArchiveBase(repoRoot string, archiveDir string) string {
	trimmed := textutil.Trim(archiveDir)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) || textutil.IsBlank(repoRoot) {
		return trimmed
	}
	return filepath.Join(repoRoot, trimmed)
}

func archiveFileBase(reportMonth string, scope ReportScope) string {
	base := fmt.Sprintf("chargeback-report-%s", strings.TrimSpace(reportMonth))
	if suffix := strings.TrimSpace(scope.ArchiveSuffix); suffix != "" {
		return base + "-" + suffix
	}
	return base
}
