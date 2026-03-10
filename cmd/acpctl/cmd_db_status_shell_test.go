// cmd_db_status_shell_test.go - Tests for typed database status and shell flows.
//
// Purpose:
//   - Cover the native `db status` and `db shell` implementations without
//     depending on live Docker or PostgreSQL services.
//
// Responsibilities:
//   - Verify status output preserves the documented section contract.
//   - Verify unreachable and degraded database states report truthfully.
//   - Verify shell invocation selection and exit-code mapping remain stable.
//
// Scope:
//   - `cmd/acpctl` database status and shell command behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub command-layer dependencies instead of using live runtime state.
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

type fakeRuntimeSummaryReader struct {
	summary db.Summary
	err     error
}

func (f fakeRuntimeSummaryReader) Summary(context.Context) (db.Summary, error) {
	return f.summary, f.err
}
func (f fakeRuntimeSummaryReader) ConfigError() error { return nil }

type fakeReadonlySummaryReader struct {
	keySummary       db.KeySummary
	keyErr           error
	budgetSummary    db.BudgetSummary
	budgetErr        error
	detectionSummary db.DetectionSummary
	detectionErr     error
}

func (f fakeReadonlySummaryReader) KeySummary(context.Context) (db.KeySummary, error) {
	return f.keySummary, f.keyErr
}

func (f fakeReadonlySummaryReader) BudgetSummary(context.Context) (db.BudgetSummary, error) {
	return f.budgetSummary, f.budgetErr
}

func (f fakeReadonlySummaryReader) DetectionSummary(context.Context) (db.DetectionSummary, error) {
	return f.detectionSummary, f.detectionErr
}

func TestRunDBStatusRendersExpectedSections(t *testing.T) {
	restore := stubDBStatusReaders(t, dbStatusReaders{
		Mode: "embedded",
		Runtime: fakeRuntimeSummaryReader{
			summary: db.Summary{
				Mode:           config.DatabaseModeEmbedded,
				DatabaseName:   "litellm",
				DatabaseUser:   "litellm",
				ContainerID:    "postgres-test",
				Ping:           db.Probe{Healthy: true},
				ExpectedTables: 4,
				Version:        "PostgreSQL 16.0",
				Size:           "24 MB",
				Connections:    3,
			},
		},
		Readonly: fakeReadonlySummaryReader{
			keySummary:       db.KeySummary{Total: 4, Active: 3, Expired: 1},
			budgetSummary:    db.BudgetSummary{Total: 2, HighUtilization: 1, Exhausted: 0},
			detectionSummary: db.DetectionSummary{SpendLogsTableExists: true, HighSeverity: 2, MediumSeverity: 1, UniqueModels24h: 5, TotalEntries24h: 12},
		},
	})
	defer restore()

	stdout, stderr := newCommandOutputFiles(t)
	exitCode := runDBStatus(context.Background(), commandRunContext{Stdout: stdout, Stderr: stderr}, struct{}{})
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("runDBStatus() exit = %d, want %d", exitCode, exitcodes.ACPExitSuccess)
	}

	output := readDBCommandOutput(t, stdout)
	for _, want := range []string{
		"1. Runtime Summary",
		"2. Schema Verification",
		"3. Virtual Keys",
		"4. Budget Usage",
		"5. Detection Summary",
		"Connectivity:",
		"reachable",
		"postgres-test",
		"High severity (24h):",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestRunDBStatusFailsWhenRuntimeUnavailable(t *testing.T) {
	restore := stubDBStatusReaders(t, dbStatusReaders{
		Mode: "embedded",
		Runtime: fakeRuntimeSummaryReader{
			summary: db.Summary{
				Ping: db.Probe{Healthy: false, Error: "database ping failed: connection refused"},
			},
			err: errors.New("database ping failed: connection refused"),
		},
		Readonly: fakeReadonlySummaryReader{},
	})
	defer restore()

	stdout, stderr := newCommandOutputFiles(t)
	exitCode := runDBStatus(context.Background(), commandRunContext{Stdout: stdout, Stderr: stderr}, struct{}{})
	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("runDBStatus() exit = %d, want %d", exitCode, exitcodes.ACPExitPrereq)
	}
	if got := readDBCommandOutput(t, stderr); !strings.Contains(got, "connection refused") {
		t.Fatalf("stderr = %q, want connection guidance", got)
	}
}

func TestRunDBStatusReportsDegradedSchemaAndMissingSpendLogs(t *testing.T) {
	restore := stubDBStatusReaders(t, dbStatusReaders{
		Mode: "embedded",
		Runtime: fakeRuntimeSummaryReader{
			summary: db.Summary{
				Mode:           config.DatabaseModeEmbedded,
				DatabaseName:   "litellm",
				DatabaseUser:   "litellm",
				Ping:           db.Probe{Healthy: true},
				ExpectedTables: 3,
				Version:        "PostgreSQL 16.0",
				Size:           "24 MB",
				Connections:    2,
			},
		},
		Readonly: fakeReadonlySummaryReader{
			keyErr:           errors.New(`database query failed: relation "LiteLLM_VerificationToken" does not exist`),
			budgetSummary:    db.BudgetSummary{Total: 1, HighUtilization: 0, Exhausted: 0},
			detectionSummary: db.DetectionSummary{SpendLogsTableExists: false},
		},
	})
	defer restore()

	stdout, stderr := newCommandOutputFiles(t)
	exitCode := runDBStatus(context.Background(), commandRunContext{Stdout: stdout, Stderr: stderr}, struct{}{})
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("runDBStatus() exit = %d, want %d", exitCode, exitcodes.ACPExitSuccess)
	}

	output := readDBCommandOutput(t, stdout)
	for _, want := range []string{
		"schema incomplete; LiteLLM initialization may still be in progress",
		`unavailable (database query failed: relation "LiteLLM_VerificationToken" does not exist)`,
		"LiteLLM_SpendLogs not present; detection metrics are not available yet",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
	if got := readDBCommandOutput(t, stderr); strings.TrimSpace(got) != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestDefaultBuildDBShellInvocationEmbeddedBuildsDockerExec(t *testing.T) {
	fakeBin := t.TempDir()
	installFakeDockerCLI(t, fakeBin)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ACP_DATABASE_MODE", "embedded")
	t.Setenv("DATABASE_URL", "")

	stdout, stderr := newCommandOutputFiles(t)
	invocation, err := defaultBuildDBShellInvocation(context.Background(), commandRunContext{
		RepoRoot: t.TempDir(),
		Stdout:   stdout,
		Stderr:   stderr,
	})
	if err != nil {
		t.Fatalf("defaultBuildDBShellInvocation() error = %v", err)
	}
	if invocation.Request.Name != "docker" {
		t.Fatalf("request name = %q, want docker", invocation.Request.Name)
	}
	if got := strings.Join(invocation.Request.Args, " "); !strings.Contains(got, "exec -i container-postgres psql -X -U litellm -d litellm") {
		t.Fatalf("docker args = %q", got)
	}
}

func TestDefaultBuildDBShellInvocationExternalRequiresDatabaseURL(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "external")
	t.Setenv("DATABASE_URL", "")

	stdout, stderr := newCommandOutputFiles(t)
	_, err := defaultBuildDBShellInvocation(context.Background(), commandRunContext{
		RepoRoot: t.TempDir(),
		Stdout:   stdout,
		Stderr:   stderr,
	})
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL not set") {
		t.Fatalf("defaultBuildDBShellInvocation() error = %v, want DATABASE_URL guidance", err)
	}
}

func TestDefaultBuildDBShellInvocationExternalRequiresPsql(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ACP_DATABASE_MODE", "external")
	t.Setenv("DATABASE_URL", "postgres://db.example.test/litellm")

	stdout, stderr := newCommandOutputFiles(t)
	_, err := defaultBuildDBShellInvocation(context.Background(), commandRunContext{
		RepoRoot: t.TempDir(),
		Stdout:   stdout,
		Stderr:   stderr,
	})
	if err == nil || !strings.Contains(err.Error(), "psql is required") {
		t.Fatalf("defaultBuildDBShellInvocation() error = %v, want psql guidance", err)
	}
}

func TestRunDBShellMapsAttachedProcessErrors(t *testing.T) {
	restoreBuild := stubDBShellBuilder(t, func(context.Context, commandRunContext) (dbShellInvocation, error) {
		return dbShellInvocation{
			Request: proc.Request{Name: "psql"},
			Mode:    "external",
			Target:  "postgres://db.example.test/litellm",
		}, nil
	})
	defer restoreBuild()

	restoreRun := stubAttachedRunner(t, func(context.Context, proc.Request) proc.Result {
		return proc.Result{
			Err: &proc.ExecError{
				Name:     "psql",
				Kind:     proc.KindExit,
				ExitCode: 2,
				Err:      errors.New("exit status 2"),
			},
			ExitCode: 2,
		}
	})
	defer restoreRun()

	stdout, stderr := newCommandOutputFiles(t)
	exitCode := runDBShell(context.Background(), commandRunContext{Stdout: stdout, Stderr: stderr}, struct{}{})
	if exitCode != exitcodes.ACPExitPrereq {
		t.Fatalf("runDBShell() exit = %d, want %d", exitCode, exitcodes.ACPExitPrereq)
	}
	if got := readDBCommandOutput(t, stderr); !strings.Contains(got, "psql exited with status 2") {
		t.Fatalf("stderr = %q, want subprocess failure", got)
	}
}

func stubDBStatusReaders(t *testing.T, readers dbStatusReaders) func() {
	t.Helper()
	original := openDBStatusReaders
	openDBStatusReaders = func(string) (dbStatusReaders, error) {
		return readers, nil
	}
	return func() {
		openDBStatusReaders = original
	}
}

func stubDBShellBuilder(t *testing.T, fn func(context.Context, commandRunContext) (dbShellInvocation, error)) func() {
	t.Helper()
	original := buildDBShellInvocation
	buildDBShellInvocation = fn
	return func() {
		buildDBShellInvocation = original
	}
}

func stubAttachedRunner(t *testing.T, fn func(context.Context, proc.Request) proc.Result) func() {
	t.Helper()
	original := runAttachedSubprocess
	runAttachedSubprocess = fn
	return func() {
		runAttachedSubprocess = original
	}
}

func newCommandOutputFiles(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	stdout, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("CreateTemp(stdout) error = %v", err)
	}
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatalf("CreateTemp(stderr) error = %v", err)
	}
	return stdout, stderr
}

func readDBCommandOutput(t *testing.T, file *os.File) string {
	t.Helper()
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("Seek(%q) error = %v", file.Name(), err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", file.Name(), err)
	}
	return string(data)
}

func installFakeDockerCLI(t *testing.T, binDir string) {
	t.Helper()
	script := `#!/bin/sh
set -eu
case "$*" in
  "compose version")
    printf 'Docker Compose version v2.0.0\n'
    ;;
  *" ps -q postgres")
    printf 'container-postgres\n'
    ;;
  *)
    printf 'unexpected docker args: %s\n' "$*" >&2
    exit 1
    ;;
esac
`
	path := filepath.Join(binDir, "docker")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
