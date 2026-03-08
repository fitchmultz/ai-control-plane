// cmd_completion.go - Completion subcommand implementation.
//
// Purpose:
//
//	Generate shell completion scripts and hidden completion suggestions from
//	the typed acpctl command-spec tree.
//
// Responsibilities:
//   - Render bash, zsh, and fish completion scripts from command specs.
//   - Resolve hidden `__complete` suggestions from the same spec metadata.
//   - Surface tracked config-driven suggestions for command arguments.
//
// Scope:
//   - Completion rendering and metadata extraction only.
//
// Usage:
//   - Used by `acpctl completion <bash|zsh|fish>` and `acpctl __complete ...`.
//
// Invariants/Assumptions:
//   - Completion output remains deterministic for equivalent repo state.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type completionShellOptions struct {
	Shell string
}

type hiddenCompleteOptions struct {
	Words []string
}

func completionCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "completion",
		Summary:     "Generate shell completion scripts",
		Description: "Generate shell completion scripts.",
		Children: []*commandSpec{
			completionShellSpec("bash", "Generate Bash completion script"),
			completionShellSpec("zsh", "Generate Zsh completion script"),
			completionShellSpec("fish", "Generate Fish completion script"),
		},
	}
}

func completionShellSpec(name string, summary string) *commandSpec {
	return &commandSpec{
		Name:        name,
		Summary:     summary,
		Description: summary + ".",
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, _ parsedCommandInput) (any, error) {
				return completionShellOptions{Shell: name}, nil
			},
			NativeRun: runCompletionShellCommand,
		},
	}
}

func hiddenCompleteCommandSpec() *commandSpec {
	return &commandSpec{
		Name:              "__complete",
		Summary:           "Hidden shell completion helper",
		Description:       "Hidden shell completion helper.",
		Hidden:            true,
		AllowTrailingArgs: true,
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
				words := append([]string(nil), input.Arguments()...)
				words = append(words, input.Trailing()...)
				return hiddenCompleteOptions{Words: words}, nil
			},
			NativeRun: runHiddenComplete,
		},
	}
}

func runCompletionShellCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(completionShellOptions)
	switch opts.Shell {
	case "bash":
		return generateBashCompletion(ctx, runCtx.Stdout)
	case "zsh":
		return generateZshCompletion(runCtx.Stdout)
	case "fish":
		return generateFishCompletion(runCtx.Stdout)
	default:
		fmt.Fprintf(runCtx.Stderr, "Error: unsupported shell: %s\n", opts.Shell)
		return exitcodes.ACPExitUsage
	}
}

func runHiddenComplete(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(hiddenCompleteOptions)
	words := append([]string(nil), opts.Words...)
	prefix := ""
	if len(words) > 0 {
		prefix = words[len(words)-1]
		words = words[:len(words)-1]
	}
	for _, suggestion := range resolveSuggestions(words, prefix, runCtx.RepoRoot) {
		fmt.Fprintln(runCtx.Stdout, suggestion)
	}
	return exitcodes.ACPExitSuccess
}

func generateBashCompletion(ctx context.Context, stdout *os.File) int {
	catalog := buildCompletionCatalog()
	var script strings.Builder

	fmt.Fprintf(&script, "_acpctl_complete() {\n")
	fmt.Fprintf(&script, "    local cur\n")
	fmt.Fprintf(&script, "    COMPREPLY=()\n")
	fmt.Fprintf(&script, "    cur=\"${COMP_WORDS[COMP_CWORD]}\"\n\n")
	fmt.Fprintf(&script, "    local commands=%q\n\n", strings.Join(catalog.RootCommands, " "))
	fmt.Fprintf(&script, "    if [[ ${COMP_CWORD} -eq 1 ]]; then\n")
	fmt.Fprintf(&script, "        COMPREPLY=( $(compgen -W \"${commands}\" -- \"${cur}\") )\n")
	fmt.Fprintf(&script, "        return 0\n")
	fmt.Fprintf(&script, "    fi\n\n")
	fmt.Fprintf(&script, "    case \"${COMP_WORDS[1]}\" in\n")
	for _, root := range catalog.RootCommands {
		subcommands := catalog.GroupSubcommands[root]
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&script, "        %s)\n", root)
		fmt.Fprintf(&script, "            local subcmds=%q\n", strings.Join(subcommands, " "))
		fmt.Fprintf(&script, "            COMPREPLY=( $(compgen -W \"${subcmds}\" -- \"${cur}\") )\n")
		fmt.Fprintf(&script, "            ;;\n")
	}
	fmt.Fprintf(&script, "        *)\n")
	fmt.Fprintf(&script, "            COMPREPLY=()\n")
	fmt.Fprintf(&script, "            ;;\n")
	fmt.Fprintf(&script, "    esac\n")
	fmt.Fprintf(&script, "}\n\n")
	fmt.Fprintf(&script, "complete -o default -F _acpctl_complete acpctl\n")

	fmt.Fprint(stdout, script.String())
	_ = ctx
	return exitcodes.ACPExitSuccess
}

func generateZshCompletion(stdout *os.File) int {
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
	for _, root := range registry.RootCommands {
		subcommands := registry.GroupSubcommands[root.Name]
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&script, "        %s)\n", root.Name)
		fmt.Fprintf(&script, "            _acpctl_%s\n", zshFunctionName(root.Name))
		fmt.Fprintf(&script, "            ;;\n")
	}
	fmt.Fprintf(&script, "        *)\n")
	fmt.Fprintf(&script, "            _files\n")
	fmt.Fprintf(&script, "            ;;\n")
	fmt.Fprintf(&script, "    esac\n")
	fmt.Fprintf(&script, "}\n\n")
	fmt.Fprintf(&script, "_acpctl_commands() {\n")
	fmt.Fprintf(&script, "    local commands=(\n")
	for _, root := range registry.RootCommands {
		fmt.Fprintf(&script, "        %q\n", zshDescribeEntry(root))
	}
	fmt.Fprintf(&script, "    )\n")
	fmt.Fprintf(&script, "    _describe -t commands 'acpctl commands' commands \"$@\"\n")
	fmt.Fprintf(&script, "}\n\n")
	for _, root := range registry.RootCommands {
		subcommands := registry.GroupSubcommands[root.Name]
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&script, "_acpctl_%s() {\n", zshFunctionName(root.Name))
		fmt.Fprintf(&script, "    local subcmds=(\n")
		for _, subcommand := range subcommands {
			fmt.Fprintf(&script, "        %q\n", zshDescribeEntry(subcommand))
		}
		fmt.Fprintf(&script, "    )\n")
		fmt.Fprintf(&script, "    _describe -t commands '%s subcommands' subcmds \"$@\"\n", root.Name)
		fmt.Fprintf(&script, "}\n\n")
	}
	fmt.Fprintf(&script, "compdef _acpctl acpctl\n")

	fmt.Fprint(stdout, script.String())
	return exitcodes.ACPExitSuccess
}

func generateFishCompletion(stdout *os.File) int {
	registry := buildCommandRegistry()
	var script strings.Builder

	fmt.Fprintf(&script, "function __acpctl_complete\n")
	fmt.Fprintf(&script, "    # Fish completion function for acpctl\n")
	fmt.Fprintf(&script, "end\n\n")
	fmt.Fprintf(&script, "complete -c acpctl -f\n\n")
	for _, root := range registry.RootCommands {
		fmt.Fprintf(
			&script,
			"complete -c acpctl -n '__fish_use_subcommand' -a %s -d %s\n",
			shellSingleQuote(root.Name),
			shellSingleQuote(root.Description),
		)
	}
	fmt.Fprintf(&script, "\n")
	for _, root := range registry.RootCommands {
		for _, subcommand := range registry.GroupSubcommands[root.Name] {
			fmt.Fprintf(
				&script,
				"complete -c acpctl -n '__fish_seen_subcommand_from %s' -a %s -d %s\n",
				root.Name,
				shellSingleQuote(subcommand.Name),
				shellSingleQuote(subcommand.Description),
			)
		}
	}

	fmt.Fprint(stdout, script.String())
	return exitcodes.ACPExitSuccess
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

func extractModelNames(repoRoot string) []string {
	litellmPath := filepath.Join(repoRoot, "demo", "config", "litellm.yaml")
	config, err := catalog.LoadLiteLLMConfig(litellmPath)
	if err != nil {
		return nil
	}
	return catalog.ApprovedModelNames(config)
}

func extractPresetNames(repoRoot string) []string {
	presetsPath := filepath.Join(repoRoot, "demo", "config", "demo_presets.yaml")
	config, err := catalog.LoadDemoPresets(presetsPath)
	if err != nil {
		return nil
	}
	return catalog.PresetNames(config)
}

func extractScenarioIDs(repoRoot string) []string {
	presetsPath := filepath.Join(repoRoot, "demo", "config", "demo_presets.yaml")
	config, err := catalog.LoadDemoPresets(presetsPath)
	if err != nil {
		return nil
	}
	return catalog.ScenarioIDs(config)
}

func extractKeyAliases(string) []string {
	return nil
}

func extractConfigKeys(repoRoot string) []string {
	var keys []string

	configPath := filepath.Join(repoRoot, "demo", "config", "test_config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return keys
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ":") {
			keys = append(keys, strings.TrimSuffix(line, ":"))
		}
	}

	return dedupeAndSort(keys)
}
