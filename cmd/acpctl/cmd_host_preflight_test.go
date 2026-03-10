// cmd_host_preflight_test.go - Tests for the native host preflight command.
//
// Purpose:
//   - Lock the typed host preflight command to the expected local prerequisite
//     and production-validation contract.
//
// Responsibilities:
//   - Cover successful host preflight execution.
//   - Cover unsupported profile usage.
//   - Cover missing local prerequisites before validation runs.
//
// Scope:
//   - Host preflight command behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub local binaries instead of requiring host tools.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunHostPreflightSucceedsForCanonicalProductionFixture(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	writeFile(t, filepath.Join(repoRoot, "deploy", "systemd", "ai-control-plane.service.tmpl"), "[Unit]\nDescription=ACP\n")
	writeFile(t, filepath.Join(repoRoot, "demo", ".env"), "ACP_DATABASE_MODE=embedded\n")
	secretsPath := filepath.Join(repoRoot, "local", "host-preflight", "secrets.env")
	writeFileWithMode(t, secretsPath, ""+
		"ACP_DATABASE_MODE=external\n"+
		"CADDY_PUBLISH_HOST=0.0.0.0\n"+
		"CADDYFILE_PATH=./config/caddy/Caddyfile.prod\n"+
		"CADDY_ACME_CA=letsencrypt\n"+
		"CADDY_DOMAIN=gateway.example.com\n"+
		"CADDY_EMAIL=ops@example.com\n"+
		"DATABASE_URL=postgresql://app:verysecurepassword@db.example.com:5432/acp?sslmode=require\n"+
		"LITELLM_MASTER_KEY=prod-master-token-abcdefghijklmnopqrstuvwxyz1234567890\n"+
		"LITELLM_PUBLISH_HOST=127.0.0.1\n"+
		"LITELLM_PUBLIC_URL=https://gateway.example.com\n"+
		"LITELLM_SALT_KEY=prod-salt-token-abcdefghijklmnopqrstuvwxyz1234567890\n"+
		"OTEL_INGEST_AUTH_TOKEN=otel-ingest-auth-token-abcdefghijklmnopqrstuvwxyz\n", 0o600)

	restore := stubHostPreflightPrereqs(t)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "host", "preflight", "--secrets-env-file", secretsPath)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stdout); !strings.Contains(got, "Host preflight passed") {
		t.Fatalf("expected success output, got %s", got)
	}
}

func TestRunHostPreflightRejectsUnsupportedProfile(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := runTestCommand(t, context.Background(), stdout, stderr, "host", "preflight", "--profile", "demo")
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "unsupported profile: demo") {
		t.Fatalf("expected unsupported profile error, got %s", got)
	}
}

func TestRunHostPreflightFailsWhenSystemctlMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	writeFile(t, filepath.Join(repoRoot, "deploy", "systemd", "ai-control-plane.service.tmpl"), "[Unit]\nDescription=ACP\n")
	writeFile(t, filepath.Join(repoRoot, "demo", ".env"), "ACP_DATABASE_MODE=embedded\n")
	secretsPath := filepath.Join(repoRoot, "local", "host-preflight", "secrets.env")
	writeFileWithMode(t, secretsPath, "LITELLM_MASTER_KEY=prod-master-token-abcdefghijklmnopqrstuvwxyz1234567890\n", 0o600)

	restore := stubHostPreflightPrereqsWithoutSystemctl(t)
	defer restore()

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr, "host", "preflight", "--secrets-env-file", secretsPath)
	})

	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("expected prereq exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "required command not found: systemctl") {
		t.Fatalf("expected systemctl error, got %s", got)
	}
}

func stubHostPreflightPrereqs(t *testing.T) func() {
	t.Helper()
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "docker"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(binDir, "make"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(binDir, "systemctl"), "#!/bin/sh\nexit 0\n")
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
	return func() {}
}

func stubHostPreflightPrereqsWithoutSystemctl(t *testing.T) func() {
	t.Helper()
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "docker"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(binDir, "make"), "#!/bin/sh\nexit 0\n")
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath)
	return func() {}
}

func writeExecutable(t *testing.T, path string, contents string) {
	t.Helper()
	writeFileWithMode(t, path, contents, 0o755)
}

func writeFileWithMode(t *testing.T, path string, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
