// cmd_host_deploy_test.go - Tests for the native host check/apply commands.
//
// Purpose:
//   - Lock the typed host deployment commands to their Ansible invocation
//     contract.
//
// Responsibilities:
//   - Cover check-mode syntax-check and playbook execution.
//   - Cover apply-mode extra-var forwarding.
//   - Cover missing ansible-playbook prerequisites.
//
// Scope:
//   - Host deployment command behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub ansible-playbook instead of requiring Ansible.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunHostCheckExecutesSyntaxCheckAndPlaybook(t *testing.T) {
	repoRoot := writeHostDeployFixtureRepo(t)
	captureFile := filepath.Join(t.TempDir(), "ansible-calls.txt")
	restore := stubHostDeployAnsible(t, captureFile)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "host", "check", "--inventory", "deploy/ansible/inventory/hosts.yml", "--limit", "gateway")
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	got := readFileFromPath(t, captureFile)
	if !strings.Contains(got, "ARGS:--syntax-check -i "+filepath.Join(repoRoot, "deploy/ansible/inventory/hosts.yml")+" "+filepath.Join(repoRoot, "deploy/ansible/playbooks/gateway_host.yml")+" --limit gateway --check") {
		t.Fatalf("expected syntax-check invocation, got %s", got)
	}
	if !strings.Contains(got, "ARGS:-i "+filepath.Join(repoRoot, "deploy/ansible/inventory/hosts.yml")+" "+filepath.Join(repoRoot, "deploy/ansible/playbooks/gateway_host.yml")+" --limit gateway --check") {
		t.Fatalf("expected playbook invocation, got %s", got)
	}
}

func TestRunHostApplyForwardsExtraVars(t *testing.T) {
	repoRoot := writeHostDeployFixtureRepo(t)
	captureFile := filepath.Join(t.TempDir(), "ansible-calls.txt")
	restore := stubHostDeployAnsible(t, captureFile)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr,
			"host", "apply",
			"--inventory", "deploy/ansible/inventory/hosts.yml",
			"--repo-path", "/opt/ai-control-plane",
			"--env-file", "/etc/ai-control-plane/secrets.env",
			"--tls-mode", "tls",
			"--public-url", "https://gateway.example.com",
			"--no-wait",
			"--skip-smoke-tests",
			"--stabilization-seconds", "15",
			"--extra-var", "custom_key=value",
		)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	got := readFileFromPath(t, captureFile)
	for _, needle := range []string{
		"acp_repo_path=/opt/ai-control-plane",
		"acp_env_file=/etc/ai-control-plane/secrets.env",
		"acp_tls_mode=tls",
		"acp_public_url=https://gateway.example.com",
		"acp_wait_for_stabilization=false",
		"acp_run_smoke_tests=false",
		"acp_stabilization_seconds=15",
		"custom_key=value",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("expected %q in capture, got %s", needle, got)
		}
	}
}

func TestRunHostCheckReturnsPrereqWhenAnsibleMissing(t *testing.T) {
	repoRoot := writeHostDeployFixtureRepo(t)
	t.Setenv("PATH", t.TempDir())
	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "host", "check", "--inventory", "deploy/ansible/inventory/hosts.yml")
	})

	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "ansible-playbook not found or not executable") {
		t.Fatalf("expected ansible error, got %s", got)
	}
}

func writeHostDeployFixtureRepo(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.yml"), "gateway:\n  hosts:\n    gateway:\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "hosts: gateway\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "ansible", "ansible.cfg"), "[defaults]\ninventory = ./inventory/hosts.yml\n")
	return repoRoot
}

func stubHostDeployAnsible(t *testing.T, captureFile string) func() {
	t.Helper()
	binDir := t.TempDir()
	script := "#!/bin/sh\nprintf 'ARGS:%s\\n' \"$*\" >>\"" + captureFile + "\"\nprintf 'ANSIBLE_CONFIG:%s\\n' \"${ANSIBLE_CONFIG:-}\" >>\"" + captureFile + "\"\nexit 0\n"
	writeFileWithMode(t, filepath.Join(binDir, "ansible-playbook"), script, 0o755)
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
	return func() {}
}
