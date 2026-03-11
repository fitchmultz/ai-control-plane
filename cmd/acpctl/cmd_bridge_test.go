// cmd_bridge_test.go - Tests for bridge-script execution helpers.
//
// Purpose:
//   - Verify bridge-script command helpers enforce repository-root and script
//     validation contracts while preserving stable exit handling.
//
// Responsibilities:
//   - Cover missing and non-executable bridge script prerequisites.
//   - Cover successful bridge script execution with repo-root propagation.
//
// Scope:
//   - Bridge-script execution helpers only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Bridge scripts execute through `/bin/bash`.
//   - ACP_REPO_ROOT remains the canonical repo-root override in tests.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunBridgeScript_MissingScriptReturnsPrereq(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("ACP_REPO_ROOT", repoRoot)

	stdout, stderr := newTestFiles(t)
	code := runBridgeScript(context.Background(), "scripts/libexec/missing.sh", "missing", nil, nil, stdout, stderr)
	if code != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq exit, got %d stderr=%s", code, readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "bridge script not found") {
		t.Fatalf("expected missing-script output, got %s", got)
	}
}

func TestRunBridgeScript_NonExecutableScriptReturnsPrereq(t *testing.T) {
	repoRoot := t.TempDir()
	scriptPath := filepath.Join(repoRoot, "scripts", "libexec", "check.sh")
	writeFile(t, scriptPath, "#!/usr/bin/env bash\nexit 0\n")
	if err := os.Chmod(scriptPath, 0o644); err != nil {
		t.Fatalf("failed to mark script non-executable: %v", err)
	}
	t.Setenv("ACP_REPO_ROOT", repoRoot)

	stdout, stderr := newTestFiles(t)
	code := runBridgeScript(context.Background(), filepath.ToSlash(strings.TrimPrefix(scriptPath, repoRoot+string(filepath.Separator))), "check", nil, nil, stdout, stderr)
	if code != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq exit, got %d stderr=%s", code, readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "bridge script is not executable") {
		t.Fatalf("expected non-executable output, got %s", got)
	}
}

func TestRunBridgeScript_ExecutesScriptWithRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	capturePath := filepath.Join(repoRoot, "captured.txt")
	relativePath := filepath.Join("scripts", "libexec", "bridge.sh")
	scriptPath := filepath.Join(repoRoot, relativePath)
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$ACP_REPO_ROOT\" \"$1\" \"$2\" >\"" + capturePath + "\"\n"
	writeFile(t, scriptPath, script)
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		t.Fatalf("failed to mark script executable: %v", err)
	}
	t.Setenv("ACP_REPO_ROOT", repoRoot)

	stdout, stderr := newTestFiles(t)
	code := runBridgeScript(context.Background(), filepath.ToSlash(relativePath), "bridge-check", []string{"prefix"}, []string{"value"}, stdout, stderr)
	if code != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}

	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("failed to read bridge capture: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(captured)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected repo root and two args, got %q", captured)
	}
	if lines[0] != repoRoot {
		t.Fatalf("expected repo root %q, got %q", repoRoot, lines[0])
	}
	if lines[1] != "prefix" || lines[2] != "value" {
		t.Fatalf("expected prefix/value args, got %q", captured)
	}
}
