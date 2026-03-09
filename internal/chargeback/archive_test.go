// archive_test.go - Permission-focused coverage for chargeback archive writes.
//
// Purpose:
//   - Verify chargeback archives honor the repo's private local-artifact policy.
//
// Responsibilities:
//   - Assert archive directories are created with private modes.
//   - Assert archived report files are written with private modes.
//
// Scope:
//   - FileArchiver filesystem behavior only.
//
// Usage:
//   - Run via `go test ./internal/chargeback`.
//
// Invariants/Assumptions:
//   - Tests run on filesystems that report owner/group/other mode bits.
package chargeback

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

func TestFileArchiverArchiveUsesPrivateModes(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	archived, err := (FileArchiver{}).Archive(repoRoot, defaultArchiveDir, "2026-02", ReportOutputs{
		Markdown: "# report\n",
		JSON:     "{\"ok\":true}\n",
		CSV:      "month,total\n2026-02,10\n",
	})
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	targetDir := filepath.Join(repoRoot, "demo", "backups", "chargeback", "2026-02")
	assertMode(t, targetDir, fsutil.PrivateDirPerm)

	if len(archived) != 3 {
		t.Fatalf("archived files = %d, want 3", len(archived))
	}
	for extension, path := range archived {
		if filepath.Dir(path) != targetDir {
			t.Fatalf("archive %s dir = %s, want %s", extension, filepath.Dir(path), targetDir)
		}
		assertMode(t, path, fsutil.PrivateFilePerm)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}
