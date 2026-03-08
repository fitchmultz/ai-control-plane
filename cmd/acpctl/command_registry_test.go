// command_registry_test.go - Tests for the typed acpctl command-spec tree.
//
// Purpose:
//
//	Verify the compiled command tree remains the single source of truth for
//	root commands, grouped subcommands, and backend ownership.
//
// Responsibilities:
//   - Ensure expected visible roots exist.
//   - Ensure hidden commands stay out of the visible registry.
//   - Ensure every leaf command resolves to exactly one backend.
//
// Scope:
//   - Command-spec structure only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Visible help/completion output is derived from the typed command tree.
package main

import (
	"strings"
	"testing"
)

func TestCommandSpec_ContainsExpectedVisibleRoots(t *testing.T) {
	registry := buildCommandRegistry()
	expected := []string{"ci", "completion", "deploy", "validate", "bridge", "onboard", "help"}
	for _, name := range expected {
		found := false
		for _, command := range registry.RootCommands {
			if command.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected visible root command %q in registry", name)
		}
	}
}

func TestCommandSpec_HiddenCommandsStayOutOfVisibleRegistry(t *testing.T) {
	registry := buildCommandRegistry()
	for _, command := range registry.RootCommands {
		if command.Name == "__complete" {
			t.Fatal("hidden command __complete leaked into visible registry")
		}
	}
}

func TestCommandSpec_AllLeavesHaveBackends(t *testing.T) {
	spec, err := loadCommandSpec()
	if err != nil {
		t.Fatalf("loadCommandSpec() error = %v", err)
	}
	var walk func(node *commandSpec, path []string)
	walk = func(node *commandSpec, path []string) {
		path = append(path, node.Name)
		if len(node.Children) == 0 {
			if node.Backend.Kind == "" {
				t.Fatalf("leaf %q is missing a backend", strings.Join(path[1:], " "))
			}
			return
		}
		for _, child := range node.Children {
			walk(child, path)
		}
	}
	for _, child := range spec.Root.Children {
		walk(child, []string{spec.Root.Name})
	}
}
