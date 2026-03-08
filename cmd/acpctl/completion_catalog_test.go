// completion_catalog_test.go - Extractor coverage for config-driven completion values.
//
// Purpose:
//   - Verify tracked-config extractors stay deterministic and resilient to missing files.
//
// Responsibilities:
//   - Cover scenario/model/preset/config-key extraction helpers.
//   - Cover missing-file fallbacks and duplicate elimination.
//
// Scope:
//   - Config-backed completion extractors only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Extractors return nil on malformed or missing tracked config files.
package main

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestExtractScenarioIDs(t *testing.T) {
	repoRoot := t.TempDir()
	writeCompletionConfigFile(t, repoRoot, filepath.Join("demo", "config", "demo_presets.yaml"), `presets:
  executive-demo:
    scenarios:
      - 1
      - 5
  duplicate-demo:
    scenarios:
      - 5
`)

	values := extractScenarioIDs(repoRoot)
	if !slices.Equal(values, []string{"1", "5"}) {
		t.Fatalf("unexpected scenario ids: %v", values)
	}
}

func TestExtractModelNamesAndPresetNames(t *testing.T) {
	repoRoot := t.TempDir()
	writeCompletionConfigFile(t, repoRoot, filepath.Join("demo", "config", "litellm.yaml"), `model_list:
  - model_name: test-model-1
  - model_name: test-model-2
  - model_name: test-model-1
`)
	writeCompletionConfigFile(t, repoRoot, filepath.Join("demo", "config", "demo_presets.yaml"), `presets:
  executive-demo: {}
  beta-demo: {}
`)

	models := extractModelNames(repoRoot)
	if !slices.Equal(models, []string{"test-model-1", "test-model-2"}) {
		t.Fatalf("unexpected models: %v", models)
	}
	presets := extractPresetNames(repoRoot)
	if !slices.Equal(presets, []string{"beta-demo", "executive-demo"}) {
		t.Fatalf("unexpected presets: %v", presets)
	}
}

func TestExtractConfigKeysAndMissingFallbacks(t *testing.T) {
	repoRoot := t.TempDir()
	writeCompletionConfigFile(t, repoRoot, filepath.Join("demo", "config", "test_config.yaml"), `
# comment
alpha:
beta:
alpha:
 nested:
`)

	keys := extractConfigKeys(repoRoot)
	if !slices.Equal(keys, []string{"alpha", "beta", "nested"}) {
		t.Fatalf("unexpected config keys: %v", keys)
	}

	if extractModelNames(t.TempDir()) != nil {
		t.Fatal("expected missing litellm config to return nil")
	}
	if extractPresetNames(t.TempDir()) != nil {
		t.Fatal("expected missing presets config to return nil")
	}
	if extractScenarioIDs(t.TempDir()) != nil {
		t.Fatal("expected missing scenario config to return nil")
	}
}
