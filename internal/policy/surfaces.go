// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Own the single source of truth for ACP deployment/config scan targets.
//
// Responsibilities:
//   - Enumerate supported deployment surfaces and applicable validators.
//   - Expand path globs into deterministic file inventories.
//   - Provide one shared policy used by security and validation engines.
//
// Scope:
//   - Repository-relative surface definitions only.
//
// Usage:
//   - Call `ExpandDeploymentSurfaces(repoRoot)` from validators and gates.
//
// Invariants/Assumptions:
//   - All deployment/config enforcement derives from this policy.
//   - Surface ordering is stable for deterministic reporting.
package policy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SurfaceKind string

const (
	SurfaceCompose    SurfaceKind = "compose"
	SurfaceHelmValues SurfaceKind = "helm-values"
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
	Path  string
	Glob  string
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
	{ID: "compose-main", Kind: SurfaceCompose, Path: "demo/docker-compose.yml", Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning}},
	{ID: "compose-offline", Kind: SurfaceCompose, Path: "demo/docker-compose.offline.yml", Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning}},
	{ID: "compose-tls", Kind: SurfaceCompose, Path: "demo/docker-compose.tls.yml", Rules: []SurfaceRule{RuleStructure, RuleHealthchecks, RuleImagePinning}},
	{ID: "demo-config-yaml", Kind: SurfaceYAML, Glob: "demo/config/**/*.yaml", Rules: []SurfaceRule{RuleStructure}},
	{ID: "demo-config-json", Kind: SurfaceJSON, Glob: "demo/config/**/*.json", Rules: []SurfaceRule{RuleStructure}},
	{ID: "helm-values", Kind: SurfaceHelmValues, Path: "deploy/helm/ai-control-plane/values.yaml", Rules: []SurfaceRule{RuleStructure, RuleImagePinning, RuleHelmContracts}},
	{ID: "helm-value-examples", Kind: SurfaceHelmValues, Glob: "deploy/helm/ai-control-plane/examples/*.yaml", Rules: []SurfaceRule{RuleStructure, RuleHelmContracts}},
	{ID: "ansible-surface", Kind: SurfaceAnsibleYML, Glob: "deploy/ansible/**/*.yml", Rules: []SurfaceRule{RuleStructure}},
	{ID: "ansible-surface-yaml", Kind: SurfaceAnsibleYML, Glob: "deploy/ansible/**/*.yaml", Rules: []SurfaceRule{RuleStructure}},
	{ID: "terraform-surface", Kind: SurfaceTerraform, Glob: "deploy/terraform/**/*.tf", Rules: []SurfaceRule{RuleStructure}},
	{ID: "dockerfiles", Kind: SurfaceDockerfile, Glob: "demo/**/Dockerfile*", Rules: []SurfaceRule{RuleStructure, RuleImagePinning}},
}

// ExpandDeploymentSurfaces resolves the canonical policy to concrete files.
func ExpandDeploymentSurfaces(repoRoot string) ([]SurfaceTarget, error) {
	targets := make([]SurfaceTarget, 0)
	for _, spec := range DeploymentSurfacePolicy {
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
	if spec.Path != "" {
		return []string{spec.Path}, nil
	}
	pattern := filepath.Join(repoRoot, filepath.FromSlash(spec.Glob))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		relPath, err := filepath.Rel(repoRoot, match)
		if err != nil {
			return nil, err
		}
		paths = append(paths, filepath.ToSlash(relPath))
	}
	sort.Strings(paths)
	return uniqStrings(paths), nil
}

func uniqStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, strings.TrimSpace(value))
	}
	return out
}
