// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement environment-variable diagnostics as a focused module.
//
// Responsibilities:
//   - Verify required env keys are set directly or via demo/.env.
//   - Safely create demo/.env from demo/.env.example when --fix is used.
//
// Scope:
//   - Environment configuration diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type envVarsSetCheck struct{}

func (c envVarsSetCheck) ID() string { return "env_vars_set" }

func (c envVarsSetCheck) Run(ctx context.Context, opts Options) CheckResult {
	requiredVars := []string{"LITELLM_MASTER_KEY", "LITELLM_SALT_KEY", "DATABASE_URL"}
	missing := []string{}
	found := []string{}
	loader := config.NewLoader()
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	for _, key := range requiredVars {
		if loader.String(key) == "" && loadEnvFromFile(envPath, key) == "" {
			missing = append(missing, key)
			continue
		}
		found = append(found, key)
	}
	if len(missing) > 0 {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Environment Variables Set",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  fmt.Sprintf("Missing required environment variables: %v", missing),
			Suggestions: []string{
				"Run: make install",
				"Or manually set: export LITELLM_MASTER_KEY=sk-...",
				"Copy .env.example to demo/.env and configure",
			},
			Details: status.ComponentDetails{
				MissingVars: missing,
			},
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Environment Variables Set",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("All required environment variables set (%d found)", len(found)),
		Details:  status.ComponentDetails{},
	}
}

func (c envVarsSetCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	examplePath := filepath.Join(opts.RepoRoot, "demo", ".env.example")
	if _, err := os.Stat(envPath); err == nil {
		return false, "", nil
	}
	if _, err := os.Stat(examplePath); err != nil {
		return false, "", nil
	}
	content, err := os.ReadFile(examplePath)
	if err != nil {
		return false, "", err
	}
	if err := os.WriteFile(envPath, content, 0o600); err != nil {
		return false, "", err
	}
	return true, "Created demo/.env from .env.example", nil
}
