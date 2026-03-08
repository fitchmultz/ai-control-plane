// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Centralize typed chargeback analytics that were previously shell-owned.
//
// Responsibilities:
//   - Calculate variance, anomaly detection, forecasts, and burn-rate metrics.
//   - Normalize budget-risk and month-boundary calculations.
//
// Non-scope:
//   - Does not execute database queries.
//   - Does not write files or send notifications.
//
// Invariants/Assumptions:
//   - Monetary values are represented as float64 USD totals.
//   - Dates are computed in local time for report-month selection and archived
//     as UTC timestamps for machine-readable outputs.
//
// Scope:
//   - Chargeback analytics and date helpers only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func resolveMonthRange(reportMonth string, now time.Time) (MonthRange, error) {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.Local()
	target := strings.TrimSpace(reportMonth)
	if target == "" {
		previous := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0)
		target = previous.Format("2006-01")
	}
	currentStart, err := time.ParseInLocation("2006-01", target, now.Location())
	if err != nil {
		return MonthRange{}, fmt.Errorf("invalid report month %q: expected YYYY-MM", target)
	}
	currentStart = time.Date(currentStart.Year(), currentStart.Month(), 1, 0, 0, 0, 0, now.Location())
	nextStart := currentStart.AddDate(0, 1, 0)
	previousStart := currentStart.AddDate(0, -1, 0)

	return MonthRange{
		ReportMonth:    currentStart.Format("2006-01"),
		MonthStart:     currentStart.Format("2006-01-02"),
		MonthEnd:       nextStart.AddDate(0, 0, -1).Format("2006-01-02"),
		PreviousStart:  previousStart.Format("2006-01-02"),
		PreviousEnd:    currentStart.AddDate(0, 0, -1).Format("2006-01-02"),
		GeneratedAtUTC: now.UTC(),
	}, nil
}

func varianceString(current float64, previous float64) string {
	if previous == 0 {
		return "N/A"
	}
	return strconv.FormatFloat(roundTo(((current-previous)/previous)*100), 'f', -1, 64)
}

func detectAnomalies(current []CostCenterAllocation, previous []CostCenterAllocation, threshold float64) []Anomaly {
	previousByCostCenter := make(map[string]CostCenterAllocation, len(previous))
	for _, row := range previous {
		previousByCostCenter[row.CostCenter] = row
	}

	anomalies := make([]Anomaly, 0)
	for _, row := range current {
		if row.CostCenter == "unknown-cc" {
			continue
		}
		previousRow, ok := previousByCostCenter[row.CostCenter]
		if !ok || previousRow.SpendAmount <= 0 {
			continue
		}
		spike := roundTo(((row.SpendAmount - previousRow.SpendAmount) / previousRow.SpendAmount) * 100)
		if spike < threshold {
			continue
		}
		anomalies = append(anomalies, Anomaly{
			CostCenter:    row.CostCenter,
			Team:          row.Team,
			CurrentSpend:  row.SpendAmount,
			PreviousSpend: previousRow.SpendAmount,
			SpikePercent:  spike,
			Type:          "spike",
		})
	}
	return anomalies
}

func forecastSpend(history []HistoricalSpend) (*float64, *float64, *float64) {
	if len(history) < 2 {
		return nil, nil, nil
	}

	chronological := make([]HistoricalSpend, 0, len(history))
	for index := len(history) - 1; index >= 0; index-- {
		chronological = append(chronological, history[index])
	}

	var (
		n     = float64(len(chronological))
		sumX  float64
		sumY  float64
		sumXY float64
		sumX2 float64
	)
	for index, row := range chronological {
		x := float64(index + 1)
		y := row.Spend
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := (n * sumX2) - (sumX * sumX)
	if denominator == 0 {
		return nil, nil, nil
	}
	slope := ((n * sumXY) - (sumX * sumY)) / denominator
	intercept := (sumY - (slope * sumX)) / n

	next1 := maxFloat(0, roundTo4((slope*float64(len(chronological)+1))+intercept))
	next2 := maxFloat(0, roundTo4((slope*float64(len(chronological)+2))+intercept))
	next3 := maxFloat(0, roundTo4((slope*float64(len(chronological)+3))+intercept))
	return &next1, &next2, &next3
}

func calculateBurnRate(totalSpend float64, monthStart string, totalBudget float64, now time.Time) (float64, *int64, string) {
	if totalSpend == 0 {
		return 0, nil, "N/A"
	}

	start, err := time.ParseInLocation("2006-01-02", monthStart, now.Location())
	if err != nil {
		return 0, nil, "N/A"
	}
	currentDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	daysElapsed := int64(currentDay.Sub(start).Hours()/24) + 1
	if daysElapsed < 1 {
		daysElapsed = 1
	}
	daily := roundTo4(totalSpend / float64(daysElapsed))

	if totalBudget <= 0 {
		return daily, nil, "N/A"
	}
	remaining := totalBudget - totalSpend
	if remaining <= 0 {
		zero := int64(0)
		return daily, &zero, "EXHAUSTED"
	}
	if daily == 0 {
		return daily, nil, "N/A"
	}

	daysRemainingValue := int64(math.Floor(remaining / daily))
	exhaustionDate := currentDay.AddDate(0, 0, int(daysRemainingValue)).Format("2006-01-02")
	return daily, &daysRemainingValue, exhaustionDate
}

func budgetRisk(month1 *float64, month2 *float64, month3 *float64, totalBudget float64, threshold float64) BudgetRisk {
	if totalBudget <= 0 || month1 == nil || month2 == nil || month3 == nil {
		return BudgetRisk{RiskLevel: "unknown", ThresholdExceeded: false}
	}
	totalForecast := *month1 + *month2 + *month3
	percent := roundTo((totalForecast / totalBudget) * 100)
	risk := BudgetRisk{RiskLevel: "low", BudgetPercent: &percent}
	switch {
	case percent >= threshold:
		risk.RiskLevel = "high"
		risk.ThresholdExceeded = true
	case percent >= 50:
		risk.RiskLevel = "medium"
	default:
		risk.RiskLevel = "low"
	}
	return risk
}

func roundTo4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func maxFloat(left float64, right float64) float64 {
	if right > left {
		return right
	}
	return left
}
