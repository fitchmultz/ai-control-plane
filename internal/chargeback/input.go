// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Own typed input normalization for chargeback report, render, and payload
//     workflows.
//
// Responsibilities:
//   - Validate command-facing values and apply canonical defaults.
//   - Decode chargeback environment payloads into typed report and notification
//     inputs.
//   - Keep env parsing and fallback behavior out of CLI adapters and workflows.
//
// Non-scope:
//   - Does not execute report workflows.
//   - Does not render output bytes.
//
// Invariants/Assumptions:
//   - Environment values may be blank or use `N/A` for nullable numeric fields.
//   - Returned inputs are fully normalized and ready for downstream workflows.
//
// Scope:
//   - Typed chargeback input adapters only.
//
// Usage:
//   - Used by CLI adapters before invoking workflow or render APIs.
package chargeback

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

type Environment interface {
	String(key string) string
	Int64Ptr(key string) *int64
	Float64Ptr(key string) *float64
	ChargebackForecast() (*float64, *float64, *float64)
	ChargebackTimestamp(now time.Time) string
}

func NewReportWorkflowInput(command ReportCommandInput, env Environment, repoRoot string, now func() time.Time) (ReportWorkflowInput, error) {
	format, err := parseReportFormat(command.Format, true)
	if err != nil {
		return ReportWorkflowInput{}, err
	}
	varianceThreshold, err := parseOptionalFloat(command.VarianceThreshold, defaultVarianceThreshold, "variance threshold")
	if err != nil {
		return ReportWorkflowInput{}, err
	}
	anomalyThreshold, err := parseOptionalFloat(command.AnomalyThreshold, defaultAnomalyThreshold, "anomaly threshold")
	if err != nil {
		return ReportWorkflowInput{}, err
	}
	budgetAlertThreshold, err := parseOptionalFloat(command.BudgetAlertThreshold, defaultBudgetAlertThreshold, "budget alert threshold")
	if err != nil {
		return ReportWorkflowInput{}, err
	}
	forecastEnabled := true
	if command.ForecastEnabled != nil {
		forecastEnabled = *command.ForecastEnabled
	}
	if now == nil {
		now = time.Now
	}
	return ReportWorkflowInput{
		Request: ReportRequest{
			ReportMonth:          strings.TrimSpace(command.ReportMonth),
			Format:               format,
			ArchiveDir:           textutil.DefaultIfBlank(command.ArchiveDir, defaultArchiveDir),
			VarianceThreshold:    varianceThreshold,
			AnomalyThreshold:     anomalyThreshold,
			ForecastEnabled:      forecastEnabled,
			BudgetAlertThreshold: budgetAlertThreshold,
			Notify:               command.Notify,
		},
		RepoRoot: repoRoot,
		Notification: NotificationConfig{
			GenericWebhookURL: envString(env, "GENERIC_WEBHOOK_URL"),
			SlackWebhookURL:   envString(env, "SLACK_WEBHOOK_URL"),
		},
		Now: now,
	}, nil
}

func NewRenderRequest(command RenderCommandInput, env Environment, now func() time.Time) (RenderRequest, error) {
	format, err := parseReportFormat(command.Format, false)
	if err != nil {
		return RenderRequest{}, err
	}
	if format != ReportFormatJSON && format != ReportFormatCSV {
		return RenderRequest{}, fmt.Errorf("--format must be one of: json, csv")
	}
	reportInput, err := decodeReportInput(env, now)
	if err != nil {
		return RenderRequest{}, err
	}
	return RenderRequest{
		Format: format,
		Input:  reportInput,
	}, nil
}

func NewPayloadRequest(command PayloadCommandInput, env Environment, now func() time.Time) (PayloadRequest, error) {
	target, err := parsePayloadTarget(command.Target)
	if err != nil {
		return PayloadRequest{}, err
	}
	if now == nil {
		now = time.Now
	}
	anomalies, err := decodeAnomalies(envString(env, "CHARGEBACK_ANOMALIES_JSON"))
	if err != nil {
		return PayloadRequest{}, fmt.Errorf("invalid CHARGEBACK_ANOMALIES_JSON: %w", err)
	}
	current := now()
	return PayloadRequest{
		Target: target,
		Generic: GenericWebhookInput{
			Event:       textutil.DefaultIfBlank(envString(env, "CHARGEBACK_PAYLOAD_EVENT"), defaultGenericNotificationEvent),
			ReportMonth: envString(env, "CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  envFloat(env, "CHARGEBACK_TOTAL_SPEND", 0),
			Variance:    envString(env, "CHARGEBACK_VARIANCE"),
			Anomalies:   anomalies,
			Timestamp:   chargebackTimestamp(env, current),
		},
		Slack: SlackWebhookInput{
			ReportMonth: envString(env, "CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  envFloat(env, "CHARGEBACK_TOTAL_SPEND", 0),
			Variance:    envString(env, "CHARGEBACK_VARIANCE"),
			Color:       textutil.DefaultIfBlank(envString(env, "CHARGEBACK_SLACK_COLOR"), defaultSlackColor),
			Epoch:       envInt64(env, "CHARGEBACK_SLACK_EPOCH", current.Unix()),
		},
	}, nil
}

func decodeReportInput(env Environment, now func() time.Time) (ReportInput, error) {
	if now == nil {
		now = time.Now
	}
	costCenters, err := decodeCostCenters(envString(env, "CHARGEBACK_COST_CENTER_JSON"))
	if err != nil {
		return ReportInput{}, fmt.Errorf("invalid CHARGEBACK_COST_CENTER_JSON: %w", err)
	}
	models, err := decodeModels(envString(env, "CHARGEBACK_MODEL_JSON"))
	if err != nil {
		return ReportInput{}, fmt.Errorf("invalid CHARGEBACK_MODEL_JSON: %w", err)
	}
	anomalies, err := decodeAnomalies(envString(env, "CHARGEBACK_ANOMALIES_JSON"))
	if err != nil {
		return ReportInput{}, fmt.Errorf("invalid CHARGEBACK_ANOMALIES_JSON: %w", err)
	}
	month1, month2, month3 := env.ChargebackForecast()
	current := now()
	return ReportInput{
		SchemaVersion:      textutil.DefaultIfBlank(envString(env, "CHARGEBACK_SCHEMA_VERSION"), defaultSchemaVersion),
		GeneratedAt:        textutil.DefaultIfBlank(envString(env, "CHARGEBACK_GENERATED_AT"), current.UTC().Format(time.RFC3339)),
		ReportMonth:        envString(env, "CHARGEBACK_REPORT_MONTH"),
		PeriodStart:        envString(env, "CHARGEBACK_MONTH_START"),
		PeriodEnd:          envString(env, "CHARGEBACK_MONTH_END"),
		TotalSpend:         envFloat(env, "CHARGEBACK_TOTAL_SPEND", 0),
		TotalRequests:      envInt64(env, "CHARGEBACK_TOTAL_REQUESTS", 0),
		TotalTokens:        envInt64(env, "CHARGEBACK_TOTAL_TOKENS", 0),
		CostCenters:        costCenters,
		Models:             models,
		Variance:           envString(env, "CHARGEBACK_VARIANCE"),
		VarianceThreshold:  envFloat(env, "CHARGEBACK_VARIANCE_THRESHOLD", defaultVarianceThreshold),
		PreviousMonthSpend: envFloat(env, "CHARGEBACK_PREV_MONTH_SPEND", 0),
		Anomalies:          anomalies,
		ForecastEnabled:    envBool(env, "CHARGEBACK_FORECAST_ENABLED", true),
		ForecastMonth1:     month1,
		ForecastMonth2:     month2,
		ForecastMonth3:     month3,
		DailyBurn:          envFloat(env, "CHARGEBACK_DAILY_BURN", 0),
		DaysRemaining:      env.Int64Ptr("CHARGEBACK_DAYS_REMAINING"),
		ExhaustionDate:     textutil.DefaultIfBlank(envString(env, "CHARGEBACK_EXHAUSTION_DATE"), "N/A"),
		TotalBudget:        envFloat(env, "CHARGEBACK_TOTAL_BUDGET", 0),
		BudgetRisk: BudgetRisk{
			RiskLevel:         textutil.DefaultIfBlank(envString(env, "CHARGEBACK_BUDGET_RISK_LEVEL"), "unknown"),
			BudgetPercent:     env.Float64Ptr("CHARGEBACK_BUDGET_RISK_PERCENT"),
			ThresholdExceeded: envBool(env, "CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED", false),
		},
		AnomalyThreshold: envFloat(env, "CHARGEBACK_ANOMALY_THRESHOLD", defaultAnomalyThreshold),
	}, nil
}

func RenderToBytes(request RenderRequest) ([]byte, error) {
	switch request.Format {
	case ReportFormatCSV:
		return renderCSVBytes(request.Input.ReportMonth, request.Input.CostCenters)
	case ReportFormatJSON:
		return renderJSONBytes(request.Input)
	default:
		return nil, fmt.Errorf("unsupported render format: %s", request.Format)
	}
}

func BuildPayload(request PayloadRequest) ([]byte, error) {
	switch request.Target {
	case PayloadTargetGeneric:
		return BuildGenericWebhookPayload(request.Generic)
	case PayloadTargetSlack:
		return BuildSlackWebhookPayload(request.Slack)
	default:
		return nil, fmt.Errorf("unsupported payload target: %s", request.Target)
	}
}

func parseReportFormat(raw string, allowAll bool) (ReportFormat, error) {
	switch ReportFormat(strings.ToLower(strings.TrimSpace(raw))) {
	case "":
		return defaultReportFormat, nil
	case ReportFormatMarkdown, ReportFormatJSON, ReportFormatCSV:
		return ReportFormat(strings.ToLower(strings.TrimSpace(raw))), nil
	case ReportFormatAll:
		if allowAll {
			return ReportFormatAll, nil
		}
	}
	if allowAll {
		return "", fmt.Errorf("--format must be one of: markdown, json, csv, all")
	}
	return "", fmt.Errorf("--format must be one of: json, csv")
}

func parsePayloadTarget(raw string) (PayloadTarget, error) {
	switch PayloadTarget(strings.ToLower(strings.TrimSpace(raw))) {
	case PayloadTargetGeneric:
		return PayloadTargetGeneric, nil
	case PayloadTargetSlack:
		return PayloadTargetSlack, nil
	default:
		return "", fmt.Errorf("--target must be one of: generic, slack")
	}
}

func parseOptionalFloat(raw string, fallback float64, field string) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", field, raw)
	}
	return value, nil
}

func decodeCostCenters(raw string) ([]CostCenterAllocation, error) {
	var values []CostCenterAllocation
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode cost centers: %w", err)
	}
	return values, nil
}

func decodeModels(raw string) ([]ModelAllocation, error) {
	var values []ModelAllocation
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	return values, nil
}

func decodeAnomalies(raw string) ([]Anomaly, error) {
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

func envString(env Environment, key string) string {
	if env == nil {
		return ""
	}
	return env.String(key)
}

func envFloat(env Environment, key string, fallback float64) float64 {
	raw := strings.TrimSpace(envString(env, key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envInt64(env Environment, key string, fallback int64) int64 {
	raw := strings.TrimSpace(envString(env, key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(env Environment, key string, fallback bool) bool {
	raw := strings.TrimSpace(envString(env, key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func chargebackTimestamp(env Environment, now time.Time) string {
	if env == nil {
		return now.UTC().Format(time.RFC3339)
	}
	return env.ChargebackTimestamp(now)
}
