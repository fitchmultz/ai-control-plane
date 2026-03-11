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
	"reflect"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestExpandDeploymentSurfacesIncludesNestedCanonicalTargets(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "presidio", "recognizers", "custom_recognizers.yaml"), "recognizers: []\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.production.yaml"), "receivers: {}\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM cgr.dev/chainguard/python@sha256:abc\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "librechat-hardened", "Dockerfile"), "FROM ghcr.io/example/librechat@sha256:def\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "---\n- hosts: gateway\n")

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
	assertSurfaceKind(t, targetKinds, "deploy/ansible/playbooks/gateway_host.yml", SurfaceAnsibleYML)
}

func TestExpandDeploymentSurfacesIncludesExplicitRequiredTargetsWhenGlobMatchesNothing(t *testing.T) {
	repoRoot := t.TempDir()

	targets, err := ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ExpandDeploymentSurfaces returned error: %v", err)
	}

	targetKinds := make(map[string]SurfaceKind, len(targets))
	for _, target := range targets {
		targetKinds[target.Path] = target.Kind
	}
	assertSurfaceKind(t, targetKinds, "demo/docker-compose.yml", SurfaceCompose)
}

func TestExpandIncubatingDeploymentSurfacesSortsAndDedupesDeterministically(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "a.yaml"), "profile: demo\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "b.yaml"), "profile: demo\n")

	spec := SurfaceSpec{
		ID:    "dedupe",
		Kind:  SurfaceHelmValues,
		Paths: []string{"deploy/incubating/helm/ai-control-plane/examples/b.yaml"},
		Globs: []string{"deploy/incubating/helm/ai-control-plane/examples/**/*.yaml"},
	}
	got, err := expandSpec(repoRoot, spec)
	if err != nil {
		t.Fatalf("expandSpec returned error: %v", err)
	}
	want := []string{
		"deploy/incubating/helm/ai-control-plane/examples/a.yaml",
		"deploy/incubating/helm/ai-control-plane/examples/b.yaml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandSpec mismatch\nwant: %v\n got: %v", want, got)
	}
}

func TestExpandIncubatingDeploymentSurfacesIncludesMovedTracks(t *testing.T) {
	repoRoot := t.TempDir()
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "templates", "deployment-litellm.yaml"), "apiVersion: apps/v1\nkind: Deployment\n")
	writePolicyFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "terraform", "examples", "aws-complete", "main.tf"), "terraform {}\n")

	targets, err := ExpandIncubatingDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ExpandIncubatingDeploymentSurfaces returned error: %v", err)
	}
	targetKinds := make(map[string]SurfaceKind, len(targets))
	for _, target := range targets {
		targetKinds[target.Path] = target.Kind
	}
	assertSurfaceKind(t, targetKinds, "deploy/incubating/helm/ai-control-plane/templates/deployment-litellm.yaml", SurfaceHelmTpl)
	assertSurfaceKind(t, targetKinds, "deploy/incubating/terraform/examples/aws-complete/main.tf", SurfaceTerraform)
}

func TestSupportedComposeSurfacesStayInPolicy(t *testing.T) {
	repoRoot := repoRootForTrackedSupportPolicyTest(t)
	matrix, err := catalog.LoadSupportMatrix(filepath.Join(repoRoot, "docs", "support-matrix.yaml"))
	if err != nil {
		t.Fatalf("LoadSupportMatrix() error = %v", err)
	}

	targets, err := ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ExpandDeploymentSurfaces() error = %v", err)
	}

	policyCompose := make(map[string]struct{})
	for _, target := range targets {
		if target.Kind == SurfaceCompose {
			policyCompose[target.Path] = struct{}{}
		}
	}

	for _, surface := range matrix.SupportedSurfaces() {
		for _, path := range surface.Paths {
			if filepath.Ext(path) != ".yml" && filepath.Ext(path) != ".yaml" {
				continue
			}
			if filepath.Base(path) == "docker-compose.yml" || filepath.Base(path) == "docker-compose.offline.yml" || filepath.Base(path) == "docker-compose.tls.yml" || filepath.Base(path) == "docker-compose.ui.yml" || filepath.Base(path) == "docker-compose.dlp.yml" {
				if _, ok := policyCompose[path]; !ok {
					t.Fatalf("supported compose surface %q from %q missing from DeploymentSurfacePolicy", path, surface.ID)
				}
			}
		}
	}
}

func repoRootForTrackedSupportPolicyTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("could not locate repo root from %s", wd)
		}
		current = parent
	}
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
