// command_observer_helpers_test.go - Tests for shared runtime observer helpers.
//
// Purpose:
//   - Lock the shared runtime observer helpers to consistent repo-root and
//     timeout handling.
//
// Responsibilities:
//   - Cover repository-root prerequisite failures.
//   - Cover timeout/cancel propagation for shared runtime collection.
//
// Scope:
//   - Shared observer helper behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests use stubbed inspectors and captured temp files only.
package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type blockingRuntimeInspector struct{}

func (blockingRuntimeInspector) Collect(ctx context.Context, _ status.Options) status.StatusReport {
	<-ctx.Done()
	return status.StatusReport{}
}

func (blockingRuntimeInspector) Close() error {
	return nil
}

func TestOpenRuntimeStatusInspectorRequiresRepoRoot(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	logger := logging.New(logging.Options{Writer: io.Discard})
	inspector, code := openRuntimeStatusInspector(commandRunContext{
		Stdout: stdout,
		Stderr: stderr,
		Logger: logger,
	}, logger, output.New(), newRuntimeStatusInspector)

	if inspector != nil {
		t.Fatal("expected nil inspector when repo root is missing")
	}
	if code != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit, got %d", code)
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Failed to detect repository root") {
		t.Fatalf("expected repo root error, got %q", got)
	}
}

func TestCollectRuntimeReportOrExitReturnsRuntimeOnTimeout(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	runCtx := commandRunContext{
		RepoRoot: t.TempDir(),
		Stdout:   stdout,
		Stderr:   stderr,
		Logger:   logging.New(logging.Options{Writer: io.Discard}),
	}

	report, code, ok := collectRuntimeReportOrExit(context.Background(), runCtx, runCtx.Logger, output.New(), blockingRuntimeInspector{}, runtimeReportCommandConfig{
		Timeout:         10 * time.Millisecond,
		TimeoutMessage:  "observer timed out",
		CanceledMessage: "observer canceled",
	})

	if ok {
		t.Fatalf("expected timeout path, got ok with report %+v", report)
	}
	if code != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit, got %d", code)
	}
	if got := readFile(t, stderr); !strings.Contains(got, "observer timed out") {
		t.Fatalf("expected timeout message, got %s", got)
	}
}
