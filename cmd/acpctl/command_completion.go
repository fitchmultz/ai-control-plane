// command_completion.go - Completion catalog and suggestion helpers.
//
// Purpose:
//
//	Derive shell-completion catalogs and contextual value suggestions from the
//	typed acpctl command tree.
//
// Responsibilities:
//   - Build root and subgroup completion catalogs from visible commands.
//   - Resolve contextual subcommand and option suggestions.
//   - Deduplicate and sort suggestion lists deterministically.
//
// Scope:
//   - Completion catalogs and suggestion filtering only.
//
// Usage:
//   - Used by completion script generation and hidden completion commands.
//
// Invariants/Assumptions:
//   - Hidden commands are never exposed through normal completion suggestions.
package main

import (
	"sort"
	"strings"
)

type commandRegistry struct {
	RootCommands     []commandDescriptor
	GroupSubcommands map[string][]commandDescriptor
}

type commandDescriptor struct {
	Name        string
	Description string
}

func buildCommandRegistry() commandRegistry {
	spec, err := loadCommandSpec()
	if err != nil {
		return commandRegistry{}
	}
	registry := commandRegistry{
		RootCommands:     make([]commandDescriptor, 0, len(spec.VisibleRoots)+1),
		GroupSubcommands: make(map[string][]commandDescriptor, len(spec.VisibleRoots)),
	}
	for _, root := range spec.VisibleRoots {
		registry.RootCommands = append(registry.RootCommands, commandDescriptor{Name: root.Name, Description: root.Summary})
		if len(root.Children) == 0 {
			continue
		}
		subcommands := make([]commandDescriptor, 0, len(root.Children))
		for _, child := range root.Children {
			if child.Hidden {
				continue
			}
			subcommands = append(subcommands, commandDescriptor{Name: child.Name, Description: child.Summary})
		}
		if len(subcommands) > 0 {
			registry.GroupSubcommands[root.Name] = subcommands
		}
	}
	registry.RootCommands = append(registry.RootCommands, commandDescriptor{Name: "help", Description: "Show this help message"})
	return registry
}

type commandCompletionCatalog struct {
	RootCommands     []string
	GroupSubcommands map[string][]string
}

func buildCompletionCatalog() commandCompletionCatalog {
	registry := buildCommandRegistry()
	catalog := commandCompletionCatalog{
		RootCommands:     make([]string, 0, len(registry.RootCommands)),
		GroupSubcommands: make(map[string][]string, len(registry.GroupSubcommands)),
	}
	for _, root := range registry.RootCommands {
		catalog.RootCommands = append(catalog.RootCommands, root.Name)
	}
	for name, subcommands := range registry.GroupSubcommands {
		values := make([]string, 0, len(subcommands))
		for _, subcommand := range subcommands {
			values = append(values, subcommand.Name)
		}
		catalog.GroupSubcommands[name] = values
	}
	return catalog
}

func resolveSuggestions(words []string, prefix string, repoRoot string) []string {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil
	}
	if len(words) == 0 {
		return filterSuggestionsByPrefix(append([]string(nil), spec.VisibleRootNames...), prefix)
	}
	current := spec.Root
	for _, word := range words {
		next := findChildCommand(current, word)
		if next == nil {
			break
		}
		current = next
	}
	if strings.Contains(prefix, "=") {
		key, value, _ := strings.Cut(prefix, "=")
		values := current.suggestValues(repoRoot, key)
		if len(values) == 0 {
			return nil
		}
		suggestions := make([]string, 0, len(values))
		for _, candidate := range values {
			if strings.HasPrefix(candidate, value) {
				suggestions = append(suggestions, key+"="+candidate)
			}
		}
		return dedupeAndSort(suggestions)
	}
	if current != spec.Root && len(current.Children) > 0 {
		names := make([]string, 0, len(current.Children))
		for _, child := range current.Children {
			if child.Hidden {
				continue
			}
			names = append(names, child.Name)
		}
		return filterSuggestionsByPrefix(dedupeAndSort(names), prefix)
	}
	if len(current.Children) == 0 {
		return filterSuggestionsByPrefix(visibleOptionSuggestions(current), prefix)
	}
	return filterSuggestionsByPrefix(append([]string(nil), spec.VisibleRootNames...), prefix)
}

func (spec *commandSpec) suggestValues(repoRoot string, key string) []string {
	values := make([]string, 0)
	for _, argument := range spec.Arguments {
		if argument.SuggestAsKey == key && argument.Suggestions != nil {
			values = append(values, argument.Suggestions(repoRoot)...)
		}
	}
	for _, option := range spec.Options {
		if option.SuggestAsKey == key && option.Suggestions != nil {
			values = append(values, option.Suggestions(repoRoot)...)
		}
	}
	return dedupeAndSort(values)
}

func dedupeAndSort(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}

func visibleOptionSuggestions(spec *commandSpec) []string {
	if len(spec.Options) == 0 {
		return nil
	}
	values := make([]string, 0, len(spec.Options)*2)
	for _, option := range spec.Options {
		values = append(values, "--"+option.Name)
		if option.Short != "" {
			values = append(values, "-"+option.Short)
		}
	}
	return dedupeAndSort(values)
}

func filterSuggestionsByPrefix(values []string, prefix string) []string {
	if len(values) == 0 || prefix == "" {
		return values
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
