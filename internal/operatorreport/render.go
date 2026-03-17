// Package operatorreport renders canonical operator-facing runtime reports.
//
// Purpose:
//   - Provide one-command operator reporting on top of the shared runtime model.
//
// Responsibilities:
//   - Render typed runtime status reports as markdown-like text or JSON.
//   - Archive generated reports using private local-only filesystem helpers.
//
// Scope:
//   - Operator-report rendering and archival only.
//
// Usage:
//   - Used by `acpctl ops report` and `make operator-report`.
//
// Invariants/Assumptions:
//   - Archived reports remain private local artifacts.
package operatorreport

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// Format identifies the supported operator-report output formats.
type Format string

const (
	// FormatMarkdown renders the shared human-readable runtime report.
	FormatMarkdown Format = "markdown"
	// FormatJSON renders the shared JSON runtime report.
	FormatJSON Format = "json"
)

// Request captures render-time report preferences.
type Request struct {
	Format Format
	Wide   bool
}

// Render formats the shared runtime report for operator-facing consumption.
func Render(report status.StatusReport, req Request) ([]byte, string, error) {
	switch req.Format {
	case FormatJSON:
		var buf bytes.Buffer
		if err := report.WriteJSON(&buf); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "json", nil
	default:
		var buf bytes.Buffer
		if err := report.WriteHuman(&buf, req.Wide); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "md", nil
	}
}

// Archive persists the rendered operator report under a private local archive root.
func Archive(repoRoot string, archiveDir string, stamp string, payload []byte, ext string) (string, error) {
	if archiveDir == "" {
		return "", nil
	}
	targetDir := filepath.Join(repoRoot, archiveDir, stamp)
	if err := fsutil.EnsurePrivateDir(targetDir); err != nil {
		return "", fmt.Errorf("create operator report directory: %w", err)
	}
	path := filepath.Join(targetDir, fmt.Sprintf("operator-report-%s.%s", stamp, ext))
	if err := fsutil.AtomicWritePrivateFile(path, payload); err != nil {
		return "", fmt.Errorf("write operator report: %w", err)
	}
	return path, nil
}
