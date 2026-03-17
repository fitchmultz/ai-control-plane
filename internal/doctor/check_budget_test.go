// check_budget_test.go - Coverage for budget doctor check adaptation.
//
// Purpose:
//   - Verify budget findings map directly from runtime status into doctor output.
//
// Responsibilities:
//   - Preserve level, severity, and shared detail propagation.
//
// Scope:
//   - Budget finding doctor check behavior only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic runtime reports.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestBudgetFindingsCheckRunPassthrough(t *testing.T) {
	result := (budgetFindingsCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("budget", status.ComponentDetails{
			TotalBudgets:           2,
			HighUtilizationBudgets: 1,
		}, status.HealthLevelWarning, "2 budgets, 1 >80% utilized"),
	})

	if result.Level != status.HealthLevelWarning || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}
