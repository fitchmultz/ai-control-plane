// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose a budget health collector that consumes the shared typed database
//	service instead of collector-local SQL command execution.
//
// Responsibilities:
//   - Convert typed budget counts into status.ComponentStatus.
//   - Preserve operator guidance for exhausted and high-utilization budgets.
//
// Non-scope:
//   - Does not execute raw SQL or mutate budget records directly.
//
// Invariants/Assumptions:
//   - Budget counts come from the shared typed database service.
//
// Scope:
//   - Budget status collection only.
//
// Usage:
//   - Construct with NewBudgetCollector(client) and call Collect(ctx).
package collectors

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// BudgetCollector checks budget utilization.
type BudgetCollector struct {
	reader db.ReadonlyServiceReader
}

// NewBudgetCollector creates a new budget collector.
func NewBudgetCollector(reader db.ReadonlyServiceReader) BudgetCollector {
	return BudgetCollector{reader: reader}
}

// Name returns the collector's domain name.
func (c BudgetCollector) Name() string {
	return "budget"
}

// Collect gathers budget status information.
func (c BudgetCollector) Collect(ctx context.Context) status.ComponentStatus {
	summary, err := c.reader.BudgetSummary(ctx)
	details := status.ComponentDetails{
		TotalBudgets:           summary.Total,
		HighUtilizationBudgets: summary.HighUtilization,
		ExhaustedBudgets:       summary.Exhausted,
	}
	if err != nil {
		return readonlyQueryWarning(c.Name(), "Could not query budget count", details, err)
	}

	if summary.Total == 0 {
		return componentStatus(c.Name(), status.HealthLevelHealthy, "No budgets configured", details)
	}

	if summary.Exhausted > 0 {
		return componentStatus(c.Name(), status.HealthLevelUnhealthy, fmt.Sprintf("%d budgets, %d exhausted", summary.Total, summary.Exhausted), details,
			"Review exhausted budgets: acpctl db status",
			"Increase budget or create new key with higher limit",
		)
	}

	if summary.HighUtilization > 0 {
		return componentStatus(c.Name(), status.HealthLevelWarning, fmt.Sprintf("%d budgets, %d >80%% utilized", summary.Total, summary.HighUtilization), details,
			"Monitor high-utilization budgets closely",
			"Consider increasing limits before exhaustion",
		)
	}

	return componentStatus(c.Name(), status.HealthLevelHealthy, fmt.Sprintf("%d budgets, all healthy", summary.Total), details)
}
