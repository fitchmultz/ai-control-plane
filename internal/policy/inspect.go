// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Provide shared surface-inspection helpers for validators.
//
// Responsibilities:
//   - Resolve target paths from repository-relative policy targets.
//   - Load deployment/config files as bytes or YAML documents.
//   - Provide shared YAML traversal helpers used across validators.
//
// Scope:
//   - Read-only inspection helpers for canonical repository surfaces.
//
// Usage:
//   - Used by `internal/security` and `internal/validation`.
//
// Invariants/Assumptions:
//   - YAML helpers operate on the document root node.
//   - Callers remain responsible for kind-specific validation semantics.
package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func TargetAbsPath(repoRoot string, target SurfaceTarget) string {
	return filepath.Join(repoRoot, filepath.FromSlash(target.Path))
}

func TargetExists(repoRoot string, target SurfaceTarget) (bool, error) {
	_, err := os.Stat(TargetAbsPath(repoRoot, target))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ReadTarget(repoRoot string, target SurfaceTarget) ([]byte, error) {
	return os.ReadFile(TargetAbsPath(repoRoot, target))
}

func LoadYAMLTarget(repoRoot string, target SurfaceTarget) (*yaml.Node, error) {
	return LoadYAMLFile(TargetAbsPath(repoRoot, target))
}

func LoadYAMLFile(path string) (*yaml.Node, error) {
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

func MappingValue(node *yaml.Node, key string) *yaml.Node {
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

func ScalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func VisitMappings(node *yaml.Node, currentPath string, fn func(node *yaml.Node, currentPath string)) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		fn(node, currentPath)
		for i := 0; i < len(node.Content); i += 2 {
			childKey := node.Content[i].Value
			childPath := childKey
			if currentPath != "" {
				childPath = currentPath + "." + childKey
			}
			VisitMappings(node.Content[i+1], childPath, fn)
		}
		return
	}
	for _, child := range node.Content {
		VisitMappings(child, currentPath, fn)
	}
}
