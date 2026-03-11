// command_compile.go - Command-tree compilation and lookup helpers.
//
// Purpose:
//
//	Compile the typed acpctl command tree into a validated lookup structure
//	used by parsing, help rendering, and completions.
//
// Responsibilities:
//   - Load and cache the compiled command tree.
//   - Validate backend ownership and duplicate command paths.
//   - Provide path and child-command lookup helpers.
//
// Scope:
//   - Command metadata compilation and lookup only.
//
// Usage:
//   - Used by parser, help rendering, completions, and dispatcher startup.
//
// Invariants/Assumptions:
//   - Leaf commands always resolve to one concrete backend.
//   - Group commands never own an execution backend.
package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	commandSpecOnce sync.Once
	commandSpecData compiledCommandSpec
	commandSpecErr  error
)

func loadCommandSpec() (compiledCommandSpec, error) {
	commandSpecOnce.Do(func() {
		commandSpecData, commandSpecErr = compileCommandSpec(acpctlCommandSpec())
	})
	return commandSpecData, commandSpecErr
}

func commandStartupError() error {
	_, err := loadCommandSpec()
	return err
}

func rootCommandSpec(children ...*commandSpec) *commandSpec {
	return &commandSpec{
		Name:        "acpctl",
		Summary:     "Typed control-plane CLI for AI Control Plane operations.",
		Description: "Typed control-plane CLI for AI Control Plane operations.",
		Examples: []string{
			"acpctl ci should-run-runtime --quiet",
			"acpctl ci wait --timeout 120",
			"acpctl env get LITELLM_MASTER_KEY",
			"acpctl chargeback report --format all",
			"acpctl benchmark baseline --requests 20 --concurrency 2",
			"acpctl onboard codex --mode subscription --verify",
			"acpctl deploy readiness-evidence run",
		},
		Sections: []commandHelpSection{
			{
				Title: "Environment",
				Lines: []string{
					"ACPCTL_MAKE_BIN  Override make executable used by delegated commands (default: make)",
					"ACP_REPO_ROOT    Override repository root detection",
				},
			},
		},
		Children: children,
	}
}

func compileCommandSpec(root *commandSpec) (compiledCommandSpec, error) {
	if root == nil {
		return compiledCommandSpec{}, errors.New("command spec root is nil")
	}
	compiled := compiledCommandSpec{
		Root:        root,
		NodesByPath: make(map[string]*commandSpec),
	}
	if err := compileCommandNode(root, nil, compiled.NodesByPath, &compiled.VisibleRoots, &compiled.VisibleRootNames); err != nil {
		return compiledCommandSpec{}, err
	}
	return compiled, nil
}

func compileCommandNode(node *commandSpec, ancestors []*commandSpec, index map[string]*commandSpec, visibleRoots *[]*commandSpec, visibleRootNames *[]string) error {
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("command spec name must not be empty")
	}
	path := append(append([]*commandSpec(nil), ancestors...), node)
	pathKey := commandPathKey(path[1:])
	if _, exists := index[pathKey]; exists {
		return fmt.Errorf("duplicate command path: %s", pathKey)
	}
	index[pathKey] = node
	if len(ancestors) == 1 && !node.Hidden {
		*visibleRoots = append(*visibleRoots, node)
		*visibleRootNames = append(*visibleRootNames, node.Name)
	}
	if len(node.Children) > 0 {
		seen := make(map[string]struct{}, len(node.Children))
		for _, child := range node.Children {
			if _, exists := seen[child.Name]; exists {
				return fmt.Errorf("duplicate child command under %s: %s", commandPathKey(path[1:]), child.Name)
			}
			seen[child.Name] = struct{}{}
			if err := compileCommandNode(child, path, index, visibleRoots, visibleRootNames); err != nil {
				return err
			}
		}
		if node.Backend.Kind != "" {
			return fmt.Errorf("group command cannot also own an execution backend: %s", commandPathKey(path[1:]))
		}
		return nil
	}
	switch node.Backend.Kind {
	case commandBackendNative:
		if node.Backend.NativeBind == nil || node.Backend.NativeRun == nil {
			return fmt.Errorf("native command missing binder or runner: %s", commandPathKey(path[1:]))
		}
	case commandBackendMake:
		if strings.TrimSpace(node.Backend.MakeTarget) == "" {
			return fmt.Errorf("make command missing target: %s", commandPathKey(path[1:]))
		}
	case commandBackendBridge:
		if strings.TrimSpace(node.Backend.BridgeRelativePath) == "" {
			return fmt.Errorf("bridge command missing script path: %s", commandPathKey(path[1:]))
		}
	default:
		return fmt.Errorf("leaf command missing backend: %s", commandPathKey(path[1:]))
	}
	return nil
}

func commandPathKey(path []*commandSpec) string {
	names := make([]string, 0, len(path))
	for _, node := range path {
		if node.Name == "acpctl" {
			continue
		}
		names = append(names, node.Name)
	}
	return strings.Join(names, " ")
}

func commandPathLabel(path []*commandSpec) string {
	if len(path) == 0 {
		return "acpctl"
	}
	names := []string{"acpctl"}
	for _, node := range path {
		if node.Name == "acpctl" {
			continue
		}
		names = append(names, node.Name)
	}
	return strings.Join(names, " ")
}

func findCommand(path []string) (*commandSpec, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil, err
	}
	current := spec.Root
	for index, part := range path {
		next := findChildCommand(current, part)
		if next == nil {
			if index == 0 {
				return nil, &commandLookupError{Kind: "root", Name: part}
			}
			return nil, &commandLookupError{Kind: "subcommand", Path: commandPathKey(pathToSpecs(path[:index])), Name: part}
		}
		current = next
	}
	return current, nil
}

func findCommandPath(path []string) ([]*commandSpec, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil, err
	}
	nodes := []*commandSpec{spec.Root}
	current := spec.Root
	for index, part := range path {
		next := findChildCommand(current, part)
		if next == nil {
			if index == 0 {
				return nil, &commandLookupError{Kind: "root", Name: part}
			}
			return nil, &commandLookupError{Kind: "subcommand", Path: commandPathKey(pathToSpecs(path[:index])), Name: part}
		}
		nodes = append(nodes, next)
		current = next
	}
	return nodes, nil
}

func pathToSpecs(path []string) []*commandSpec {
	nodes := make([]*commandSpec, 0, len(path))
	for _, name := range path {
		nodes = append(nodes, &commandSpec{Name: name})
	}
	return nodes
}

func findChildCommand(node *commandSpec, name string) *commandSpec {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}
