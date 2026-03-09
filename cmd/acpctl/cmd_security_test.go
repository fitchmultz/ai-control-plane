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

const defaultSecretsPolicyFixture = `{
  "schema_version": "1.0.0",
  "policy_id": "tracked-secret-scan-v1",
  "description": "Tracked-file repository secret scanning policy for deterministic CI-friendly leak detection.",
  "path_rules": [
    {
      "id": "tracked-env-file",
      "message": "tracked environment file",
      "patterns": ["**/.env"]
    },
    {
      "id": "private-key-file",
      "message": "suspicious private-key filename",
      "patterns": ["**/id_rsa", "**/id_ed25519"]
    },
    {
      "id": "secret-bearing-file",
      "message": "suspicious certificate/key archive filename",
      "patterns": ["**/*.pem", "**/*.p12", "**/*.pfx"]
    }
  ],
  "content_rules": [
    {
      "id": "private-key-block",
      "message": "private key material",
      "pattern": "-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----"
    },
    {
      "id": "aws-access-key-id",
      "message": "AWS access key ID",
      "pattern": "\\bAKIA[0-9A-Z]{16}\\b"
    },
    {
      "id": "github-token",
      "message": "GitHub token",
      "pattern": "\\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\\b"
    },
    {
      "id": "slack-token",
      "message": "Slack token",
      "pattern": "\\bxox[baprs]-[A-Za-z0-9-]{20,}\\b"
    },
    {
      "id": "google-api-key",
      "message": "Google API key",
      "pattern": "\\bAIza[0-9A-Za-z_-]{20,}\\b"
    },
    {
      "id": "openai-style-key",
      "message": "OpenAI-style API key",
      "pattern": "\\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\\b"
    }
  ],
  "placeholder_exemptions": [
    {
      "id": "demo-env-example-placeholders",
      "path_patterns": ["demo/.env.example"],
      "allowed_substrings": ["change-me"],
      "allow_empty_assignment": true
    },
    {
      "id": "test-placeholder-fixtures",
      "path_patterns": ["**/*_test.go", "**/tests/**"],
      "allowed_substrings": ["sk-test-", "change-me", "sk-litellm-"]
    },
    {
      "id": "helm-example-placeholders",
      "path_patterns": ["deploy/helm/ai-control-plane/examples/**"],
      "allowed_substrings": ["sk-demo-", "sk-offline-demo-"]
    },
    {
      "id": "docs-placeholder-examples",
      "path_patterns": ["README.md", "demo/README.md", "docs/**"],
      "allowed_substrings": ["change-me", "sk-demo-", "sk-offline-demo-", "sk-your-", "sk-personal-", "sk-litellm-"]
    }
  ]
}`

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

func TestRunSecretsAudit_IgnoresTrackedDeletedFilesInWorkingTree(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":          "fixture\n",
			"internal/legacy.go": "package legacy\n",
			"demo/.env.example":  "OPENAI_API_KEY=\n",
		},
	})

	if err := os.Remove(filepath.Join(repoRoot, "internal", "legacy.go")); err != nil {
		t.Fatalf("remove tracked file from worktree: %v", err)
	}

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success with deleted tracked file, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
}

func TestRunSecretsAudit_FailsOnMalformedPolicy(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":         "fixture\n",
			"demo/.env.example": "OPENAI_API_KEY=\n",
		},
		SecretsPolicy: `{
  "schema_version": "1.0.0",
  "policy_id": "broken",
  "content_rules": [
    {
      "id": "openai-style-key",
      "message": "OpenAI-style API key",
      "pattern": "("
    }
  ]
}`,
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runSecretsAudit(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stderr), "SECRET_SCAN_POLICY.json") {
		t.Fatalf("expected policy error in stderr, got %s", readFile(t, stderr))
	}
}

func TestRunValidatePublicHygiene_FailsOnTrackedLocalOnlyFile(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureRepo(t, repoRoot, securityFixtureOptions{
		TrackedFiles: map[string]string{
			"README.md":         "fixture\n",
			".scratchpad.md":    "notes\n",
			"demo/.env.example": "OPENAI_API_KEY=\n",
		},
	})

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidatePublicHygiene(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stderr), ".scratchpad.md") {
		t.Fatalf("expected scratchpad violation, got %s", readFile(t, stderr))
	}
}

func TestRunValidateSupplyChain_FailsOnNonDigestPinnedImage(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","allowlist":[],"severity_policy":{"fail_on":["high"]}}`)
	writeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  app:\n    image: example/app:latest\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateSupplyChain(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stderr), "demo/docker-compose.yml") {
		t.Fatalf("expected compose violation, got %s", readFile(t, stderr))
	}
}

func TestRunValidateSupplyChain_IgnoresCommentedImageLines(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"), `{"policy_id":"policy","allowlist":[],"severity_policy":{"fail_on":["high"]}}`)
	writeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  app:\n    # image: example/app:latest\n    image: example/app:1.2.3@sha256:abcdef\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateSupplyChain(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success when only commented image line is non-digest, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
}

func TestRunValidateLicense_FailsOnRestrictedReferenceOutsideDocs(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "docs", "policy", "THIRD_PARTY_LICENSE_MATRIX.json"), `{"schema_version":1,"policy_id":"policy","scan_scope":{"include":["cmd/**/*.go"],"exclude":["cmd/acpctl/cmd_security.go"]},"restricted_components":[{"name":"litellm-enterprise","match":{"content_regex":["import litellm.enterprise"]}}]}`)
	writeFile(t, filepath.Join(repoRoot, "cmd", "sample.go"), "package sample\n// import litellm.enterprise\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateLicense(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stderr), "cmd/sample.go") {
		t.Fatalf("expected restricted reference finding, got %s", readFile(t, stderr))
	}
}

func TestRunValidateLicense_RespectsExcludePatterns(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "docs", "policy", "THIRD_PARTY_LICENSE_MATRIX.json"), `{"schema_version":1,"policy_id":"policy","scan_scope":{"include":["cmd/**/*.go"],"exclude":["cmd/acpctl/cmd_security.go"]},"restricted_components":[{"name":"litellm-enterprise","match":{"content_regex":["import litellm.enterprise"]}}]}`)
	writeFile(t, filepath.Join(repoRoot, "cmd", "acpctl", "cmd_security.go"), "package main\n// import litellm.enterprise\n")

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateLicense(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success for excluded enforcement file, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
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
	TrackedFiles  map[string]string
	SecretsPolicy string
}

func writeSecurityFixtureRepo(t *testing.T, repoRoot string, opts securityFixtureOptions) {
	t.Helper()
	policyJSON := opts.SecretsPolicy
	if strings.TrimSpace(policyJSON) == "" {
		policyJSON = defaultSecretsPolicyFixture
	}
	writeFile(t, filepath.Join(repoRoot, "docs", "policy", "SECRET_SCAN_POLICY.json"), policyJSON)
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
