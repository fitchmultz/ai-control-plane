// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Enforce tracked-file hygiene for local-only repository artifacts.
//
// Responsibilities:
//   - Identify tracked files that must remain local-only.
//   - Centralize the repository's local-artifact allowlist.
//   - Return deterministic violation paths for CLI and CI reporting.
//
// Scope:
//   - Tracked-path policy checks only.
//
// Usage:
//   - Called by `acpctl validate public-hygiene`.
//
// Invariants/Assumptions:
//   - `.gitkeep` and `.gitignore` marker files remain allowed.
//   - Violations are repository-relative and sorted.
package security

import (
	"strings"

	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
)

func ValidatePublicHygiene(trackedFiles []string) []string {
	violations := validationissues.NewIssues(len(trackedFiles))
	for _, relPath := range trackedFiles {
		if !IsLocalOnlyTrackedPath(relPath) {
			continue
		}
		if strings.HasSuffix(relPath, "/.gitkeep") || strings.HasSuffix(relPath, "/.gitignore") {
			continue
		}
		violations.Add(relPath)
	}
	return violations.Sorted()
}

func IsLocalOnlyTrackedPath(relPath string) bool {
	switch {
	case relPath == ".env":
		return true
	case relPath == "demo/.env":
		return true
	case strings.HasPrefix(relPath, "demo/") && strings.HasSuffix(relPath, "/.env"):
		return true
	case strings.HasPrefix(relPath, "demo/logs/"):
		return true
	case strings.HasPrefix(relPath, "demo/backups/"):
		return true
	case strings.HasPrefix(relPath, "handoff-packet/"):
		return true
	case strings.HasPrefix(relPath, ".ralph/"):
		return true
	case strings.HasPrefix(relPath, "docs/presentation/slides-internal/"):
		return true
	case strings.HasPrefix(relPath, "docs/presentation/slides-external/") && strings.HasSuffix(relPath, ".png"):
		return true
	case relPath == ".scratchpad.md":
		return true
	default:
		return false
	}
}
