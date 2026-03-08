// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Verify canonical deployment surfaces resolve nested repository targets.
//
// Responsibilities:
//   - Prove recursive surface expansion reaches nested deployment files.
//   - Lock down surface kinds for key canonical targets.
//   - Prevent regressions back to shallow glob behavior.
//
// Scope:
//   - Unit tests for deployment-surface policy expansion only.
//
// Usage:
//   - Run with `go test ./internal/policy`.
//
// Invariants/Assumptions:
//   - Tests use temporary repositories.
//   - Surface resolution stays deterministic for equivalent fixtures.
package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandDeploymentSurfacesIncludesNestedCanonicalTargets(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "presidio", "recognizers", "custom_recognizers.yaml"), "recognizers: []\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.production.yaml"), "receivers: {}\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM cgr.dev/chainguard/python@sha256:abc\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "librechat-hardened", "Dockerfile"), "FROM ghcr.io/example/librechat@sha256:def\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "templates", "deployment-litellm.yaml"), "apiVersion: apps/v1\nkind: Deployment\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "---\n- hosts: gateway\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "terraform", "examples", "aws-complete", "main.tf"), "terraform {}\n")

	targets, err := ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ExpandDeploymentSurfaces returned error: %v", err)
	}
	targetKinds := make(map[string]SurfaceKind, len(targets))
	for _, target := range targets {
		targetKinds[target.Path] = target.Kind
	}
	assertSurfaceKind(t, targetKinds, "demo/config/presidio/recognizers/custom_recognizers.yaml", SurfaceYAML)
	assertSurfaceKind(t, targetKinds, "demo/config/otel-collector/config.production.yaml", SurfaceYAML)
	assertSurfaceKind(t, targetKinds, "demo/images/litellm-hardened/Dockerfile", SurfaceDockerfile)
	assertSurfaceKind(t, targetKinds, "demo/images/librechat-hardened/Dockerfile", SurfaceDockerfile)
	assertSurfaceKind(t, targetKinds, "deploy/helm/ai-control-plane/templates/deployment-litellm.yaml", SurfaceHelmTpl)
	assertSurfaceKind(t, targetKinds, "deploy/ansible/playbooks/gateway_host.yml", SurfaceAnsibleYML)
	assertSurfaceKind(t, targetKinds, "deploy/terraform/examples/aws-complete/main.tf", SurfaceTerraform)
}

func assertSurfaceKind(t *testing.T, got map[string]SurfaceKind, path string, want SurfaceKind) {
	t.Helper()
	kind, ok := got[path]
	if !ok {
		t.Fatalf("expected target %q to be included, got %v", path, got)
	}
	if kind != want {
		t.Fatalf("expected target %q kind %q, got %q", path, want, kind)
	}
}

func writePolicyFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
