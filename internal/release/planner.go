// planner.go - Release bundle planning logic
//
// Purpose: Plan release bundle structure and content
//
// Responsibilities:
//   - Define canonical deployment asset list
//   - Plan bundle file structure
//   - Validate required files exist
//
// Non-scope:
//   - Does not copy files or create bundles
//   - Does not compute checksums
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package release

import (
	"fmt"
	"os"
	"path/filepath"
)

// Canonical deployment asset list (single source of truth)
var CanonicalPaths = []string{
	"Makefile",
	"README.md",
	"demo/docker-compose.yml",
	"demo/docker-compose.offline.yml",
	"demo/docker-compose.tls.yml",
	"demo/config/litellm.yaml",
	"demo/config/litellm-offline.yaml",
	"demo/config/detection_rules.yaml",
	"docs/deployment/PRODUCTION_HANDOFF_RUNBOOK.md",
	"docs/DEPLOYMENT.md",
}

// Plan represents the planned bundle structure
type Plan struct {
	Version      string
	RepoRoot     string
	OutputDir    string
	BundlePath   string
	PayloadFiles []string
	StageDir     string
	PayloadDir   string
}

// CreatePlan creates a release bundle plan
func CreatePlan(config *Config, repoRoot string) (*Plan, error) {
	// Ensure output directory is absolute
	outputDir := config.OutputDir
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(repoRoot, outputDir)
	}

	bundleName := fmt.Sprintf("ai-control-plane-deploy-%s.tar.gz", config.Version)
	bundlePath := filepath.Join(outputDir, bundleName)

	plan := &Plan{
		Version:    config.Version,
		RepoRoot:   repoRoot,
		OutputDir:  outputDir,
		BundlePath: bundlePath,
	}

	return plan, nil
}

// ValidateSourceFiles checks that all canonical files exist
func ValidateSourceFiles(repoRoot string, verbose bool) ([]string, error) {
	var missing []string
	var found []string

	for _, relPath := range CanonicalPaths {
		src := filepath.Join(repoRoot, relPath)
		if _, err := os.Stat(src); err != nil {
			missing = append(missing, relPath)
		} else {
			found = append(found, relPath)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("required bundle files missing: %v", missing)
	}

	return found, nil
}

// GetBundleName generates the bundle filename from version
func GetBundleName(version string) string {
	return fmt.Sprintf("ai-control-plane-deploy-%s.tar.gz", version)
}
