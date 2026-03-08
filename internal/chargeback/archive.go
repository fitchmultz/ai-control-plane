// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Persist rendered report outputs using repo-standard filesystem safety.
//
// Responsibilities:
//   - Resolve archive destinations relative to the repo root when needed.
//   - Create archive directories.
//   - Write report artifacts atomically.
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
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

type FileArchiver struct{}

func (FileArchiver) Archive(repoRoot string, archiveDir string, reportMonth string, outputs ReportOutputs) (map[string]string, error) {
	archiveBase := resolveArchiveBase(repoRoot, archiveDir)
	if archiveBase == "" {
		return map[string]string{}, nil
	}
	targetDir := filepath.Join(archiveBase, reportMonth)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create archive directory: %w", err)
	}

	files := map[string]string{
		"md":   outputs.Markdown,
		"json": outputs.JSON,
		"csv":  outputs.CSV,
	}
	paths := make(map[string]string, len(files))
	for extension, content := range files {
		path := filepath.Join(targetDir, fmt.Sprintf("chargeback-report-%s.%s", reportMonth, extension))
		if err := fsutil.AtomicWriteFile(path, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write archive %s: %w", path, err)
		}
		paths[extension] = path
	}
	return paths, nil
}

func resolveArchiveBase(repoRoot string, archiveDir string) string {
	trimmed := strings.TrimSpace(archiveDir)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) || strings.TrimSpace(repoRoot) == "" {
		return trimmed
	}
	return filepath.Join(repoRoot, trimmed)
}
