// runner_test.go - Deterministic tests for the status command runner.
//
// Purpose:
//
//	Verify the runner adapter preserves stdout/stderr capture and explicit
//	timeout/cancel classification without depending on host shell binaries.
//
// Responsibilities:
//   - Test MockRunner behavior.
//   - Test DefaultRunner subprocess execution using helper-process fixtures.
//   - Test timeout, cancel, and exit-status classification.
//   - Test output sanitization.
//
// Scope:
//   - Covers the internal/status/runner package only.
//
// Usage:
//   - Run via `go test ./internal/status/runner`.
//
// Invariants/Assumptions:
//   - Helper-process tests re-exec the current Go test binary.
//   - No test depends on host `sleep`, `echo`, or `false`.
package runner

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunnerHelperProcess(t *testing.T) {
	t.Helper()
	sep := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		return
	}

	switch os.Args[sep+1] {
	case "stdout-stderr":
		fmt.Fprint(os.Stdout, os.Args[sep+2])
		fmt.Fprint(os.Stderr, os.Args[sep+3])
		os.Exit(0)
	case "exit":
		code, _ := strconv.Atoi(os.Args[sep+2])
		fmt.Fprint(os.Stderr, "failed")
		os.Exit(code)
	case "sleep":
		d, _ := time.ParseDuration(os.Args[sep+2])
		time.Sleep(d)
		os.Exit(0)
	case "block":
		time.Sleep(24 * time.Hour)
	default:
		os.Exit(99)
	}
}

func helperArgs(mode string, args ...string) []string {
	return append([]string{"-test.run=TestRunnerHelperProcess", "--", mode}, args...)
}

func TestMockRunner(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("echo hello", &Result{
		Stdout:   "hello\n",
		ExitCode: 0,
	})

	result := mock.Run(context.Background(), "echo", "hello")
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("Expected 'hello\\n', got %q", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestMockRunner_NoResponse(t *testing.T) {
	mock := NewMockRunner()

	result := mock.Run(context.Background(), "unknown", "command")
	if result.Error == nil {
		t.Error("Expected error for missing mock response")
	}
	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1, got %d", result.ExitCode)
	}
}

func TestMockRunner_ErrorResponse(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("failing command", &Result{
		Stderr:   "error message",
		ExitCode: 1,
		Error:    context.DeadlineExceeded,
	})

	result := mock.Run(context.Background(), "failing", "command")
	if result.Error == nil {
		t.Error("Expected error")
	}
	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr != "error message" {
		t.Errorf("Expected stderr 'error message', got %q", result.Stderr)
	}
}

func TestSanitizeForDisplay(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "normal output\n", expected: "normal output\n"},
		{input: "password=secret123\n", expected: "[REDACTED]\n"},
		{input: "SECRET_KEY=abc123\n", expected: "[REDACTED]\n"},
		{input: "token=Bearer xyz\n", expected: "[REDACTED]\n"},
		{input: "api_key=12345\n", expected: "[REDACTED]\n"},
		{input: "error: connection failed\n", expected: "error: connection failed\n"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeForDisplay(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForDisplay(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultRunner_WithWorkDir(t *testing.T) {
	runner := NewDefaultRunner("/tmp")
	if runner.WorkDir != "/tmp" {
		t.Errorf("Expected WorkDir '/tmp', got %q", runner.WorkDir)
	}
}

func TestDefaultRunner_Integration(t *testing.T) {
	runner := NewDefaultRunner("")

	result := runner.Run(context.Background(), os.Args[0], helperArgs("stdout-stderr", "test-out", "test-err")...)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Stdout != "test-out" {
		t.Fatalf("stdout = %q, want test-out", result.Stdout)
	}
	if result.Stderr != "test-err" {
		t.Fatalf("stderr = %q, want test-err", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestDefaultRunner_FailingCommand(t *testing.T) {
	runner := NewDefaultRunner("")

	result := runner.Run(context.Background(), os.Args[0], helperArgs("exit", "7")...)
	if result.Error == nil {
		t.Fatal("expected error")
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", result.ExitCode)
	}
}

func TestDefaultRunner_Timeout(t *testing.T) {
	runner := NewDefaultRunner("")
	runner.Timeout = 20 * time.Millisecond

	result := runner.Run(context.Background(), os.Args[0], helperArgs("block")...)
	if result.Error == nil {
		t.Fatalf("expected timeout error, got %+v", result)
	}
	if !result.TimedOut {
		t.Fatalf("expected timeout result, got %+v", result)
	}
	if result.Canceled {
		t.Fatalf("expected timeout, not canceled: %+v", result)
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit code = %d, want -1", result.ExitCode)
	}
}

func TestDefaultRunner_Canceled(t *testing.T) {
	runner := NewDefaultRunner("")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := runner.Run(ctx, os.Args[0], helperArgs("block")...)
	if result.Error == nil {
		t.Fatalf("expected canceled error, got %+v", result)
	}
	if !result.Canceled {
		t.Fatalf("expected canceled result, got %+v", result)
	}
	if result.TimedOut {
		t.Fatalf("expected canceled result, not timeout: %+v", result)
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit code = %d, want -1", result.ExitCode)
	}
}

func TestResult_Struct(t *testing.T) {
	result := &Result{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 42,
	}

	if result.Stdout != "output" {
		t.Error("Stdout field incorrect")
	}
	if result.Stderr != "error" {
		t.Error("Stderr field incorrect")
	}
	if result.ExitCode != 42 {
		t.Error("ExitCode field incorrect")
	}
}

func TestMockRunner_PrefixMatch(t *testing.T) {
	mock := NewMockRunner()
	pattern := strings.Repeat("a", 120)
	mock.SetResponse(pattern, &Result{Stdout: "matched"})

	result := mock.Run(context.Background(), strings.Repeat("a", 120))
	if result.Stdout != "matched" {
		t.Fatalf("expected prefix match result, got %+v", result)
	}
}
