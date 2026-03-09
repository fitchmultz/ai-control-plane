// cmd_env_test.go - Tests for strict env command behavior.
//
// Purpose:
//   - Verify `acpctl env get` reads `.env` data without shell execution.
//
// Responsibilities:
//   - Validate success, missing-key, and malformed-file behavior.
//
// Non-scope:
//   - Does not test shell wrappers that call the command.
//   - Does not test repository root detection.
//
// Invariants/Assumptions:
//   - Tests use temporary env files.
//   - Command output is written to provided stdout/stderr files.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestPrintEnvGetHelpStatesNonExecutingContract(t *testing.T) {
	t.Parallel()

	stdout, stdoutPath := tempOutputFile(t)
	exitCode := run(context.Background(), []string{"env", "get", "--help"}, stdout, stdout)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("run(env get --help) exit = %d, want %d", exitCode, exitcodes.ACPExitSuccess)
	}

	output := readOutputFile(t, stdoutPath)
	if !strings.Contains(output, "Prefer this over sourcing env files or grepping secrets from them.") {
		t.Fatalf("help output missing non-executing contract guidance: %q", output)
	}
}

func TestRunEnvGetCommandReturnsLiteralValue(t *testing.T) {
	t.Parallel()

	envPath := writeEnvCommandFixture(t, ""+
		"PAYLOAD=$(touch /tmp/pwned)\n"+
		"LITELLM_MASTER_KEY=sk-test-123\n")
	stdout, stdoutPath := tempOutputFile(t)
	stderr, _ := tempOutputFile(t)

	exitCode := runEnvGetCommand(context.Background(), []string{"--file", envPath, "LITELLM_MASTER_KEY"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("runEnvGetCommand() exit = %d, want %d", exitCode, exitcodes.ACPExitSuccess)
	}

	if got := strings.TrimSpace(readOutputFile(t, stdoutPath)); got != "sk-test-123" {
		t.Fatalf("stdout = %q, want %q", got, "sk-test-123")
	}
}

func TestRunEnvGetCommandMissingKeyReturnsDomain(t *testing.T) {
	t.Parallel()

	envPath := writeEnvCommandFixture(t, "DATABASE_URL=postgresql://demo\n")
	stdout, _ := tempOutputFile(t)
	stderr, stderrPath := tempOutputFile(t)

	exitCode := runEnvGetCommand(context.Background(), []string{"--file", envPath, "LITELLM_MASTER_KEY"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("runEnvGetCommand() exit = %d, want %d", exitCode, exitcodes.ACPExitDomain)
	}

	if got := readOutputFile(t, stderrPath); !strings.Contains(got, "LITELLM_MASTER_KEY not found") {
		t.Fatalf("stderr = %q, want missing-key message", got)
	}
}

func TestRunEnvGetCommandMalformedFileReturnsPrereq(t *testing.T) {
	t.Parallel()

	envPath := writeEnvCommandFixture(t, "export BAD=value\n")
	stdout, _ := tempOutputFile(t)
	stderr, stderrPath := tempOutputFile(t)

	exitCode := runEnvGetCommand(context.Background(), []string{"--file", envPath, "BAD"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("runEnvGetCommand() exit = %d, want %d", exitCode, exitcodes.ACPExitPrereq)
	}

	if got := readOutputFile(t, stderrPath); !strings.Contains(got, "invalid env line") && !strings.Contains(got, "invalid env key") {
		t.Fatalf("stderr = %q, want invalid env error", got)
	}
}

func writeEnvCommandFixture(t *testing.T, content string) string {
	t.Helper()

	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return envPath
}

func tempOutputFile(t *testing.T) (*os.File, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "output.txt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = file.Close()
	})
	return file, path
}

func readOutputFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	return string(data)
}
