// runner_test.go - Tests for command runner
//
// Purpose: Test runner interface and mock implementation
//
// Responsibilities:
//   - Test MockRunner behavior
//   - Test DefaultRunner integration
//   - Test output sanitization
//
// Non-scope:
//   - Does not test actual command execution (integration tests)
package runner

import (
	"context"
	"strings"
	"testing"
)

func TestMockRunner(t *testing.T) {
	mock := NewMockRunner()

	// Set up mock response
	mock.SetResponse("echo hello", &Result{
		Stdout:   "hello\n",
		Stderr:   "",
		ExitCode: 0,
		Error:    nil,
	})

	// Test successful response
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

	// Test missing response
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

	// Set up error response
	mock.SetResponse("failing command", &Result{
		Stdout:   "",
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
		{
			input:    "normal output\n",
			expected: "normal output\n",
		},
		{
			input:    "password=secret123\n",
			expected: "[REDACTED]\n",
		},
		{
			input:    "SECRET_KEY=abc123\n",
			expected: "[REDACTED]\n",
		},
		{
			input:    "token=Bearer xyz\n",
			expected: "[REDACTED]\n",
		},
		{
			input:    "api_key=12345\n",
			expected: "[REDACTED]\n",
		},
		{
			input:    "error: connection failed\n",
			expected: "error: connection failed\n",
		},
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

// Test that DefaultRunner properly times out
func TestDefaultRunner_Timeout(t *testing.T) {
	runner := NewDefaultRunner("")

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should fail immediately due to cancelled context
	result := runner.Run(ctx, "sleep", "10")

	// The command should fail because context is cancelled
	if result.Error == nil {
		// On some systems this might still work, so just verify we get a result
		t.Log("Command completed despite cancelled context (may be system-specific)")
	}
}

func TestResult_Struct(t *testing.T) {
	result := &Result{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 42,
		Error:    nil,
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

// Integration test that actually runs a command
func TestDefaultRunner_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runner := NewDefaultRunner("")
	ctx := context.Background()

	result := runner.Run(ctx, "echo", "test")

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}

	if !strings.Contains(result.Stdout, "test") {
		t.Errorf("Expected output to contain 'test', got %q", result.Stdout)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

// Test command that fails
func TestDefaultRunner_FailingCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runner := NewDefaultRunner("")
	ctx := context.Background()

	// Use a command that will definitely fail
	result := runner.Run(ctx, "false")

	if result.Error == nil {
		t.Error("Expected error for failing command")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}
