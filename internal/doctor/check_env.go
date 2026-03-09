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
	loader := config.NewLoader().WithRepoRoot(opts.RepoRoot)
	statusResult := loader.RequiredRuntimeEnv(requiredVars)
	if len(statusResult.Missing) > 0 {
		return withCheckDetails(
			newCheckResult(c.ID(), "Environment Variables Set", status.HealthLevelUnhealthy, SeverityPrereq, fmt.Sprintf("Missing required environment variables: %v", statusResult.Missing)),
			status.ComponentDetails{
				MissingVars: statusResult.Missing,
			},
			"Run: make install",
			"Or manually set: export LITELLM_MASTER_KEY=sk-...",
			"Copy .env.example to demo/.env and configure",
		)
	}
	return withCheckDetails(
		newCheckResult(c.ID(), "Environment Variables Set", status.HealthLevelHealthy, SeverityDomain, fmt.Sprintf("All required environment variables set (%d found)", len(statusResult.Found))),
		status.ComponentDetails{},
	)
}

func (c envVarsSetCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	envStatus, err := config.NewLoader().WithRepoRoot(opts.RepoRoot).RepoEnvStatus(ctx)
	if err != nil {
		return false, "", err
	}
	envPath := envStatus.Path
	examplePath := filepath.Join(opts.RepoRoot, "demo", ".env.example")
	if envStatus.Exists {
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
