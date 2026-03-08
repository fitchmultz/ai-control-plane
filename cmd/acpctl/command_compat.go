// command_compat.go - Compatibility helpers for existing tests and legacy help.
//
// Purpose:
//
//	Provide narrow compatibility shims while the typed command-spec layer
//	replaces the old registry API surface.
//
// Responsibilities:
//   - Expose first-level root/subcommand descriptors for legacy helpers.
//   - Support delegated-group compatibility tests against the new command tree.
//
// Scope:
//   - Transitional compatibility for the cmd/acpctl package only.
//
// Usage:
//   - Consumed by older command help functions and tests in this package.
//
// Invariants/Assumptions:
//   - These helpers reflect the typed command tree and do not define new metadata.
package main

import (
	"context"
	"os"
)

type subcommandDefinition struct {
	commandDescriptor
}

type rootCommandDefinition struct {
	commandDescriptor
	Subcommands []subcommandDefinition
}

func lookupNativeRootCommand(name string) (rootCommandDefinition, error) {
	node, err := findCommand([]string{name})
	if err != nil {
		return rootCommandDefinition{}, err
	}
	definition := rootCommandDefinition{
		commandDescriptor: commandDescriptor{
			Name:        node.Name,
			Description: node.Summary,
		},
		Subcommands: make([]subcommandDefinition, 0, len(node.Children)),
	}
	for _, child := range node.Children {
		if child.Hidden {
			continue
		}
		definition.Subcommands = append(definition.Subcommands, subcommandDefinition{
			commandDescriptor: commandDescriptor{
				Name:        child.Name,
				Description: child.Summary,
			},
		})
	}
	return definition, nil
}

func lookupDelegatedGroup(name string) (*commandSpec, bool) {
	node, err := findCommand([]string{name})
	if err != nil || len(node.Children) == 0 {
		return nil, false
	}
	return node, true
}

func runDelegatedGroup(ctx context.Context, root *commandSpec, args []string, stdout *os.File, stderr *os.File) int {
	invocation, err := parseInvocation(append([]string{root.Name}, args...))
	if err != nil {
		return 64
	}
	return executeInvocation(ctx, invocation, stdout, stderr)
}
