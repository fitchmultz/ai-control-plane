// Package chargeback provides canonical report rendering helpers.
//
// Purpose:
//   - Centralize machine-safe chargeback JSON, CSV, and webhook payload generation.
//
// Responsibilities:
//   - Decode structured chargeback allocation/anomaly inputs from JSON.
//   - Render the chargeback report JSON schema with deterministic formatting.
//   - Emit RFC 4180-compliant, spreadsheet-safe CSV output.
//   - Build generic and Slack webhook payloads without shell string interpolation.
//
// Non-scope:
//   - Does not execute database queries.
//   - Does not calculate spend analytics or forecasting inputs.
//
// Invariants/Assumptions:
//   - Structured collection inputs are valid JSON arrays or empty.
//   - Numeric counts are integral in the upstream JSON payloads.
//   - Spreadsheet-safe escaping prefixes risky text cells with a leading apostrophe.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

type CostCenterAllocation struct {
	CostCenter     string  `json:"cost_center"`
	Team           string  `json:"team"`
	RequestCount   int64   `json:"request_count"`
	TokenCount     int64   `json:"token_count"`
	SpendAmount    float64 `json:"spend_amount"`
	PercentOfTotal float64 `json:"percent_of_total"`
}

type ModelAllocation struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TokenCount   int64   `json:"token_count"`
	SpendAmount  float64 `json:"spend_amount"`
}

type Anomaly struct {
	CostCenter    string  `json:"cost_center"`
	Team          string  `json:"team"`
	CurrentSpend  float64 `json:"current_spend"`
	PreviousSpend float64 `json:"previous_spend"`
	SpikePercent  float64 `json:"spike_percent"`
	Type          string  `json:"type"`
}

type BudgetRisk struct {
	RiskLevel         string   `json:"risk_level"`
	BudgetPercent     *float64 `json:"budget_percent"`
	ThresholdExceeded bool     `json:"threshold_exceeded"`
}

type ReportInput struct {
	SchemaVersion      string
	GeneratedAt        string
	ReportMonth        string
	PeriodStart        string
	PeriodEnd          string
	TotalSpend         float64
	TotalRequests      int64
	TotalTokens        int64
	CostCenters        []CostCenterAllocation
	Models             []ModelAllocation
	Variance           string
	VarianceThreshold  float64
	PreviousMonthSpend float64
	Anomalies          []Anomaly
	ForecastEnabled    bool
	ForecastMonth1     *float64
	ForecastMonth2     *float64
	ForecastMonth3     *float64
	DailyBurn          float64
	DaysRemaining      *int64
	ExhaustionDate     string
	TotalBudget        float64
	BudgetRisk         BudgetRisk
	AnomalyThreshold   float64
}

type GenericWebhookInput struct {
	Event       string
	ReportMonth string
	TotalSpend  float64
	Variance    string
	Anomalies   []Anomaly
	Timestamp   string
}

type SlackWebhookInput struct {
	ReportMonth string
	TotalSpend  float64
	Variance    string
	Color       string
	Epoch       int64
}

func DecodeCostCenters(raw string) ([]CostCenterAllocation, error) {
	var values []CostCenterAllocation
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode cost centers: %w", err)
	}
	return values, nil
}

func DecodeModels(raw string) ([]ModelAllocation, error) {
	var values []ModelAllocation
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	return values, nil
}

func DecodeAnomalies(raw string) ([]Anomaly, error) {
	var values []Anomaly
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode anomalies: %w", err)
	}
	return values, nil
}

func decodeJSONArray(raw string, target any) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "[]"
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func RenderJSON(w io.Writer, input ReportInput) error {
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

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func RenderCSV(w io.Writer, reportMonth string, rows []CostCenterAllocation) error {
	writer := csv.NewWriter(w)
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
		return fmt.Errorf("write csv: %w", err)
	}
	writer.Flush()
	return writer.Error()
}

func BuildGenericWebhookPayload(input GenericWebhookInput) ([]byte, error) {
	payload := struct {
		Event       string    `json:"event"`
		ReportMonth string    `json:"report_month"`
		TotalSpend  float64   `json:"total_spend"`
		Variance    string    `json:"variance_percent"`
		Anomalies   []Anomaly `json:"anomalies"`
		Timestamp   string    `json:"timestamp"`
	}{
		Event:       input.Event,
		ReportMonth: input.ReportMonth,
		TotalSpend:  input.TotalSpend,
		Variance:    input.Variance,
		Anomalies:   input.Anomalies,
		Timestamp:   input.Timestamp,
	}
	return json.Marshal(payload)
}

func BuildSlackWebhookPayload(input SlackWebhookInput) ([]byte, error) {
	payload := struct {
		Text        string `json:"text"`
		Attachments []struct {
			Color  string `json:"color"`
			Title  string `json:"title"`
			Fields []struct {
				Title string `json:"title"`
				Value string `json:"value"`
				Short bool   `json:"short"`
			} `json:"fields"`
			Footer string `json:"footer"`
			TS     int64  `json:"ts"`
		} `json:"attachments"`
	}{
		Text: "📊 Monthly Chargeback Report Generated",
	}

	attachment := struct {
		Color  string `json:"color"`
		Title  string `json:"title"`
		Fields []struct {
			Title string `json:"title"`
			Value string `json:"value"`
			Short bool   `json:"short"`
		} `json:"fields"`
		Footer string `json:"footer"`
		TS     int64  `json:"ts"`
	}{
		Color:  input.Color,
		Title:  "Report for " + input.ReportMonth,
		Footer: "AI Control Plane",
		TS:     input.Epoch,
	}
	attachment.Fields = []struct {
		Title string `json:"title"`
		Value string `json:"value"`
		Short bool   `json:"short"`
	}{
		{Title: "Total Spend", Value: fmt.Sprintf("$%.2f", input.TotalSpend), Short: true},
		{Title: "Variance", Value: input.Variance + "%", Short: true},
	}
	payload.Attachments = append(payload.Attachments, attachment)
	return json.Marshal(payload)
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
