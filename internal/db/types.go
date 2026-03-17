// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Define the shared ACP database types and narrow service interfaces.
//
// Responsibilities:
//   - Describe runtime probe and summary payloads.
//   - Describe readonly inventory summaries and chargeback-supporting payloads.
//   - Publish narrow interfaces consumed by callers across status/doctor/ops flows.
//
// Scope:
//   - Shared database service types only.
//
// Usage:
//   - Construct a `Connector`, then derive runtime, readonly, admin, or
//   - chargeback services from it as needed.
//
// Invariants/Assumptions:
//   - Public interfaces expose typed workflows, not generic SQL execution.
package db

import (
	"context"
	"io"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

const (
	defaultDBName          = "litellm"
	defaultDBUser          = "litellm"
	containerLookupTimeout = 5 * time.Second
	queryTimeout           = 10 * time.Second
)

// Probe captures a typed outcome for a single database operation.
type Probe struct {
	Operation string        `json:"operation"`
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// Summary captures typed database runtime health.
type Summary struct {
	Mode           config.DatabaseMode `json:"mode"`
	DatabaseName   string              `json:"database_name"`
	DatabaseUser   string              `json:"database_user"`
	ContainerID    string              `json:"container_id,omitempty"`
	Ping           Probe               `json:"ping"`
	ExpectedTables int                 `json:"expected_tables,omitempty"`
	Version        string              `json:"version,omitempty"`
	Size           string              `json:"size,omitempty"`
	Connections    int                 `json:"connections,omitempty"`
}

// KeySummary captures typed virtual key counts.
type KeySummary struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Expired int `json:"expired"`
}

// BudgetSummary captures typed budget utilization counts.
type BudgetSummary struct {
	Total           int `json:"total"`
	HighUtilization int `json:"high_utilization"`
	Exhausted       int `json:"exhausted"`
}

// DetectionSummary captures typed spend-log finding counts.
type DetectionSummary struct {
	SpendLogsTableExists bool `json:"spend_logs_table_exists"`
	HighSeverity         int  `json:"high_severity"`
	MediumSeverity       int  `json:"medium_severity"`
	UniqueModels24h      int  `json:"unique_models_24h"`
	TotalEntries24h      int  `json:"total_entries_24h"`
}

// ChargebackMetricsSummary captures typed aggregate request/token counts.
type ChargebackMetricsSummary struct {
	TotalRequests int64 `json:"total_requests"`
	TotalTokens   int64 `json:"total_tokens"`
}

// RuntimeServiceReader narrows runtime health access to typed summaries.
type RuntimeServiceReader interface {
	Summary(ctx context.Context) (Summary, error)
	ConfigError() error
}

// ReadonlyServiceReader narrows readonly metrics access for collectors.
type ReadonlyServiceReader interface {
	KeySummary(ctx context.Context) (KeySummary, error)
	BudgetSummary(ctx context.Context) (BudgetSummary, error)
	DetectionSummary(ctx context.Context) (DetectionSummary, error)
}

// BackupServiceReader narrows admin flows to backup and restore operations.
type BackupServiceReader interface {
	Backup(ctx context.Context) (string, error)
	Restore(ctx context.Context, sqlReader io.Reader) error
}

// SQLExecutor narrows release-declared SQL migration execution.
type SQLExecutor interface {
	Execute(ctx context.Context, databaseName string, sql string) error
}
