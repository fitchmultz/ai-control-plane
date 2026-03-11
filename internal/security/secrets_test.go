// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify tracked secret-scan policy loading, validation, and matching behavior.
//
// Responsibilities:
//   - Cover canonical secret policy decoding and malformed-policy failures.
//   - Verify path-rule, content-rule, and placeholder-exemption enforcement.
//   - Lock down deterministic finding ordering for CI-facing scanner output.
//
// Scope:
//   - Unit tests for the tracked-file secret scanner only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Tests provision isolated temporary repositories with tracked policy fixtures.
//   - Policy failures surface as loader errors, not synthetic findings.
package security

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
      "id": "docs-placeholder-examples",
      "path_patterns": ["README.md", "demo/README.md", "docs/**"],
      "allowed_substrings": ["change-me", "sk-demo-", "sk-offline-demo-", "sk-your-", "sk-personal-", "sk-litellm-"]
    }
  ]
}`

func TestLoadSecretsPolicy_MissingFileFails(t *testing.T) {
	repoRoot := t.TempDir()

	_, err := loadSecretsPolicy(repoRoot)
	if err == nil {
		t.Fatal("expected missing policy file error")
	}
	if !strings.Contains(err.Error(), "SECRET_SCAN_POLICY.json") {
		t.Fatalf("expected policy path in error, got %v", err)
	}
}

func TestLoadSecretsPolicy_RejectsMissingPolicyID(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "docs", "policy", "SECRET_SCAN_POLICY.json"), `{
  "schema_version": "1.0.0",
  "policy_id": "",
  "path_rules": [
    {
      "id": "tracked-env-file",
      "message": "tracked environment file",
      "patterns": ["**/.env"]
    }
  ]
}`)

	_, err := loadSecretsPolicy(repoRoot)
	if err == nil {
		t.Fatal("expected missing policy_id error")
	}
	if !strings.Contains(err.Error(), "missing policy_id") {
		t.Fatalf("expected missing policy_id error, got %v", err)
	}
}

func TestLoadSecretsPolicy_RejectsInvalidRegex(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "docs", "policy", "SECRET_SCAN_POLICY.json"), `{
  "schema_version": "1.0.0",
  "policy_id": "broken-regex",
  "content_rules": [
    {
      "id": "openai-style-key",
      "message": "OpenAI-style API key",
      "pattern": "("
    }
  ]
}`)

	_, err := loadSecretsPolicy(repoRoot)
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
	if !strings.Contains(err.Error(), "missing closing") {
		t.Fatalf("expected regex compilation error, got %v", err)
	}
}

func TestLoadSecretsPolicy_RejectsDuplicateIDs(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "docs", "policy", "SECRET_SCAN_POLICY.json"), `{
  "schema_version": "1.0.0",
  "policy_id": "duplicate-id",
  "path_rules": [
    {
      "id": "duplicate",
      "message": "tracked environment file",
      "patterns": ["**/.env"]
    }
  ],
  "content_rules": [
    {
      "id": "duplicate",
      "message": "OpenAI-style API key",
      "pattern": "\\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\\b"
    }
  ]
}`)

	_, err := loadSecretsPolicy(repoRoot)
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), `duplicate policy id "duplicate"`) {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestAuditTrackedSecrets_UsesTrackedPolicyRulesDeterministically(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecretsPolicyFixture(t, repoRoot, defaultSecretsPolicyFixture)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "config", "provider.txt"), "OPENAI_API_KEY=sk-"+strings.Repeat("a", 24)+"\n")
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "keys", "id_rsa"), "placeholder key filename\n")
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "ops", ".env"), "OPENAI_API_KEY=\n")
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "docs", "guide.md"), "OPENAI_API_KEY=sk-demo-"+strings.Repeat("a", 24)+"\n")

	findings, err := AuditTrackedSecrets(repoRoot, []string{
		"keys/id_rsa",
		"docs/guide.md",
		"config/provider.txt",
		"ops/.env",
	})
	if err != nil {
		t.Fatalf("AuditTrackedSecrets returned error: %v", err)
	}

	want := []Finding{
		{Path: "config/provider.txt", Line: 1, RuleID: "openai-style-key", Message: "OpenAI-style API key"},
		{Path: "keys/id_rsa", RuleID: "private-key-file", Message: "suspicious private-key filename"},
		{Path: "ops/.env", RuleID: "tracked-env-file", Message: "tracked environment file"},
	}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("unexpected findings:\n got %#v\nwant %#v", findings, want)
	}
}

func TestAuditTrackedSecrets_AppliesPlaceholderExemptionsFromPolicy(t *testing.T) {
	repoRoot := t.TempDir()
	writeSecretsPolicyFixture(t, repoRoot, `{
  "schema_version": "1.0.0",
  "policy_id": "placeholder-policy",
  "path_rules": [],
  "content_rules": [
    {
      "id": "any-assignment",
      "message": "assignment detected",
      "pattern": "=[[:space:]]*$"
    }
  ],
  "placeholder_exemptions": [
    {
      "id": "demo-env-empty-values",
      "path_patterns": ["demo/.env.example"],
      "allowed_substrings": [],
      "allow_empty_assignment": true
    }
  ]
}`)
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "demo", ".env.example"), "OPENAI_API_KEY=\n")
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "config", "provider.env"), "OPENAI_API_KEY=\n")

	findings, err := AuditTrackedSecrets(repoRoot, []string{"config/provider.env", "demo/.env.example"})
	if err != nil {
		t.Fatalf("AuditTrackedSecrets returned error: %v", err)
	}
	want := []Finding{
		{Path: "config/provider.env", Line: 1, RuleID: "any-assignment", Message: "assignment detected"},
	}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("unexpected findings:\n got %#v\nwant %#v", findings, want)
	}
}

func writeSecretsPolicyFixture(t *testing.T, repoRoot string, policyJSON string) {
	t.Helper()
	writeSecurityFixtureFile(t, filepath.Join(repoRoot, "docs", "policy", "SECRET_SCAN_POLICY.json"), policyJSON)
}
