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

import (
	"fmt"
	"strings"
)

func Trim(value string) string {
	return strings.TrimSpace(value)
}

func LowerTrim(value string) string {
	return strings.ToLower(Trim(value))
}

func IsBlank(value string) bool {
	return Trim(value) == ""
}

func DefaultIfBlank(value string, fallback string) string {
	if IsBlank(value) {
		return fallback
	}
	return value
}

func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if !IsBlank(value) {
			return Trim(value)
		}
	}
	return ""
}

func EqualFoldTrimmed(left string, right string) bool {
	return strings.EqualFold(Trim(left), Trim(right))
}

func RequireNonBlank(name string, value string) error {
	if IsBlank(value) {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}
