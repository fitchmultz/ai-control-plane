// completion_test_helpers_test.go - Shared helpers for completion unit tests.
//
// Purpose:
//   - Centralize temporary repo/config setup for focused completion suites.
//
// Responsibilities:
//   - Capture generated shell scripts for renderer tests.
//   - Materialize tracked config fixtures for extractor tests.
//
// Scope:
//   - Test-only helpers for cmd/acpctl completion suites.
//
// Usage:
//   - Imported implicitly by other completion `_test.go` files.
//
// Invariants/Assumptions:
//   - Helpers remain deterministic for equivalent temp repo fixtures.
package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func captureCompletionScript(t *testing.T, shell string) string {
	t.Helper()
	stdout, stderr := newTestFiles(t)

	if exitCode := run(context.Background(), []string{"completion", shell}, stdout, stderr); exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("completion %s failed: %d stderr=%s", shell, exitCode, readFile(t, stderr))
	}

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatalf("seek stdout: %v", err)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(stdout); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}

func writeCompletionConfigFile(t *testing.T, repoRoot string, relativePath string, contents string) string {
	t.Helper()
	return testutil.WriteRepoFile(t, repoRoot, relativePath, contents)
}
