// check_env_test.go - Focused coverage for environment-variable doctor checks.
//
// Purpose:
//   - Verify required env detection and `--fix` behavior for doctor env checks.
//
// Responsibilities:
//   - Cover missing/present env cases and env-file fix behavior.
//
// Scope:
//   - Environment configuration diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Environment-mutating tests stay non-parallel.
package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestEnvVarsSetCheckID(t *testing.T) {
	t.Parallel()
	if (envVarsSetCheck{}).ID() != "env_vars_set" {
		t.Fatalf("expected ID env_vars_set")
	}
}

func TestEnvVarsSetCheckRunWithMissingVars(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "")
	t.Setenv("LITELLM_SALT_KEY", "")
	t.Setenv("DATABASE_URL", "")

	result := (envVarsSetCheck{}).Run(context.Background(), Options{RepoRoot: t.TempDir()})
	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy result, got %v", result.Level)
	}
	if result.Severity != SeverityPrereq {
		t.Fatalf("expected prereq severity, got %v", result.Severity)
	}
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for missing vars")
	}
}

func TestEnvVarsSetCheckRunWithVarsSet(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "test-key")
	t.Setenv("LITELLM_SALT_KEY", "test-salt")
	t.Setenv("DATABASE_URL", "test-db-url")

	result := (envVarsSetCheck{}).Run(context.Background(), Options{})
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy result, got %v", result.Level)
	}
}

func TestEnvVarsSetCheckFixCreatesRepoEnv(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	demoDir := filepath.Join(repoRoot, "demo")
	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(demoDir, ".env.example"), []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	applied, message, err := (envVarsSetCheck{}).Fix(context.Background(), Options{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Fix() error = %v", err)
	}
	if !applied || message == "" {
		t.Fatalf("expected fix to apply, got applied=%t message=%q", applied, message)
	}
	if _, err := os.Stat(filepath.Join(demoDir, ".env")); err != nil {
		t.Fatalf("expected .env to be created: %v", err)
	}
}

func TestEnvVarsSetCheckFixSkipsExistingEnv(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	demoDir := filepath.Join(repoRoot, "demo")
	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(demoDir, ".env"), []byte("existing\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	applied, message, err := (envVarsSetCheck{}).Fix(context.Background(), Options{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Fix() error = %v", err)
	}
	if applied || message != "" {
		t.Fatalf("expected no-op fix, got applied=%t message=%q", applied, message)
	}
}
