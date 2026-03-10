// completion_shell_test.go - Renderer-focused completion coverage.
//
// Purpose:
//   - Verify shell completion commands render deterministic scripts.
//
// Responsibilities:
//   - Cover shell subcommand execution and invariant command presence.
//
// Scope:
//   - Completion script rendering only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests assert stable contract markers, not entire script bodies.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestCompletionShellCommandBash(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := run(context.Background(), []string{"completion", "bash"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}

	content := readFile(t, stdout)
	if !strings.Contains(content, "_acpctl_complete") {
		t.Fatalf("expected bash completion function, got %s", content)
	}
	if !strings.Contains(content, "complete -o default -F _acpctl_complete acpctl") {
		t.Fatalf("expected bash complete line, got %s", content)
	}
}

func TestGeneratedCompletionScriptsFollowCommandTree(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		content := captureCompletionScript(t, shell)
		for _, root := range []string{"ci", "completion", "deploy", "validate", "helm"} {
			if !strings.Contains(content, root) {
				t.Fatalf("%s completion missing root command %q", shell, root)
			}
		}
		if strings.Contains(content, "bridge") {
			t.Fatalf("%s completion leaked hidden root command %q", shell, "bridge")
		}
		for _, subcommand := range []string{"wait", "artifact-retention", "compose-healthchecks", "service-restart"} {
			if !strings.Contains(content, subcommand) {
				t.Fatalf("%s completion missing subcommand %q", shell, subcommand)
			}
		}
	}
}
