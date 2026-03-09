// Package exitcodes wires generation for shell exit-code constants.
//
// Purpose:
//   - Keep shell wrappers synchronized with the canonical Go exit-code contract.
//
// Responsibilities:
//   - Provide the `go generate` entrypoint for exit-code shell output.
//
// Scope:
//   - Generation wiring only.
//
// Usage:
//   - Run `go generate ./internal/exitcodes`.
//
// Invariants/Assumptions:
//   - `internal/exitcodes/exitcodes.go` remains the source of truth.
package exitcodes

//go:generate go run ./gen
