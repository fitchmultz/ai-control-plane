// env_test.go - Coverage for repo env-file and required-variable helpers.
//
// Purpose:
//   - Verify repo env status and required runtime env attribution.
//
// Responsibilities:
//   - Cover normalized env-file lookup behavior.
//   - Cover process-vs-repo source attribution and missing-key reporting.
//
// Scope:
//   - Env-file and required-key behavior only.
//
// Usage:
//   - Run via `go test ./internal/config`.
//
// Invariants/Assumptions:
//   - Tests use temp files instead of real repo state.
package config

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestEnvFileLookupNormalizesAndTrimsValues(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "demo.env")
	if err := os.WriteFile(envPath, []byte("VALUE=  abc  \n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	envFile := NewEnvFile(filepath.Join(filepath.Dir(envPath), ".", filepath.Base(envPath)))
	if envFile.Path() != envPath {
		t.Fatalf("Path() = %q, want %q", envFile.Path(), envPath)
	}
	value, ok, err := envFile.Lookup("VALUE")
	if err != nil || !ok {
		t.Fatalf("Lookup() error = %v ok=%t", err, ok)
	}
	if value != "abc" {
		t.Fatalf("Lookup() = %q", value)
	}
}

func TestLoaderRepoEnvStatus(t *testing.T) {
	repoRoot := t.TempDir()
	loader := NewLoader().WithRepoRoot(repoRoot)

	status, err := loader.RepoEnvStatus(context.Background())
	if err != nil {
		t.Fatalf("RepoEnvStatus() error = %v", err)
	}
	if status.Exists {
		t.Fatal("expected missing env file before creation")
	}

	envPath := filepath.Join(repoRoot, "demo", ".env")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(envPath, []byte("KEY=value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	status, err = loader.RepoEnvStatus(context.Background())
	if err != nil {
		t.Fatalf("RepoEnvStatus() error = %v", err)
	}
	if !status.Exists || status.Path != envPath {
		t.Fatalf("unexpected RepoEnvStatus() = %+v", status)
	}
}

func TestLoaderRequiredRuntimeEnvTracksSources(t *testing.T) {
	loader := NewTestLoader(map[string]string{
		"PROCESS_ONLY": "one",
		"OVERRIDE":     "process",
	}, "/repo", map[string]string{
		"REPO_ONLY": "two",
		"OVERRIDE":  "repo",
	})

	status := loader.RequiredRuntimeEnv([]string{"PROCESS_ONLY", "REPO_ONLY", "OVERRIDE", "MISSING", ""})
	if !slices.Equal(status.Found, []string{"OVERRIDE", "PROCESS_ONLY", "REPO_ONLY"}) {
		t.Fatalf("Found = %v", status.Found)
	}
	if !slices.Equal(status.Missing, []string{"MISSING"}) {
		t.Fatalf("Missing = %v", status.Missing)
	}
	if status.Sources["PROCESS_ONLY"] != "process" || status.Sources["REPO_ONLY"] != "repo" || status.Sources["OVERRIDE"] != "process" {
		t.Fatalf("unexpected Sources = %#v", status.Sources)
	}
}
