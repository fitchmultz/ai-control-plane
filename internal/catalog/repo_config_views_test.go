// repo_config_views_test.go - Derived-view coverage for tracked repository config.
//
// Purpose:
//   - Verify deterministic sorting and deduplication for tracked repo config views.
//
// Responsibilities:
//   - Cover approved model names, preset names, and scenario IDs.
//
// Scope:
//   - Pure view/helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/catalog`.
//
// Invariants/Assumptions:
//   - Returned slices are unique and sorted.
package catalog

import (
	"slices"
	"testing"
)

func TestApprovedModelNames(t *testing.T) {
	names := ApprovedModelNames(LiteLLMConfig{
		ModelList: []LiteLLMModel{
			{ModelName: " z-model "},
			{ModelName: "a-model"},
			{ModelName: "a-model"},
			{ModelName: ""},
		},
	})
	if !slices.Equal(names, []string{"a-model", "z-model"}) {
		t.Fatalf("ApprovedModelNames() = %v", names)
	}
}

func TestPresetNames(t *testing.T) {
	names := PresetNames(DemoPresetsFile{
		Presets: map[string]DemoPreset{
			" executive-demo ": {},
			"beta":             {},
			"":                 {},
		},
	})
	if !slices.Equal(names, []string{"beta", "executive-demo"}) {
		t.Fatalf("PresetNames() = %v", names)
	}
}

func TestScenarioIDs(t *testing.T) {
	ids := ScenarioIDs(DemoPresetsFile{
		Presets: map[string]DemoPreset{
			"a": {Scenarios: []int{5, 1}},
			"b": {Scenarios: []int{1, 9}},
		},
	})
	if !slices.Equal(ids, []string{"1", "5", "9"}) {
		t.Fatalf("ScenarioIDs() = %v", ids)
	}
}
