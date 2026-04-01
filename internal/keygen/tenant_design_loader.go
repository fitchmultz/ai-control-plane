// Package keygen provides typed virtual-key lifecycle workflows.
//
// Purpose:
//   - Centralize tracked tenant-design loading for key planning and lifecycle
//     enforcement.
//
// Responsibilities:
//   - Resolve the repository root for tenant-aware workflows.
//   - Load the configured tenant design file from the canonical repo path.
//   - Keep tenant-design path resolution consistent across key helpers.
//
// Scope:
//   - Tenant design loading only.
//
// Usage:
//   - Used by tenant-scoped request planning and lifecycle scope helpers.
//
// Invariants/Assumptions:
//   - Empty tenant config path falls back to the canonical tracked design file.
//   - Callers provide context when repo-root resolution may depend on config.
package keygen

import (
	"context"
	"fmt"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
)

func resolveKeyRepoRoot(ctx context.Context, repoRoot string) (string, error) {
	resolvedRepoRoot, err := config.NewLoader().WithRepoRoot(repoRoot).RequireRepoRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	return resolvedRepoRoot, nil
}

func loadTenantDesign(ctx context.Context, repoRoot string, tenantConfigPath string) (tenant.Design, error) {
	resolvedRepoRoot, err := resolveKeyRepoRoot(ctx, repoRoot)
	if err != nil {
		return tenant.Design{}, err
	}
	designPath := repopath.ResolveRepoPath(resolvedRepoRoot, tenant.DefaultDesignPath)
	if trimmed := strings.TrimSpace(tenantConfigPath); trimmed != "" {
		designPath = repopath.ResolveRepoPath(resolvedRepoRoot, trimmed)
	}
	return tenant.LoadFile(designPath)
}
