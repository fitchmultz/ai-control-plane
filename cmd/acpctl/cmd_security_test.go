// cmd_security_test.go - Tests for security validation command behavior
//
// Purpose: Verify the typed security command surface stays deterministic and honest.
// Responsibilities:
//   - Exercise fixture-backed tracked-file secrets auditing.
//   - Verify placeholder/example content behavior intentionally.
//   - Confirm aggregate security validation delegates to the Make-owned gate.
// Scope:
//   - Unit coverage for `acpctl validate secrets-audit` and validate security routing.
// Usage:
//   - Run with `go test ./cmd/acpctl`.
// Invariants/Assumptions:
//   - Tests operate on isolated temp git repos instead of the live checkout.
//   - Aggregate security policy remains owned by `make security-gate`.

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunSecretsAudit_Help(t *testing.T) {
	stdout, stderr := newTestFiles(t)

	exitCode := runSecretsAudit(context.Background(), []string{"help"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected help to succeed, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stdout), "deterministic tracked-file secrets audit") {
		t.Fatalf("expected tracked-file help text, got %s", readFile(t, stdout))
	}
}

func TestRunSecretsAudit_CleanTrackedRepoPasses(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":         "clean fixture\n",
			"demo/.env.example": "LITELLM_MASTER_KEY=sk-litellm-master-change-me\nLITELLM_SALT_KEY=sk-litellm-salt-change-me\n",
		},
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "Secrets audit passed") {
		t.Fatalf("expected success output, got %s", readFile(t, stdout))
	}
}

func TestRunSecretsAudit_FailsOnTrackedPrivateKeyFile(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":         "fixture\n",
			"deploy/id_rsa":     "not a real key, but tracked like one\n",
			"demo/.env.example": "OPENAI_API_KEY=\n",
		},
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "[private-key-file]") {
		t.Fatalf("expected private-key filename finding, got %s", readFile(t, stdout))
	}
}

func TestRunSecretsAudit_FailsOnSecretContent(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":           "fixture\n",
			"config/provider.txt": "OPENAI_API_KEY=" + "sk-" + strings.Repeat("a", 24) + "\n",
			"demo/.env.example":   "OPENAI_API_KEY=\n",
		},
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "[openai-style-key]") {
		t.Fatalf("expected OpenAI-style key finding, got %s", readFile(t, stdout))
	}
}

func TestRunSecretsAudit_AllowsDocumentedExamplePlaceholders(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md": "fixture\n",
			"demo/.env.example": strings.Join([]string{
				"# PLACEHOLDER: safe committed example",
				"LITELLM_MASTER_KEY=sk-litellm-master-change-me",
				"LITELLM_SALT_KEY=sk-litellm-salt-change-me",
				"# Format: AIza...",
				"GEMINI_API_KEY=",
				"",
			}, "\n"),
		},
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected placeholder examples to pass, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
}

func TestRunDelegatedGroup_ValidateSecurityDelegatesToSecurityGate(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md": "fixture\n",
		},
	})

	recordPath := filepath.Join(repoRoot, "make-invocation.txt")
	fakeMake := filepath.Join(repoRoot, "fake-make.sh")
	writeFile(t, fakeMake, strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$@\" > \"" + recordPath + "\"",
		"exit 0",
		"",
	}, "\n"))
	if err := os.Chmod(fakeMake, 0o755); err != nil {
		t.Fatalf("chmod fake make: %v", err)
	}

	validateGroup, ok := lookupDelegatedGroup("validate")
	if !ok {
		t.Fatalf("expected validate group to exist")
	}

	originalMakeBin := os.Getenv("ACPCTL_MAKE_BIN")
	if err := os.Setenv("ACPCTL_MAKE_BIN", fakeMake); err != nil {
		t.Fatalf("set ACPCTL_MAKE_BIN: %v", err)
	}
	defer func() {
		if originalMakeBin == "" {
			_ = os.Unsetenv("ACPCTL_MAKE_BIN")
			return
		}
		_ = os.Setenv("ACPCTL_MAKE_BIN", originalMakeBin)
	}()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runDelegatedGroup(context.Background(), validateGroup, []string{"security"}, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected delegated security success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}

	recorded := strings.TrimSpace(readFileFromPath(t, recordPath))
	if recorded != "security-gate" {
		t.Fatalf("expected make to receive security-gate, got %q", recorded)
	}
}

func TestRunDelegatedGroup_ValidateSecretsAuditHelpUsesNativeHelp(t *testing.T) {
	validateGroup, ok := lookupDelegatedGroup("validate")
	if !ok {
		t.Fatalf("expected validate group to exist")
	}

	stdout, stderr := newTestFiles(t)
	exitCode := runDelegatedGroup(context.Background(), validateGroup, []string{"secrets-audit", "help"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected native help success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}

	helpOutput := readFile(t, stdout)
	if !strings.Contains(helpOutput, "deterministic tracked-file secrets audit") {
		t.Fatalf("expected native help output, got %s", helpOutput)
	}
	if strings.Contains(helpOutput, "Delegates to make target") {
		t.Fatalf("expected native help rather than delegated help, got %s", helpOutput)
	}
}

type securityFixtureOptions struct {
	TrackedFiles map[string]string
}

func writeSecurityFixtureRepo(t *testing.T, repoRoot string, opts securityFixtureOptions) {
	t.Helper()
	for relPath, contents := range opts.TrackedFiles {
		writeFile(t, filepath.Join(repoRoot, relPath), contents)
	}
	runGit(t, repoRoot, "init", "-q")
	runGit(t, repoRoot, "add", ".")
}

func runGit(t *testing.T, repoRoot string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v output=%s", strings.Join(args, " "), err, output)
	}
}

func readFileFromPath(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
