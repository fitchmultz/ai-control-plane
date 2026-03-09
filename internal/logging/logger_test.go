// logger_test.go - Tests for structured workflow logger helpers.
//
// Purpose:
//   - Verify the logging package seeds deterministic loggers and context
//     propagation for internal workflow packages.
//
// Responsibilities:
//   - Cover text/json logger construction.
//   - Cover context storage and retrieval helpers.
//   - Verify error-field helper output remains stable.
//
// Scope:
//   - Unit tests for internal/logging only.
//
// Usage:
//   - Run via `go test ./internal/logging`.
//
// Invariants/Assumptions:
//   - Logging helpers remain lightweight and dependency-free.
package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestWithLoggerAndFromContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Options{
		Writer: &buf,
		Format: FormatText,
		Level:  slog.LevelInfo,
		Attrs:  []slog.Attr{slog.String("component", "test")},
	})

	ctx := WithLogger(context.Background(), logger)
	FromContext(ctx).Info("workflow.start")

	output := buf.String()
	if !strings.Contains(output, "workflow.start") {
		t.Fatalf("expected message in log output, got %q", output)
	}
	if !strings.Contains(output, "component=test") {
		t.Fatalf("expected component attr in log output, got %q", output)
	}
}

func TestFromContextFallsBackToNop(t *testing.T) {
	logger := FromContext(context.Background())
	if logger == nil {
		t.Fatal("expected fallback logger")
	}
}

func TestErrHelper(t *testing.T) {
	attr := Err(context.DeadlineExceeded)
	if attr.Key != "error" {
		t.Fatalf("attr key = %q, want error", attr.Key)
	}
	if attr.Value.String() == "" {
		t.Fatal("expected non-empty error string")
	}
}
