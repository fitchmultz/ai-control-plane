// completion_suggestions_test.go - Suggestion-tree coverage for hidden completion paths.
//
// Purpose:
//   - Verify hidden completion suggestion resolution from the typed command tree.
//
// Responsibilities:
//   - Cover root/subcommand suggestions and value completion for keyed arguments.
//
// Scope:
//   - Hidden completion suggestion behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Suggestions derive from typed command metadata rather than shell-specific code.
package main

import (
	"slices"
	"testing"
)

func TestResolveSuggestionsRootAndSubcommands(t *testing.T) {
	rootSuggestions := resolveSuggestions(nil, "", t.TempDir())
	if !slices.Contains(rootSuggestions, "ci") || !slices.Contains(rootSuggestions, "deploy") || !slices.Contains(rootSuggestions, "tenant") {
		t.Fatalf("expected root suggestions to include current command tree, got %v", rootSuggestions)
	}

	deploySuggestions := resolveSuggestions([]string{"deploy"}, "", t.TempDir())
	if !slices.Contains(deploySuggestions, "release-bundle") || !slices.Contains(deploySuggestions, "readiness-evidence") || !slices.Contains(deploySuggestions, "assessor-packet") {
		t.Fatalf("expected deploy suggestions from command tree, got %v", deploySuggestions)
	}
}

func TestResolveSuggestionsFallsBackWhenPathIsUnknown(t *testing.T) {
	rootSuggestions := resolveSuggestions([]string{"not-a-command"}, "", t.TempDir())
	if !slices.Contains(rootSuggestions, "ci") || !slices.Contains(rootSuggestions, "deploy") {
		t.Fatalf("expected root fallback suggestions, got %v", rootSuggestions)
	}
}

func TestResolveSuggestionsFiltersRootsByPrefix(t *testing.T) {
	rootSuggestions := resolveSuggestions(nil, "de", t.TempDir())
	if !slices.Equal(rootSuggestions, []string{"deploy"}) {
		t.Fatalf("expected prefix-filtered root suggestions, got %v", rootSuggestions)
	}
}

func TestResolveSuggestionsFiltersKnownSubcommandsByPrefix(t *testing.T) {
	deploySuggestions := resolveSuggestions([]string{"deploy"}, "re", t.TempDir())
	if !slices.Equal(deploySuggestions, []string{"readiness-evidence", "release-bundle"}) {
		t.Fatalf("expected prefix-filtered deploy suggestions, got %v", deploySuggestions)
	}
}
