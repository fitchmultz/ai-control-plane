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
	if err := WriteArtifacts(runDir, []Artifact{{Path: "summary.json", Body: []byte("{}\n"), Perm: 0o644}}); err != nil {
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
