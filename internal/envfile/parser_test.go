// parser_test.go - Regression tests for strict .env parsing.
//
// Purpose:
//   - Verify repository .env parsing remains data-only and deterministic.
//
// Responsibilities:
//   - Confirm values are read literally without shell execution.
//   - Confirm malformed lines are rejected.
//   - Confirm comments and quoted values are handled safely.
//
// Non-scope:
//   - Does not test shell integration entrypoints.
//   - Does not validate file permission policy.
//
// Invariants/Assumptions:
//   - Tests run against temporary files only.
//   - Command substitution text is treated as literal data.
package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupFileReturnsLiteralValueWithoutShellExecution(t *testing.T) {
	t.Parallel()

	envPath := writeEnvFixture(t, ""+
		"# comment\n"+
		"EVIL=$(touch /tmp/pwned)\n"+
		"LITELLM_MASTER_KEY=sk-test-123\n")

	value, ok, err := LookupFile(envPath, "LITELLM_MASTER_KEY")
	if err != nil {
		t.Fatalf("LookupFile() error = %v", err)
	}
	if !ok {
		t.Fatal("LookupFile() did not find key")
	}
	if value != "sk-test-123" {
		t.Fatalf("LookupFile() value = %q, want %q", value, "sk-test-123")
	}
}

func TestLookupFileRejectsMalformedLine(t *testing.T) {
	t.Parallel()

	envPath := writeEnvFixture(t, "export BAD=value\n")

	if _, _, err := LookupFile(envPath, "BAD"); err == nil {
		t.Fatal("LookupFile() expected malformed line error")
	}
}

func TestLookupFileTrimsPairedQuotes(t *testing.T) {
	t.Parallel()

	envPath := writeEnvFixture(t, "LITELLM_SALT_KEY=\"quoted value\"\n")

	value, ok, err := LookupFile(envPath, "LITELLM_SALT_KEY")
	if err != nil {
		t.Fatalf("LookupFile() error = %v", err)
	}
	if !ok {
		t.Fatal("LookupFile() did not find quoted key")
	}
	if value != "quoted value" {
		t.Fatalf("LookupFile() value = %q, want %q", value, "quoted value")
	}
}

func writeEnvFixture(t *testing.T, content string) string {
	t.Helper()

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return envPath
}
