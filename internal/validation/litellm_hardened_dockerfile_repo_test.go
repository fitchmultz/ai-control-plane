// litellm_hardened_dockerfile_repo_test.go - Hardened LiteLLM Dockerfile regression tests.
//
// Purpose:
//   - Guard the tracked hardened LiteLLM image contract against regressions that
//   - break local non-root startup.
//
// Responsibilities:
//   - Verify the Dockerfile keeps Prisma's required shared library available.
//   - Verify the Dockerfile preserves writable migration/cache path overrides.
//   - Verify the Dockerfile patches Prisma's generated client cache paths away
//   - from the root-owned default.
//
// Scope:
//   - Repository-tracked Dockerfile contract assertions only.
//
// Usage:
//   - Run via `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - The hardened image continues to run as a non-root LiteLLM user.
//   - Prisma startup must not depend on traversing `/root/.cache/...`.
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrackedLiteLLMHardenedDockerfileKeepsRuntimeStartupContract(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	dockerfilePath := filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile")
	data, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", dockerfilePath, err)
	}
	content := string(data)

	for _, needle := range []string{
		"apk add --no-cache libatomic",
		"LITELLM_MIGRATION_DIR=/app/.local/share/litellm_proxy_extras",
		"text.replace('/root/.cache/prisma-python', '/app/.cache/prisma-python')",
		"text.replace('/node_modules/prisma/query-engine-', '/node_modules/@prisma/engines/query-engine-')",
		"/usr/lib/python3.13/site-packages/litellm_proxy_extras/schema.prisma",
		"dedupe it so Prisma validation and the",
		"prisma validate >/dev/null",
		"-name '*_baseline_diff' -exec rm -rf {} +",
		"exec litellm \\\"$@\\\"",
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("tracked hardened Dockerfile missing %q", needle)
		}
	}
}
