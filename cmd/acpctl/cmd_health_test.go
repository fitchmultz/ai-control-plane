// cmd_health_test.go - Tests for the runtime health command.
//
// Purpose:
//   - Lock the health command to stable prerequisite, cancellation, and report
//     rendering behavior.
//
// Responsibilities:
//   - Cover healthy runtime success.
//   - Cover missing Docker prerequisites.
//   - Cover canceled execution failures.
//
// Scope:
//   - Command-level health behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub the shared inspector instead of requiring a live runtime.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type stubHealthInspector struct {
	collect func(ctx context.Context, opts status.Options) status.StatusReport
}

func (s stubHealthInspector) Collect(ctx context.Context, opts status.Options) status.StatusReport {
	return s.collect(ctx, opts)
}

func (s stubHealthInspector) Close() error {
	return nil
}

func TestRunHealthCommandSucceedsWhenRuntimeIsHealthy(t *testing.T) {
	restore := stubHealthDeps(t, stubHealthInspector{
		collect: func(context.Context, status.Options) status.StatusReport {
			return status.StatusReport{
				Overall: status.HealthLevelHealthy,
				Components: map[string]status.ComponentStatus{
					"gateway":  {Name: "gateway", Level: status.HealthLevelHealthy, Message: "Gateway is responding"},
					"database": {Name: "database", Level: status.HealthLevelHealthy, Message: "Database is responding"},
				},
			}
		},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "health")
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stdout); !strings.Contains(got, "Gateway") || !strings.Contains(got, "Overall: HEALTHY") {
		t.Fatalf("expected health report output, got %s", got)
	}
}

func TestRunHealthCommandReturnsRuntimeErrorWhenCanceled(t *testing.T) {
	restore := stubHealthDeps(t, stubHealthInspector{
		collect: func(ctx context.Context, opts status.Options) status.StatusReport {
			<-ctx.Done()
			return status.StatusReport{
				Overall: status.HealthLevelHealthy,
				Components: map[string]status.ComponentStatus{
					"gateway": {Name: "gateway", Level: status.HealthLevelHealthy, Message: "Gateway is responding"},
				},
			}
		},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runTestCommand(t, ctx, stdout, stderr, "health")
	})

	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Health check canceled") {
		t.Fatalf("expected canceled output, got %s", got)
	}
}

func TestRunHealthCommandReturnsPrereqWhenDockerMissing(t *testing.T) {
	restore := stubHealthDepsWithoutDocker(t)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "health")
	})

	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq failure, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Docker not found") {
		t.Fatalf("expected docker prereq output, got %s", got)
	}
}

func stubHealthDeps(t *testing.T, inspector runtimeStatusInspector) func() {
	t.Helper()

	originalInspector := newHealthInspector
	newHealthInspector = func(string) runtimeStatusInspector {
		return inspector
	}

	binDir := t.TempDir()
	dockerPath := filepath.Join(binDir, "docker")
	if err := os.WriteFile(dockerPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write docker stub: %v", err)
	}
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)

	return func() {
		newHealthInspector = originalInspector
	}
}

func stubHealthDepsWithoutDocker(t *testing.T) func() {
	t.Helper()

	originalInspector := newHealthInspector
	newHealthInspector = func(string) runtimeStatusInspector {
		t.Fatal("inspector should not be constructed when docker is missing")
		return nil
	}
	t.Setenv("PATH", t.TempDir())

	return func() {
		newHealthInspector = originalInspector
	}
}
