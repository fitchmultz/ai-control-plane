// cmd_cert_test.go - Tests for the typed certificate command surface.
//
// Purpose:
//   - Verify certificate commands are registered and enforce basic CLI validation.
//
// Responsibilities:
//   - Ensure the `cert` root command exists.
//   - Ensure invalid threshold parsing returns usage errors.
//
// Scope:
//   - Certificate command behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Parsing failures should stop before any Docker-dependent runtime work.
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestCertCommandRegistered(t *testing.T) {
	t.Parallel()

	spec, err := loadCommandSpec()
	if err != nil {
		t.Fatalf("loadCommandSpec: %v", err)
	}
	if findChildCommand(spec.Root, "cert") == nil {
		t.Fatalf("expected cert command to be registered")
	}
}

func TestRunCertCheckRejectsInvalidThreshold(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "0.1.0\n")
	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "cert", "check", "--threshold-days", "nope")
	})
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "invalid threshold-days") {
		t.Fatalf("expected invalid threshold-days error, got %s", got)
	}
}

func TestRunCertRenewRejectsInvalidThreshold(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "0.1.0\n")
	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "cert", "renew", "--threshold-days", "nope")
	})
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "invalid threshold-days") {
		t.Fatalf("expected invalid threshold-days error, got %s", got)
	}
}
