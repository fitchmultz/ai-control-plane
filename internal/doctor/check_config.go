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
package doctor

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type configValidCheck struct{}

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
		return CheckResult{
			ID:       c.ID(),
			Name:     "Config Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "Required deployment configuration files are missing",
			Suggestions: []string{
				"Ensure repository is complete",
				"Run: make install",
			},
			Details: map[string]any{"missing_files": missingFiles},
		}
	}
	if _, err := os.Stat(filepath.Join(opts.RepoRoot, "demo", ".env")); err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Config Valid",
			Level:    status.HealthLevelWarning,
			Severity: SeverityPrereq,
			Message:  "Environment file demo/.env is missing",
			Suggestions: []string{
				"Run: make install-env",
				"Populate required environment variables in demo/.env",
			},
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Config Valid",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "Deployment configuration files are present",
	}
}

func (c configValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
