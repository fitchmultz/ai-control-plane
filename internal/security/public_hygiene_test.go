// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify public-hygiene local-only path detection and violation shaping.
//
// Responsibilities:
//   - Cover the local-only tracked-path allowlist.
//   - Verify marker files are exempted from violations.
//   - Lock down deterministic sorted output for CI-facing checks.
//
// Scope:
//   - Unit tests for public-hygiene validation only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Local-only path rules are repository-relative.
//   - Marker files remain allowed even under local-only prefixes.
package security

import (
	"reflect"
	"testing"
)

func TestValidatePublicHygiene_SortsViolationsAndSkipsMarkerFiles(t *testing.T) {
	t.Parallel()

	got := ValidatePublicHygiene([]string{
		"docs/presentation/slides-external/demo.png",
		"demo/logs/.gitkeep",
		"demo/.env",
		".scratchpad.md",
		"demo/backups/.gitignore",
		".env",
		"docs/guide.md",
	})

	want := []string{
		".env",
		".scratchpad.md",
		"demo/.env",
		"docs/presentation/slides-external/demo.png",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidatePublicHygiene() = %v, want %v", got, want)
	}
}

func TestIsLocalOnlyTrackedPath_CoversCanonicalPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: "demo/.env", want: true},
		{path: "demo/tenant/.env", want: true},
		{path: "demo/logs/runtime.log", want: true},
		{path: "demo/backups/archive.tar.gz", want: true},
		{path: "handoff-packet/report.md", want: true},
		{path: ".ralph/state.json", want: true},
		{path: "docs/presentation/slides-internal/notes.md", want: true},
		{path: "docs/presentation/slides-external/slide-1.png", want: true},
		{path: "docs/presentation/slides-external/slide-1.pdf", want: false},
		{path: ".scratchpad.md", want: true},
		{path: "docs/guide.md", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := IsLocalOnlyTrackedPath(tt.path); got != tt.want {
				t.Fatalf("IsLocalOnlyTrackedPath(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}
