// role_contract.go - Repository-backed RBAC contract helpers for key workflows.
//
// Purpose:
//   - Load the tracked RBAC contract and approved online model aliases for key
//     planning and lifecycle workflows.
//
// Responsibilities:
//   - Resolve the canonical repository root through internal/config.
//   - Load demo/config/roles.yaml and demo/config/model_catalog.yaml.
//   - Return deterministic role and model inputs for key-domain workflows.
//
// Scope:
//   - Repository-backed helper loading only.
//
// Usage:
//   - Used by validator, request-planning, and lifecycle helpers.
//
// Invariants/Assumptions:
//   - Approved gateway model aliases come from the tracked online model catalog.
//   - The repo root is resolved through internal/config rather than direct env access.
package keygen

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/rbac"
)

func loadTrackedRoleContract() (rbac.Config, []string, error) {
	repoRoot, err := config.NewLoader().RequireRepoRoot(context.Background())
	if err != nil {
		return rbac.Config{}, nil, fmt.Errorf("resolve repo root: %w", err)
	}

	cfg, err := rbac.LoadFile(repopath.DemoConfigPath(repoRoot, "roles.yaml"))
	if err != nil {
		return rbac.Config{}, nil, fmt.Errorf("load RBAC config: %w", err)
	}
	modelCatalog, err := catalog.LoadModelCatalog(repopath.DemoConfigPath(repoRoot, "model_catalog.yaml"))
	if err != nil {
		return rbac.Config{}, nil, fmt.Errorf("load model catalog: %w", err)
	}

	return cfg, modelCatalog.OnlineAliases(), nil
}
