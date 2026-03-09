// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement repository configuration diagnostics as a focused module.
//
// Responsibilities:
//   - Verify required tracked configuration files exist.
//   - Warn when demo/.env is missing.
//
// Scope:
//   - Repository configuration diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package doctor

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type configValidCheck struct{ noFixCheck }

func (c configValidCheck) ID() string { return "config_valid" }

func (c configValidCheck) Run(ctx context.Context, opts Options) CheckResult {
	requiredFiles := []string{"demo/docker-compose.yml", "demo/config/litellm.yaml"}
	missingFiles := make([]string, 0, len(requiredFiles))
	for _, relPath := range requiredFiles {
		if _, err := os.Stat(filepath.Join(opts.RepoRoot, relPath)); err != nil {
			missingFiles = append(missingFiles, relPath)
		}
	}
	if len(missingFiles) > 0 {
		return withCheckDetails(
			newCheckResult(c.ID(), "Config Valid", status.HealthLevelUnhealthy, SeverityPrereq, "Required deployment configuration files are missing"),
			status.ComponentDetails{MissingFiles: missingFiles},
			"Ensure repository is complete",
			"Run: make install",
		)
	}
	envStatus, err := config.NewLoader().WithRepoRoot(opts.RepoRoot).RepoEnvStatus(ctx)
	if err != nil || !envStatus.Exists {
		return withCheckDetails(
			newCheckResult(c.ID(), "Config Valid", status.HealthLevelWarning, SeverityPrereq, "Environment file demo/.env is missing"),
			status.ComponentDetails{},
			"Run: make install-env",
			"Populate required environment variables in demo/.env",
		)
	}
	return newCheckResult(c.ID(), "Config Valid", status.HealthLevelHealthy, SeverityDomain, "Deployment configuration files are present")
}
