// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Verify structural deployment and repository-policy validation behavior.
//
// Responsibilities:
//   - Cover compose healthcheck enforcement.
//   - Cover helm contract enforcement.
//   - Cover header and direct-env policy checks.
//
// Scope:
//   - Unit tests for validation package behavior only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixture repositories.
//   - Validation output remains deterministic for equivalent fixtures.
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateComposeHealthchecksFlagsMissingTest(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  app:\n    image: example/app:1@sha256:abc\n    healthcheck:\n      interval: 5s\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")

	issues, err := ValidateComposeHealthchecks(repoRoot)
	if err != nil {
		t.Fatalf("ValidateComposeHealthchecks returned error: %v", err)
	}
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), `service "app" healthcheck must define test`) {
		t.Fatalf("expected missing test issue, got %v", issues)
	}
}

func TestValidateDeploymentSurfacesFlagsHelmContractDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","severity_policy":{"fail_on":["CRITICAL"]}}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "values.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "examples", "values.demo.yaml"), "profile: demo\ndemo:\n  enabled: true\n")

	issues, err := ValidateDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDeploymentSurfaces returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "values.yaml: profile must be production") {
		t.Fatalf("expected helm profile drift issue, got %v", issues)
	}
}

func TestValidateGoHeadersFlagsMissingSections(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "sample.go"), "package sample\n")

	issues, err := ValidateGoHeaders(repoRoot)
	if err != nil {
		t.Fatalf("ValidateGoHeaders returned error: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "internal/sample.go") {
		t.Fatalf("expected missing header issue, got %v", issues)
	}
}

func TestValidateDirectEnvAccessFlagsForbiddenCalls(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "sample.go"), "package sample\nimport \"os\"\nfunc value() string { return os.Getenv(\"X\") }\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "config", "allowed.go"), "package config\nimport \"os\"\nfunc value() string { return os.Getenv(\"X\") }\n")

	issues, err := ValidateDirectEnvAccess(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDirectEnvAccess returned error: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "internal/sample.go") {
		t.Fatalf("expected one forbidden env-access issue, got %v", issues)
	}
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
