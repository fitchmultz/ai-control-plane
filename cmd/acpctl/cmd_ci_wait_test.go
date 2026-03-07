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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type fakeCIWaitCompose struct {
	ps string
}

func (f fakeCIWaitCompose) PS(context.Context) (string, error) {
	return f.ps, nil
}

type fakeCIWaitGateway struct {
	healthy   bool
	masterKey bool
}

func (f fakeCIWaitGateway) Health(context.Context) (bool, int, error) {
	if f.healthy {
		return true, 200, nil
	}
	return false, 503, fmt.Errorf("not ready")
}

func (f fakeCIWaitGateway) HasMasterKey() bool {
	return f.masterKey
}

func TestRunCIWaitCommand_SucceedsWhenServicesReady(t *testing.T) {
	restore := stubCIWaitDeps(t, fakeCIWaitCompose{ps: "postgres healthy\nlitellm healthy\n"}, fakeCIWaitGateway{healthy: true, masterKey: true})
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
	restore := stubCIWaitDeps(t, fakeCIWaitCompose{ps: "postgres starting\nlitellm starting\n"}, fakeCIWaitGateway{healthy: false, masterKey: true})
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
	restore := stubCIWaitDeps(t, fakeCIWaitCompose{ps: "postgres starting\nlitellm starting\n"}, fakeCIWaitGateway{healthy: false, masterKey: true})
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

func stubCIWaitDeps(t *testing.T, compose ciWaitCompose, gateway ciWaitGateway) func() {
	t.Helper()
	originalCompose := newCIWaitCompose
	originalGateway := newCIWaitGateway
	newCIWaitCompose = func(string) (ciWaitCompose, error) { return compose, nil }
	newCIWaitGateway = func() ciWaitGateway { return gateway }

	binDir := t.TempDir()
	dockerPath := filepath.Join(binDir, "docker")
	writeFile(t, dockerPath, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(dockerPath, 0o755); err != nil {
		t.Fatalf("chmod fake docker: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("ACP_REPO_ROOT", t.TempDir())

	return func() {
		newCIWaitCompose = originalCompose
		newCIWaitGateway = originalGateway
	}
}
