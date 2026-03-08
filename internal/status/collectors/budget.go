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
	client *db.Client
}

// NewBudgetCollector creates a new budget collector.
func NewBudgetCollector(client *db.Client) BudgetCollector {
	return BudgetCollector{client: client}
}

// Name returns the collector's domain name.
func (c BudgetCollector) Name() string {
	return "budget"
}

// Collect gathers budget status information.
func (c BudgetCollector) Collect(ctx context.Context) status.ComponentStatus {
	summary, err := c.client.BudgetSummary(ctx)
	details := status.ComponentDetails{
		TotalBudgets:           summary.Total,
		HighUtilizationBudgets: summary.HighUtilization,
		ExhaustedBudgets:       summary.Exhausted,
	}
	if err != nil {
		details.Error = err.Error()
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Could not query budget count",
			Details: details,
			Suggestions: []string{
				"Table may not exist yet - LiteLLM creates tables on first use",
			},
		}
	}

	if summary.Total == 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelHealthy,
			Message: "No budgets configured",
			Details: details,
		}
	}

	if summary.Exhausted > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("%d budgets, %d exhausted", summary.Total, summary.Exhausted),
			Details: details,
			Suggestions: []string{
				"Review exhausted budgets: acpctl db status",
				"Increase budget or create new key with higher limit",
			},
		}
	}

	if summary.HighUtilization > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("%d budgets, %d >80%% utilized", summary.Total, summary.HighUtilization),
			Details: details,
			Suggestions: []string{
				"Monitor high-utilization budgets closely",
				"Consider increasing limits before exhaustion",
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: fmt.Sprintf("%d budgets, all healthy", summary.Total),
		Details: details,
	}
}
