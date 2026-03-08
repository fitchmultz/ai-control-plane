// check_helpers_test.go - Focused coverage for shared doctor helper functions.
//
// Purpose:
//   - Verify small shared helper utilities behave deterministically.
//
// Responsibilities:
//   - Cover first-non-empty-line and output sanitization helpers.
//
// Scope:
//   - Helper functions only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Sanitization must drop secret-bearing lines conservatively.
package doctor

import "testing"

func TestFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single line", input: "abc123", expected: "abc123"},
		{name: "first line selected", input: "id-one\nid-two", expected: "id-one"},
		{name: "skips blank lines", input: "\n\n container-id ", expected: "container-id"},
		{name: "all blank", input: "\n\t\n", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := firstNonEmptyLine(tt.input); got != tt.expected {
				t.Fatalf("firstNonEmptyLine(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal output\nsecret key=hidden\nmore output",
			expected: "normal output\nmore output",
		},
		{
			input:    "password=test123\ntoken=abc",
			expected: "",
		},
		{
			input:    "API_KEY=sk-xxx\nDATABASE_URL=postgres://user:pass@host",
			expected: "",
		},
		{
			input:    "regular log line\nanother line",
			expected: "regular log line\nanother line",
		},
	}

	for _, tt := range tests {
		if result := sanitizeOutput(tt.input); result != tt.expected {
			t.Fatalf("sanitizeOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
