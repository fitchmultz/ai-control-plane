// cmd_string_defaults.go - Shared string fallback helpers for CLI adapters.
//
// Purpose:
//   - Provide tiny string fallback helpers used by multiple acpctl command
//     adapters.
//
// Responsibilities:
//   - Normalize empty command values to canonical defaults.
//
// Non-scope:
//   - Does not own command-specific validation.
//   - Does not read environment variables.
//
// Invariants/Assumptions:
//   - Blank and whitespace-only strings should resolve to the fallback.
//
// Scope:
//   - CLI adapter string helpers only.
//
// Usage:
//   - Used by command binders that need consistent fallback behavior.
package main

import "strings"

func stringDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
