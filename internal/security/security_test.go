// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify canonical supply-chain scanning covers repository deployment surfaces.
//
// Responsibilities:
//   - Cover Compose image digest enforcement.
//   - Cover Dockerfile base-image digest enforcement.
//   - Lock down policy-field and supported-surface edge cases.
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
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestValidateSupplyChainPolicyFlagsComposeImageDigestDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  gateway:\n    image: ghcr.io/example/gateway:latest\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, `demo/docker-compose.yml: service "gateway" image must be digest pinned`) {
		t.Fatalf("expected compose digest issue, got %v", issues)
	}
}

func TestValidateSupplyChainPolicyAllowsDigestPinnedComposeImages(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  gateway:\n    image: ghcr.io/example/gateway@sha256:abc123\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected digest-pinned compose image to pass, got %v", issues)
	}
}

func TestValidateSupplyChainPolicyFlagsDockerfileBaseImageDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM python:3.14-alpine\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "base image must be digest pinned") {
		t.Fatalf("expected Dockerfile digest issue, got %v", issues)
	}
}

func TestValidateSupplyChainPolicyFlagsNestedDockerfileCoverage(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM python:3.14-alpine\n")
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "librechat-hardened", "Dockerfile"), "FROM node:22-alpine\n")

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "demo/images/litellm-hardened/Dockerfile") {
		t.Fatalf("expected litellm hardened Dockerfile issue, got %v", issues)
	}
	if !strings.Contains(joined, "demo/images/librechat-hardened/Dockerfile") {
		t.Fatalf("expected librechat hardened Dockerfile issue, got %v", issues)
	}
}

func TestValidateSupplyChainPolicyFlagsMissingPolicyFields(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"","severity_policy":{"fail_on":[]}}`)

	issues, err := ValidateSupplyChainPolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateSupplyChainPolicy returned error: %v", err)
	}
	if len(issues) != 1 || issues[0] != "demo/config/supply_chain_vulnerability_policy.json: missing required policy fields" {
		t.Fatalf("expected policy field issue, got %v", issues)
	}
}

func writeSecurityFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Clean(path), content)
}
