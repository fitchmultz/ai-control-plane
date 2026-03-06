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
	fmt.Fprint(out, `Usage: acpctl completion <bash|zsh|fish>

Generate shell completion scripts.

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
	script := `_acpctl_complete() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Main commands
    local commands="ci files status doctor bridge completion deploy validate db key host demo terraform help"
    
    # Complete main command
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi
    
    # Subcommand completions
    case "${COMP_WORDS[1]}" in
        ci)
            local ci_cmds="should-run-runtime help"
            COMPREPLY=( $(compgen -W "${ci_cmds}" -- ${cur}) )
            ;;
        files)
            local file_cmds="sync-helm help"
            COMPREPLY=( $(compgen -W "${file_cmds}" -- ${cur}) )
            ;;
        deploy)
            local deploy_cmds="up down restart health logs ps up-production up-offline up-tls help"
            COMPREPLY=( $(compgen -W "${deploy_cmds}" -- ${cur}) )
            ;;
        validate)
            local validate_cmds="lint config detections siem-queries help"
            COMPREPLY=( $(compgen -W "${validate_cmds}" -- ${cur}) )
            ;;
        db)
            local db_cmds="status backup restore shell dr-drill help"
            COMPREPLY=( $(compgen -W "${db_cmds}" -- ${cur}) )
            ;;
        key)
            local key_cmds="gen revoke gen-dev gen-lead rbac-whoami rbac-roles help"
            COMPREPLY=( $(compgen -W "${key_cmds}" -- ${cur}) )
            ;;
        host)
            local host_cmds="preflight check apply install service-status upgrade-status help"
            COMPREPLY=( $(compgen -W "${host_cmds}" -- ${cur}) )
            ;;
        demo)
            local demo_cmds="scenario all preset snapshot restore status help"
            COMPREPLY=( $(compgen -W "${demo_cmds}" -- ${cur}) )
            ;;
        terraform)
            local tf_cmds="init plan apply destroy fmt validate help"
            COMPREPLY=( $(compgen -W "${tf_cmds}" -- ${cur}) )
            ;;
        *)
            COMPREPLY=()
            ;;
    esac
}

complete -o default -F _acpctl_complete acpctl
`
	fmt.Fprint(stdout, script)
	return exitcodes.ACPExitSuccess
}

func generateZshCompletion(stdout *os.File, stderr *os.File) int {
	script := `#compdef acpctl

_acpctl() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :_acpctl_commands' \
        '*:: :->args'

    case "$line[1]" in
        ci)
            _acpctl_ci
            ;;
        files)
            _acpctl_files
            ;;
        status)
            _arguments \
                '--json[Output in JSON format]' \
                '--wide[Show extended details]' \
                '--watch[Watch mode]'
            ;;
        doctor)
            _arguments \
                '--json[Output in JSON format]' \
                '--wide[Show extended details]' \
                '--fix[Attempt auto-remediation]'
            ;;
        *)
            _files
            ;;
    esac
}

_acpctl_commands() {
    local commands=(
        "ci:CI and local gate helpers"
        "files:Typed local file synchronization helpers"
        "status:Aggregated system health overview"
        "doctor:Environment preflight diagnostics"
        "bridge:Execute mapped legacy script implementations"
        "completion:Generate shell completion scripts"
        "deploy:Service lifecycle and deployment helpers"
        "validate:Validation and policy checks"
        "db:Database operations"
        "key:Virtual key operations"
        "host:Host deployment and service operations"
        "demo:Demo scenarios and state operations"
        "terraform:Terraform provisioning helpers"
        "help:Show help message"
    )
    _describe -t commands 'acpctl commands' commands "$@"
}

_acpctl_ci() {
    local subcmds=(
        "should-run-runtime:Decide whether runtime checks should run"
        "help:Show help"
    )
    _describe -t commands 'ci subcommands' subcmds "$@"
}

_acpctl_files() {
    local subcmds=(
        "sync-helm:Synchronize files into Helm chart"
        "help:Show help"
    )
    _describe -t commands 'files subcommands' subcmds "$@"
}

compdef _acpctl acpctl
`
	fmt.Fprint(stdout, script)
	return exitcodes.ACPExitSuccess
}

func generateFishCompletion(stdout *os.File, stderr *os.File) int {
	script := `function __acpctl_complete
    # Fish completion function for acpctl
end

complete -c acpctl -f

# Main commands
complete -c acpctl -n '__fish_use_subcommand' -a "ci" -d "CI and local gate helpers"
complete -c acpctl -n '__fish_use_subcommand' -a "files" -d "File synchronization helpers"
complete -c acpctl -n '__fish_use_subcommand' -a "status" -d "System health overview"
complete -c acpctl -n '__fish_use_subcommand' -a "doctor" -d "Environment diagnostics"
complete -c acpctl -n '__fish_use_subcommand' -a "bridge" -d "Execute legacy scripts"
complete -c acpctl -n '__fish_use_subcommand' -a "completion" -d "Generate completions"
complete -c acpctl -n '__fish_use_subcommand' -a "deploy" -d "Service lifecycle"
complete -c acpctl -n '__fish_use_subcommand' -a "validate" -d "Validation checks"
complete -c acpctl -n '__fish_use_subcommand' -a "db" -d "Database operations"
complete -c acpctl -n '__fish_use_subcommand' -a "key" -d "Virtual key operations"
complete -c acpctl -n '__fish_use_subcommand' -a "host" -d "Host deployment"
complete -c acpctl -n '__fish_use_subcommand' -a "demo" -d "Demo scenarios"
complete -c acpctl -n '__fish_use_subcommand' -a "terraform" -d "Terraform helpers"

# CI subcommands
complete -c acpctl -n '__fish_seen_subcommand_from ci' -a "should-run-runtime" -d "Decide runtime checks"

# Files subcommands
complete -c acpctl -n '__fish_seen_subcommand_from files' -a "sync-helm" -d "Sync Helm files"

# Status options
complete -c acpctl -n '__fish_seen_subcommand_from status' -l json -d "JSON output"
complete -c acpctl -n '__fish_seen_subcommand_from status' -l wide -d "Extended details"
complete -c acpctl -n '__fish_seen_subcommand_from status' -l watch -d "Watch mode"

# Doctor options
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l json -d "JSON output"
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l wide -d "Extended details"
complete -c acpctl -n '__fish_seen_subcommand_from doctor' -l fix -d "Auto-remediation"
`
	fmt.Fprint(stdout, script)
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
		if lastWord == "bridge" {
			return catalog.bridgeNames
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
	return completionCatalog{
		rootCommands: []string{"ci", "files", "status", "doctor", "bridge", "completion", "deploy", "validate", "db", "key", "host", "demo", "terraform", "help"},
		groupSubcommands: map[string][]string{
			"deploy":    {"up", "down", "restart", "health", "logs", "ps", "up-production", "up-offline", "up-tls"},
			"validate":  {"lint", "config", "detections", "siem-queries", "network-contract", "supply-chain", "secrets-audit"},
			"db":        {"status", "backup", "restore", "shell", "dr-drill"},
			"key":       {"gen", "revoke", "gen-dev", "gen-lead", "rbac-whoami", "rbac-roles"},
			"host":      {"preflight", "check", "apply", "install", "service-status", "upgrade-status"},
			"demo":      {"scenario", "all", "preset", "snapshot", "restore", "status"},
			"terraform": {"init", "plan", "apply", "destroy", "fmt", "validate"},
		},
		bridgeNames: []string{"host_deploy", "host_install", "host_preflight", "host_upgrade_slots", "onboard", "prepare_secrets_env", "prod_smoke_helm", "prod_smoke_test", "release_bundle", "switch_claude_mode"},
		modelNames:  extractModelNames(repoRoot),
		presetNames: extractPresetNames(repoRoot),
		scenarioIDs: extractScenarioIDs(repoRoot),
		keyAliases:  extractKeyAliases(repoRoot),
	}
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
