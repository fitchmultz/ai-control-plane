// Package catalog loads tracked repository configuration data.
//
// Purpose:
//
//	Provide typed loaders for stable, committed repository configuration used
//	by completions, validations, and operator workflows.
//
// Responsibilities:
//   - Load LiteLLM model metadata from committed YAML files.
//   - Load demo preset metadata and derive canonical scenario identifiers.
//   - Expose deterministic, sorted views over tracked configuration values.
//
// Scope:
//   - Read-only access to tracked repository configuration under demo/config.
//
// Usage:
//   - Used by command completions and validation workflows.
//
// Invariants/Assumptions:
//   - Only tracked repo data is considered canonical; local/scratch paths are excluded.
//   - Returned name slices are unique and sorted.
package catalog

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type LiteLLMConfig struct {
	ModelList []LiteLLMModel `yaml:"model_list"`
}

type LiteLLMModel struct {
	ModelName string `yaml:"model_name"`
}

type DemoPresetsFile struct {
	Presets map[string]DemoPreset `yaml:"presets"`
}

type DemoPreset struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Scenarios   []int  `yaml:"scenarios"`
}

func LoadLiteLLMConfig(path string) (LiteLLMConfig, error) {
	var config LiteLLMConfig
	if err := loadYAMLFile(path, &config); err != nil {
		return LiteLLMConfig{}, err
	}
	return config, nil
}

func LoadDemoPresets(path string) (DemoPresetsFile, error) {
	var config DemoPresetsFile
	if err := loadYAMLFile(path, &config); err != nil {
		return DemoPresetsFile{}, err
	}
	return config, nil
}

func ApprovedModelNames(config LiteLLMConfig) []string {
	models := make([]string, 0, len(config.ModelList))
	seen := make(map[string]struct{}, len(config.ModelList))
	for _, model := range config.ModelList {
		name := strings.TrimSpace(model.ModelName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	sort.Strings(models)
	return models
}

func PresetNames(config DemoPresetsFile) []string {
	names := make([]string, 0, len(config.Presets))
	for name := range config.Presets {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	sort.Strings(names)
	return names
}

func ScenarioIDs(config DemoPresetsFile) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, preset := range config.Presets {
		for _, scenarioID := range preset.Scenarios {
			id := fmt.Sprintf("%d", scenarioID)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func loadYAMLFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found", path)
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
