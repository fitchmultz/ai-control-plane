// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Centralize domain types shared by chargeback input decoding, workflow
//     orchestration, storage, rendering, archival, and notification delivery.
//
// Responsibilities:
//   - Describe report requests, runtime context, collected spend data, and
//     rendered execution results.
//   - Keep shared types out of CLI wrappers and database adapters.
//
// Non-scope:
//   - Does not parse CLI flags directly.
//   - Does not execute database queries or HTTP requests.
//
// Invariants/Assumptions:
//   - Report requests target a single reporting month and validated output
//     format.
//   - Render outputs are deterministic for equivalent inputs.
//
// Scope:
//   - Shared chargeback domain types only.
//
// Usage:
//   - Used through package exports and CLI entrypoints as applicable.
package chargeback

import (
	"context"
	"time"
)

const (
	defaultReportFormat             ReportFormat = "markdown"
	defaultArchiveDir                            = "demo/backups/chargeback"
	defaultVarianceThreshold                     = 15.0
	defaultAnomalyThreshold                      = 200.0
	defaultBudgetAlertThreshold                  = 80.0
	defaultGenericNotificationEvent              = "chargeback_report_generated"
	defaultSlackColor                            = "good"
	defaultSchemaVersion                         = "1.0.0"
)

type ReportFormat string

const (
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatCSV      ReportFormat = "csv"
	ReportFormatAll      ReportFormat = "all"
)

type PayloadTarget string

const (
	PayloadTargetGeneric PayloadTarget = "generic"
	PayloadTargetSlack   PayloadTarget = "slack"
)

type ReportCommandInput struct {
	ReportMonth          string
	Format               string
	ArchiveDir           string
	VarianceThreshold    string
	AnomalyThreshold     string
	BudgetAlertThreshold string
	ForecastEnabled      *bool
	Notify               bool
}

type RenderCommandInput struct {
	Format string
}

type PayloadCommandInput struct {
	Target string
}

type ReportRequest struct {
	ReportMonth          string
	Format               ReportFormat
	ArchiveDir           string
	VarianceThreshold    float64
	AnomalyThreshold     float64
	ForecastEnabled      bool
	BudgetAlertThreshold float64
	Notify               bool
}

type NotificationConfig struct {
	GenericWebhookURL string
	SlackWebhookURL   string
}

type ReportWorkflowInput struct {
	Request      ReportRequest
	RepoRoot     string
	Notification NotificationConfig
	Now          func() time.Time
}

type RenderRequest struct {
	Format ReportFormat
	Input  ReportInput
}

type PayloadRequest struct {
	Target  PayloadTarget
	Generic GenericWebhookInput
	Slack   SlackWebhookInput
}

type Metrics struct {
	TotalRequests int64
	TotalTokens   int64
}

type HistoricalSpend struct {
	Month string  `json:"month"`
	Spend float64 `json:"spend"`
}

type PrincipalSpend struct {
	Principal    string  `json:"principal"`
	Team         string  `json:"team"`
	CostCenter   string  `json:"cost_center"`
	RequestCount int64   `json:"request_count"`
	SpendAmount  float64 `json:"spend_amount"`
}

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

type MonthRange struct {
	ReportMonth    string
	MonthStart     string
	MonthEnd       string
	PreviousStart  string
	PreviousEnd    string
	GeneratedAtUTC time.Time
}

type ReportData struct {
	Range            MonthRange
	Input            ReportInput
	TopPrincipals    []PrincipalSpend
	VarianceExceeded bool
	HasAnomalies     bool
}

type ReportOutputs struct {
	Markdown string
	JSON     string
	CSV      string
	Archived map[string]string
}

type ReportResult struct {
	Data    ReportData
	Outputs ReportOutputs
}

type Store interface {
	CostCenterAllocations(ctx context.Context, monthStart string, monthEnd string) ([]CostCenterAllocation, error)
	ModelAllocations(ctx context.Context, monthStart string, monthEnd string) ([]ModelAllocation, error)
	TopPrincipals(ctx context.Context, monthStart string, monthEnd string, limit int) ([]PrincipalSpend, error)
	TotalSpend(ctx context.Context, monthStart string, monthEnd string) (float64, error)
	Metrics(ctx context.Context, monthStart string, monthEnd string) (Metrics, error)
	HistoricalSpend(ctx context.Context, monthsBack int, monthStart string) ([]HistoricalSpend, error)
	TotalBudget(ctx context.Context) (float64, error)
}

type ReportRenderer interface {
	Render(data ReportData) (ReportOutputs, error)
}

type ReportArchiver interface {
	Archive(repoRoot string, archiveDir string, reportMonth string, outputs ReportOutputs) (map[string]string, error)
}

type ReportNotifier interface {
	Notify(ctx context.Context, config NotificationConfig, data ReportData) error
}
