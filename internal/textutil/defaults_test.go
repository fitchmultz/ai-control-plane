// Package textutil provides tiny shared text normalization helpers.
//
// Purpose:
//   - Validate the package-level trim/default helpers used across ACP packages.
//
// Responsibilities:
//   - Lock in blank detection, lower-trim normalization, and required-value checks.
//
// Scope:
//   - Unit tests for generic text helpers only.
//
// Usage:
//   - Run via `go test ./internal/textutil`.
//
// Invariants/Assumptions:
//   - Helpers never collapse internal whitespace.
package textutil

import "testing"

func TestHelpers(t *testing.T) {
	t.Parallel()

	if got := Trim("  value  "); got != "value" {
		t.Fatalf("Trim() = %q, want value", got)
	}
	if got := LowerTrim("  VALUE  "); got != "value" {
		t.Fatalf("LowerTrim() = %q, want value", got)
	}
	if !IsBlank(" \n\t ") {
		t.Fatal("IsBlank() should treat whitespace-only input as blank")
	}
	if got := DefaultIfBlank(" ", "fallback"); got != "fallback" {
		t.Fatalf("DefaultIfBlank() = %q, want fallback", got)
	}
	if got := FirstNonBlank("", " ", " value "); got != "value" {
		t.Fatalf("FirstNonBlank() = %q, want value", got)
	}
	if !EqualFoldTrimmed(" HTTPS ", "https") {
		t.Fatal("EqualFoldTrimmed() should compare trimmed values")
	}
	if err := RequireNonBlank("customer", " "); err == nil {
		t.Fatal("RequireNonBlank() should reject blank input")
	}
}
