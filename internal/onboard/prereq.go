// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Load onboarding prerequisites from canonical configuration sources before
//	workflow execution begins.
//
// Responsibilities:
//   - Validate repo-root and demo/.env availability when gateway auth is required.
//   - Resolve required gateway secrets through internal/config.
//   - Keep prerequisite lookups out of the coordinator.
//
// Scope:
//   - Onboarding prerequisite resolution only.
//
// Usage:
//   - Called by Run before key generation or verification.
//
// Invariants/Assumptions:
//   - Process env remains the highest precedence source.
//   - Direct mode skips gateway-key prerequisites.
package onboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func loadPrerequisites(opts Options) (prerequisites, error) {
	if opts.Mode == "direct" {
		return prerequisites{}, nil
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return prerequisites{}, fmt.Errorf("repo root is required")
	}
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	if _, err := os.Stat(envPath); err != nil {
		return prerequisites{}, fmt.Errorf("missing %s. Run: make install-env", envPath)
	}
	masterKey := config.NewLoader().WithRepoRoot(opts.RepoRoot).Gateway(true).MasterKey
	if strings.TrimSpace(masterKey) == "" {
		return prerequisites{}, fmt.Errorf("LITELLM_MASTER_KEY is not set (%s)", envPath)
	}
	return prerequisites{MasterKey: masterKey}, nil
}
