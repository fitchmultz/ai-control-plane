// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Render the finance-facing Markdown chargeback report from typed inputs.
//
// Responsibilities:
//   - Format executive summary, allocation tables, top principals, anomalies,
//     and forecast sections without shell text scraping.
//
// Non-scope:
//   - Does not execute database queries.
//   - Does not emit JSON or CSV formats.
//
// Invariants/Assumptions:
//   - Markdown is deterministic for equivalent inputs.
//   - Rendered values are presentation-friendly rather than machine-oriented.
//
// Scope:
//   - Markdown formatting only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"fmt"
	"strings"
)

func RenderMarkdown(data ReportData) string {
	var builder strings.Builder
	input := data.Input
	coverage := coveragePercent(input.TotalSpend, unattributedSpend(input.CostCenters))
	unattributed := unattributedSpend(input.CostCenters)

	builder.WriteString("# Financial Chargeback Report\n\n")
	builder.WriteString(fmt.Sprintf("**Reporting Period:** %s to %s  \n", input.PeriodStart, input.PeriodEnd))
	builder.WriteString(fmt.Sprintf("**Generated:** %s  \n", input.GeneratedAt))
	builder.WriteString("**Report Type:** Chargeback Allocation\n\n")
	builder.WriteString("---\n\n")
	builder.WriteString("## Executive Summary\n\n")
	builder.WriteString("| Metric | Value |\n")
	builder.WriteString("|--------|-------|\n")
	builder.WriteString(fmt.Sprintf("| **Total AI Spend** | $%.2f |\n", input.TotalSpend))
	builder.WriteString(fmt.Sprintf("| **Total Requests** | %d |\n", input.TotalRequests))
	builder.WriteString(fmt.Sprintf("| **Total Tokens** | %d |\n", input.TotalTokens))
	builder.WriteString(fmt.Sprintf("| **Attribution Coverage** | %.2f%% |\n", coverage))
	builder.WriteString(fmt.Sprintf("| **Unattributed Usage** | $%.2f |\n\n", unattributed))

	builder.WriteString("### Month-over-Month Variance\n\n")
	builder.WriteString("| Metric | Value |\n")
	builder.WriteString("|--------|-------|\n")
	builder.WriteString(fmt.Sprintf("| **Previous Month Spend** | $%.2f |\n", input.PreviousMonthSpend))
	if strings.EqualFold(input.Variance, "N/A") {
		builder.WriteString("| **Variance** | N/A (no previous data) |\n\n")
	} else {
		status := "Within Threshold"
		if data.VarianceExceeded {
			status = "THRESHOLD EXCEEDED"
		}
		builder.WriteString(fmt.Sprintf("| **Variance** | %s%% %s |\n\n", input.Variance, status))
	}

	builder.WriteString("---\n\n")
	builder.WriteString("## Allocation by Cost Center\n\n")
	builder.WriteString("| Cost Center | Team | Requests | Tokens | Spend | % of Total |\n")
	builder.WriteString("|-------------|------|----------|--------|-------|------------|\n")
	for _, row := range input.CostCenters {
		builder.WriteString(fmt.Sprintf("| %s | %s | %d | %d | $%.2f | %.2f%% |\n", row.CostCenter, row.Team, row.RequestCount, row.TokenCount, row.SpendAmount, row.PercentOfTotal))
	}
	builder.WriteString("\n---\n\n")

	builder.WriteString("## Top Principals by Spend\n\n")
	builder.WriteString("| Principal | Team | Cost Center | Requests | Spend |\n")
	builder.WriteString("|-----------|------|-------------|----------|-------|\n")
	for _, row := range data.TopPrincipals {
		builder.WriteString(fmt.Sprintf("| %s | %s | %s | %d | $%.2f |\n", row.Principal, row.Team, row.CostCenter, row.RequestCount, row.SpendAmount))
	}
	builder.WriteString("\n---\n\n")

	builder.WriteString("## Spend by Model\n\n")
	builder.WriteString("| Model | Requests | Tokens | Spend |\n")
	builder.WriteString("|-------|----------|--------|-------|\n")
	for _, row := range input.Models {
		builder.WriteString(fmt.Sprintf("| %s | %d | %d | $%.2f |\n", row.Model, row.RequestCount, row.TokenCount, row.SpendAmount))
	}

	if len(input.Anomalies) > 0 {
		builder.WriteString("\n---\n\n")
		builder.WriteString("## Anomalies Detected\n\n")
		builder.WriteString("| Cost Center | Type | Current Spend | Previous Spend | Spike % |\n")
		builder.WriteString("|-------------|------|---------------|----------------|---------|\n")
		for _, anomaly := range input.Anomalies {
			builder.WriteString(fmt.Sprintf("| %s | %s | $%.2f | $%.2f | %.2f%% |\n", anomaly.CostCenter, anomaly.Type, anomaly.CurrentSpend, anomaly.PreviousSpend, anomaly.SpikePercent))
		}
	}

	if input.ForecastEnabled && input.ForecastMonth1 != nil && input.ForecastMonth2 != nil && input.ForecastMonth3 != nil {
		builder.WriteString("\n---\n\n")
		builder.WriteString("## Spend Forecast\n\n")
		builder.WriteString("_Predictions based on linear regression of historical spend trends._\n")
		builder.WriteString("_Confidence: Estimates may vary +/- 20% from actual spend._\n\n")
		builder.WriteString("### 3-Month Projection\n\n")
		builder.WriteString("| Period | Predicted Spend |\n")
		builder.WriteString("|--------|-----------------|\n")
		builder.WriteString(fmt.Sprintf("| Month +1 | $%.2f |\n", *input.ForecastMonth1))
		builder.WriteString(fmt.Sprintf("| Month +2 | $%.2f |\n", *input.ForecastMonth2))
		builder.WriteString(fmt.Sprintf("| Month +3 | $%.2f |\n", *input.ForecastMonth3))
		builder.WriteString(fmt.Sprintf("| **3-Mo Total** | **$%.2f** |\n\n", *input.ForecastMonth1+*input.ForecastMonth2+*input.ForecastMonth3))

		builder.WriteString("### Burn Rate Analysis\n\n")
		builder.WriteString("| Metric | Value |\n")
		builder.WriteString("|--------|-------|\n")
		builder.WriteString(fmt.Sprintf("| **Daily Average** | $%.2f |\n", input.DailyBurn))
		if input.DaysRemaining != nil {
			builder.WriteString(fmt.Sprintf("| **Days Until Budget Exhaustion** | %d |\n", *input.DaysRemaining))
			builder.WriteString(fmt.Sprintf("| **Projected Exhaustion Date** | %s |\n", input.ExhaustionDate))
		} else {
			builder.WriteString("| **Days Until Budget Exhaustion** | N/A (no budget set) |\n")
		}
		if input.TotalBudget > 0 {
			builder.WriteString(fmt.Sprintf("| **Total Budget** | $%.2f |\n", input.TotalBudget))
		}
		builder.WriteString("\n")
		switch input.BudgetRisk.RiskLevel {
		case "high":
			builder.WriteString("### Budget Alert\n\n")
			builder.WriteString("**Risk Level: HIGH** - Forecasted spend is projected to exceed the budget alert threshold.\n\n")
		case "medium":
			builder.WriteString("### Budget Notice\n\n")
			builder.WriteString("**Risk Level: MEDIUM** - Forecasted spend is approaching budget thresholds.\n\n")
		}
	}

	builder.WriteString("---\n\n")
	builder.WriteString("*Report generated by AI Control Plane - Chargeback Reporting*\n")
	builder.WriteString(fmt.Sprintf("*Schema Version: %s*\n", input.SchemaVersion))
	return builder.String()
}
