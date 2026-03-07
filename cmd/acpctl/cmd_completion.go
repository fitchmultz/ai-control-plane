// cmd_completion.go - Completion subcommand implementation
//
// Purpose: Generate shell completion scripts
// Responsibilities:
//   - Generate bash, zsh, and fish completions
//   - Output completion scripts to stdout
//
// Non-scope:
//   - Does not install completions to system directories
//   - Does not detect user's shell

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func runCompletionSubcommand(args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Error: Shell type required (bash, zsh, fish)")
		fmt.Fprintln(stderr, "Usage: acpctl completion <bash|zsh|fish>")
		return exitcodes.ACPExitUsage
	}

	shell := strings.ToLower(args[0])

	switch shell {
	case "bash":
		return generateBashCompletion(stdout, stderr)
	case "zsh":
		return generateZshCompletion(stdout, stderr)
	case "fish":
		return generateFishCompletion(stdout, stderr)
	case "help", "--help", "-h":
		printCompletionHelp(stdout)
		return exitcodes.ACPExitSuccess
	default:
		fmt.Fprintf(stderr, "Error: Unknown shell type: %s\n", shell)
		fmt.Fprintln(stderr, "Supported shells: bash, zsh, fish")
		return exitcodes.ACPExitUsage
	}
}

func printCompletionHelp(out *os.File) {
	command := mustLookupNativeCommand("completion")

	fmt.Fprint(out, `Usage: acpctl completion <bash|zsh|fish>

Generate shell completion scripts.

Supported shells:
`)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(out, "  %-5s %s\n", subcommand.Name, subcommand.Description)
	}
	fmt.Fprint(out, `

To install completions:

  Bash:
    acpctl completion bash > /usr/local/etc/bash_completion.d/acpctl

  Zsh:
    acpctl completion zsh > "${fpath[1]}/_acpctl"

  Fish:
    acpctl completion fish > ~/.config/fish/completions/acpctl.fish

Exit codes:
  0   Success
  64  Usage error
`)
}

func generateBashCompletion(stdout *os.File, stderr *os.File) int {
	catalog := buildCompletionCatalog(detectRepoRoot())
	var script strings.Builder

	fmt.Fprintf(&script, "_acpctl_complete() {\n")
	fmt.Fprintf(&script, "    local cur\n")
	fmt.Fprintf(&script, "    COMPREPLY=()\n")
	fmt.Fprintf(&script, "    cur=\"${COMP_WORDS[COMP_CWORD]}\"\n\n")
	fmt.Fprintf(&script, "    local commands=%q\n\n", bashWordList(catalog.rootCommands))
	fmt.Fprintf(&script, "    if [[ ${COMP_CWORD} -eq 1 ]]; then\n")
	fmt.Fprintf(&script, "        COMPREPLY=( $(compgen -W \"${commands}\" -- \"${cur}\") )\n")
	fmt.Fprintf(&script, "        return 0\n")
	fmt.Fprintf(&script, "    fi\n\n")
	fmt.Fprintf(&script, "    case \"${COMP_WORDS[1]}\" in\n")
	script.WriteString(renderBashSubcommandCases(catalog))
	fmt.Fprintf(&script, "        *)\n")
	fmt.Fprintf(&script, "            COMPREPLY=()\n")
	fmt.Fprintf(&script, "            ;;\n")
	fmt.Fprintf(&script, "    esac\n")
	fmt.Fprintf(&script, "}\n\n")
	fmt.Fprintf(&script, "complete -o default -F _acpctl_complete acpctl\n")

	fmt.Fprint(stdout, script.String())
	return exitcodes.ACPExitSuccess
}

func generateZshCompletion(stdout *os.File, stderr *os.File) int {
	registry := buildCommandRegistry()
	var script strings.Builder

	fmt.Fprintf(&script, "#compdef acpctl\n\n")
	fmt.Fprintf(&script, "_acpctl() {\n")
	fmt.Fprintf(&script, "    local curcontext=\"$curcontext\" state line\n")
	fmt.Fprintf(&script, "    typeset -A opt_args\n\n")
	fmt.Fprintf(&script, "    _arguments -C \\\n")
	fmt.Fprintf(&script, "        '1: :_acpctl_commands' \\\n")
	fmt.Fprintf(&script, "        '*:: :->args'\n\n")
	fmt.Fprintf(&script, "    case \"$line[1]\" in\n")
	script.WriteString(renderZshSubcommandCases(registry))
	fmt.Fprintf(&script, "        *)\n")
	fmt.Fprintf(&script, "            _files\n")
	fmt.Fprintf(&script, "            ;;\n")
	fmt.Fprintf(&script, "    esac\n")
	fmt.Fprintf(&script, "}\n\n")
	fmt.Fprintf(&script, "_acpctl_commands() {\n")
	fmt.Fprintf(&script, "    local commands=(\n")
	script.WriteString(renderZshDescribeEntries(registry.RootCommands))
	fmt.Fprintf(&script, "    )\n")
	fmt.Fprintf(&script, "    _describe -t commands 'acpctl commands' commands \"$@\"\n")
	fmt.Fprintf(&script, "}\n\n")
	script.WriteString(renderZshSubcommandFunctions(registry))
	fmt.Fprintf(&script, "compdef _acpctl acpctl\n")

	fmt.Fprint(stdout, script.String())
	return exitcodes.ACPExitSuccess
}

func generateFishCompletion(stdout *os.File, stderr *os.File) int {
	registry := buildCommandRegistry()
	var script strings.Builder

	fmt.Fprintf(&script, "function __acpctl_complete\n")
	fmt.Fprintf(&script, "    # Fish completion function for acpctl\n")
	fmt.Fprintf(&script, "end\n\n")
	fmt.Fprintf(&script, "complete -c acpctl -f\n\n")
	for _, command := range registry.RootCommands {
		fmt.Fprintf(
			&script,
			"complete -c acpctl -n '__fish_use_subcommand' -a %s -d %s\n",
			shellSingleQuote(command.Name),
			shellSingleQuote(command.Description),
		)
	}
	fmt.Fprintf(&script, "\n")
	for _, command := range registry.RootCommands {
		for _, subcommand := range registry.GroupSubcommands[command.Name] {
			fmt.Fprintf(
				&script,
				"complete -c acpctl -n '__fish_seen_subcommand_from %s' -a %s -d %s\n",
				command.Name,
				shellSingleQuote(subcommand.Name),
				shellSingleQuote(subcommand.Description),
			)
		}
	}

	fmt.Fprint(stdout, script.String())
	return exitcodes.ACPExitSuccess
}

func runHiddenComplete(args []string, stdout *os.File, stderr *os.File) int {
	// Hidden completion helper for Cobra compatibility if needed in future
	// Currently not used but reserved for compatibility
	return exitcodes.ACPExitSuccess
}

// =============================================================================
// Completion Catalog and Dynamic Suggestions (for testing)
// =============================================================================

// completionCatalog holds all completion data
type completionCatalog struct {
	rootCommands     []string
	groupSubcommands map[string][]string
	bridgeNames      []string
	modelNames       []string
	presetNames      []string
	scenarioIDs      []string
	keyAliases       []string
}

// resolveSuggestions returns suggestions based on context
func resolveSuggestions(words []string, prefix string, catalog completionCatalog) []string {
	var suggestions []string

	// Handle KEY=VALUE style prefixes
	if strings.HasPrefix(prefix, "ALIAS=") {
		for _, alias := range catalog.keyAliases {
			suggestions = append(suggestions, "ALIAS="+alias)
		}
		return dedupeAndSort(suggestions)
	}
	if strings.HasPrefix(prefix, "MODEL=") {
		for _, model := range catalog.modelNames {
			suggestions = append(suggestions, "MODEL="+model)
		}
		return dedupeAndSort(suggestions)
	}
	if strings.HasPrefix(prefix, "SCENARIO=") {
		for _, id := range catalog.scenarioIDs {
			suggestions = append(suggestions, "SCENARIO="+id)
		}
		return dedupeAndSort(suggestions)
	}

	// If we have context words, suggest based on context
	if len(words) > 0 {
		lastWord := words[len(words)-1]
		if subcmds, ok := catalog.groupSubcommands[lastWord]; ok {
			return subcmds
		}
	}

	// Default: return root commands
	return catalog.rootCommands
}

// dedupeAndSort removes duplicates and sorts a slice
func dedupeAndSort(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	// Simple bubble sort for consistency
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// buildCompletionCatalog creates a catalog from the repository
func buildCompletionCatalog(repoRoot string) completionCatalog {
	registry := buildCommandRegistry()
	catalog := completionCatalog{
		rootCommands:     make([]string, 0, len(registry.RootCommands)),
		groupSubcommands: make(map[string][]string, len(registry.GroupSubcommands)),
		bridgeNames:      make([]string, 0, len(registry.BridgeCommands)),
		modelNames:       extractModelNames(repoRoot),
		presetNames:      extractPresetNames(repoRoot),
		scenarioIDs:      extractScenarioIDs(repoRoot),
		keyAliases:       extractKeyAliases(repoRoot),
	}

	for _, command := range registry.RootCommands {
		catalog.rootCommands = append(catalog.rootCommands, command.Name)
	}
	for groupName, subcommands := range registry.GroupSubcommands {
		names := make([]string, 0, len(subcommands))
		for _, subcommand := range subcommands {
			names = append(names, subcommand.Name)
		}
		catalog.groupSubcommands[groupName] = names
	}
	for _, bridge := range registry.BridgeCommands {
		catalog.bridgeNames = append(catalog.bridgeNames, bridge.Name)
	}

	return catalog
}

func bashWordList(values []string) string {
	return strings.Join(values, " ")
}

func renderBashSubcommandCases(catalog completionCatalog) string {
	registry := buildCommandRegistry()
	var script strings.Builder
	for _, command := range registry.RootCommands {
		subcommands := catalog.groupSubcommands[command.Name]
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&script, "        %s)\n", command.Name)
		fmt.Fprintf(&script, "            local subcmds=%q\n", bashWordList(subcommands))
		fmt.Fprintf(&script, "            COMPREPLY=( $(compgen -W \"${subcmds}\" -- \"${cur}\") )\n")
		fmt.Fprintf(&script, "            ;;\n")
	}
	return script.String()
}

func renderZshSubcommandCases(registry commandRegistry) string {
	var script strings.Builder
	for _, command := range registry.RootCommands {
		if len(registry.GroupSubcommands[command.Name]) == 0 {
			continue
		}
		fmt.Fprintf(&script, "        %s)\n", command.Name)
		fmt.Fprintf(&script, "            _acpctl_%s\n", zshFunctionName(command.Name))
		fmt.Fprintf(&script, "            ;;\n")
	}
	return script.String()
}

func renderZshSubcommandFunctions(registry commandRegistry) string {
	var script strings.Builder
	for _, command := range registry.RootCommands {
		subcommands := registry.GroupSubcommands[command.Name]
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&script, "_acpctl_%s() {\n", zshFunctionName(command.Name))
		fmt.Fprintf(&script, "    local subcmds=(\n")
		script.WriteString(renderZshDescribeEntries(subcommands))
		fmt.Fprintf(&script, "    )\n")
		fmt.Fprintf(&script, "    _describe -t commands '%s subcommands' subcmds \"$@\"\n", command.Name)
		fmt.Fprintf(&script, "}\n\n")
	}
	return script.String()
}

func renderZshDescribeEntries(commands []commandDescriptor) string {
	var script strings.Builder
	for _, command := range commands {
		fmt.Fprintf(&script, "        %q\n", zshDescribeEntry(command))
	}
	return script.String()
}

func zshDescribeEntry(command commandDescriptor) string {
	description := strings.ReplaceAll(command.Description, `\`, `\\`)
	description = strings.ReplaceAll(description, ":", `\:`)
	return command.Name + ":" + description
}

func zshFunctionName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\"'\"'`) + "'"
}

// extractModelNames extracts model names from litellm.yaml
func extractModelNames(repoRoot string) []string {
	var models []string
	seen := make(map[string]bool)

	litellmPath := filepath.Join(repoRoot, "demo", "config", "litellm.yaml")
	data, err := os.ReadFile(litellmPath)
	if err != nil {
		return models
	}

	// Simple line-based parsing to extract model_name values
	// Handles both: "model_name: value" and "  - model_name: value" formats
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		// Check if line contains model_name (could be "- model_name:" or just "model_name:")
		if strings.Contains(trimmed, "model_name:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				model := strings.TrimSpace(parts[1])
				// Remove quotes if present
				model = strings.Trim(model, `"'`)
				if model != "" && !seen[model] {
					seen[model] = true
					models = append(models, model)
				}
			}
		}
	}

	return dedupeAndSort(models)
}

// extractPresetNames extracts preset names from demo_presets.yaml
func extractPresetNames(repoRoot string) []string {
	var presets []string

	presetsPath := filepath.Join(repoRoot, "demo", "config", "demo_presets.yaml")
	data, err := os.ReadFile(presetsPath)
	if err != nil {
		return presets
	}

	inPresets := false
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, "presets:") && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			inPresets = true
			continue
		}
		// Check for preset names (2-space indent followed by name:)
		if inPresets && len(line) >= 2 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
			trimmed := strings.TrimSpace(line)
			if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
				preset := strings.TrimSuffix(trimmed, ":")
				if preset != "" && preset != "presets" {
					presets = append(presets, preset)
				}
			}
		}
		// Exit presets section when we hit a non-indented line (except blank)
		if inPresets && line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "#") {
			if !strings.HasPrefix(line, "presets:") {
				break
			}
		}
	}

	return dedupeAndSort(presets)
}

// extractScenarioIDs extracts scenario IDs from scenario files
func extractScenarioIDs(repoRoot string) []string {
	var ids []string
	seen := make(map[string]bool)

	scenarioDir := filepath.Join(repoRoot, "local", "scripts", "demo_scenarios")
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		return ids
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match scenario_N_*.sh pattern
		if strings.HasPrefix(name, "scenario_") && strings.HasSuffix(name, ".sh") {
			parts := strings.Split(name, "_")
			if len(parts) >= 2 {
				id := parts[1]
				if _, err := filepath.Match("[0-9]*", id); err == nil && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}

	return dedupeAndSort(ids)
}

// extractKeyAliases extracts key aliases from scenario scripts
func extractKeyAliases(repoRoot string) []string {
	var aliases []string
	seen := make(map[string]bool)

	scenarioDir := filepath.Join(repoRoot, "local", "scripts", "demo_scenarios")
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		return aliases
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(scenarioDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Look for KEY_ALIAS patterns
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, "KEY_ALIAS=") || strings.Contains(line, "SCENARIO_KEY_ALIAS=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					alias := strings.TrimSpace(parts[1])
					// Remove quotes
					alias = strings.Trim(alias, `"'`)
					if alias != "" && !seen[alias] {
						seen[alias] = true
						aliases = append(aliases, alias)
					}
				}
			}
		}
	}

	return dedupeAndSort(aliases)
}

// extractConfigKeys extracts top-level keys from config files
func extractConfigKeys(repoRoot string) []string {
	var keys []string

	// Try to read a test config if it exists
	configPath := filepath.Join(repoRoot, "demo", "config", "test_config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return keys
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ":") {
			key := strings.TrimSuffix(line, ":")
			keys = append(keys, key)
		}
	}

	return dedupeAndSort(keys)
}
