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
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

// Result contains the outcome of a command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
	TimedOut bool
	Canceled bool
	NotFound bool
}

// Runner executes commands
type Runner interface {
	Run(ctx context.Context, name string, arg ...string) *Result
}

// DefaultRunner is the production command runner
type DefaultRunner struct {
	WorkDir string
	Timeout time.Duration
}

// NewDefaultRunner creates a new default runner
func NewDefaultRunner(workDir string) *DefaultRunner {
	return &DefaultRunner{
		WorkDir: workDir,
		Timeout: 10 * time.Second,
	}
}

// Run executes a command and returns the result
func (r *DefaultRunner) Run(ctx context.Context, name string, arg ...string) *Result {
	res := proc.Run(ctx, proc.Request{
		Name:    name,
		Args:    arg,
		Dir:     r.WorkDir,
		Timeout: r.Timeout,
	})

	return &Result{
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: res.ExitCode,
		Error:    res.Err,
		TimedOut: proc.IsTimeout(res.Err),
		Canceled: proc.IsCanceled(res.Err),
		NotFound: proc.IsNotFound(res.Err),
	}
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
