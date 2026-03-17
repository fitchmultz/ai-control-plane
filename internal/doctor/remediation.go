// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Provide bounded remediation helpers for safe runtime recovery.
//
// Responsibilities:
//   - Start minimal ACP runtime services needed for common doctor fixes.
//   - Preserve compose project scoping and explicit env-file usage.
//
// Scope:
//   - Doctor auto-remediation helpers only.
//
// Usage:
//   - Used by fix-capable doctor checks.
//
// Invariants/Assumptions:
//   - Remediation is limited to safe compose service start actions.
package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

var runComposeUp = func(ctx context.Context, repoRoot string, services ...string) error {
	compose, err := docker.NewACPCompose(repoRoot, nil)
	if err != nil {
		return fmt.Errorf("docker compose unavailable: %w", err)
	}
	upCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	envFile := filepath.Join(repoRoot, "demo", ".env")
	if err := compose.UpServices(upCtx, envFile, true, services); err != nil {
		return fmt.Errorf("docker compose up failed: %s", sanitizeOutput(err.Error()))
	}
	return nil
}

func recoverGatewayRuntime(ctx context.Context, opts Options) (bool, string, error) {
	component, ok := runtimeComponent(opts, "gateway")
	if !ok || component.Level == status.HealthLevelHealthy || !component.Details.MasterKeyConfigured {
		return false, "", nil
	}

	services := []string{"litellm"}
	if dbComponent, ok := runtimeComponent(opts, "database"); ok && strings.EqualFold(dbComponent.Details.Mode, "embedded") {
		services = append([]string{"postgres"}, services...)
	}

	if err := runComposeUp(ctx, opts.RepoRoot, services...); err != nil {
		return false, "", err
	}
	return true, "Started ACP runtime services for gateway recovery", nil
}

func recoverEmbeddedDatabase(ctx context.Context, opts Options) (bool, string, error) {
	component, ok := runtimeComponent(opts, "database")
	if !ok || component.Level == status.HealthLevelHealthy || !strings.EqualFold(component.Details.Mode, "embedded") {
		return false, "", nil
	}
	if err := runComposeUp(ctx, opts.RepoRoot, "postgres"); err != nil {
		return false, "", err
	}
	return true, "Started embedded PostgreSQL for database recovery", nil
}
