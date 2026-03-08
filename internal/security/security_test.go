// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify canonical supply-chain scanning covers non-compose deployment surfaces.
//
// Responsibilities:
//   - Cover Helm image digest enforcement.
//   - Cover Dockerfile base-image digest enforcement.
//
// Scope:
//   - Unit tests for internal security validators only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixture repositories.
//   - Findings remain stable and human-readable.
package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSupplyChainPolicyFlagsHelmDigestDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "values.yaml"), "profile: production\ndemo:\n  enabled: false\nlitellm:\n  image:\n    repository: ghcr.io/example/app\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "must declare a non-empty image digest") {
		t.Fatalf("expected helm digest issue, got %v", issues)
	}
}

func TestValidateSupplyChainPolicyFlagsDockerfileBaseImageDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "mock_upstream", "Dockerfile"), "FROM python:3.14-alpine\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "base image must be digest pinned") {
		t.Fatalf("expected Dockerfile digest issue, got %v", issues)
	}
}

func writeSecurityFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
