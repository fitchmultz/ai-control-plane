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
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type envVarsSetCheck struct{}

func (c envVarsSetCheck) ID() string { return "env_vars_set" }

func (c envVarsSetCheck) Run(ctx context.Context, opts Options) CheckResult {
	requiredVars := []string{"LITELLM_MASTER_KEY", "LITELLM_SALT_KEY", "DATABASE_URL"}
	missing := []string{}
	found := []string{}
	for _, key := range requiredVars {
		if os.Getenv(key) == "" {
			envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
			if val := loadEnvFromFile(envPath, key); val == "" {
				missing = append(missing, key)
			} else {
				found = append(found, key)
			}
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
			Details: map[string]any{
				"missing_vars": missing,
				"found_vars":   len(found),
			},
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Environment Variables Set",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("All required environment variables set (%d found)", len(found)),
		Details:  map[string]any{"found_vars": len(found)},
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
