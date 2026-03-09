// cmd_bind_helpers.go - Shared command binder helpers.
//
// Purpose:
//   - Centralize small command-layer binder patterns reused across multiple
//     command specs.
//
// Responsibilities:
//   - Provide reusable no-option binders for native commands.
//   - Keep repeated binder boilerplate out of leaf command definitions.
//
// Scope:
//   - Command-layer binder helpers only.
//
// Usage:
//   - Used by command specs that do not need parsed options or need to return a
//     fixed typed payload.
//
// Invariants/Assumptions:
//   - Helpers remain behaviorally trivial and side-effect free.
package main

func bindNoOptions(_ commandBindContext, _ parsedCommandInput) (any, error) {
	return struct{}{}, nil
}

func bindStaticOptions[T any](value T) func(commandBindContext, parsedCommandInput) (any, error) {
	return func(_ commandBindContext, _ parsedCommandInput) (any, error) {
		return value, nil
	}
}
