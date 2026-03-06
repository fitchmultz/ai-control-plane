// cmd_delegated_test validates delegated Make command execution safeguards.
//
// Purpose:
//
//	Ensure delegated command execution honors executable validation and
//	returns deterministic exit codes for missing/invalid make binaries.
//
// Responsibilities:
//   - Verify missing ACPCTL_MAKE_BIN paths return prerequisite exit code.
//   - Verify non-executable ACPCTL_MAKE_BIN paths return prerequisite exit code.
//   - Verify executable ACPCTL_MAKE_BIN paths execute delegated targets.
//
// Scope:
//   - Covers delegated command execution safeguards only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`
//
// Invariants/Assumptions:
//   - Exit-code contract remains 0/1/2/3/64.
//   - Delegated command execution uses ACPCTL_MAKE_BIN when provided.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunMakeTarget_MissingOverrideReturnsPrereq(t *testing.T) {
	t.Setenv("ACPCTL_MAKE_BIN", filepath.Join(t.TempDir(), "missing-make"))
	t.Setenv("ACP_REPO_ROOT", t.TempDir())

	if code := runMakeTarget("up", nil, os.Stdout, os.Stderr); code != exitcodes.ACPExitPrereq {
		t.Fatalf("expected exit code %d, got %d", exitcodes.ACPExitPrereq, code)
	}
}

func TestRunMakeTarget_NonExecutableOverrideReturnsPrereq(t *testing.T) {
	tmpDir := t.TempDir()
	nonExec := filepath.Join(tmpDir, "not-executable")
	if err := os.WriteFile(nonExec, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("failed to write non-executable stub: %v", err)
	}

	t.Setenv("ACPCTL_MAKE_BIN", nonExec)
	t.Setenv("ACP_REPO_ROOT", tmpDir)

	if code := runMakeTarget("up", nil, os.Stdout, os.Stderr); code != exitcodes.ACPExitPrereq {
		t.Fatalf("expected exit code %d, got %d", exitcodes.ACPExitPrereq, code)
	}
}

func TestRunMakeTarget_ExecutableOverrideRunsTarget(t *testing.T) {
	tmpDir := t.TempDir()
	capturePath := filepath.Join(tmpDir, "captured.txt")
	makeStub := filepath.Join(tmpDir, "make-stub.sh")

	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" >\"" + capturePath + "\"\n"
	if err := os.WriteFile(makeStub, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write executable stub: %v", err)
	}

	t.Setenv("ACPCTL_MAKE_BIN", makeStub)
	t.Setenv("ACP_REPO_ROOT", tmpDir)

	if code := runMakeTarget("demo-target", []string{"FLAG=1"}, os.Stdout, os.Stderr); code != exitcodes.ACPExitSuccess {
		t.Fatalf("expected exit code %d, got %d", exitcodes.ACPExitSuccess, code)
	}

	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("failed to read captured arguments: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(captured)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected captured target and argument, got: %q", captured)
	}
	if lines[0] != "demo-target" {
		t.Fatalf("expected first arg demo-target, got %q", lines[0])
	}
	if lines[1] != "FLAG=1" {
		t.Fatalf("expected second arg FLAG=1, got %q", lines[1])
	}
}
