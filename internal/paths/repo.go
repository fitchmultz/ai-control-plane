// Package paths provides canonical repository-relative path helpers.
//
// Purpose:
//   - Centralize repeated ACP repository path construction.
//
// Responsibilities:
//   - Build canonical demo/config/logs/backups/env paths from a repo root.
//   - Resolve user-provided relative paths against the repository root.
//   - Keep repository layout assumptions out of command handlers.
//
// Scope:
//   - Repository-relative path helpers only.
//
// Usage:
//   - Used by config and command packages that need canonical repo paths.
//
// Invariants/Assumptions:
//   - Absolute paths are preserved and cleaned.
//   - Relative paths are always resolved from the supplied repo root.
package paths

import (
	"path/filepath"
	"strings"
)

// FromRepoRoot joins path parts onto the repository root.
func FromRepoRoot(repoRoot string, parts ...string) string {
	segments := []string{strings.TrimSpace(repoRoot)}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		segments = append(segments, filepath.FromSlash(trimmed))
	}
	return filepath.Join(segments...)
}

// ResolveRepoPath resolves a potentially relative path from the repository root.
func ResolveRepoPath(repoRoot string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Join(strings.TrimSpace(repoRoot), filepath.Clean(trimmed))
}

// DemoPath returns a path under demo/.
func DemoPath(repoRoot string, parts ...string) string {
	return FromRepoRoot(repoRoot, append([]string{"demo"}, parts...)...)
}

// DemoConfigPath returns a path under demo/config.
func DemoConfigPath(repoRoot string, parts ...string) string {
	return FromRepoRoot(repoRoot, append([]string{"demo", "config"}, parts...)...)
}

// DemoLogsPath returns a path under demo/logs.
func DemoLogsPath(repoRoot string, parts ...string) string {
	return FromRepoRoot(repoRoot, append([]string{"demo", "logs"}, parts...)...)
}

// DemoBackupsPath returns a path under demo/backups.
func DemoBackupsPath(repoRoot string, parts ...string) string {
	return FromRepoRoot(repoRoot, append([]string{"demo", "backups"}, parts...)...)
}

// DemoEnvPath returns the canonical demo/.env path.
func DemoEnvPath(repoRoot string) string {
	return DemoPath(repoRoot, ".env")
}

// ReleaseBundlesPath returns the canonical release bundle directory.
func ReleaseBundlesPath(repoRoot string) string {
	return DemoLogsPath(repoRoot, "release-bundles")
}
