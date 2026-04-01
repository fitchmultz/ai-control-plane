// cmd_key_lifecycle_test.go - Tests for key lifecycle command bindings.
//
// Purpose:
//   - Verify tenant-scope CLI validation for key lifecycle commands.
//
// Responsibilities:
//   - Cover partial tenant-scope flag rejection before runtime execution.
//   - Keep lifecycle command contracts aligned with the tracked tenant design.
//
// Scope:
//   - Command-layer validation only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stay parse/bind only and do not require a live gateway or database.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestKeyLifecycleCommandsRejectWorkspaceWithoutOrganization(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)

	tests := []struct {
		name   string
		args   []string
		stderr string
	}{
		{
			name:   "list",
			args:   []string{"key", "list", "--workspace", "claims-adjuster"},
			stderr: "--organization is required when --workspace is set",
		},
		{
			name:   "inspect",
			args:   []string{"key", "inspect", "demo", "--workspace", "claims-adjuster"},
			stderr: "--organization is required when --workspace is set",
		},
		{
			name:   "rotate",
			args:   []string{"key", "rotate", "demo", "--workspace", "claims-adjuster", "--dry-run"},
			stderr: "--organization is required when --workspace is set",
		},
		{
			name:   "revoke",
			args:   []string{"key", "revoke", "demo", "--workspace", "claims-adjuster"},
			stderr: "--organization is required when --workspace is set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := newTestFiles(t)
			code := withRepoRoot(t, repoRoot, func() int {
				return runTestCommand(t, context.Background(), stdout, stderr, tt.args...)
			})
			if code != exitcodes.ACPExitUsage {
				t.Fatalf("runTestCommand(%s) exit = %d stdout=%s stderr=%s", tt.name, code, readFile(t, stdout), readFile(t, stderr))
			}
			if got := readFile(t, stderr); !strings.Contains(got, tt.stderr) {
				t.Fatalf("stderr = %q", got)
			}
		})
	}
}
