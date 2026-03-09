// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Verify chargeback analytics and month-range helpers directly.
//
// Responsibilities:
//   - Cover report month resolution, anomaly detection, and forecasting.
//   - Verify burn-rate and budget-risk branch handling.
//   - Lock down deterministic rounded outputs for report generation.
//
// Scope:
//   - Unit tests for analytics helpers only.
//
// Usage:
//   - Run with `go test ./internal/chargeback`.
//
// Invariants/Assumptions:
//   - Tests use fixed clocks and pure inputs.
//   - Helper outputs remain deterministic for equivalent inputs.
package chargeback

import (
	"testing"
	"time"
)

func TestResolveMonthRange_DefaultAndInvalid(t *testing.T) {
	now := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.FixedZone("MDT", -6*60*60))

	got, err := resolveMonthRange("", now)
	if err != nil {
		t.Fatalf("resolveMonthRange default returned error: %v", err)
	}
	if got.ReportMonth != "2026-02" || got.MonthStart != "2026-02-01" || got.MonthEnd != "2026-02-28" {
		t.Fatalf("unexpected default month range: %+v", got)
	}

	if _, err := resolveMonthRange("2026/02", now); err == nil {
		t.Fatal("expected invalid month format to fail")
	}
}

func TestVarianceStringAndDetectAnomalies(t *testing.T) {
	t.Parallel()

	if got := varianceString(10, 0); got != "N/A" {
		t.Fatalf("varianceString zero previous = %q", got)
	}
	if got := varianceString(150, 100); got != "50" {
		t.Fatalf("varianceString growth = %q", got)
	}

	anomalies := detectAnomalies(
		[]CostCenterAllocation{
			{CostCenter: "1001", Team: "platform", SpendAmount: 200},
			{CostCenter: "unknown-cc", Team: "unknown", SpendAmount: 1000},
			{CostCenter: "1002", Team: "ops", SpendAmount: 100},
		},
		[]CostCenterAllocation{
			{CostCenter: "1001", Team: "platform", SpendAmount: 50},
			{CostCenter: "1002", Team: "ops", SpendAmount: 95},
		},
		100,
	)
	if len(anomalies) != 1 {
		t.Fatalf("expected one anomaly, got %+v", anomalies)
	}
	if anomalies[0].CostCenter != "1001" || anomalies[0].SpikePercent != 300 {
		t.Fatalf("unexpected anomaly: %+v", anomalies[0])
	}
}

func TestForecastSpendBurnRateAndBudgetRisk(t *testing.T) {
	t.Parallel()

	month1, month2, month3 := forecastSpend([]HistoricalSpend{
		{Month: "2026-03", Spend: 30},
		{Month: "2026-02", Spend: 20},
		{Month: "2026-01", Spend: 10},
	})
	if month1 == nil || month2 == nil || month3 == nil {
		t.Fatal("expected forecast values")
	}
	if *month1 != 40 || *month2 != 50 || *month3 != 60 {
		t.Fatalf("unexpected forecast values: %v %v %v", *month1, *month2, *month3)
	}
	if a, b, c := forecastSpend([]HistoricalSpend{{Month: "2026-03", Spend: 10}}); a != nil || b != nil || c != nil {
		t.Fatal("expected insufficient history to return nil forecasts")
	}

	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	daily, remaining, date := calculateBurnRate(150, "2026-03-01", 300, now)
	if daily != 10 || remaining == nil || *remaining != 15 || date != "2026-03-30" {
		t.Fatalf("unexpected burn rate result: daily=%v remaining=%v date=%s", daily, remaining, date)
	}

	daily, remaining, date = calculateBurnRate(0, "2026-03-01", 300, now)
	if daily != 0 || remaining != nil || date != "N/A" {
		t.Fatalf("unexpected zero-spend burn rate result: daily=%v remaining=%v date=%s", daily, remaining, date)
	}

	risk := budgetRisk(month1, month2, month3, 120, 90)
	if risk.RiskLevel != "high" || !risk.ThresholdExceeded {
		t.Fatalf("expected high budget risk, got %+v", risk)
	}
	unknown := budgetRisk(nil, month2, month3, 120, 90)
	if unknown.RiskLevel != "unknown" || unknown.BudgetPercent != nil {
		t.Fatalf("expected unknown risk for nil forecast, got %+v", unknown)
	}
}
