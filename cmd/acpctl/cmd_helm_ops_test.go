// cmd_helm_ops_test.go - Tests for Helm validation and smoke gates.
//
// Purpose:
//   - Keep Helm operator commands truthful and non-stubbed.
//
// Responsibilities:
//   - Cover success and validation drift failures.
//   - Cover missing helm prerequisite and lint failure handling.
//   - Ensure unsupported extra arguments fail instead of being ignored.
//
// Scope:
//   - Command-level Helm gate behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub helm lint execution and use fixture repos for typed validation.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

func TestRunHelmSmokeCommandSucceedsOnValidHelmSurface(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	restore := stubHelmLint(t, proc.Result{})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runHelmSmokeCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stdout); !stringsContainAll(got, "=== Helm Smoke Checks ===", "Helm smoke checks passed") {
		t.Fatalf("expected smoke success output, got %s", got)
	}
}

func TestRunHelmSmokeCommandFailsOnHelmContractDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "values.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	restore := stubHelmLint(t, proc.Result{})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runHelmSmokeCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !stringsContainAll(got, "Helm deployment surface validation failed", "profile must be production") {
		t.Fatalf("expected contract drift output, got %s", got)
	}
}

func TestRunHelmSmokeCommandReturnsPrereqWhenHelmMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	restore := stubHelmLint(t, proc.Result{
		Err: &proc.ExecError{Name: "helm", Kind: proc.KindNotFound, ExitCode: 127, Err: fmt.Errorf("exec: helm not found")},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runHelmSmokeCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !stringsContainAll(got, "helm not found in PATH") {
		t.Fatalf("expected missing-helm output, got %s", got)
	}
}

func TestRunHelmSmokeCommandReturnsDomainFailureWhenLintFails(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	restore := stubHelmLint(t, proc.Result{
		Err: &proc.ExecError{Name: "helm", Args: []string{"lint"}, Kind: proc.KindExit, ExitCode: 1, Err: fmt.Errorf("exit status 1")},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runHelmSmokeCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !stringsContainAll(got, "helm lint failed") {
		t.Fatalf("expected helm lint failure output, got %s", got)
	}
}

func TestRunHelmSmokeCommandReturnsRuntimeWhenLintCannotStart(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	restore := stubHelmLint(t, proc.Result{
		Err: &proc.ExecError{Name: "helm", Args: []string{"lint"}, Kind: proc.KindStart, ExitCode: -1, Err: fmt.Errorf("proc.Run requires a non-empty command name")},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runHelmSmokeCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !stringsContainAll(got, "helm lint could not start") {
		t.Fatalf("expected helm lint start failure output, got %s", got)
	}
}

func TestRunHelmSmokeCommandRejectsUnsupportedArguments(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := runHelmSmokeCommand(context.Background(), []string{"NAMESPACE=acp"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage failure, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stderr); !stringsContainAll(got, "unknown option: NAMESPACE=acp", "Usage: acpctl helm smoke") {
		t.Fatalf("expected usage output, got %s", got)
	}
}

func stubHelmLint(t *testing.T, result proc.Result) func() {
	t.Helper()

	original := runHelmLint
	runHelmLint = func(context.Context, string, *os.File, *os.File) proc.Result {
		return result
	}
	return func() {
		runHelmLint = original
	}
}

func stringsContainAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
