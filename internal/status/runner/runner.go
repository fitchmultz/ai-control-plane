// runner.go - Command execution abstraction for status collectors
//
// Purpose: Provide an interface for command execution that can be mocked in tests
//
// Responsibilities:
//   - Execute commands with timeout context
//   - Return stdout, stderr, and exit codes
//   - Allow mocking for unit tests
//
// Non-scope:
//   - Does not retry failed commands
//   - Does not parse command output
package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Result contains the outcome of a command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// Runner executes commands
type Runner interface {
	Run(ctx context.Context, name string, arg ...string) *Result
}

// DefaultRunner is the production command runner
type DefaultRunner struct {
	WorkDir string
}

// NewDefaultRunner creates a new default runner
func NewDefaultRunner(workDir string) *DefaultRunner {
	return &DefaultRunner{WorkDir: workDir}
}

// Run executes a command and returns the result
func (r *DefaultRunner) Run(ctx context.Context, name string, arg ...string) *Result {
	// Ensure context has timeout
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, arg...)
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	// Use CombinedOutput to capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	result := &Result{
		Stdout: string(output),
	}

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// CombinedOutput already includes stderr in output
			result.Stderr = string(exitErr.Stderr)
		} else {
			result.ExitCode = -1
		}
	}

	return result
}

// MockRunner is a test double for command execution
type MockRunner struct {
	Responses map[string]*Result
}

// NewMockRunner creates a new mock runner
func NewMockRunner() *MockRunner {
	return &MockRunner{
		Responses: make(map[string]*Result),
	}
}

// SetResponse configures a mock response for a command
func (m *MockRunner) SetResponse(command string, result *Result) {
	m.Responses[normalizeWhitespace(command)] = result
}

// Run returns the mocked result for a command
func (m *MockRunner) Run(ctx context.Context, name string, arg ...string) *Result {
	cmd := name + " " + strings.Join(arg, " ")
	normalizedCmd := normalizeWhitespace(cmd)

	// Try exact match first
	if result, ok := m.Responses[normalizedCmd]; ok {
		return result
	}

	// Try prefix match for SQL queries (match first 100 chars)
	if len(normalizedCmd) > 100 {
		prefix := normalizedCmd[:100]
		for pattern, result := range m.Responses {
			if len(pattern) > 100 && pattern[:100] == prefix {
				return result
			}
		}
	}

	return &Result{
		Error:    fmt.Errorf("no mock response for: %s", normalizedCmd),
		ExitCode: -1,
	}
}

// normalizeWhitespace collapses multiple whitespace characters into single spaces
func normalizeWhitespace(s string) string {
	var result strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				result.WriteRune(' ')
				inSpace = true
			}
		} else {
			result.WriteRune(r)
			inSpace = false
		}
	}
	return strings.TrimSpace(result.String())
}

// SanitizeForDisplay removes sensitive information from output for display
func SanitizeForDisplay(output string) string {
	// Remove potential passwords or keys
	lines := strings.Split(output, "\n")
	var sanitized []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "key=") ||
			strings.Contains(lower, "token=") {
			sanitized = append(sanitized, "[REDACTED]")
		} else {
			sanitized = append(sanitized, line)
		}
	}
	return strings.Join(sanitized, "\n")
}
