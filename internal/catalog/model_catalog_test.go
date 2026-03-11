// model_catalog_test.go - Coverage for tracked model-catalog helpers.
//
// Purpose:
//   - Verify model catalog loading, views, and validation behavior.
//
// Responsibilities:
//   - Cover not-found and parse failures for the tracked model catalog.
//   - Cover alias views, managed-browser defaults, and duplicate detection.
//
// Scope:
//   - Model catalog loader and view behavior only.
//
// Usage:
//   - Run via `go test ./internal/catalog`.
//
// Invariants/Assumptions:
//   - Fixtures are temporary and deterministic.
package catalog

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadModelCatalogErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadModelCatalog(missing); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}

	badPath := filepath.Join(t.TempDir(), "bad.yaml")
	writeModelCatalogFile(t, badPath, "online_models: [")
	if _, err := LoadModelCatalog(badPath); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestModelCatalogViewsAndValidation(t *testing.T) {
	config := ModelCatalog{
		OnlineModels: []CatalogModel{
			{Alias: " claude-sonnet-4-5 ", UpstreamModel: "anthropic/claude-sonnet-4-5", CredentialEnv: "ANTHROPIC_API_KEY", ManagedUIDefault: true},
			{Alias: "openai-gpt5.2", UpstreamModel: "openai/gpt-5.2", CredentialEnv: "OPENAI_API_KEY", ManagedUIDefault: true},
			{Alias: "openai-gpt5.2", UpstreamModel: "openai/gpt-5.2", CredentialEnv: "OPENAI_API_KEY"},
		},
		OfflineModels: []CatalogModel{
			{Alias: "mock-gpt", UpstreamModel: "openai/mock-gpt"},
			{Alias: "mock-claude", UpstreamModel: "openai/mock-claude"},
		},
	}

	if !slices.Equal(config.ManagedUIDefaultAliases(), []string{"claude-sonnet-4-5", "openai-gpt5.2"}) {
		t.Fatalf("ManagedUIDefaultAliases() = %v", config.ManagedUIDefaultAliases())
	}
	if config.ManagedUIDefaultModel() != "claude-sonnet-4-5" {
		t.Fatalf("ManagedUIDefaultModel() = %q", config.ManagedUIDefaultModel())
	}
	if !slices.Equal(ApprovedModelNames(config.OnlineLiteLLMConfig()), []string{"claude-sonnet-4-5", "openai-gpt5.2"}) {
		t.Fatalf("OnlineLiteLLMConfig() aliases mismatch")
	}
	issues := ValidateModelCatalog(config)
	if len(issues) == 0 {
		t.Fatalf("expected duplicate alias issue")
	}
	if !slices.Equal(config.OfflineAliases(), []string{"mock-claude", "mock-gpt"}) {
		t.Fatalf("OfflineAliases() = %v", config.OfflineAliases())
	}
	if !slices.Equal(ApprovedModelNames(config.OfflineLiteLLMConfig()), []string{"mock-claude", "mock-gpt"}) {
		t.Fatalf("OfflineLiteLLMConfig() aliases mismatch")
	}
}

func writeModelCatalogFile(t *testing.T, path string, content string) {
	t.Helper()
	writeModelCatalogDir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestModelCatalogValidationCoversMissingFields(t *testing.T) {
	config := ModelCatalog{
		OnlineModels: []CatalogModel{
			{Alias: "missing-credential", UpstreamModel: "openai/missing"},
		},
	}

	issues := ValidateModelCatalog(config)
	if len(issues) == 0 {
		t.Fatal("expected validation issues")
	}
	joined := strings.Join(issues, "\n")
	for _, want := range []string{
		"credential_env is required for online models",
		"offline_models must contain at least one model",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected issue %q, got %v", want, issues)
		}
	}

	noDefault := ModelCatalog{
		OnlineModels: []CatalogModel{
			{Alias: "openai-gpt5.2", UpstreamModel: "openai/gpt-5.2", CredentialEnv: "OPENAI_API_KEY"},
		},
		OfflineModels: []CatalogModel{
			{Alias: "mock-gpt", UpstreamModel: "openai/mock-gpt"},
		},
	}
	if got := noDefault.ManagedUIDefaultModel(); got != "" {
		t.Fatalf("ManagedUIDefaultModel() = %q, want empty", got)
	}
}

func writeModelCatalogDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
