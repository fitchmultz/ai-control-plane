// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Add focused unit coverage for helper branches not exercised by the main
//     repository-policy tests.
//
// Responsibilities:
//   - Cover incubating Helm image validation helpers directly.
//   - Cover content-scanner skip logic and strict secrets-policy validation.
//
// Scope:
//   - Helper-level security validation coverage only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixtures and deterministic direct helper calls.
package security

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"gopkg.in/yaml.v3"
)

func TestValidateHelmImagesAndHelpers(t *testing.T) {
	repoRoot := t.TempDir()
	target := policy.SurfaceTarget{
		Kind: policy.SurfaceHelmValues,
		Path: "deploy/incubating/helm/ai-control-plane/values.yaml",
	}

	writeSecurityFixtureFile(t, filepath.Join(repoRoot, target.Path), ""+
		"chargeback:\n"+
		"  image:\n"+
		"    repository: ghcr.io/example/acpctl\n"+
		"localTool:\n"+
		"  image:\n"+
		"    repository: ai-control-plane/acpctl\n"+
		"    tag: local\n")

	issues, err := validateHelmImages(repoRoot, target)
	if err != nil {
		t.Fatalf("validateHelmImages() error = %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "must declare a non-empty image digest") {
		t.Fatalf("unexpected issues: %v", issues)
	}

	if !isLocalImageOverride(mustYAMLMappingNode(t, "repository: ai-control-plane/acpctl\ntag: local\n"), "ai-control-plane/acpctl") {
		t.Fatal("expected local image override to be allowed")
	}
	if !isDigestPinnedImage("${IMAGE:-ghcr.io/example/app@sha256:abc123}") {
		t.Fatal("expected defaulted env digest form to be treated as pinned")
	}
	if isDigestPinnedImage("ghcr.io/example/app:latest") {
		t.Fatal("expected tag-only image to be treated as unpinned")
	}
}

func mustYAMLMappingNode(t *testing.T, content string) *yaml.Node {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if len(root.Content) == 0 {
		t.Fatal("expected mapping node content")
	}
	return root.Content[0]
}

func TestShouldScanContentAndValidateSecretsPolicyBranches(t *testing.T) {
	if shouldScanContent("demo/archive.tgz", []byte("content")) {
		t.Fatal("expected archive extension to skip content scanning")
	}
	if shouldScanContent("demo/image.png", []byte("content")) {
		t.Fatal("expected binary-like extension to skip content scanning")
	}
	if shouldScanContent("demo/config.txt", []byte{0xff, 0xfe}) {
		t.Fatal("expected invalid UTF-8 to skip content scanning")
	}
	if !shouldScanContent("demo/config.txt", []byte("safe text")) {
		t.Fatal("expected UTF-8 text file to be scanned")
	}

	err := validateSecretsPolicy(SecretsPolicy{
		SchemaVersion: "1.0.0",
		PolicyID:      "missing-pattern",
		PathRules: []SecretPathRule{
			{ID: "tracked-env-file", Message: "tracked environment file"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "must declare at least one pattern") {
		t.Fatalf("expected missing pattern error, got %v", err)
	}

	err = validateSecretsPolicy(SecretsPolicy{
		SchemaVersion: "1.0.0",
		PolicyID:      "missing-placeholder-allowance",
		ContentRules: []SecretContentRule{
			{ID: "openai-style-key", Message: "OpenAI-style API key", Pattern: "\\bsk-[A-Za-z0-9_-]{20,}\\b"},
		},
		PlaceholderExemptions: []SecretPlaceholderExemption{
			{ID: "docs", PathPatterns: []string{"docs/**"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "must allow at least one placeholder substring or empty assignment") {
		t.Fatalf("expected placeholder allowance error, got %v", err)
	}
}
