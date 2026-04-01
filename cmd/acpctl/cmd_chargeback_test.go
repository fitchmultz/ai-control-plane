// cmd_chargeback_test.go - Tests for the chargeback command surface.
//
// Purpose:
//   - Verify tenant-scoped chargeback command validation.
//
// Responsibilities:
//   - Cover CLI validation for partial tenant-scope flags on chargeback report.
//
// Scope:
//   - Command-layer chargeback report behavior only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl`.
//
// Invariants/Assumptions:
//   - Tests stay validation-focused and do not require a live database.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestChargebackReportRejectsWorkspaceWithoutOrganization(t *testing.T) {
	repoRoot := t.TempDir()
	writeTenantCommandFixtureRepo(t, repoRoot)
	stdout, stderr := newTestFiles(t)

	code := withRepoRoot(t, repoRoot, func() int {
		return runTestCommand(t, context.Background(), stdout, stderr,
			"chargeback", "report",
			"--workspace", "claims-adjuster",
		)
	})
	if code != exitcodes.ACPExitUsage {
		t.Fatalf("runTestCommand(chargeback report partial scope) exit = %d stdout=%s stderr=%s", code, readFile(t, stdout), readFile(t, stderr))
	}
	if got := readFile(t, stderr); !strings.Contains(got, "--organization is required") {
		t.Fatalf("stderr = %q", got)
	}
}
