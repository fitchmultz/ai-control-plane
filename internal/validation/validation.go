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
		if !policy.HasRule(target.Rules, policy.RuleStructure) {
			continue
		}
		targetIssues, err := validateStructureForTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		issues = append(issues, targetIssues...)
		if policy.HasRule(target.Rules, policy.RuleHelmContracts) {
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
		if target.Kind != policy.SurfaceCompose || !policy.HasRule(target.Rules, policy.RuleHealthchecks) {
			continue
		}
		targetIssues, err := validateComposeHealthchecksForTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		issues = append(issues, targetIssues...)
	}
	sort.Strings(issues)
	return issues, nil
}

func validateStructureForTarget(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	exists, err := policy.TargetExists(repoRoot, target)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{fmt.Sprintf("%s: missing required file", target.Path)}, nil
	}
	switch target.Kind {
	case policy.SurfaceCompose, policy.SurfaceHelmChart, policy.SurfaceHelmValues, policy.SurfaceAnsibleYML, policy.SurfaceYAML:
		if _, err := policy.LoadYAMLTarget(repoRoot, target); err != nil {
			return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}, nil
		}
	case policy.SurfaceJSON, policy.SurfaceHelmSchema:
		data, err := policy.ReadTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return []string{fmt.Sprintf("%s: invalid JSON: %v", target.Path, err)}, nil
		}
	case policy.SurfaceHelmTpl:
		return validateHelmTemplateStructure(repoRoot, target)
	case policy.SurfaceTerraform:
		data, err := policy.ReadTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(data)) == "" {
			return []string{fmt.Sprintf("%s: empty Terraform source", target.Path)}, nil
		}
	case policy.SurfaceDockerfile:
		data, err := policy.ReadTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(strings.ToUpper(string(data)), "FROM ") {
			return []string{fmt.Sprintf("%s: Dockerfile must declare at least one FROM instruction", target.Path)}, nil
		}
	}
	return nil, nil
}

func validateHelmTemplateStructure(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	data, err := policy.ReadTarget(repoRoot, target)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return []string{fmt.Sprintf("%s: Helm template must not be empty", target.Path)}, nil
	}
	if !strings.Contains(trimmed, "apiVersion:") || !strings.Contains(trimmed, "kind:") {
		if isTemplateOnlyHelmFile(trimmed) {
			return nil, nil
		}
		return []string{fmt.Sprintf("%s: Helm template must declare apiVersion and kind", target.Path)}, nil
	}
	return nil, nil
}

func isTemplateOnlyHelmFile(content string) bool {
	lines := strings.Split(content, "\n")
	inTemplateComment := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if inTemplateComment {
			if strings.Contains(trimmed, "*/}}") {
				inTemplateComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "{{/*") {
			if !strings.Contains(trimmed, "*/}}") {
				inTemplateComment = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "{{") || strings.HasPrefix(trimmed, "}}") {
			continue
		}
		return false
	}
	return true
}

func validateComposeHealthchecksForTarget(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	root, err := policy.LoadYAMLTarget(repoRoot, target)
	if err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}, nil
	}
	servicesNode := policy.MappingValue(root, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return []string{fmt.Sprintf("%s: compose file must define services", target.Path)}, nil
	}
	issues := make([]string, 0)
	for i := 0; i < len(servicesNode.Content); i += 2 {
		serviceName := servicesNode.Content[i].Value
		serviceNode := servicesNode.Content[i+1]
		if policy.MappingValue(serviceNode, "image") == nil && policy.MappingValue(serviceNode, "build") == nil {
			continue
		}
		healthcheck := policy.MappingValue(serviceNode, "healthcheck")
		if healthcheck == nil || healthcheck.Kind != yaml.MappingNode {
			issues = append(issues, fmt.Sprintf("%s: service %q must define a healthcheck mapping", target.Path, serviceName))
			continue
		}
		testNode := policy.MappingValue(healthcheck, "test")
		if testNode == nil {
			issues = append(issues, fmt.Sprintf("%s: service %q healthcheck must define test", target.Path, serviceName))
			continue
		}
		switch testNode.Kind {
		case yaml.SequenceNode:
			if len(testNode.Content) == 0 {
				issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must not be empty", target.Path, serviceName))
			}
		case yaml.ScalarNode:
			if strings.TrimSpace(testNode.Value) == "" {
				issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must not be empty", target.Path, serviceName))
			}
		default:
			issues = append(issues, fmt.Sprintf("%s: service %q healthcheck test must be a string or sequence", target.Path, serviceName))
		}
	}
	return issues, nil
}

func validateHelmContracts(repoRoot string, target policy.SurfaceTarget) []string {
	root, err := policy.LoadYAMLTarget(repoRoot, target)
	if err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML: %v", target.Path, err)}
	}
	profile := policy.ScalarValue(policy.MappingValue(root, "profile"))
	demoEnabled := policy.ScalarValue(policy.MappingValue(policy.MappingValue(root, "demo"), "enabled"))
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
