// Package textutil provides tiny shared text normalization helpers.
//
// Purpose:
//   - Centralize small string normalization rules reused across ACP packages.
//
// Responsibilities:
//   - Apply canonical fallback behavior for blank or whitespace-only strings.
//
// Scope:
//   - Tiny text helpers only.
//
// Usage:
//   - Used by CLI parsing, config loading, and typed input normalization.
//
// Invariants/Assumptions:
//   - Whitespace-only input is treated as blank.
//   - Non-blank values are returned unchanged.
package textutil

import "strings"

func DefaultIfBlank(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
