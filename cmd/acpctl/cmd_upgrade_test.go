// cmd_upgrade_test.go - Tests for the typed upgrade command surface.
//
// Purpose:
//   - Verify upgrade commands honor the typed upgrade framework guardrails.
//
// Responsibilities:
//   - Ensure unsupported upgrade paths fail as domain errors.
//
// Scope:
//   - Upgrade command behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - The current framework release ships without explicit in-place edges.
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunUpgradePlanRejectsUnsupportedPath(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "0.1.0\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "upgrade", "plan", "--from", "0.0.9")
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "unsupported upgrade path") {
		t.Fatalf("expected unsupported path error, got %s", got)
	}
}

func TestRunUpgradeExecuteRejectsUnsupportedPathWithDomainExit(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "0.1.0\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "upgrade", "execute", "--from", "0.0.9")
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "unsupported upgrade path") {
		t.Fatalf("expected unsupported path error, got %s", got)
	}
}
