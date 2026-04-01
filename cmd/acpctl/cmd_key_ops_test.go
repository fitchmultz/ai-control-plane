// cmd_key_ops_test.go - Tests for the key-generation command surface.
//
// Purpose:
//   - Verify workspace-scoped key-generation bindings and dry-run output.
//
// Responsibilities:
//   - Cover workspace-scoped alias/model planning from the tenant design.
//   - Cover CLI validation for partial tenant-scope flags.
//
// Scope:
//   - Command-layer key-generation behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stay dry-run only and do not require a live gateway.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunKeyGenDryRunAppliesTenantWorkspaceScope(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newTestFiles(t)

	code := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr,
			"key", "gen", "svc-claims",
			"--organization", "falcon-insurance",
			"--workspace", "claims-adjuster",
			"--budget", "10.00",
			"--dry-run",
		)
	})
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("runTestCommand(key gen dry-run) exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}
	output := readFile(t, stdout)
	for _, snippet := range []string{
		"Alias: falcon-insurance--claims-adjuster--svc-claims__cc-1100",
		"Tenant Scope: falcon-insurance/claims-adjuster",
		"Role: developer",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected %q in output, got %s", snippet, output)
		}
	}
}

func TestRunKeyGenRejectsPartialTenantScopeFlags(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newTestFiles(t)

	code := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr,
			"key", "gen", "svc-claims",
			"--organization", "falcon-insurance",
			"--dry-run",
		)
	})
	if code != exitcodes.ACPExitUsage {
		t.Fatalf("runTestCommand(key gen partial scope) exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "--organization and --workspace must be provided together") {
		t.Fatalf("stderr = %q", got)
	}
}
