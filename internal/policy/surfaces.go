// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Own the single source of truth for ACP deployment/config scan targets.
//
// Responsibilities:
//   - Enumerate supported deployment surfaces and applicable validators.
//   - Resolve exact files and recursive path scopes into deterministic targets.
//   - Provide shared rule metadata consumed by security and validation engines.
//
// Scope:
//   - Repository-relative deployment/config policy only.
//
// Usage:
//   - Call `ExpandDeploymentSurfaces(repoRoot)` from validators and gates.
//
// Invariants/Assumptions:
//   - All deployment/config enforcement derives from this policy.
//   - Surface ordering is stable for deterministic reporting.
package policy

import (
	"path/filepath"
	"sort"
)

type SurfaceKind string

const (
	SurfaceCompose    SurfaceKind = "compose"
	SurfaceHelmValues SurfaceKind = "helm-values"
	SurfaceHelmChart  SurfaceKind = "helm-chart"
	SurfaceHelmSchema SurfaceKind = "helm-schema"
	SurfaceHelmTpl    SurfaceKind = "helm-template"
	SurfaceAnsibleYML SurfaceKind = "ansible-yaml"
	SurfaceTerraform  SurfaceKind = "terraform"
	SurfaceDockerfile SurfaceKind = "dockerfile"
	SurfaceJSON       SurfaceKind = "json"
	SurfaceYAML       SurfaceKind = "yaml"
)

type SurfaceRule string

const (
	RuleStructure     SurfaceRule = "structure"
	RuleHealthchecks  SurfaceRule = "healthchecks"
	RuleImagePinning  SurfaceRule = "image-pinning"
	RuleHelmContracts SurfaceRule = "helm-contracts"
)

// SurfaceSpec defines one canonical deployment or config surface.
type SurfaceSpec struct {
	ID    string
	Kind  SurfaceKind
	Paths []string
	Globs []string
	Rules []SurfaceRule
}

// SurfaceTarget is one concrete resolved file target.
type SurfaceTarget struct {
	ID    string
	Kind  SurfaceKind
	Path  string
	Rules []SurfaceRule
}

// DeploymentSurfacePolicy is the canonical scan/config enforcement scope.
var DeploymentSurfacePolicy = []SurfaceSpec{
	{
		ID:    "compose-main",
		Kind:  SurfaceCompose,
		Paths: []string{"demo/docker-compose.yml"},
		Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning},
	},
	{
		ID:    "compose-offline",
		Kind:  SurfaceCompose,
		Paths: []string{"demo/docker-compose.offline.yml"},
		Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning},
	},
	{
		ID:    "compose-tls",
		Kind:  SurfaceCompose,
		Paths: []string{"demo/docker-compose.tls.yml"},
		Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning},
	},
	{
		ID:    "demo-config-yaml",
		Kind:  SurfaceYAML,
		Globs: []string{"demo/config/**/*.yaml", "demo/config/**/*.yml"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "demo-config-json",
		Kind:  SurfaceJSON,
		Globs: []string{"demo/config/**/*.json"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "ansible-surface",
		Kind:  SurfaceAnsibleYML,
		Globs: []string{"deploy/ansible/**/*.yml", "deploy/ansible/**/*.yaml"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "dockerfiles",
		Kind:  SurfaceDockerfile,
		Globs: []string{"demo/**/Dockerfile*"},
		Rules: []SurfaceRule{RuleStructure, RuleImagePinning},
	},
}

// IncubatingDeploymentSurfacePolicy captures non-supported tracked deployment
// assets retained for explicit internal checks only.
var IncubatingDeploymentSurfacePolicy = []SurfaceSpec{
	{
		ID:    "helm-chart",
		Kind:  SurfaceHelmChart,
		Paths: []string{"deploy/incubating/helm/ai-control-plane/Chart.yaml"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "helm-schema",
		Kind:  SurfaceHelmSchema,
		Paths: []string{"deploy/incubating/helm/ai-control-plane/values.schema.json"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "helm-values",
		Kind:  SurfaceHelmValues,
		Paths: []string{"deploy/incubating/helm/ai-control-plane/values.yaml"},
		Rules: []SurfaceRule{RuleStructure, RuleImagePinning, RuleHelmContracts},
	},
	{
		ID:    "helm-value-examples",
		Kind:  SurfaceHelmValues,
		Globs: []string{"deploy/incubating/helm/ai-control-plane/examples/**/*.yaml", "deploy/incubating/helm/ai-control-plane/examples/**/*.yml"},
		Rules: []SurfaceRule{RuleStructure, RuleHelmContracts},
	},
	{
		ID:    "helm-templates",
		Kind:  SurfaceHelmTpl,
		Globs: []string{"deploy/incubating/helm/ai-control-plane/templates/**/*.yaml", "deploy/incubating/helm/ai-control-plane/templates/**/*.yml"},
		Rules: []SurfaceRule{RuleStructure},
	},
	{
		ID:    "terraform-surface",
		Kind:  SurfaceTerraform,
		Globs: []string{"deploy/incubating/terraform/**/*.tf"},
		Rules: []SurfaceRule{RuleStructure},
	},
}

// ExpandDeploymentSurfaces resolves the canonical policy to concrete files.
func ExpandDeploymentSurfaces(repoRoot string) ([]SurfaceTarget, error) {
	return expandPolicy(repoRoot, DeploymentSurfacePolicy)
}

// ExpandIncubatingDeploymentSurfaces resolves the incubating policy to concrete files.
func ExpandIncubatingDeploymentSurfaces(repoRoot string) ([]SurfaceTarget, error) {
	return expandPolicy(repoRoot, IncubatingDeploymentSurfacePolicy)
}

func expandPolicy(repoRoot string, specs []SurfaceSpec) ([]SurfaceTarget, error) {
	targets := make([]SurfaceTarget, 0)
	for _, spec := range specs {
		paths, err := expandSpec(repoRoot, spec)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			targets = append(targets, SurfaceTarget{
				ID:    spec.ID,
				Kind:  spec.Kind,
				Path:  path,
				Rules: append([]SurfaceRule(nil), spec.Rules...),
			})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	return targets, nil
}

func expandSpec(repoRoot string, spec SurfaceSpec) ([]string, error) {
	paths := make([]string, 0, len(spec.Paths))
	for _, path := range spec.Paths {
		if path == "" {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Clean(path)))
	}
	if len(spec.Globs) == 0 {
		sort.Strings(paths)
		return uniqStrings(paths), nil
	}
	matches, err := WalkScopeFiles(repoRoot, PathScope{Include: spec.Globs})
	if err != nil {
		return nil, err
	}
	paths = append(paths, matches...)
	sort.Strings(paths)
	return uniqStrings(paths), nil
}

func HasRule(rules []SurfaceRule, target SurfaceRule) bool {
	for _, rule := range rules {
		if rule == target {
			return true
		}
	}
	return false
}

func uniqStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(filepath.Clean(value))
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
