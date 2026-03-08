// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Coordinate the report-generation workflow across store, render, archive,
//     and notification layers.
//
// Responsibilities:
//   - Collect chargeback data from the typed store.
//   - Compute analytics and assemble canonical report data.
//   - Delegate rendering, archival, and notification side effects to focused
//     collaborators.
//
// Non-scope:
//   - Does not parse CLI flags or environment variables.
//   - Does not embed SQL or HTTP client setup logic.
//
// Invariants/Assumptions:
//   - Input has already been normalized by the chargeback input adapter.
//   - Collaborators are explicit and replaceable for tests.
//
// Scope:
//   - Chargeback report workflow only.
//
// Usage:
//   - Used by CLI composition and workflow-focused tests.
package chargeback

import (
	"context"
	"fmt"
	"io"
	"time"
)

type ReportWorkflow struct {
	Store    Store
	Renderer ReportRenderer
	Archiver ReportArchiver
	Notifier ReportNotifier
}

func NewReportWorkflow(store Store) ReportWorkflow {
	return ReportWorkflow{
		Store:    store,
		Renderer: DefaultRenderer{},
		Archiver: FileArchiver{},
		Notifier: WebhookNotifier{},
	}
}

func (w ReportWorkflow) Run(ctx context.Context, input ReportWorkflowInput) (ReportResult, error) {
	if w.Store == nil {
		return ReportResult{}, fmt.Errorf("chargeback report workflow requires a store")
	}
	if input.Now == nil {
		return ReportResult{}, fmt.Errorf("chargeback report workflow requires a clock")
	}
	if w.Renderer == nil {
		w.Renderer = DefaultRenderer{}
	}
	if w.Archiver == nil {
		w.Archiver = FileArchiver{}
	}
	if w.Notifier == nil {
		w.Notifier = WebhookNotifier{}
	}

	monthRange, err := resolveMonthRange(input.Request.ReportMonth, input.Now())
	if err != nil {
		return ReportResult{}, err
	}
	data, err := collectReportData(ctx, w.Store, input.Request, monthRange, input.Now)
	if err != nil {
		return ReportResult{}, err
	}
	outputs, err := w.Renderer.Render(data)
	if err != nil {
		return ReportResult{}, err
	}
	archived, err := w.Archiver.Archive(input.RepoRoot, input.Request.ArchiveDir, data.Range.ReportMonth, outputs)
	if err != nil {
		return ReportResult{}, err
	}
	outputs.Archived = archived
	if input.Request.Notify {
		if err := w.Notifier.Notify(ctx, input.Notification, data); err != nil {
			return ReportResult{}, err
		}
	}
	return ReportResult{Data: data, Outputs: outputs}, nil
}

func WriteSelectedOutput(out io.Writer, format ReportFormat, outputs ReportOutputs) error {
	if out == nil {
		out = io.Discard
	}
	switch format {
	case ReportFormatMarkdown, ReportFormatAll:
		_, err := io.WriteString(out, outputs.Markdown)
		return err
	case ReportFormatJSON:
		_, err := io.WriteString(out, outputs.JSON)
		return err
	case ReportFormatCSV:
		_, err := io.WriteString(out, outputs.CSV)
		return err
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func collectReportData(ctx context.Context, store Store, request ReportRequest, monthRange MonthRange, now func() time.Time) (ReportData, error) {
	currentCostCenters, err := store.CostCenterAllocations(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect cost center allocations: %w", err)
	}
	previousCostCenters, err := store.CostCenterAllocations(ctx, monthRange.PreviousStart, monthRange.PreviousEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect previous cost center allocations: %w", err)
	}
	models, err := store.ModelAllocations(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect model allocations: %w", err)
	}
	principals, err := store.TopPrincipals(ctx, monthRange.MonthStart, monthRange.MonthEnd, 20)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect top principals: %w", err)
	}
	totalSpend, err := store.TotalSpend(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect total spend: %w", err)
	}
	previousSpend, err := store.TotalSpend(ctx, monthRange.PreviousStart, monthRange.PreviousEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect previous total spend: %w", err)
	}
	metrics, err := store.Metrics(ctx, monthRange.MonthStart, monthRange.MonthEnd)
	if err != nil {
		return ReportData{}, fmt.Errorf("collect metrics: %w", err)
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
	if request.ForecastEnabled {
		history, historyErr := store.HistoricalSpend(ctx, 6, monthRange.MonthStart)
		if historyErr != nil {
			return ReportData{}, fmt.Errorf("collect historical spend: %w", historyErr)
		}
		forecast1, forecast2, forecast3 = forecastSpend(history)
		totalBudget, err = store.TotalBudget(ctx)
		if err != nil {
			return ReportData{}, fmt.Errorf("collect total budget: %w", err)
		}
		dailyBurn, daysRemain, exhaustDate = calculateBurnRate(totalSpend, monthRange.MonthStart, totalBudget, now())
		risk = budgetRisk(forecast1, forecast2, forecast3, totalBudget, request.BudgetAlertThreshold)
	}

	variance := varianceString(totalSpend, previousSpend)
	anomalies := detectAnomalies(currentCostCenters, previousCostCenters, request.AnomalyThreshold)
	reportInput := ReportInput{
		SchemaVersion:      defaultSchemaVersion,
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
		VarianceThreshold:  request.VarianceThreshold,
		PreviousMonthSpend: previousSpend,
		Anomalies:          anomalies,
		ForecastEnabled:    request.ForecastEnabled,
		ForecastMonth1:     forecast1,
		ForecastMonth2:     forecast2,
		ForecastMonth3:     forecast3,
		DailyBurn:          dailyBurn,
		DaysRemaining:      daysRemain,
		ExhaustionDate:     exhaustDate,
		TotalBudget:        totalBudget,
		BudgetRisk:         risk,
		AnomalyThreshold:   request.AnomalyThreshold,
	}
	return ReportData{
		Range:            monthRange,
		Input:            reportInput,
		TopPrincipals:    principals,
		VarianceExceeded: varianceThresholdExceeded(variance, request.VarianceThreshold),
		HasAnomalies:     len(anomalies) > 0,
	}, nil
}
