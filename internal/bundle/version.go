// version.go - Release bundle version source helpers.
//
// Purpose:
//   - Resolve and validate the tracked repository release version.
//
// Responsibilities:
//   - Read the default version from the root VERSION file.
//   - Validate explicit version inputs for bundle and readiness workflows.
//
// Scope:
//   - Release-version parsing and validation only.
//
// Usage:
//   - Used by release bundle and readiness evidence workflows.
//
// Invariants/Assumptions:
//   - VERSION is the primary tracked release-version source of truth.
//   - Missing or blank VERSION falls back to "dev" for local-only workflows.
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// ValidateVersion validates release-bundle and readiness version inputs.
func ValidateVersion(version string) error {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return fmt.Errorf("version is required")
	}
	if trimmed == "dev" {
		return nil
	}
	if !semverPattern.MatchString(trimmed) {
		return fmt.Errorf("version %q must be semantic versioning like 0.1.0 or 0.1.0-rc.1", trimmed)
	}
	return nil
}

// GetDefaultVersion returns the tracked repository version or "dev".
func GetDefaultVersion(repoRoot string) string {
	path := filepath.Join(repoRoot, "VERSION")
	data, err := os.ReadFile(path)
	if err != nil {
		return "dev"
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "dev"
	}
	return version
}
