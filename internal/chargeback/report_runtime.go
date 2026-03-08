// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Coordinate typed chargeback report generation, archival, and notification
//     delivery across one canonical workflow.
//
// Responsibilities:
//   - Collect report data from the typed store.
//   - Render Markdown, JSON, and CSV outputs.
//   - Archive rendered outputs and optionally deliver notifications.
//
// Non-scope:
//   - Does not parse CLI flags.
//   - Does not own shell wrapper behavior.
//
// Invariants/Assumptions:
//   - Notification delivery is explicit via `Notify`.
//   - Archive paths resolve relative to RepoRoot when not absolute.
//
// Scope:
//   - Top-level report workflow only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultReportFormat             = "markdown"
	defaultArchiveDir               = "demo/backups/chargeback"
	defaultVarianceThreshold        = 15
	defaultAnomalyThreshold         = 200
	defaultBudgetAlertThreshold     = 80
	defaultGenericNotificationEvent = "chargeback_report_generated"
)

func GenerateReport(ctx context.Context, store Store, opts ReportOptions) (ReportResult, error) {
	opts = withReportDefaults(opts)
	monthRange, err := resolveMonthRange(opts.ReportMonth, opts.Now())
	if err != nil {
		return ReportResult{}, err
	}

	currentCostCenters, err := store.CostCenterAllocations(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect cost center allocations: %w", err)
	}
	previousCostCenters, err := store.CostCenterAllocations(ctx, monthRange.PreviousStart, monthRange.PreviousEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect previous cost center allocations: %w", err)
	}
	models, err := store.ModelAllocations(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect model allocations: %w", err)
	}
	principals, err := store.TopPrincipals(ctx, monthRange.MonthStart, monthRange.MonthEnd, 20)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect top principals: %w", err)
	}
	totalSpend, err := store.TotalSpend(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect total spend: %w", err)
	}
	previousSpend, err := store.TotalSpend(ctx, monthRange.PreviousStart, monthRange.PreviousEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect previous total spend: %w", err)
	}
	metrics, err := store.Metrics(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportResult{}, fmt.Errorf("collect metrics: %w", err)
	}

	var (
		forecast1   *float64
		forecast2   *float64
		forecast3   *float64
		dailyBurn   float64
		daysRemain  *int64
		exhaustDate = "N/A"
		totalBudget float64
		risk        = BudgetRisk{RiskLevel: "unknown", ThresholdExceeded: false}
	)
	if opts.ForecastEnabled {
		history, historyErr := store.HistoricalSpend(ctx, 6, monthRange.MonthStart)
		if historyErr != nil {
			return ReportResult{}, fmt.Errorf("collect historical spend: %w", historyErr)
		}
		forecast1, forecast2, forecast3 = forecastSpend(history)
		totalBudget, err = store.TotalBudget(ctx)
		if err != nil {
			return ReportResult{}, fmt.Errorf("collect total budget: %w", err)
		}
		dailyBurn, daysRemain, exhaustDate = calculateBurnRate(totalSpend, monthRange.MonthStart, totalBudget, opts.Now())
		risk = budgetRisk(forecast1, forecast2, forecast3, totalBudget, opts.BudgetAlertThreshold)
	}

	variance := varianceString(totalSpend, previousSpend)
	anomalies := detectAnomalies(currentCostCenters, previousCostCenters, opts.AnomalyThreshold)
	input := ReportInput{
		SchemaVersion:      "1.0.0",
		GeneratedAt:        monthRange.GeneratedAtUTC.Format(time.RFC3339),
		ReportMonth:        monthRange.ReportMonth,
		PeriodStart:        monthRange.MonthStart,
		PeriodEnd:          monthRange.MonthEnd,
		TotalSpend:         totalSpend,
		TotalRequests:      metrics.TotalRequests,
		TotalTokens:        metrics.TotalTokens,
		CostCenters:        currentCostCenters,
		Models:             models,
		Variance:           variance,
		VarianceThreshold:  opts.VarianceThreshold,
		PreviousMonthSpend: previousSpend,
		Anomalies:          anomalies,
		ForecastEnabled:    opts.ForecastEnabled,
		ForecastMonth1:     forecast1,
		ForecastMonth2:     forecast2,
		ForecastMonth3:     forecast3,
		DailyBurn:          dailyBurn,
		DaysRemaining:      daysRemain,
		ExhaustionDate:     exhaustDate,
		TotalBudget:        totalBudget,
		BudgetRisk:         risk,
		AnomalyThreshold:   opts.AnomalyThreshold,
	}
	data := ReportData{
		Range:            monthRange,
		Input:            input,
		TopPrincipals:    principals,
		VarianceExceeded: varianceThresholdExceeded(variance, opts.VarianceThreshold),
		HasAnomalies:     len(anomalies) > 0,
	}

	outputs, err := renderOutputs(data)
	if err != nil {
		return ReportResult{}, err
	}
	archived, err := archiveOutputs(opts, data.Range.ReportMonth, outputs)
	if err != nil {
		return ReportResult{}, err
	}
	outputs.Archived = archived

	if opts.Notify {
		if err := sendNotifications(ctx, opts, data); err != nil {
			return ReportResult{}, err
		}
	}
	return ReportResult{Data: data, Outputs: outputs}, nil
}

func WriteSelectedOutput(result ReportResult, opts ReportOptions) error {
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = defaultReportFormat
	}
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	switch format {
	case "markdown":
		_, err := io.WriteString(out, result.Outputs.Markdown)
		return err
	case "json":
		_, err := io.WriteString(out, result.Outputs.JSON)
		return err
	case "csv":
		_, err := io.WriteString(out, result.Outputs.CSV)
		return err
	case "all":
		_, err := io.WriteString(out, result.Outputs.Markdown)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func withReportDefaults(opts ReportOptions) ReportOptions {
	if opts.Format == "" {
		opts.Format = defaultReportFormat
	}
	if opts.ArchiveDir == "" {
		opts.ArchiveDir = defaultArchiveDir
	}
	if opts.VarianceThreshold == 0 {
		opts.VarianceThreshold = defaultVarianceThreshold
	}
	if opts.AnomalyThreshold == 0 {
		opts.AnomalyThreshold = defaultAnomalyThreshold
	}
	if opts.BudgetAlertThreshold == 0 {
		opts.BudgetAlertThreshold = defaultBudgetAlertThreshold
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	return opts
}

func renderOutputs(data ReportData) (ReportOutputs, error) {
	var (
		jsonBuffer bytes.Buffer
		csvBuffer  bytes.Buffer
	)
	if err := RenderJSON(&jsonBuffer, data.Input); err != nil {
		return ReportOutputs{}, fmt.Errorf("render json: %w", err)
	}
	if err := RenderCSV(&csvBuffer, data.Input.ReportMonth, data.Input.CostCenters); err != nil {
		return ReportOutputs{}, fmt.Errorf("render csv: %w", err)
	}
	return ReportOutputs{
		Markdown: RenderMarkdown(data),
		JSON:     jsonBuffer.String(),
		CSV:      csvBuffer.String(),
		Archived: map[string]string{},
	}, nil
}

func archiveOutputs(opts ReportOptions, reportMonth string, outputs ReportOutputs) (map[string]string, error) {
	archiveBase := resolveArchiveBase(opts.RepoRoot, opts.ArchiveDir)
	if archiveBase == "" {
		return map[string]string{}, nil
	}
	targetDir := filepath.Join(archiveBase, reportMonth)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create archive directory: %w", err)
	}

	paths := map[string]string{}
	files := map[string]string{
		"md":   outputs.Markdown,
		"json": outputs.JSON,
		"csv":  outputs.CSV,
	}
	for extension, content := range files {
		path := filepath.Join(targetDir, fmt.Sprintf("chargeback-report-%s.%s", reportMonth, extension))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write archive %s: %w", path, err)
		}
		paths[extension] = path
	}
	return paths, nil
}

func resolveArchiveBase(repoRoot string, archiveDir string) string {
	trimmed := strings.TrimSpace(archiveDir)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) || strings.TrimSpace(repoRoot) == "" {
		return trimmed
	}
	return filepath.Join(repoRoot, trimmed)
}

func sendNotifications(ctx context.Context, opts ReportOptions, data ReportData) error {
	if strings.TrimSpace(opts.GenericWebhookURL) == "" && strings.TrimSpace(opts.SlackWebhookURL) == "" {
		return nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if strings.TrimSpace(opts.GenericWebhookURL) != "" {
		payload, err := BuildGenericWebhookPayload(GenericWebhookInput{
			Event:       defaultGenericNotificationEvent,
			ReportMonth: data.Input.ReportMonth,
			TotalSpend:  data.Input.TotalSpend,
			Variance:    data.Input.Variance,
			Anomalies:   data.Input.Anomalies,
			Timestamp:   data.Range.GeneratedAtUTC.Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("build generic notification payload: %w", err)
		}
		if err := postJSON(ctx, client, opts.GenericWebhookURL, payload); err != nil {
			return fmt.Errorf("send generic notification: %w", err)
		}
	}
	if strings.TrimSpace(opts.SlackWebhookURL) != "" {
		color := "good"
		if data.VarianceExceeded {
			color = "danger"
		}
		payload, err := BuildSlackWebhookPayload(SlackWebhookInput{
			ReportMonth: data.Input.ReportMonth,
			TotalSpend:  data.Input.TotalSpend,
			Variance:    data.Input.Variance,
			Color:       color,
			Epoch:       data.Range.GeneratedAtUTC.Unix(),
		})
		if err != nil {
			return fmt.Errorf("build slack notification payload: %w", err)
		}
		if err := postJSON(ctx, client, opts.SlackWebhookURL, payload); err != nil {
			return fmt.Errorf("send slack notification: %w", err)
		}
	}
	return nil
}

func postJSON(ctx context.Context, client *http.Client, targetURL string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP %d", response.StatusCode)
	}
	return nil
}
