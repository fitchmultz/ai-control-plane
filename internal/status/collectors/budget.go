// budget.go implements the LiteLLM budget status collector.
//
// Purpose:
//
//	Inspect configured budgets in LiteLLM and report remaining-capacity health
//	using the repository's operator-facing status model.
//
// Responsibilities:
//   - Resolve the PostgreSQL container for budget queries.
//   - Count total, high-utilization, and exhausted budgets.
//   - Surface actionable health messages based on remaining budget semantics.
//
// Scope:
//   - Covers budget inventory and remaining-capacity analysis only.
//
// Usage:
//   - Construct `NewBudgetCollector(repoRoot)` and call `Collect(ctx)`.
//
// Invariants/Assumptions:
//   - LiteLLM stores remaining budget in `LiteLLM_BudgetTable.budget`, not spent amount.
//   - Remaining budget percentage is calculated from `budget / max_budget`.
package collectors

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

// BudgetCollector checks budget utilization.
type BudgetCollector struct {
	RepoRoot string
	runner   runner.Runner
	compose  containerIDResolver
}

// NewBudgetCollector creates a new budget collector
func NewBudgetCollector(repoRoot string) *BudgetCollector {
	return &BudgetCollector{
		RepoRoot: repoRoot,
		runner:   newCollectorRunner(repoRoot),
		compose:  newCollectorCompose(repoRoot),
	}
}

// SetRunner sets a custom runner (for testing)
func (c *BudgetCollector) SetRunner(r runner.Runner) {
	c.runner = r
}

// SetContainerResolver sets a custom container resolver (for testing)
func (c *BudgetCollector) SetContainerResolver(resolver containerIDResolver) {
	c.compose = resolver
}

// Name returns the collector's domain name.
func (c BudgetCollector) Name() string {
	return "budget"
}

// Collect gathers budget status information.
func (c BudgetCollector) Collect(ctx context.Context) status.ComponentStatus {
	// Check docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Docker not available",
		}
	}

	runtime := resolveCollectorRuntime(c.RepoRoot, c.runner, c.compose)
	containerID, err := resolvePostgresContainer(ctx, runtime)
	if err != nil {
		details := make(map[string]any)
		details["lookup_error"] = runner.SanitizeForDisplay(err.Error())
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "PostgreSQL unavailable",
			Details: details,
		}
	}

	// Get total budget count
	countQuery := `SELECT COUNT(*) FROM "LiteLLM_BudgetTable";`
	countResult := runPostgresQuery(ctx, runtime, containerID, countQuery)

	if countResult.Error != nil {
		details := map[string]any{
			"exit_code": countResult.ExitCode,
		}
		if countResult.Stderr != "" {
			details["stderr"] = runner.SanitizeForDisplay(countResult.Stderr)
		}
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Could not query budget count",
			Details: details,
			Suggestions: []string{
				"Table may not exist yet - LiteLLM creates tables on first use",
				fmt.Sprintf("Query error: %s", runner.SanitizeForDisplay(countResult.Stderr)),
			},
		}
	}

	countStr := strings.TrimSpace(countResult.Stdout)
	totalCount, err := strconv.Atoi(countStr)
	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Failed to parse budget count",
			Details: map[string]any{
				"raw_output":  countResult.Stdout,
				"parse_error": err.Error(),
			},
		}
	}

	details := map[string]any{
		"total_budgets": totalCount,
	}

	if totalCount == 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelHealthy,
			Message: "No budgets configured",
			Details: details,
		}
	}

	// Check for budgets with high utilization (>80% used, i.e., <20% remaining).
	// Using <= 20 to include budgets exactly at the 20% remaining threshold.
	highUtilQuery := `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE max_budget > 0 AND (budget::float / max_budget::float * 100) <= 20;
	`
	highUtilResult := runPostgresQuery(ctx, runtime, containerID, highUtilQuery)

	highUtilCount := 0
	if highUtilResult.Error == nil {
		highUtilCount, _ = strconv.Atoi(strings.TrimSpace(highUtilResult.Stdout))
		details["high_utilization_budgets"] = highUtilCount
	} else if highUtilResult.Stderr != "" {
		details["high_util_query_error"] = runner.SanitizeForDisplay(highUtilResult.Stderr)
	}

	// Check for exhausted budgets (no remaining budget).
	// budget <= 0 means 0% or negative remaining (100%+ used).
	exhaustedQuery := `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE budget <= 0;
	`
	exhaustedResult := runPostgresQuery(ctx, runtime, containerID, exhaustedQuery)

	exhaustedCount := 0
	if exhaustedResult.Error == nil {
		exhaustedCount, _ = strconv.Atoi(strings.TrimSpace(exhaustedResult.Stdout))
		details["exhausted_budgets"] = exhaustedCount
	} else if exhaustedResult.Stderr != "" {
		details["exhausted_query_error"] = runner.SanitizeForDisplay(exhaustedResult.Stderr)
	}

	// Determine status level
	if exhaustedCount > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("%d budgets, %d exhausted", totalCount, exhaustedCount),
			Details: details,
			Suggestions: []string{
				"Review exhausted budgets: acpctl db status",
				"Increase budget or create new key with higher limit",
			},
		}
	}

	if highUtilCount > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("%d budgets, %d >80%% utilized", totalCount, highUtilCount),
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
		Message: fmt.Sprintf("%d budgets, all healthy", totalCount),
		Details: details,
	}
}
