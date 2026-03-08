// artifact_run_test.go - Tests for shared artifact-run primitives.
//
// Purpose:
//
//	Verify collision-safe run creation and inventory-backed verification.
//
// Responsibilities:
//   - Ensure same-timestamp run creation stays unique.
//   - Confirm inventory verification detects drift.
//
// Scope:
//   - Covers internal/artifactrun behavior only.
//
// Usage:
//   - Run via `go test ./internal/artifactrun`.
//
// Invariants/Assumptions:
//   - Tests use temporary directories and local-only generated files.
package artifactrun

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

func TestCreateAvoidsTimestampCollisions(t *testing.T) {
	outputRoot := t.TempDir()
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	first, err := Create(outputRoot, "readiness", now)
	if err != nil {
		t.Fatalf("Create() first error = %v", err)
	}
	second, err := Create(outputRoot, "readiness", now)
	if err != nil {
		t.Fatalf("Create() second error = %v", err)
	}

	if first.ID == second.ID {
		t.Fatalf("run IDs collided: %q", first.ID)
	}
	if first.Directory == second.Directory {
		t.Fatalf("run directories collided: %q", first.Directory)
	}
}

func TestFinalizeAndVerifyDetectInventoryDrift(t *testing.T) {
	runDir := t.TempDir()
	if err := WriteArtifacts(runDir, []Artifact{{Path: "summary.json", Body: []byte("{}\n")}}); err != nil {
		t.Fatalf("WriteArtifacts() error = %v", err)
	}
	if _, err := Finalize(runDir, filepath.Dir(runDir), FinalizeOptions{InventoryName: "inventory.txt"}); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "inventory.txt"), []byte("wrong.txt\n"), 0o644); err != nil {
		t.Fatalf("overwrite inventory: %v", err)
	}
	err := Verify(runDir, VerifyOptions{InventoryName: "inventory.txt", RequiredFiles: []string{"summary.json"}})
	if err == nil {
		t.Fatal("expected inventory mismatch")
	}
}

func TestCreateAndWriteArtifactsUsePrivateModes(t *testing.T) {
	outputRoot := filepath.Join(t.TempDir(), "evidence")
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	run, err := Create(outputRoot, "readiness", now)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := mustMode(t, outputRoot); got != fsutil.PrivateDirPerm {
		t.Fatalf("output root mode = %04o, want %04o", got, fsutil.PrivateDirPerm)
	}
	if got := mustMode(t, run.Directory); got != fsutil.PrivateDirPerm {
		t.Fatalf("run dir mode = %04o, want %04o", got, fsutil.PrivateDirPerm)
	}

	if err := WriteArtifacts(run.Directory, []Artifact{{Path: "summary.json", Body: []byte("{}\n")}}); err != nil {
		t.Fatalf("WriteArtifacts() error = %v", err)
	}
	if got := mustMode(t, filepath.Join(run.Directory, "summary.json")); got != fsutil.PrivateFilePerm {
		t.Fatalf("artifact mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
	}

	if err := WriteJSON(filepath.Join(run.Directory, "summary-extra.json"), map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if got := mustMode(t, filepath.Join(run.Directory, "summary-extra.json")); got != fsutil.PrivateFilePerm {
		t.Fatalf("json mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
	}

	if _, err := Finalize(run.Directory, outputRoot, FinalizeOptions{InventoryName: "inventory.txt", LatestPointers: []string{"latest.txt"}}); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if got := mustMode(t, filepath.Join(run.Directory, "inventory.txt")); got != fsutil.PrivateFilePerm {
		t.Fatalf("inventory mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
	}
	if got := mustMode(t, filepath.Join(outputRoot, "latest.txt")); got != fsutil.PrivateFilePerm {
		t.Fatalf("latest pointer mode = %04o, want %04o", got, fsutil.PrivateFilePerm)
	}
}

func mustMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%s) error = %v", path, err)
	}
	return info.Mode().Perm()
}
