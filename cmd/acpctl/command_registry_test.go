// command_registry_test.go - Tests for the canonical acpctl command catalog.
//
// Purpose:
//
//	Verify the unified command catalog remains the single source of truth for
//	root commands, grouped subcommands, and execution ownership.
//
// Responsibilities:
//   - Ensure expected root commands exist and hidden commands stay hidden.
//   - Ensure grouped subcommands have exactly one execution owner.
//   - Ensure help/completion registry output stays aligned with the catalog.
//
// Scope:
//   - Catalog metadata behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Each grouped subcommand is native, Make-backed, or bridge-backed, but never multiple.
//   - The visible registry is derived from the canonical catalog.
package main

import "testing"

func TestCommandCatalog_ContainsExpectedRoots(t *testing.T) {
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

func TestCommandCatalog_GroupedSubcommandsHaveSingleOwner(t *testing.T) {
	catalog := buildCommandCatalog()
	for _, root := range catalog.RootCommands {
		if root.NativeRun != nil {
			continue
		}
		for _, subcommand := range root.Subcommands {
			owners := 0
			if subcommand.NativeRun != nil {
				owners++
			}
			if subcommand.MakeTarget != "" {
				owners++
			}
			if subcommand.ScriptRelativePath != "" {
				owners++
			}
			if owners != 1 {
				t.Fatalf("%s %s should have exactly one owner, got native=%t make=%q script=%q",
					root.Name, subcommand.Name, subcommand.NativeRun != nil, subcommand.MakeTarget, subcommand.ScriptRelativePath)
			}
		}
	}
}

func TestCommandCatalog_HiddenCommandsStayOutOfVisibleRegistry(t *testing.T) {
	registry := buildCommandRegistry()
	for _, command := range registry.RootCommands {
		if command.Name == "__complete" {
			t.Fatal("hidden command __complete leaked into visible registry")
		}
	}
}
