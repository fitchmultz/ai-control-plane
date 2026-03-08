// runtime_service_test.go - Coverage for typed runtime database inspection.
//
// Purpose:
//   - Verify runtime service health adaptation and summary shaping.
//
// Responsibilities:
//   - Cover nil guards, config-error passthrough, accessibility checks, and summary output.
//
// Scope:
//   - Runtime inspection behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests avoid Docker by exercising external-mode summaries with sqlmock.
package db

import (
	"context"
	"errors"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func TestRuntimeServiceRequiresConnector(t *testing.T) {
	service := &RuntimeService{}

	if _, err := service.Summary(context.Background()); err == nil {
		t.Fatal("expected connector guard error")
	}
	if err := service.ConfigError(); err == nil {
		t.Fatal("expected connector guard error")
	}
}

func TestRuntimeServiceConfigErrorPassthrough(t *testing.T) {
	service := NewRuntimeService(&Connector{
		settings: config.DatabaseSettings{AmbiguousErr: errors.New("ambiguous")},
	})

	if err := service.ConfigError(); err == nil || err.Error() != "ambiguous" {
		t.Fatalf("unexpected ConfigError() = %v", err)
	}
}

func TestRuntimeServiceIsAccessibleReflectsPing(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	mock.ExpectPing()

	if !NewRuntimeService(connector).IsAccessible(context.Background()) {
		t.Fatal("expected external database to be accessible")
	}
}

func TestRuntimeServiceSummaryReturnsProbeError(t *testing.T) {
	service := NewRuntimeService(&Connector{
		settings: config.DatabaseSettings{AmbiguousErr: errors.New("ambiguous")},
	})

	summary, err := service.Summary(context.Background())
	if err == nil {
		t.Fatal("expected summary error")
	}
	if summary.Ping.Error != "ambiguous" {
		t.Fatalf("unexpected summary ping error: %+v", summary)
	}
}

func TestRuntimeServiceSummaryExternalSuccess(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewRuntimeService(connector)

	mock.ExpectPing()
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('LiteLLM_VerificationToken', 'LiteLLM_UserTable', 'LiteLLM_BudgetTable', 'LiteLLM_SpendLogs');
	`, exactQueryRows("count").AddRow("4"))
	expectExactQuery(mock, "SELECT version();", exactQueryRows("version").AddRow("PostgreSQL 17.2"))
	expectExactQuery(mock, "SELECT pg_size_pretty(pg_database_size('litellm'));", exactQueryRows("size").AddRow("42 MB"))
	expectExactQuery(mock, "SELECT COUNT(*) FROM pg_stat_activity WHERE datname = 'litellm';", exactQueryRows("count").AddRow("6"))

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.Mode != config.DatabaseModeExternal || summary.DatabaseName != "litellm" || summary.DatabaseUser != "litellm" {
		t.Fatalf("unexpected identity fields: %+v", summary)
	}
	if !summary.Ping.Healthy || summary.ExpectedTables != 4 || summary.Version != "PostgreSQL 17.2" || summary.Size != "42 MB" || summary.Connections != 6 {
		t.Fatalf("unexpected runtime summary: %+v", summary)
	}
}
