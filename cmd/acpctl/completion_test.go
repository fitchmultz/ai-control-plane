// completion_test.go - Tests for command-spec-driven completion behavior.
//
// Purpose:
//
//	Verify completion scripts and hidden completion suggestions are generated
//	from the typed command-spec tree instead of ad hoc registry scraping.
//
// Responsibilities:
//   - Test shell completion command dispatch.
//   - Test root/subcommand suggestion resolution.
//   - Test tracked config extraction helpers.
//   - Ensure generated scripts contain current command tree entries.
//
// Scope:
//   - Completion-layer behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests use temp repositories for config-driven suggestions.
package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestCompletionShellCommand_Bash(t *testing.T) {
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

func TestResolveSuggestions_RootAndSubcommands(t *testing.T) {
	rootSuggestions := resolveSuggestions(nil, "", t.TempDir())
	if !slices.Contains(rootSuggestions, "ci") || !slices.Contains(rootSuggestions, "deploy") {
		t.Fatalf("expected root suggestions to include current command tree, got %v", rootSuggestions)
	}

	deploySuggestions := resolveSuggestions([]string{"deploy"}, "", t.TempDir())
	if !slices.Contains(deploySuggestions, "up") || !slices.Contains(deploySuggestions, "readiness-evidence") {
		t.Fatalf("expected deploy suggestions from command tree, got %v", deploySuggestions)
	}
}

func TestResolveSuggestions_ConfigDrivenScenarioValues(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "demo", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	writeFile(t, filepath.Join(configDir, "demo_presets.yaml"), `presets:
  executive-demo:
    scenarios:
      - 1
      - 5
`)

	// Transitional legacy specs do not yet attach this provider directly, so assert the extractor.
	values := extractScenarioIDs(repoRoot)
	if !slices.Equal(values, []string{"1", "5"}) {
		t.Fatalf("unexpected scenario ids: %v", values)
	}
}

func TestExtractModelNames(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "demo", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	writeFile(t, filepath.Join(configDir, "litellm.yaml"), `---
model_list:
  - model_name: test-model-1
  - model_name: test-model-2
  - model_name: test-model-1
`)

	models := extractModelNames(tmpDir)
	if !slices.Equal(models, []string{"test-model-1", "test-model-2"}) {
		t.Fatalf("unexpected models: %v", models)
	}
}

func TestGeneratedCompletionScriptsFollowCommandTree(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		content := captureCompletionScript(t, shell)
		for _, root := range []string{"ci", "completion", "bridge", "deploy", "validate", "helm"} {
			if !strings.Contains(content, root) {
				t.Fatalf("%s completion missing root command %q", shell, root)
			}
		}
		for _, subcommand := range []string{"wait", "host_preflight", "artifact-retention", "compose-healthchecks"} {
			if !strings.Contains(content, subcommand) {
				t.Fatalf("%s completion missing subcommand %q", shell, subcommand)
			}
		}
	}
}

func captureCompletionScript(t *testing.T, shell string) string {
	t.Helper()
	stdout, err := os.CreateTemp("", "acpctl_completion_stdout")
	if err != nil {
		t.Fatalf("create stdout: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr, err := os.CreateTemp("", "acpctl_completion_stderr")
	if err != nil {
		t.Fatalf("create stderr: %v", err)
	}
	defer os.Remove(stderr.Name())

	if exitCode := run(context.Background(), []string{"completion", shell}, stdout, stderr); exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("completion %s failed: %d stderr=%s", shell, exitCode, readFile(t, stderr))
	}

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatalf("seek stdout: %v", err)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(stdout); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
