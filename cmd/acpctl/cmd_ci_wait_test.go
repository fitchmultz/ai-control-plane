// cmd_ci_wait_test.go - Deterministic tests for the CI wait command.
//
// Purpose:
//
//	Verify the command's single authoritative timeout/cancellation path without
//	depending on a live Docker daemon or gateway.
//
// Responsibilities:
//   - Test successful readiness detection.
//   - Test command timeout returns domain failure.
//   - Test parent cancellation returns runtime failure promptly.
//
// Scope:
//   - Covers command-layer behavior for `acpctl ci wait`.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests inject fake compose and gateway clients.
//   - Tests provide a fake docker binary on PATH for prerequisite checks.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type fakeCIWaitInspector struct {
	reports []status.StatusReport
	index   int
}

func (f *fakeCIWaitInspector) Collect(context.Context, status.Options) status.StatusReport {
	if len(f.reports) == 0 {
		return status.StatusReport{}
	}
	if f.index >= len(f.reports) {
		return f.reports[len(f.reports)-1]
	}
	report := f.reports[f.index]
	f.index++
	return report
}

func (f *fakeCIWaitInspector) Close() error {
	return nil
}

func TestRunCIWaitCommand_SucceedsWhenServicesReady(t *testing.T) {
	restore := stubCIWaitDeps(t, &fakeCIWaitInspector{reports: []status.StatusReport{
		readyCIWaitReport(),
	}})
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := runCIWaitCommand(context.Background(), nil, stdout, stderr)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "All services are healthy and ready") {
		t.Fatalf("expected success output, got %s", readFile(t, stdout))
	}
}

func TestRunCIWaitCommand_TimeoutReturnsDomain(t *testing.T) {
	restore := stubCIWaitDeps(t, &fakeCIWaitInspector{reports: []status.StatusReport{
		pendingCIWaitReport(),
	}})
	defer restore()

	stdout, stderr := newTestFiles(t)
	start := time.Now()
	exitCode := runCIWaitCommand(context.Background(), []string{"--timeout", "1"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain exit, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("timeout path took too long: %s", time.Since(start))
	}
	if !strings.Contains(readFile(t, stdout), "Timeout: Services did not become healthy within 1s") {
		t.Fatalf("expected timeout output, got %s", readFile(t, stdout))
	}
}

func TestRunCIWaitCommand_CanceledReturnsRuntime(t *testing.T) {
	restore := stubCIWaitDeps(t, &fakeCIWaitInspector{reports: []status.StatusReport{
		pendingCIWaitReport(),
	}})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, stderr := newTestFiles(t)
	exitCode := runCIWaitCommand(ctx, []string{"--timeout", "1"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stderr), "CI wait canceled") {
		t.Fatalf("expected cancel output, got %s", readFile(t, stderr))
	}
}

func stubCIWaitDeps(t *testing.T, inspector ciWaitInspector) func() {
	t.Helper()
	originalInspector := newCIWaitInspector
	newCIWaitInspector = func(string) ciWaitInspector { return inspector }

	binDir := t.TempDir()
	dockerPath := filepath.Join(binDir, "docker")
	writeFile(t, dockerPath, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(dockerPath, 0o755); err != nil {
		t.Fatalf("chmod fake docker: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("ACP_REPO_ROOT", t.TempDir())

	return func() {
		newCIWaitInspector = originalInspector
	}
}

func readyCIWaitReport() status.StatusReport {
	return status.StatusReport{
		Overall: status.HealthLevelHealthy,
		Components: map[string]status.ComponentStatus{
			"gateway":  {Name: "gateway", Level: status.HealthLevelHealthy, Message: "Gateway is responding"},
			"database": {Name: "database", Level: status.HealthLevelHealthy, Message: "Connected"},
		},
	}
}

func pendingCIWaitReport() status.StatusReport {
	return status.StatusReport{
		Overall: status.HealthLevelUnhealthy,
		Components: map[string]status.ComponentStatus{
			"gateway":  {Name: "gateway", Level: status.HealthLevelUnhealthy, Message: "Gateway unreachable"},
			"database": {Name: "database", Level: status.HealthLevelWarning, Message: "Database accessible, but schema is incomplete"},
		},
	}
}
