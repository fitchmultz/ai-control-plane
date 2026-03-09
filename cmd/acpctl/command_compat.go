// command_compat.go - Shared command tree path lookup helpers.
//
// Purpose:
//   - Provide shared command tree path lookup helpers for help rendering.
//
// Responsibilities:
//   - Resolve typed command paths from the canonical command tree.
//   - Surface stable lookup errors for help and adapter flows.
//
// Scope:
//   - Command-tree path lookup only.
//
// Usage:
//   - Used by help rendering and command path adapters in this package.
//
// Invariants/Assumptions:
//   - These helpers reflect the typed command tree and do not define metadata.
package main

func findCommandPath(parts []string) ([]*commandSpec, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return nil, err
	}
	path := []*commandSpec{spec.Root}
	current := spec.Root
	for _, part := range parts {
		next := findChildCommand(current, part)
		if next == nil {
			return nil, &commandLookupError{Kind: "subcommand", Path: commandPathLabel(path[1:]), Name: part}
		}
		path = append(path, next)
		current = next
	}
	return path, nil
}
