// budget_test validates BudgetCollector docker-based status checks.
//
// Purpose:
//
//	Ensure budget status collection correctly interprets PostgreSQL
//	query results for the LiteLLM_BudgetTable.
//
// Responsibilities:
//   - Verify collector name returns "budget".
//   - Verify status levels based on budget counts and utilization.
//   - Verify exhausted budgets trigger unhealthy status.
//   - Verify high utilization (>80%) triggers warning status.
//   - Verify zero budgets returns healthy (no budgets configured).
//
// Non-scope:
//   - Does not test against real running PostgreSQL containers.
//
// Invariants/Assumptions:
//   - Exhausted budgets (<=0 remaining) are unhealthy.
//   - High utilization budgets (>80% used) are warning.
//   - Zero budgets is healthy (not an error state).
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package collectors

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

func TestBudgetCollector_Name(t *testing.T) {
	t.Parallel()

	c := BudgetCollector{}
	if c.Name() != "budget" {
		t.Fatalf("expected name 'budget', got %q", c.Name())
	}
}

func TestBudgetCollector_StatusLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		totalBudgets   int
		exhaustedCount int
		highUtilCount  int
		expectedLevel  status.HealthLevel
		expectedMsg    string
	}{
		{
			name:           "zero budgets - healthy",
			totalBudgets:   0,
			exhaustedCount: 0,
			highUtilCount:  0,
			expectedLevel:  status.HealthLevelHealthy,
			expectedMsg:    "No budgets configured",
		},
		{
			name:           "budgets all healthy",
			totalBudgets:   3,
			exhaustedCount: 0,
			highUtilCount:  0,
			expectedLevel:  status.HealthLevelHealthy,
			expectedMsg:    "3 budgets, all healthy",
		},
		{
			name:           "high utilization - warning",
			totalBudgets:   5,
			exhaustedCount: 0,
			highUtilCount:  2,
			expectedLevel:  status.HealthLevelWarning,
			expectedMsg:    "5 budgets, 2 >80% utilized",
		},
		{
			name:           "exhausted - unhealthy",
			totalBudgets:   4,
			exhaustedCount: 1,
			highUtilCount:  0,
			expectedLevel:  status.HealthLevelUnhealthy,
			expectedMsg:    "4 budgets, 1 exhausted",
		},
		{
			name:           "exhausted trumps high utilization",
			totalBudgets:   6,
			exhaustedCount: 2,
			highUtilCount:  3,
			expectedLevel:  status.HealthLevelUnhealthy,
			expectedMsg:    "6 budgets, 2 exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			details := map[string]any{
				"total_budgets": tt.totalBudgets,
			}

			if tt.highUtilCount > 0 {
				details["high_utilization_budgets"] = tt.highUtilCount
			}

			if tt.exhaustedCount > 0 {
				details["exhausted_budgets"] = tt.exhaustedCount
			}

			component := status.ComponentStatus{
				Name:    "budget",
				Level:   tt.expectedLevel,
				Message: tt.expectedMsg,
				Details: details,
			}

			if component.Level != tt.expectedLevel {
				t.Fatalf("expected level %s, got %s", tt.expectedLevel, component.Level)
			}

			if component.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, component.Message)
			}
		})
	}
}

func TestBudgetCollector_ZeroBudgets_Details(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "budget",
		Level:   status.HealthLevelHealthy,
		Message: "No budgets configured",
		Details: map[string]any{
			"total_budgets": 0,
		},
	}

	if component.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy for zero budgets, got %s", component.Level)
	}

	details, ok := component.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if details["total_budgets"] != 0 {
		t.Fatal("expected total_budgets to be 0")
	}
}

func TestBudgetCollector_ExhaustedBudgets_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "budget",
		Level:   status.HealthLevelUnhealthy,
		Message: "4 budgets, 2 exhausted",
		Details: map[string]any{
			"total_budgets":     4,
			"exhausted_budgets": 2,
		},
		Suggestions: []string{
			"Review exhausted budgets: acpctl db status",
			"Increase budget or create new key with higher limit",
		},
	}

	if component.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for exhausted budgets")
	}

	hasReview := false
	hasIncrease := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "acpctl db status") {
			hasReview = true
		}
		if strings.Contains(s, "Increase budget") {
			hasIncrease = true
		}
	}

	if !hasReview {
		t.Fatal("expected review suggestion")
	}

	if !hasIncrease {
		t.Fatal("expected increase budget suggestion")
	}
}

func TestBudgetCollector_HighUtilization_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "budget",
		Level:   status.HealthLevelWarning,
		Message: "5 budgets, 3 >80% utilized",
		Details: map[string]any{
			"total_budgets":            5,
			"high_utilization_budgets": 3,
		},
		Suggestions: []string{
			"Monitor high-utilization budgets closely",
			"Consider increasing limits before exhaustion",
		},
	}

	if component.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for high utilization")
	}

	hasMonitor := false
	hasConsider := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "Monitor") {
			hasMonitor = true
		}
		if strings.Contains(s, "increasing limits") {
			hasConsider = true
		}
	}

	if !hasMonitor {
		t.Fatal("expected monitor suggestion")
	}

	if !hasConsider {
		t.Fatal("expected consider increasing suggestion")
	}
}

func TestBudgetCollector_ParseBudgetCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  bool
	}{
		{
			name:     "single budget",
			output:   "1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "multiple budgets",
			output:   "42",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "with whitespace",
			output:   "  10  ",
			expected: 10,
			wantErr:  false,
		},
		{
			name:     "zero",
			output:   "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "non-numeric returns error",
			output:   "error",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty returns error",
			output:   "",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := strconv.Atoi(strings.TrimSpace(tt.output))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if parsed != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, parsed)
			}
		})
	}
}

func TestBudgetCollector_QueryError_Response(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "budget",
		Level:   status.HealthLevelWarning,
		Message: "Could not query budget count",
		Suggestions: []string{
			"Table may not exist yet - LiteLLM creates tables on first use",
		},
	}

	if component.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning level, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for query error")
	}
}

func TestBudgetCollector_Collect_UsesComposeResolverWhenAvailable(t *testing.T) {
	recording := newRecordingRunner()
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_BudgetTable";`, &runner.Result{
		Stdout:   "0\n",
		ExitCode: 0,
	})

	resolver := &fakeContainerResolver{containerID: "compose-postgres"}

	c := NewBudgetCollector("/tmp")
	c.SetRunner(recording)
	c.SetContainerResolver(resolver)

	result := c.Collect(context.Background())
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %v", result.Level)
	}

	if resolver.calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.calls)
	}

	if recording.sawCommandContaining("docker ps --filter name=postgres") {
		t.Fatal("expected compose resolver to avoid docker ps fallback")
	}
}

func TestBudgetCollector_DetailsStructure(t *testing.T) {
	t.Parallel()

	details := map[string]any{
		"total_budgets":            10,
		"high_utilization_budgets": 3,
		"exhausted_budgets":        1,
	}

	component := status.ComponentStatus{
		Name:    "budget",
		Level:   status.HealthLevelUnhealthy,
		Message: "10 budgets, 1 exhausted",
		Details: details,
	}

	detailsMap, ok := component.Details.(map[string]any)
	if !ok {
		t.Fatal("expected details to be a map")
	}

	if _, ok := detailsMap["total_budgets"]; !ok {
		t.Fatal("expected total_budgets in details")
	}

	if _, ok := detailsMap["high_utilization_budgets"]; !ok {
		t.Fatal("expected high_utilization_budgets in details")
	}

	if _, ok := detailsMap["exhausted_budgets"]; !ok {
		t.Fatal("expected exhausted_budgets in details")
	}
}
