// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Define typed report-generation structures shared by chargeback collection,
//     analysis, rendering, archival, and notification flows.
//
// Responsibilities:
//   - Describe report options, collected spend data, and execution results.
//   - Keep shared types out of CLI wrappers and database adapters.
//
// Non-scope:
//   - Does not execute database queries.
//   - Does not implement CLI argument parsing.
//
// Invariants/Assumptions:
//   - Report options describe one target reporting month.
//   - Render outputs are deterministic for equivalent inputs.
//
// Scope:
//   - Shared chargeback report types only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"context"
	"io"
	"time"
)

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

type MonthRange struct {
	ReportMonth    string
	MonthStart     string
	MonthEnd       string
	PreviousStart  string
	PreviousEnd    string
	GeneratedAtUTC time.Time
}

type ReportOptions struct {
	ReportMonth          string
	Format               string
	ArchiveDir           string
	VarianceThreshold    float64
	AnomalyThreshold     float64
	ForecastEnabled      bool
	BudgetAlertThreshold float64
	Notify               bool
	GenericWebhookURL    string
	SlackWebhookURL      string
	RepoRoot             string
	Stdout               io.Writer
	Stderr               io.Writer
	Now                  func() time.Time
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
