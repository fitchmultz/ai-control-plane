// repo_config_load_test.go - Loader coverage for tracked repository config.
//
// Purpose:
//   - Verify tracked YAML config loaders report missing and parse failures clearly.
//
// Responsibilities:
//   - Cover LiteLLM and demo-preset loader success/error paths.
//
// Scope:
//   - YAML loading behavior only.
//
// Usage:
//   - Run via `go test ./internal/catalog`.
//
// Invariants/Assumptions:
//   - Tests operate on temp fixture files only.
package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLiteLLMConfig(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadLiteLLMConfig(missing); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}

	path := filepath.Join(t.TempDir(), "litellm.yaml")
	writeCatalogFile(t, path, "model_list:\n  - model_name: gpt-5.2\n")
	config, err := LoadLiteLLMConfig(path)
	if err != nil {
		t.Fatalf("LoadLiteLLMConfig() error = %v", err)
	}
	if len(config.ModelList) != 1 || config.ModelList[0].ModelName != "gpt-5.2" {
		t.Fatalf("unexpected config: %+v", config)
	}
}

func TestLoadDemoPresets(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadDemoPresets(missing); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}

	path := filepath.Join(t.TempDir(), "presets.yaml")
	writeCatalogFile(t, path, "presets: [")
	if _, err := LoadDemoPresets(path); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func writeCatalogFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
