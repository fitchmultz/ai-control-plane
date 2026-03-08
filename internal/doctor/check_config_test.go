// check_config_test.go - Focused coverage for repository configuration checks.
//
// Purpose:
//   - Verify doctor config checks classify missing files and repo env warnings correctly.
//
// Responsibilities:
//   - Cover missing required files, missing env warning, and healthy repo config.
//
// Scope:
//   - Repository configuration diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests construct temp repo fixtures explicitly.
package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestConfigValidCheckID(t *testing.T) {
	t.Parallel()
	if (configValidCheck{}).ID() != "config_valid" {
		t.Fatalf("expected ID config_valid")
	}
}

func TestConfigValidCheckRunMissingRequiredFiles(t *testing.T) {
	t.Parallel()

	result := (configValidCheck{}).Run(context.Background(), Options{RepoRoot: t.TempDir()})
	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy result, got %v", result.Level)
	}
}

func TestConfigValidCheckRunMissingEnvWarning(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeDoctorFile(t, repoRoot, "demo/docker-compose.yml", "services: {}\n")
	writeDoctorFile(t, repoRoot, "demo/config/litellm.yaml", "model_list: []\n")

	result := (configValidCheck{}).Run(context.Background(), Options{RepoRoot: repoRoot})
	if result.Level != status.HealthLevelWarning || result.Severity != SeverityPrereq {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestConfigValidCheckRunHealthy(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeDoctorFile(t, repoRoot, "demo/docker-compose.yml", "services: {}\n")
	writeDoctorFile(t, repoRoot, "demo/config/litellm.yaml", "model_list: []\n")
	writeDoctorFile(t, repoRoot, "demo/.env", "KEY=value\n")

	result := (configValidCheck{}).Run(context.Background(), Options{RepoRoot: repoRoot})
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy result, got %+v", result)
	}
}

func writeDoctorFile(t *testing.T, repoRoot string, relativePath string, contents string) {
	t.Helper()
	path := filepath.Join(repoRoot, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
