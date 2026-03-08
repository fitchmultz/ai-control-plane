// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Render canonical chargeback report outputs.
//
// Responsibilities:
//   - Serialize the report JSON schema with deterministic formatting.
//   - Emit RFC 4180-compliant spreadsheet-safe CSV output.
//   - Fan out the full report output set for workflow archival and routing.
//
// Non-scope:
//   - Does not decode environment variables.
//   - Does not deliver notifications.
//
// Invariants/Assumptions:
//   - Rendered values are deterministic for equivalent typed inputs.
//   - Spreadsheet-safe escaping prefixes risky text cells with a leading
//     apostrophe.
//
// Scope:
//   - Chargeback report rendering only.
//
// Usage:
//   - Used by the report workflow and render-focused tests.
package chargeback

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

type DefaultRenderer struct{}

func (DefaultRenderer) Render(data ReportData) (ReportOutputs, error) {
	jsonPayload, err := renderJSONBytes(data.Input)
	if err != nil {
		return ReportOutputs{}, fmt.Errorf("render json: %w", err)
	}
	csvPayload, err := renderCSVBytes(data.Input.ReportMonth, data.Input.CostCenters)
	if err != nil {
		return ReportOutputs{}, fmt.Errorf("render csv: %w", err)
	}
	return ReportOutputs{
		Markdown: RenderMarkdown(data),
		JSON:     string(jsonPayload),
		CSV:      string(csvPayload),
		Archived: map[string]string{},
	}, nil
}

func RenderJSON(w io.Writer, input ReportInput) error {
	payload, err := renderJSONBytes(input)
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func RenderCSV(w io.Writer, reportMonth string, rows []CostCenterAllocation) error {
	payload, err := renderCSVBytes(reportMonth, rows)
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

func renderJSONBytes(input ReportInput) ([]byte, error) {
	payload := reportPayload{
		SchemaVersion: input.SchemaVersion,
		ReportMetadata: reportMetadata{
			GeneratedAt: input.GeneratedAt,
			ReportMonth: input.ReportMonth,
			PeriodStart: input.PeriodStart,
			PeriodEnd:   input.PeriodEnd,
		},
		ExecutiveSummary: executiveSummary{
			TotalSpend:                 input.TotalSpend,
			TotalRequests:              input.TotalRequests,
			TotalTokens:                input.TotalTokens,
			AttributionCoveragePercent: coveragePercent(input.TotalSpend, unattributedSpend(input.CostCenters)),
			UnattributedSpend:          unattributedSpend(input.CostCenters),
		},
		AllocationsByCostCenter: input.CostCenters,
		AllocationsByModel:      input.Models,
		VarianceAnalysis: varianceAnalysis{
			PreviousMonthSpend:        input.PreviousMonthSpend,
			VariancePercent:           varianceValue(input.Variance),
			VarianceThreshold:         input.VarianceThreshold,
			VarianceThresholdExceeded: varianceThresholdExceeded(input.Variance, input.VarianceThreshold),
		},
		Anomalies: input.Anomalies,
		Forecast: forecastSection{
			Enabled:        input.ForecastEnabled,
			Methodology:    "linear_regression",
			ConfidenceNote: "Estimates based on historical trends; actual spend may vary +/- 20%",
			Predictions: forecastPredictions{
				Month1: input.ForecastMonth1,
				Month2: input.ForecastMonth2,
				Month3: input.ForecastMonth3,
			},
			BurnRate: burnRate{
				DailyAverage:        input.DailyBurn,
				DaysUntilExhaustion: input.DaysRemaining,
				ExhaustionDate:      normalizeExhaustionDate(input.ExhaustionDate),
			},
			BudgetAnalysis: budgetAnalysis{
				TotalBudget:    input.TotalBudget,
				RiskAssessment: input.BudgetRisk,
			},
		},
		Configuration: reportConfiguration{
			VarianceThresholdPercent: input.VarianceThreshold,
			AnomalyThresholdPercent:  input.AnomalyThreshold,
		},
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func renderCSVBytes(reportMonth string, rows []CostCenterAllocation) ([]byte, error) {
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	records := [][]string{{
		"CostCenter",
		"Team",
		"SpendAmount",
		"RequestCount",
		"TokenCount",
		"PercentOfTotal",
		"ReportMonth",
	}}

	for _, row := range rows {
		records = append(records, []string{
			spreadsheetSafe(row.CostCenter),
			spreadsheetSafe(row.Team),
			formatFloat(row.SpendAmount),
			strconv.FormatInt(row.RequestCount, 10),
			strconv.FormatInt(row.TokenCount, 10),
			formatFloat(row.PercentOfTotal),
			spreadsheetSafe(reportMonth),
		})
	}

	if err := writer.WriteAll(records); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type reportPayload struct {
	SchemaVersion           string                 `json:"schema_version"`
	ReportMetadata          reportMetadata         `json:"report_metadata"`
	ExecutiveSummary        executiveSummary       `json:"executive_summary"`
	AllocationsByCostCenter []CostCenterAllocation `json:"allocations_by_cost_center"`
	AllocationsByModel      []ModelAllocation      `json:"allocations_by_model"`
	VarianceAnalysis        varianceAnalysis       `json:"variance_analysis"`
	Anomalies               []Anomaly              `json:"anomalies"`
	Forecast                forecastSection        `json:"forecast"`
	Configuration           reportConfiguration    `json:"configuration"`
}

type reportMetadata struct {
	GeneratedAt string `json:"generated_at"`
	ReportMonth string `json:"report_month"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
}

type executiveSummary struct {
	TotalSpend                 float64 `json:"total_spend"`
	TotalRequests              int64   `json:"total_requests"`
	TotalTokens                int64   `json:"total_tokens"`
	AttributionCoveragePercent float64 `json:"attribution_coverage_percent"`
	UnattributedSpend          float64 `json:"unattributed_spend"`
}

type varianceAnalysis struct {
	PreviousMonthSpend        float64 `json:"previous_month_spend"`
	VariancePercent           any     `json:"variance_percent"`
	VarianceThreshold         float64 `json:"variance_threshold"`
	VarianceThresholdExceeded bool    `json:"variance_threshold_exceeded"`
}

type forecastSection struct {
	Enabled        bool                `json:"enabled"`
	Methodology    string              `json:"methodology"`
	ConfidenceNote string              `json:"confidence_note"`
	Predictions    forecastPredictions `json:"predictions"`
	BurnRate       burnRate            `json:"burn_rate"`
	BudgetAnalysis budgetAnalysis      `json:"budget_analysis"`
}

type forecastPredictions struct {
	Month1 *float64 `json:"month_1"`
	Month2 *float64 `json:"month_2"`
	Month3 *float64 `json:"month_3"`
}

type burnRate struct {
	DailyAverage        float64 `json:"daily_average"`
	DaysUntilExhaustion *int64  `json:"days_until_exhaustion"`
	ExhaustionDate      string  `json:"exhaustion_date"`
}

type budgetAnalysis struct {
	TotalBudget    float64    `json:"total_budget"`
	RiskAssessment BudgetRisk `json:"risk_assessment"`
}

type reportConfiguration struct {
	VarianceThresholdPercent float64 `json:"variance_threshold_percent"`
	AnomalyThresholdPercent  float64 `json:"anomaly_threshold_percent"`
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func spreadsheetSafe(value string) string {
	if value == "" {
		return value
	}
	trimmed := strings.TrimLeft(value, " \t")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	}
	if value[0] == '\t' || value[0] == '\r' || value[0] == '\n' {
		return "'" + value
	}
	return value
}

func unattributedSpend(rows []CostCenterAllocation) float64 {
	for _, row := range rows {
		if row.CostCenter == "unknown-cc" {
			return row.SpendAmount
		}
	}
	return 0
}

func coveragePercent(totalSpend float64, unattributed float64) float64 {
	if totalSpend == 0 {
		return 100
	}
	return roundTo(((totalSpend - unattributed) / totalSpend) * 100)
}

func roundTo(value float64) float64 {
	return math.Round(value*100) / 100
}

func varianceValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "N/A") {
		return "N/A"
	}
	if value, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return value
	}
	return trimmed
}

func varianceThresholdExceeded(raw string, threshold float64) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "N/A") {
		return false
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return false
	}
	return value >= threshold || value <= -threshold
}

func normalizeExhaustionDate(value string) string {
	if strings.TrimSpace(value) == "" {
		return "N/A"
	}
	return value
}
