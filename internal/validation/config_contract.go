// config_contract.go - Machine-readable config contract validation.
//
// Purpose:
//   - Validate supported tracked config files against the repository's
//     machine-readable config contract and cross-file invariants.
//
// Responsibilities:
//   - Load the tracked config contract manifest.
//   - Validate YAML files against JSON Schemas.
//   - Enforce cross-file invariants such as RBAC-to-model alias consistency
//     and allowed inventory overlay values.
//
// Scope:
//   - Tracked configuration validation only.
//
// Usage:
//   - Called by ValidateDeploymentConfig and repository contract tests.
//
// Invariants/Assumptions:
//   - docs/contracts/config/contract.yaml is the tracked config-contract source.
package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/rbac"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

const configContractManifestRelativePath = "docs/contracts/config/contract.yaml"

type configContractManifest struct {
	Version int `yaml:"version"`
	Schemas []struct {
		ID     string `yaml:"id"`
		Path   string `yaml:"path"`
		Schema string `yaml:"schema"`
	} `yaml:"schemas"`
	Naming struct {
		ModelAliasPattern string `yaml:"model_alias_pattern"`
		RoleNamePattern   string `yaml:"role_name_pattern"`
		PresetNamePattern string `yaml:"preset_name_pattern"`
	} `yaml:"naming"`
	Runtime struct {
		AllowedOverlays []string `yaml:"allowed_overlays"`
	} `yaml:"runtime"`
}

type litellmContract struct {
	ModelList []struct {
		ModelName string `yaml:"model_name"`
	} `yaml:"model_list"`
}

type presetContract struct {
	Presets map[string]struct{} `yaml:"presets"`
}

// ValidateConfigContract validates tracked config files against the manifest.
func ValidateConfigContract(repoRoot string) ([]string, error) {
	manifest, err := loadConfigContractManifest(repoRoot)
	if err != nil {
		return nil, err
	}

	issues := NewIssues()
	for _, schemaEntry := range manifest.Schemas {
		issues.Extend(validateSchemaEntry(repoRoot, schemaEntry.Path, schemaEntry.Schema))
	}

	modelPattern, err := regexp.Compile(manifest.Naming.ModelAliasPattern)
	if err != nil {
		return nil, fmt.Errorf("compile model alias pattern: %w", err)
	}
	rolePattern, err := regexp.Compile(manifest.Naming.RoleNamePattern)
	if err != nil {
		return nil, fmt.Errorf("compile role name pattern: %w", err)
	}
	presetPattern, err := regexp.Compile(manifest.Naming.PresetNamePattern)
	if err != nil {
		return nil, fmt.Errorf("compile preset name pattern: %w", err)
	}

	knownModels, modelIssues, err := loadLiteLLMAliases(repopath.DemoConfigPath(repoRoot, "litellm.yaml"), modelPattern)
	if err != nil {
		return nil, err
	}
	issues.Extend(modelIssues)

	rbacConfig, err := rbac.LoadFile(repopath.DemoConfigPath(repoRoot, "roles.yaml"))
	if err != nil {
		return nil, err
	}
	issues.Extend(rbacConfig.ValidateKnownModels(knownModels, rolePattern))

	issues.Extend(validatePresetNames(repopath.DemoConfigPath(repoRoot, "demo_presets.yaml"), presetPattern))
	issues.Extend(validateInventoryOverlayContract(repopath.FromRepoRoot(repoRoot, "deploy", "ansible", "inventory", "hosts.example.yml"), manifest.Runtime.AllowedOverlays))

	return issues.Sorted(), nil
}

func loadConfigContractManifest(repoRoot string) (configContractManifest, error) {
	path := repopath.FromRepoRoot(repoRoot, configContractManifestRelativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return configContractManifest{}, fmt.Errorf("read config contract manifest: %w", err)
	}
	var manifest configContractManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return configContractManifest{}, fmt.Errorf("parse config contract manifest: %w", err)
	}
	if manifest.Version <= 0 {
		return configContractManifest{}, fmt.Errorf("config contract manifest version must be positive")
	}
	if len(manifest.Schemas) == 0 {
		return configContractManifest{}, fmt.Errorf("config contract manifest must declare at least one schema")
	}
	return manifest, nil
}

func validateSchemaEntry(repoRoot string, documentRelPath string, schemaRelPath string) []string {
	documentPath := repopath.FromRepoRoot(repoRoot, documentRelPath)
	schemaPath := repopath.FromRepoRoot(repoRoot, schemaRelPath)

	jsonBytes, err := yamlFileToJSONBytes(documentPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", documentRelPath, err)}
	}

	schemaURL := url.URL{Scheme: "file", Path: filepath.ToSlash(schemaPath)}
	result, err := gojsonschema.Validate(
		gojsonschema.NewReferenceLoader(schemaURL.String()),
		gojsonschema.NewBytesLoader(jsonBytes),
	)
	if err != nil {
		return []string{fmt.Sprintf("%s: schema validation error: %v", documentRelPath, err)}
	}
	if result.Valid() {
		return nil
	}

	issues := NewIssues(len(result.Errors()))
	for _, issue := range result.Errors() {
		issues.Addf("%s: %s", documentRelPath, issue.String())
	}
	return issues.Sorted()
}

func yamlFileToJSONBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeYAML(document))
}

func normalizeYAML(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, inner := range typed {
			out[key] = normalizeYAML(inner)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, inner := range typed {
			out[fmt.Sprint(key)] = normalizeYAML(inner)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, inner := range typed {
			out[i] = normalizeYAML(inner)
		}
		return out
	default:
		return typed
	}
}

func loadLiteLLMAliases(path string, pattern *regexp.Regexp) (map[string]struct{}, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read LiteLLM config: %w", err)
	}
	var cfg litellmContract
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("parse LiteLLM config: %w", err)
	}

	known := make(map[string]struct{}, len(cfg.ModelList))
	issues := NewIssues()
	for _, model := range cfg.ModelList {
		name := strings.TrimSpace(model.ModelName)
		if name == "" {
			issues.Add("demo/config/litellm.yaml: model_list entries must define model_name")
			continue
		}
		if pattern != nil && !pattern.MatchString(name) {
			issues.Addf("demo/config/litellm.yaml: model_name %q does not match contract pattern", name)
		}
		if _, exists := known[name]; exists {
			issues.Addf("demo/config/litellm.yaml: duplicate model_name %q", name)
			continue
		}
		known[name] = struct{}{}
	}
	return known, issues.Sorted(), nil
}

func validatePresetNames(path string, pattern *regexp.Regexp) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("demo/config/demo_presets.yaml: %v", err)}
	}
	var cfg presetContract
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return []string{fmt.Sprintf("demo/config/demo_presets.yaml: %v", err)}
	}
	issues := NewIssues()
	for name := range cfg.Presets {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			issues.Add("demo/config/demo_presets.yaml: preset names must not be blank")
			continue
		}
		if pattern != nil && !pattern.MatchString(trimmed) {
			issues.Addf("demo/config/demo_presets.yaml: preset name %q does not match contract pattern", name)
		}
	}
	return issues.Sorted()
}

func validateInventoryOverlayContract(path string, allowed []string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return []string{fmt.Sprintf("deploy/ansible/inventory/hosts.example.yml: %v", err)}
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, overlay := range allowed {
		trimmed := strings.TrimSpace(overlay)
		if trimmed != "" {
			allowedSet[trimmed] = struct{}{}
		}
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return []string{fmt.Sprintf("deploy/ansible/inventory/hosts.example.yml: %v", err)}
	}

	issues := NewIssues()
	var walk func(node *yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil {
			return
		}
		if node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := node.Content[i]
				value := node.Content[i+1]
				if key.Value == "acp_runtime_overlays" {
					for _, overlay := range overlayValues(value) {
						if _, ok := allowedSet[overlay]; !ok {
							issues.Addf("deploy/ansible/inventory/hosts.example.yml: unsupported acp_runtime_overlays value %q", overlay)
						}
					}
				}
				walk(value)
			}
			return
		}
		for _, child := range node.Content {
			walk(child)
		}
	}
	walk(&root)
	return issues.Sorted()
}

func overlayValues(node *yaml.Node) []string {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return nil
		}
		return []string{value}
	case yaml.SequenceNode:
		values := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			trimmed := strings.TrimSpace(child.Value)
			if trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values
	default:
		return nil
	}
}
