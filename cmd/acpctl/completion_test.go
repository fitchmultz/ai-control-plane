// completion_test.go - User-facing completion command contract tests.
//
// Purpose:
//   - Verify the public completion command help and default behavior.
//
// Responsibilities:
//   - Prove help output advertises supported shells.
//   - Lock down the no-subcommand fallback contract.
//
// Scope:
//   - User-facing completion command behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - `acpctl completion` remains a help-rendering group command.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestCompletionCommandHelpListsSupportedShells(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := run(context.Background(), []string{"completion", "--help"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	content := readFile(t, stdout)
	for _, shell := range []string{"bash", "zsh", "fish"} {
		if !strings.Contains(content, shell) {
			t.Fatalf("expected help output to mention %q, got %s", shell, content)
		}
	}
}

func TestCompletionCommandWithoutShellReturnsHelp(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := run(context.Background(), []string{"completion"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	content := readFile(t, stdout)
	if !strings.Contains(content, "Usage: acpctl completion <subcommand>") {
		t.Fatalf("expected help output, got %s", content)
	}
}
