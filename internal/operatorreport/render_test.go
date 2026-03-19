// render_test.go - Coverage for operator-report rendering helpers.
//
// Purpose:
//   - Verify operator reports render and archive deterministically.
//
// Responsibilities:
//   - Cover markdown and JSON rendering paths.
//   - Verify archived reports use private local filesystem modes.
//
// Scope:
//   - Operator-report helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/operatorreport`.
//
// Invariants/Assumptions:
//   - Tests use temp directories instead of repository state.
package operatorreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestRenderSupportsMarkdownJSONAndHTML(t *testing.T) {
	report := status.StatusReport{
		Overall: status.HealthLevelHealthy,
		Components: map[string]status.ComponentStatus{
			"gateway": {Name: "gateway", Level: status.HealthLevelHealthy, Message: "ok", Details: status.ComponentDetails{BaseURL: "http://127.0.0.1:4000"}},
		},
		Timestamp: "2026-03-17T00:00:00Z",
		Duration:  "10ms",
	}

	payload, ext, err := Render(report, Request{Format: FormatMarkdown})
	if err != nil {
		t.Fatalf("Render(markdown) error = %v", err)
	}
	if ext != "md" || !strings.Contains(string(payload), "AI Control Plane Status") {
		t.Fatalf("unexpected markdown payload: ext=%q payload=%q", ext, payload)
	}

	payload, ext, err = Render(report, Request{Format: FormatJSON})
	if err != nil {
		t.Fatalf("Render(json) error = %v", err)
	}
	if ext != "json" || !strings.Contains(string(payload), `"overall": "healthy"`) {
		t.Fatalf("unexpected json payload: ext=%q payload=%q", ext, payload)
	}

	payload, ext, err = Render(report, Request{Format: FormatHTML, Wide: true})
	if err != nil {
		t.Fatalf("Render(html) error = %v", err)
	}
	if ext != "html" || !strings.Contains(string(payload), "AI Control Plane Operator Dashboard") || !strings.Contains(string(payload), "base_url: http://127.0.0.1:4000") {
		t.Fatalf("unexpected html payload: ext=%q payload=%q", ext, payload)
	}
}

func TestArchiveWritesPrivateArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	path, err := Archive(repoRoot, "demo/backups/operator-reports", "2026-03-17T000000Z", []byte("report"), "md")
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("2026-03-17T000000Z", "operator-report-2026-03-17T000000Z.md")) {
		t.Fatalf("unexpected archive path: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("archive file mode = %#o, want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat(dir) error = %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("archive dir mode = %#o, want 0700", dirInfo.Mode().Perm())
	}
}
