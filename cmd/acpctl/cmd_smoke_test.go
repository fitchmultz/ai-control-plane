// cmd_smoke_test.go - Tests for the runtime smoke command.
//
// Purpose:
//   - Lock the smoke command to truthful runtime pass/fail behavior.
//
// Responsibilities:
//   - Cover healthy runtime success.
//   - Cover missing Docker prerequisites.
//   - Cover missing auth, gateway failure, and canceled execution failures.
//
// Scope:
//   - Command-level smoke behavior only.
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

type stubSmokeInspector struct {
	collect func(ctx context.Context, opts status.Options) status.StatusReport
}

func (s stubSmokeInspector) Collect(ctx context.Context, opts status.Options) status.StatusReport {
	return s.collect(ctx, opts)
}

func (s stubSmokeInspector) Close() error {
	return nil
}

func TestRunSmokeTestCommandSucceedsWhenRuntimeIsHealthy(t *testing.T) {
	restore := stubSmokeDeps(t, stubSmokeInspector{
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
		return runSmokeTestCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stdout); !strings.Contains(got, "Runtime smoke checks passed") {
		t.Fatalf("expected success output, got %s", got)
	}
}

func TestRunSmokeTestCommandFailsWhenAuthIsMissing(t *testing.T) {
	restore := stubSmokeDeps(t, stubSmokeInspector{
		collect: func(context.Context, status.Options) status.StatusReport {
			return status.StatusReport{
				Overall: status.HealthLevelWarning,
				Components: map[string]status.ComponentStatus{
					"gateway":  {Name: "gateway", Level: status.HealthLevelWarning, Message: "LITELLM_MASTER_KEY not set; authorized gateway checks skipped"},
					"database": {Name: "database", Level: status.HealthLevelHealthy, Message: "Database is responding"},
				},
			}
		},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runSmokeTestCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "required components are not ready") || !strings.Contains(got, "authorized gateway checks skipped") {
		t.Fatalf("expected missing-auth output, got %s", got)
	}
}

func TestRunSmokeTestCommandFailsWhenGatewayIsUnavailable(t *testing.T) {
	restore := stubSmokeDeps(t, stubSmokeInspector{
		collect: func(context.Context, status.Options) status.StatusReport {
			return status.StatusReport{
				Overall: status.HealthLevelUnhealthy,
				Components: map[string]status.ComponentStatus{
					"gateway":  {Name: "gateway", Level: status.HealthLevelUnhealthy, Message: "Gateway unreachable: connection refused"},
					"database": {Name: "database", Level: status.HealthLevelHealthy, Message: "Database is responding"},
				},
			}
		},
	})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runSmokeTestCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Gateway unreachable") {
		t.Fatalf("expected gateway failure details, got %s", got)
	}
}

func TestRunSmokeTestCommandReturnsRuntimeErrorWhenCanceled(t *testing.T) {
	restore := stubSmokeDeps(t, stubSmokeInspector{
		collect: func(ctx context.Context, opts status.Options) status.StatusReport {
			<-ctx.Done()
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runSmokeTestCommand(ctx, nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Smoke check canceled") {
		t.Fatalf("expected canceled output, got %s", got)
	}
}

func TestRunSmokeTestCommandReturnsPrereqWhenDockerMissing(t *testing.T) {
	restore := stubSmokeDepsWithoutDocker(t)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runSmokeTestCommand(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq failure, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Docker not found") {
		t.Fatalf("expected docker prereq output, got %s", got)
	}
}

func stubSmokeDeps(t *testing.T, inspector smokeInspector) func() {
	t.Helper()

	originalInspector := newSmokeInspector
	newSmokeInspector = func(string) smokeInspector {
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
		newSmokeInspector = originalInspector
	}
}

func stubSmokeDepsWithoutDocker(t *testing.T) func() {
	t.Helper()

	originalInspector := newSmokeInspector
	newSmokeInspector = func(string) smokeInspector {
		t.Fatal("inspector should not be constructed when docker is missing")
		return nil
	}
	t.Setenv("PATH", t.TempDir())

	return func() {
		newSmokeInspector = originalInspector
	}
}
