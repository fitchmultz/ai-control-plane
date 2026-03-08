// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Replace heuristic command-local validators with typed structural checks.
//
// Responsibilities:
//   - Validate canonical deployment/config surfaces structurally.
//   - Enforce compose healthcheck requirements with parsed YAML.
//   - Enforce repository policy checks such as header shape and env ownership.
//
// Scope:
//   - Read-only validation of committed repository content and source policy.
//
// Usage:
//   - Called by `acpctl validate` adapters and CI make targets.
//
// Invariants/Assumptions:
//   - Validators share `internal/policy` as the single source of scan scope.
//   - Findings are deterministic and safe to run repeatedly.
package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"gopkg.in/yaml.v3"
)

func ValidateDeploymentSurfaces(repoRoot string) ([]string, error) {
	targets, err := policy.ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		return nil, err
	}
	issues := make([]string, 0)
	for _, target := range targets {
		if !hasRule(target.Rules, policy.RuleStructure) {
			continue
		}
		targetIssues, err := validateStructureForTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		issues = append(issues, targetIssues...)
		if hasRule(target.Rules, policy.RuleHelmContracts) {
			issues = append(issues, validateHelmContracts(repoRoot, target)...)
		}
	}
	sort.Strings(issues)
	return issues, nil
}

func ValidateComposeHealthchecks(repoRoot string) ([]string, error) {
	targets, err := policy.ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		return nil, err
	}
	issues := make([]string, 0)
	for _, target := range targets {
		if target.Kind != policy.SurfaceCompose || !hasRule(target.Rules, policy.RuleHealthchecks) {
			continue
		}
		targetIssues, err := validateComposeHealthchecksForTarget(filepath.Join(repoRoot, filepath.FromSlash(target.Path)), target.Path)
		if err != nil {
			return nil, err
		}
		issues = append(issues, targetIssues...)
	}
	sort.Strings(issues)
	return issues, nil
}

func validateStructureForTarget(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	absPath := filepath.Join(repoRoot, filepath.FromSlash(target.Path))
	if _, err := os.Stat(absPath); err != nil {
		return []string{fmt.Sprintf("%s: missing required file", target.Path)}, nil
	}
	switch target.Kind {
	case policy.SurfaceCompose, policy.SurfaceHelmValues, policy.SurfaceAnsibleYML, policy.SurfaceYAML:
		if _, err := loadYAMLRoot(absPath); err != nil {
			return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}, nil
		}
	case policy.SurfaceJSON:
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return []string{fmt.Sprintf("%s: invalid JSON: %v", target.Path, err)}, nil
		}
	case policy.SurfaceTerraform:
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(data)) == "" {
			return []string{fmt.Sprintf("%s: empty Terraform source", target.Path)}, nil
		}
	case policy.SurfaceDockerfile:
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(strings.ToUpper(string(data)), "FROM ") {
			return []string{fmt.Sprintf("%s: Dockerfile must declare at least one FROM instruction", target.Path)}, nil
		}
	}
	return nil, nil
}

func validateComposeHealthchecksForTarget(path string, relPath string) ([]string, error) {
	root, err := loadYAMLRoot(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML: %v", relPath, err)}, nil
	}
	servicesNode := mappingValue(root, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return []string{fmt.Sprintf("%s: compose file must define services", relPath)}, nil
	}
	issues := make([]string, 0)
	for i := 0; i < len(servicesNode.Content); i += 2 {
		serviceName := servicesNode.Content[i].Value
		serviceNode := servicesNode.Content[i+1]
		if mappingValue(serviceNode, "image") == nil && mappingValue(serviceNode, "build") == nil {
			continue
		}
		healthcheck := mappingValue(serviceNode, "healthcheck")
		if healthcheck == nil || healthcheck.Kind != yaml.MappingNode {
			issues = append(issues, fmt.Sprintf("%s: service %q must define a healthcheck mapping", relPath, serviceName))
			continue
		}
		testNode := mappingValue(healthcheck, "test")
		if testNode == nil {
			issues = append(issues, fmt.Sprintf("%s: service %q healthcheck must define test", relPath, serviceName))
			continue
		}
		switch testNode.Kind {
		case yaml.SequenceNode:
			if len(testNode.Content) == 0 {
				issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must not be empty", relPath, serviceName))
			}
		case yaml.ScalarNode:
			if strings.TrimSpace(testNode.Value) == "" {
				issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must not be empty", relPath, serviceName))
			}
		default:
			issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must be a string or sequence", relPath, serviceName))
		}
	}
	return issues, nil
}

func validateHelmContracts(repoRoot string, target policy.SurfaceTarget) []string {
	root, err := loadYAMLRoot(filepath.Join(repoRoot, filepath.FromSlash(target.Path)))
	if err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}
	}
	profile := scalarValue(mappingValue(root, "profile"))
	demoEnabled := scalarValue(mappingValue(mappingValue(root, "demo"), "enabled"))
	issues := make([]string, 0)
	switch target.Path {
	case "deploy/helm/ai-control-plane/values.yaml":
		if profile != "production" {
			issues = append(issues, fmt.Sprintf("%s: profile must be production", target.Path))
		}
		if demoEnabled != "false" {
			issues = append(issues, fmt.Sprintf("%s: demo.enabled must be false", target.Path))
		}
	default:
		if strings.Contains(target.Path, "values.demo.yaml") || strings.Contains(target.Path, "values.offline.yaml") {
			if profile != "demo" {
				issues = append(issues, fmt.Sprintf("%s: profile must be demo", target.Path))
			}
			if demoEnabled != "true" {
				issues = append(issues, fmt.Sprintf("%s: demo.enabled must be true", target.Path))
			}
		}
	}
	return issues
}

func loadYAMLRoot(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}
	return root.Content[0], nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func hasRule(rules []policy.SurfaceRule, target policy.SurfaceRule) bool {
	for _, rule := range rules {
		if rule == target {
			return true
		}
	}
	return false
}
