// Package catalog loads tracked repository configuration data.
//
// Purpose:
//
//	Provide typed loaders for stable, committed repository configuration used
//	by completions, validations, operator workflows, and generated references.
//
// Responsibilities:
//   - Load LiteLLM model metadata from committed YAML files.
//   - Load the host-first model catalog used as the approved-model source.
//   - Load demo preset metadata and derive canonical scenario identifiers.
//   - Expose deterministic, sorted views over tracked configuration values.
//
// Scope:
//   - Read-only access to tracked repository configuration under demo/config
//     and docs/.
//
// Usage:
//   - Used by command completions, validation workflows, and doc generation.
//
// Invariants/Assumptions:
//   - Only tracked repo data is considered canonical; local/scratch paths are excluded.
//   - Returned name slices are unique and sorted.
package catalog

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// ModelCatalog is the tracked source of truth for approved online/offline
// model aliases and managed browser defaults.
type ModelCatalog struct {
	OnlineModels  []CatalogModel `yaml:"online_models"`
	OfflineModels []CatalogModel `yaml:"offline_models"`
}

// CatalogModel captures one approved model entry.
type CatalogModel struct {
	Alias            string `yaml:"alias"`
	UpstreamModel    string `yaml:"upstream_model"`
	CredentialEnv    string `yaml:"credential_env"`
	ManagedUIDefault bool   `yaml:"managed_ui_default"`
}

// LoadModelCatalog loads the tracked model catalog.
func LoadModelCatalog(path string) (ModelCatalog, error) {
	var config ModelCatalog
	if err := loadYAMLFile(path, &config); err != nil {
		return ModelCatalog{}, err
	}
	return config, nil
}

// OnlineAliases returns unique sorted online aliases.
func (c ModelCatalog) OnlineAliases() []string {
	return catalogAliases(c.OnlineModels)
}

// OfflineAliases returns unique sorted offline aliases.
func (c ModelCatalog) OfflineAliases() []string {
	return catalogAliases(c.OfflineModels)
}

// ManagedUIDefaultAliases returns aliases exposed by the managed browser UI.
func (c ModelCatalog) ManagedUIDefaultAliases() []string {
	values := make([]string, 0, len(c.OnlineModels))
	for _, model := range c.OnlineModels {
		if model.ManagedUIDefault {
			values = append(values, strings.TrimSpace(model.Alias))
		}
	}
	return dedupeSorted(values)
}

// OnlineLiteLLMConfig returns an alias-only LiteLLM config view for
// validation workflows that need the approved online alias set.
func (c ModelCatalog) OnlineLiteLLMConfig() LiteLLMConfig {
	return liteLLMConfigFromAliases(c.OnlineAliases())
}

// OfflineLiteLLMConfig returns an alias-only LiteLLM config view for
// validation workflows that need the approved offline alias set.
func (c ModelCatalog) OfflineLiteLLMConfig() LiteLLMConfig {
	return liteLLMConfigFromAliases(c.OfflineAliases())
}

// ValidateModelCatalog returns structural issues for the tracked model catalog.
func ValidateModelCatalog(config ModelCatalog) []string {
	issues := make([]string, 0)
	seen := make(map[string]string)
	validateSet := func(section string, models []CatalogModel, requireCredential bool) {
		if len(models) == 0 {
			issues = append(issues, fmt.Sprintf("%s must contain at least one model", section))
			return
		}
		for index, model := range models {
			prefix := fmt.Sprintf("%s[%d]", section, index)
			alias := strings.TrimSpace(model.Alias)
			if alias == "" {
				issues = append(issues, prefix+": alias is required")
			}
			upstream := strings.TrimSpace(model.UpstreamModel)
			if upstream == "" {
				issues = append(issues, prefix+": upstream_model is required")
			}
			credentialEnv := strings.TrimSpace(model.CredentialEnv)
			if requireCredential && credentialEnv == "" {
				issues = append(issues, prefix+": credential_env is required for online models")
			}
			if alias == "" {
				continue
			}
			if owner, exists := seen[alias]; exists {
				issues = append(issues, prefix+": duplicate alias "+alias+" already defined in "+owner)
				continue
			}
			seen[alias] = prefix
		}
	}
	validateSet("online_models", config.OnlineModels, true)
	validateSet("offline_models", config.OfflineModels, false)
	return dedupeSorted(issues)
}

// ManagedUIDefaultModel returns the first configured managed browser default.
func (c ModelCatalog) ManagedUIDefaultModel() string {
	values := c.ManagedUIDefaultAliases()
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func catalogAliases(models []CatalogModel) []string {
	values := make([]string, 0, len(models))
	for _, model := range models {
		values = append(values, strings.TrimSpace(model.Alias))
	}
	return dedupeSorted(values)
}

func liteLLMConfigFromAliases(aliases []string) LiteLLMConfig {
	config := LiteLLMConfig{ModelList: make([]LiteLLMModel, 0, len(aliases))}
	for _, alias := range dedupeSorted(aliases) {
		config.ModelList = append(config.ModelList, LiteLLMModel{ModelName: alias})
	}
	return config
}

func dedupeSorted(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	sort.Strings(clean)
	return slices.Compact(clean)
}
