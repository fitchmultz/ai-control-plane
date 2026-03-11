// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Add focused coverage for helper branches that underpin the host-first
//     validation contract.
//
// Responsibilities:
//   - Cover default-profile config validation behavior.
//   - Cover issue accumulator helpers and direct structural validators.
//
// Scope:
//   - Helper-level validation coverage only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixture repositories and deterministic helper calls.
package validation

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
)

func TestValidateDeploymentConfigDefaultsToDemoProfile(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")

	issues, err := ValidateDeploymentConfig(repoRoot, ConfigValidationOptions{})
	if err != nil {
		t.Fatalf("ValidateDeploymentConfig() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected demo/default validation to accept minimal surfaces, got %v", issues)
	}
}

func TestIssuesHelpersAndDirectStructureValidation(t *testing.T) {
	issues := NewIssues()
	issues.Add("")
	issues.Add("b")
	issues.Addf("%s", "a")
	issues.Extend([]string{"", "c"})
	if issues.Len() != 3 {
		t.Fatalf("Len() = %d", issues.Len())
	}
	if !slices.Equal(issues.Sorted(), []string{"a", "b", "c"}) {
		t.Fatalf("Sorted() = %v", issues.Sorted())
	}

	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "sample.json"), `{"ok":true}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: []\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "terraform", "example.tf"), "terraform {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "template.yaml"), "{{/* helper */}}\n{{ include \"acp.validate\" . }}\n")

	for _, target := range []policy.SurfaceTarget{
		{Kind: policy.SurfaceJSON, Path: "demo/config/sample.json"},
		{Kind: policy.SurfaceTerraform, Path: "deploy/incubating/terraform/example.tf"},
		{Kind: policy.SurfaceHelmTpl, Path: "deploy/incubating/helm/template.yaml"},
	} {
		targetIssues, err := validateStructureForTarget(repoRoot, target)
		if err != nil {
			t.Fatalf("validateStructureForTarget(%s) error = %v", target.Path, err)
		}
		if len(targetIssues) != 0 {
			t.Fatalf("expected no issues for %s, got %v", target.Path, targetIssues)
		}
	}

	if !isHelmSurface(policy.SurfaceHelmValues) || isHelmSurface(policy.SurfaceCompose) {
		t.Fatal("isHelmSurface() returned unexpected values")
	}
	if !isTemplateOnlyHelmFile("{{/* helper */}}\n{{ include \"acp.validate\" . }}\n") {
		t.Fatal("expected helper-only Helm template to be allowed")
	}
	if isTemplateOnlyHelmFile("apiVersion: v1\nkind: Service\n") {
		t.Fatal("expected concrete manifest to not be classified as helper-only")
	}

	composeIssues, err := validateComposeHealthchecksForTarget(repoRoot, policy.SurfaceTarget{
		Kind: policy.SurfaceCompose,
		Path: "demo/docker-compose.yml",
	})
	if err != nil {
		t.Fatalf("validateComposeHealthchecksForTarget() error = %v", err)
	}
	if len(composeIssues) != 1 || !strings.Contains(composeIssues[0], "compose file must define services") {
		t.Fatalf("unexpected compose issues: %v", composeIssues)
	}
}
