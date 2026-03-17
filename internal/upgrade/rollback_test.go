// rollback_test.go - Tests for upgrade rollback guards.
//
// Purpose:
//   - Verify rollback safety checks and persisted-run parsing.
//
// Responsibilities:
//   - Require rollback execution from the recorded previous release checkout.
//
// Scope:
//   - Upgrade rollback tests only.
//
// Usage:
//   - Run via `go test ./internal/upgrade`.
//
// Invariants/Assumptions:
//   - Rollback must not run from the wrong checkout.
package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
)

func TestRollbackRequiresPreviousReleaseCheckout(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "VERSION"), []byte("0.2.0\n"), 0o644); err != nil {
		t.Fatalf("write VERSION: %v", err)
	}

	runDir := t.TempDir()
	summary := Summary{
		FromVersion:        "0.1.0",
		ToVersion:          "0.2.0",
		ConfigBackupPath:   filepath.Join(runDir, configBackupName),
		DatabaseBackupPath: filepath.Join(runDir, databaseBackup),
	}
	if err := os.WriteFile(summary.ConfigBackupPath, []byte("KEY=value\n"), 0o600); err != nil {
		t.Fatalf("write config backup: %v", err)
	}
	if err := os.WriteFile(summary.DatabaseBackupPath, []byte("not-gzip"), 0o600); err != nil {
		t.Fatalf("write database backup: %v", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if _, err := artifactrun.Finalize(runDir, t.TempDir(), artifactrun.FinalizeOptions{InventoryName: InventoryName}); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	_, err := Rollback(context.Background(), RollbackOptions{
		RepoRoot: repoRoot,
		RunDir:   runDir,
	})
	if err == nil {
		t.Fatal("expected rollback checkout guard to fail")
	}
	if !strings.Contains(err.Error(), "current checkout VERSION is 0.2.0 but target version is 0.1.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
