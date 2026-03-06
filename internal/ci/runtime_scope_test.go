// runtime_scope_test validates CI runtime-scope decisions.
//
// Purpose:
//
//	Ensure path classification and conservative fallbacks match the prior shell
//	implementation for ci_should_run_runtime behavior.
//
// Responsibilities:
//   - Verify docs/test-only skip behavior.
//   - Verify runtime-impacting run behavior.
//   - Verify CI_FULL override and conservative fallbacks.
//
// Non-scope:
//   - Does not execute any runtime checks.
//
// Invariants/Assumptions:
//   - Exit-code mapping is tested at CLI/wrapper level outside this file.
package ci

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsTruthy(t *testing.T) {
	t.Parallel()
	truthy := []string{"1", "true", "TRUE", "True", "yes", "YES", "Yes"}
	for _, value := range truthy {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if !IsTruthy(value) {
				t.Fatalf("expected %q to be truthy", value)
			}
		})
	}
	if IsTruthy("0") {
		t.Fatal("expected 0 to be non-truthy")
	}
}

func TestDecideRuntimeScopeWithExplicitPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	result, err := DecideRuntimeScope(ctx, DecisionOptions{
		Paths:  []string{"docs/README.md", "AGENTS.md", "scripts/tests/onboard_test.sh"},
		CIFull: "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldRun {
		t.Fatalf("expected docs/tests-only paths to skip runtime checks")
	}

	result, err = DecideRuntimeScope(ctx, DecisionOptions{
		Paths:  []string{"Makefile"},
		CIFull: "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldRun {
		t.Fatalf("expected Makefile to require runtime checks")
	}
}

func TestDecideRuntimeScopeCIFullOverride(t *testing.T) {
	t.Parallel()
	result, err := DecideRuntimeScope(context.Background(), DecisionOptions{
		Paths:  []string{"docs/README.md"},
		CIFull: "1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldRun {
		t.Fatal("expected CI_FULL to force runtime checks")
	}
}

func TestDecideRuntimeScopeNoGitFallsBackToRun(t *testing.T) {
	t.Parallel()

	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", ""); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", originalPath)
	})

	result, err := DecideRuntimeScope(context.Background(), DecisionOptions{
		RepoRoot: "/tmp",
		CIFull:   "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldRun {
		t.Fatal("expected conservative run when git is unavailable")
	}
}

func TestDecideRuntimeScopeFromGitChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmp := t.TempDir()
	mustRun(t, tmp, "git", "init")
	mustRun(t, tmp, "git", "config", "user.email", "ci-runtime-test@example.com")
	mustRun(t, tmp, "git", "config", "user.name", "CI Runtime Test")

	mustWriteFile(t, filepath.Join(tmp, "README.md"), "# test\n")
	mustRun(t, tmp, "git", "add", "README.md")
	mustRun(t, tmp, "git", "commit", "-m", "init")

	mustWriteFile(t, filepath.Join(tmp, "docs", "README.md"), "# docs\n")

	result, err := DecideRuntimeScope(context.Background(), DecisionOptions{
		RepoRoot: tmp,
		CIFull:   "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldRun {
		t.Fatalf("expected docs-only git change to skip runtime checks")
	}

	mustWriteFile(t, filepath.Join(tmp, "Makefile"), "all:\n\t@echo ok\n")
	result, err = DecideRuntimeScope(context.Background(), DecisionOptions{
		RepoRoot: tmp,
		CIFull:   "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldRun {
		t.Fatalf("expected Makefile git change to run runtime checks")
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %v\n%s\nerror: %v", name, args, string(out), err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
