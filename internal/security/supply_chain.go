// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Enforce supply-chain policy across canonical deployment surfaces.
//
// Responsibilities:
//   - Validate the supply-chain policy document contract.
//   - Enforce digest pinning for compose, Helm values, and Dockerfiles.
//   - Consume the shared canonical surface inventory from `internal/policy`.
//
// Scope:
//   - Read-only supply-chain and image provenance validation only.
//
// Usage:
//   - Called by `acpctl validate supply-chain`.
//
// Invariants/Assumptions:
//   - Every image-pinning check derives from canonical deployment surfaces.
//   - Findings are deterministic and sorted for stable CI output.
package security

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/policy"
	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
	"gopkg.in/yaml.v3"
)

func ValidateSupplyChainPolicy(repoRoot string) ([]string, error) {
	data, err := os.ReadFile(repopath.DemoConfigPath(repoRoot, "supply_chain_vulnerability_policy.json"))
	if err != nil {
		return nil, err
	}
	var policyDoc SupplyChainPolicy
	if err := json.Unmarshal(data, &policyDoc); err != nil {
		return nil, err
	}
	issues := validationissues.NewIssues()
	if policyDoc.PolicyID == "" || len(policyDoc.SeverityPolicy.FailOn) == 0 {
		issues.Add("demo/config/supply_chain_vulnerability_policy.json: missing required policy fields")
	}
	targets, err := policy.ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if !policy.HasRule(target.Rules, policy.RuleImagePinning) {
			continue
		}
		targetIssues, err := validatePinnedImagesForTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		issues.Extend(targetIssues)
	}
	return issues.Sorted(), nil
}

func validatePinnedImagesForTarget(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	exists, err := policy.TargetExists(repoRoot, target)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	switch target.Kind {
	case policy.SurfaceCompose:
		return validateComposeImages(repoRoot, target)
	case policy.SurfaceHelmValues:
		return validateHelmImages(repoRoot, target)
	case policy.SurfaceDockerfile:
		return validateDockerfileBaseImages(repoRoot, target)
	default:
		return nil, nil
	}
}

func validateComposeImages(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	root, err := policy.LoadYAMLTarget(repoRoot, target)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", target.Path, err)}, nil
	}
	servicesNode := policy.MappingValue(root, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return []string{fmt.Sprintf("%s: missing services mapping", target.Path)}, nil
	}
	issues := make([]string, 0)
	for i := 0; i < len(servicesNode.Content); i += 2 {
		serviceName := servicesNode.Content[i].Value
		serviceNode := servicesNode.Content[i+1]
		imageNode := policy.MappingValue(serviceNode, "image")
		if imageNode == nil {
			continue
		}
		image := strings.TrimSpace(imageNode.Value)
		if isDigestPinnedImage(image) {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s: service %q image must be digest pinned (got %q)", target.Path, serviceName, image))
	}
	return issues, nil
}

func validateHelmImages(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	root, err := policy.LoadYAMLTarget(repoRoot, target)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", target.Path, err)}, nil
	}
	issues := make([]string, 0)
	policy.VisitMappings(root, "", func(node *yaml.Node, currentPath string) {
		repository := policy.MappingValue(node, "repository")
		if repository == nil || strings.TrimSpace(repository.Value) == "" {
			return
		}
		if isLocalImageOverride(node, strings.TrimSpace(repository.Value)) {
			return
		}
		digest := policy.MappingValue(node, "digest")
		if digest == nil || strings.TrimSpace(digest.Value) == "" {
			issues = append(issues, fmt.Sprintf("%s: %s must declare a non-empty image digest for repository %q", target.Path, currentPath, strings.TrimSpace(repository.Value)))
		}
	})
	return issues, nil
}

func isLocalImageOverride(node *yaml.Node, repository string) bool {
	tag := policy.MappingValue(node, "tag")
	if tag == nil {
		return false
	}
	return strings.HasPrefix(repository, "ai-control-plane/") && strings.TrimSpace(tag.Value) == "local"
}

func validateDockerfileBaseImages(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	data, err := policy.ReadTarget(repoRoot, target)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	issues := make([]string, 0)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 || isDigestPinnedImage(fields[1]) {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s:%d: base image must be digest pinned (got %q)", target.Path, lineNumber, fields[1]))
	}
	return issues, scanner.Err()
}

func isDigestPinnedImage(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "@sha256:") {
		return true
	}
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "${"), "}")
		parts := strings.SplitN(inner, ":-", 2)
		if len(parts) == 2 {
			return strings.Contains(parts[1], "@sha256:")
		}
	}
	return false
}
