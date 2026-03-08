// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Verify shared inspection helpers cover nested repository fixtures safely.
//
// Responsibilities:
//   - Prove recursive include/exclude traversal behavior.
//   - Cover YAML loading and nested mapping traversal helpers.
//
// Scope:
//   - Unit tests for `internal/policy` helper functions only.
//
// Usage:
//   - Run with `go test ./internal/policy`.
//
// Invariants/Assumptions:
//   - Helpers return repository-relative, deterministic results.
package policy

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWalkScopeFilesHonorsRecursiveIncludeAndExcludeForNestedFixtures(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "nested", "keep.yaml"), "value: keep\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "nested", "skip.json"), "{}\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "nested", "excluded", "drop.yaml"), "value: drop\n")

	got, err := WalkScopeFiles(repoRoot, PathScope{
		Include: []string{"demo/config/**/*"},
		Exclude: []string{"demo/config/**/excluded/**", "demo/config/**/*.json"},
	})
	if err != nil {
		t.Fatalf("WalkScopeFiles() error = %v", err)
	}
	want := []string{"demo/config/nested/keep.yaml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WalkScopeFiles() mismatch\nwant: %v\n got: %v", want, got)
	}
}

func TestWalkScopeFilesSkipsGitVendorAndNodeModules(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, ".git", "config"), "ignored\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "vendor", "module", "values.yaml"), "ignored: true\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "node_modules", "pkg", "config.yaml"), "ignored: true\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "docs", "keep.yaml"), "kept: true\n")

	got, err := WalkScopeFiles(repoRoot, PathScope{Include: []string{"**/*.yaml"}})
	if err != nil {
		t.Fatalf("WalkScopeFiles() error = %v", err)
	}
	want := []string{"docs/keep.yaml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WalkScopeFiles() mismatch\nwant: %v\n got: %v", want, got)
	}
}

func TestLoadYAMLFileRejectsEmptyDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty yaml: %v", err)
	}
	if _, err := LoadYAMLFile(path); err == nil {
		t.Fatal("expected empty YAML document to fail")
	}
}

func TestVisitMappingsReportsNestedDotPaths(t *testing.T) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte("top:\n  child:\n    leaf: value\n"), &root); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	var paths []string
	VisitMappings(root.Content[0], "", func(_ *yaml.Node, currentPath string) {
		paths = append(paths, currentPath)
	})
	want := []string{"", "top", "top.child"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("VisitMappings() mismatch\nwant: %v\n got: %v", want, paths)
	}
}
