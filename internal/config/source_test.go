// source_test.go - Coverage for loader lookup and repo-root resolution helpers.
//
// Purpose:
//   - Verify typed config loader lookup precedence and repo-root behavior.
//
// Responsibilities:
//   - Cover process-only and repo-aware lookups.
//   - Cover parsing helpers and explicit repo-root overrides.
//
// Scope:
//   - Loader helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/config`.
//
// Invariants/Assumptions:
//   - Tests avoid shelling out to git by using explicit repo-root inputs.
package config

import (
	"context"
	"strings"
	"testing"
)

func TestLoaderLookupPrecedenceAndDefaults(t *testing.T) {
	loader := NewTestLoader(map[string]string{
		"PROCESS_ONLY": " process ",
		"BOOL_TRUE":    "true",
		"FLOAT_VALUE":  "1.5",
		"INT_VALUE":    "9",
	}, "/repo", map[string]string{
		"PROCESS_ONLY": " repo should not win ",
		"REPO_ONLY":    " repo ",
	})

	if value, ok, err := loader.LookupProcess("PROCESS_ONLY"); err != nil || !ok || value != " process " {
		t.Fatalf("LookupProcess() = %q, %t, %v", value, ok, err)
	}
	if value, ok, err := loader.LookupRepoAware("REPO_ONLY"); err != nil || !ok || value != " repo " {
		t.Fatalf("LookupRepoAware() = %q, %t, %v", value, ok, err)
	}
	if got := loader.String("PROCESS_ONLY"); got != "process" {
		t.Fatalf("String() = %q", got)
	}
	if got := loader.RepoAwareString("REPO_ONLY"); got != "repo" {
		t.Fatalf("RepoAwareString() = %q", got)
	}
	if got := loader.StringDefault("MISSING", "fallback"); got != "fallback" {
		t.Fatalf("StringDefault() = %q", got)
	}
	if got := loader.RepoAwareStringDefault("MISSING", "fallback"); got != "fallback" {
		t.Fatalf("RepoAwareStringDefault() = %q", got)
	}
	if !loader.BoolDefault("BOOL_TRUE", false) {
		t.Fatal("BoolDefault() should parse true")
	}
	if loader.BoolDefault("BAD_BOOL", true) != true {
		t.Fatal("BoolDefault() should fall back")
	}
	if value := loader.Float64Ptr("FLOAT_VALUE"); value == nil || *value != 1.5 {
		t.Fatalf("Float64Ptr() = %v", value)
	}
	if value := loader.Float64Ptr("BAD_FLOAT"); value != nil {
		t.Fatalf("Float64Ptr() = %v, want nil", value)
	}
	if value := loader.Int64Ptr("INT_VALUE"); value == nil || *value != 9 {
		t.Fatalf("Int64Ptr() = %v", value)
	}
	if value := loader.Int64Ptr("BAD_INT"); value != nil {
		t.Fatalf("Int64Ptr() = %v, want nil", value)
	}
}

func TestLoaderWithRepoRootResetsRepoFileCache(t *testing.T) {
	loader := NewTestLoader(nil, "/repo-a", map[string]string{"VALUE": "a"})
	clone := loader.WithRepoRoot("/repo-b")

	if clone == loader {
		t.Fatal("expected WithRepoRoot to clone the loader")
	}
	if clone.repoRoot != "/repo-b" {
		t.Fatalf("clone repoRoot = %q", clone.repoRoot)
	}
	if clone.repoFile != nil {
		t.Fatal("expected repo file cache to be cleared")
	}
}

func TestLoaderRepoSourceUsesRepoRoot(t *testing.T) {
	loader := NewLoader().WithRepoRoot("/repo-root")

	source, err := loader.repoSource()
	if err != nil {
		t.Fatalf("repoSource() error = %v", err)
	}
	fileSource, ok := source.(fileEnvSource)
	if !ok {
		t.Fatalf("repoSource() type = %T", source)
	}
	if !strings.HasSuffix(fileSource.path, "/repo-root/demo/.env") {
		t.Fatalf("repoSource() path = %q", fileSource.path)
	}
}

func TestLoaderRepoRootUsesExplicitProcessValue(t *testing.T) {
	loader := NewLoader()
	t.Setenv("ACP_REPO_ROOT", "/explicit/repo")

	repoRoot, err := loader.RepoRoot(context.Background())
	if err != nil {
		t.Fatalf("RepoRoot() error = %v", err)
	}
	if repoRoot != "/explicit/repo" {
		t.Fatalf("RepoRoot() = %q", repoRoot)
	}
}

func TestLoaderRequireRepoRootRejectsEmptyValue(t *testing.T) {
	loader := NewTestLoader(nil, "", nil)

	if _, err := loader.RequireRepoRoot(context.Background()); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}
