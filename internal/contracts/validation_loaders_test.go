// validation_loaders_test.go - Loader-focused coverage for contract validation.
//
// Purpose:
//   - Verify YAML contract loading and small utility helpers.
//
// Responsibilities:
//   - Cover not-found and parse failures for tracked YAML contracts.
//   - Cover approved-model and enabled-rule utility helpers.
//
// Scope:
//   - Loader and utility behavior only.
//
// Usage:
//   - Run via `go test ./internal/contracts`.
//
// Invariants/Assumptions:
//   - Fixtures are temporary and deterministic.
package contracts

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestLoadDetectionRulesFileErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadDetectionRulesFile(missing); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}

	badPath := filepath.Join(t.TempDir(), "bad.yaml")
	writeContractFile(t, badPath, "detection_rules: [")
	if _, err := LoadDetectionRulesFile(badPath); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestLoadSIEMQueriesFileErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadSIEMQueriesFile(missing); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}

	badPath := filepath.Join(t.TempDir(), "bad.yaml")
	writeContractFile(t, badPath, "siem_queries: [")
	if _, err := LoadSIEMQueriesFile(badPath); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestApprovedModelsAndEnabledRuleCounts(t *testing.T) {
	models := ApprovedModels(catalog.LiteLLMConfig{
		ModelList: []catalog.LiteLLMModel{
			{ModelName: " z-model "},
			{ModelName: "a-model"},
			{ModelName: "a-model"},
			{ModelName: ""},
		},
	})
	if !slices.Equal(models, []string{"a-model", "z-model"}) {
		t.Fatalf("ApprovedModels() = %v", models)
	}

	count := CountEnabledDetectionRules(DetectionRulesFile{
		DetectionRules: []DetectionRule{
			{Enabled: true},
			{Enabled: false},
			{Enabled: true},
		},
	})
	if count != 2 {
		t.Fatalf("CountEnabledDetectionRules() = %d, want 2", count)
	}
}
