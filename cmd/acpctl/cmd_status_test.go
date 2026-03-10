// cmd_status_test.go - Tests for the runtime status command.
//
// Purpose:
//   - Verify the shared status command path keeps watch-mode behavior aligned
//     with the consolidated runtime observer helpers.
//
// Responsibilities:
//   - Cover canceled watch-mode execution.
//
// Scope:
//   - Status command behavior only.
//
// Usage:
//   - Run with `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stub the shared runtime inspector instead of requiring a live runtime.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunStatusWatchReturnsRuntimeWhenCollectionCanceled(t *testing.T) {
	originalInspector := newStatusInspector
	newStatusInspector = func(string) runtimeStatusInspector {
		return blockingRuntimeInspector{}
	}
	defer func() {
		newStatusInspector = originalInspector
	}()

	stdout, stderr := newTestFiles(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exitCode := withRepoRoot(t, t.TempDir(), func() int {
		return runTestCommand(t, ctx, stdout, stderr, "status", "--watch", "--interval", "1")
	})

	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit, got %d stdout=%s stderr=%s", exitCode, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "Status check canceled") {
		t.Fatalf("expected canceled output, got %s", got)
	}
}
