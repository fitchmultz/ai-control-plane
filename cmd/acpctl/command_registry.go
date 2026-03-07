// command_registry.go - Canonical acpctl command metadata registry
//
// Purpose:
//
//	Provide the single source of truth for visible acpctl command metadata
//	used by root help and shell completion generation.
//
// Responsibilities:
//   - Define native root commands and their visible subcommands.
//   - Aggregate delegated groups from delegatedGroups.
//   - Aggregate bridge script names from bridgedScripts.
//   - Keep user-facing command ordering deterministic.
//
// Scope:
//   - Metadata and lookup helpers only; execution remains in command handlers.
//
// Usage:
//   - Consumed by main.go and cmd_completion.go.
//
// Invariants/Assumptions:
//   - Hidden/internal commands stay out of visible command lists.
//   - delegatedGroups and bridgedScripts remain authoritative for delegated and
//     bridge names respectively.
package main

import (
	"context"
	"os"
)

type commandDescriptor struct {
	Name        string
	Description string
}

type nativeCommandDefinition struct {
	commandDescriptor
	Run         func(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int
	Subcommands []commandDescriptor
	Hidden      bool
}

type commandRegistry struct {
	RootCommands     []commandDescriptor
	GroupSubcommands map[string][]commandDescriptor
	BridgeCommands   []commandDescriptor
}

func lookupNativeCommand(name string) (nativeCommandDefinition, bool) {
	for _, command := range nativeCommandDefinitions() {
		if command.Name == name {
			return command, true
		}
	}
	return nativeCommandDefinition{}, false
}

func mustLookupNativeCommand(name string) nativeCommandDefinition {
	command, ok := lookupNativeCommand(name)
	if !ok {
		panic("missing native command definition: " + name)
	}
	return command
}

func buildCommandRegistry() commandRegistry {
	nativeCommands := nativeCommandDefinitions()
	registry := commandRegistry{
		RootCommands:     make([]commandDescriptor, 0, len(nativeCommands)+len(delegatedGroups)+1),
		GroupSubcommands: make(map[string][]commandDescriptor, len(nativeCommands)+len(delegatedGroups)),
		BridgeCommands:   make([]commandDescriptor, 0, len(bridgedScripts)),
	}

	for _, command := range nativeCommands {
		if command.Hidden {
			continue
		}

		registry.RootCommands = append(registry.RootCommands, command.commandDescriptor)

		subcommands := append([]commandDescriptor(nil), command.Subcommands...)
		if command.Name == "bridge" {
			for _, script := range bridgedScripts {
				registry.BridgeCommands = append(registry.BridgeCommands, commandDescriptor{
					Name:        script.Name,
					Description: script.Description,
				})
			}
			subcommands = append(subcommands, registry.BridgeCommands...)
		}

		if len(subcommands) > 0 {
			registry.GroupSubcommands[command.Name] = subcommands
		}
	}

	for _, group := range delegatedGroups {
		registry.RootCommands = append(registry.RootCommands, commandDescriptor{
			Name:        group.Name,
			Description: group.Description,
		})

		subcommands := make([]commandDescriptor, 0, len(group.Subcommands))
		for _, subcommand := range group.Subcommands {
			subcommands = append(subcommands, commandDescriptor{
				Name:        subcommand.Name,
				Description: subcommand.Description,
			})
		}
		registry.GroupSubcommands[group.Name] = subcommands
	}

	registry.RootCommands = append(registry.RootCommands, commandDescriptor{
		Name:        "help",
		Description: "Show this help message",
	})

	return registry
}

func nativeCommandDefinitions() []nativeCommandDefinition {
	return []nativeCommandDefinition{
		{
			commandDescriptor: commandDescriptor{
				Name:        "ci",
				Description: "CI and local gate helpers",
			},
			Run: runCISubcommand,
			Subcommands: []commandDescriptor{
				{Name: "should-run-runtime", Description: "Decide whether runtime checks should run"},
				{Name: "wait", Description: "Wait for services to become healthy"},
			},
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "files",
				Description: "Typed local file synchronization helpers",
			},
			Run: runFilesSubcommand,
			Subcommands: []commandDescriptor{
				{Name: "sync-helm", Description: "Synchronize canonical repository files into Helm chart files/"},
			},
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "status",
				Description: "Aggregated system health overview",
			},
			Run: runStatusCommand,
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "health",
				Description: "Run service health checks",
			},
			Run: runHealthCommand,
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "doctor",
				Description: "Environment preflight diagnostics",
			},
			Run: runDoctorCommand,
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "benchmark",
				Description: "Lightweight local performance baseline",
			},
			Run: runBenchmarkCommand,
			Subcommands: []commandDescriptor{
				{Name: "baseline", Description: "Run the local gateway performance baseline"},
			},
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "bridge",
				Description: "Execute mapped legacy script implementations directly",
			},
			Run: runBridgeSubcommand,
		},
		{
			commandDescriptor: commandDescriptor{
				Name:        "completion",
				Description: "Generate shell completion scripts",
			},
			Run: runCompletionSubcommand,
			Subcommands: []commandDescriptor{
				{Name: "bash", Description: "Generate Bash completion script"},
				{Name: "zsh", Description: "Generate Zsh completion script"},
				{Name: "fish", Description: "Generate Fish completion script"},
			},
		},
		{
			commandDescriptor: commandDescriptor{
				Name: "__complete",
			},
			Run:    runHiddenComplete,
			Hidden: true,
		},
	}
}
