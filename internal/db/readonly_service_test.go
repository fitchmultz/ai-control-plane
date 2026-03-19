// readonly_service_test.go - Coverage for typed readonly database summaries.
//
// Purpose:
//   - Verify readonly services translate SQL results into typed summaries.
//
// Responsibilities:
//   - Cover nil-guard behavior and summary calculations.
//   - Exercise detection-table present and absent flows.
//
// Scope:
//   - Readonly summary behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - SQL responses are deterministic via sqlmock.
package db

import (
	"context"
	"testing"
)

func TestReadonlyServiceRequiresConnector(t *testing.T) {
	service := &ReadonlyService{}

	if _, err := service.KeySummary(context.Background()); err == nil {
		t.Fatal("expected connector guard error")
	}
	if _, err := service.BudgetSummary(context.Background()); err == nil {
		t.Fatal("expected connector guard error")
	}
	if _, err := service.DetectionSummary(context.Background()); err == nil {
		t.Fatal("expected connector guard error")
	}
	if _, err := service.TrafficSummary(context.Background()); err == nil {
		t.Fatal("expected connector guard error")
	}
}

func TestReadonlyServiceKeySummary(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `SELECT COUNT(*) FROM "LiteLLM_VerificationToken";`, exactQueryRows("count").AddRow("10"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_VerificationToken"
		WHERE expires IS NULL OR expires > NOW();
	`, exactQueryRows("count").AddRow("7"))

	summary, err := service.KeySummary(context.Background())
	if err != nil {
		t.Fatalf("KeySummary() error = %v", err)
	}
	if summary.Total != 10 || summary.Active != 7 || summary.Expired != 3 {
		t.Fatalf("unexpected key summary: %+v", summary)
	}
}

func TestReadonlyServiceBudgetSummary(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `SELECT COUNT(*) FROM "LiteLLM_BudgetTable";`, exactQueryRows("count").AddRow("8"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE max_budget > 0 AND (budget::float / max_budget::float * 100) <= 20;
	`, exactQueryRows("count").AddRow("2"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE budget <= 0;
	`, exactQueryRows("count").AddRow("1"))

	summary, err := service.BudgetSummary(context.Background())
	if err != nil {
		t.Fatalf("BudgetSummary() error = %v", err)
	}
	if summary.Total != 8 || summary.HighUtilization != 2 || summary.Exhausted != 1 {
		t.Fatalf("unexpected budget summary: %+v", summary)
	}
}

func TestReadonlyServiceDetectionSummaryWithoutSpendLogsTable(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`, exactQueryRows("count").AddRow("0"))

	summary, err := service.DetectionSummary(context.Background())
	if err != nil {
		t.Fatalf("DetectionSummary() error = %v", err)
	}
	if summary.SpendLogsTableExists {
		t.Fatalf("expected zero-value summary when table is absent: %+v", summary)
	}
}

func TestReadonlyServiceDetectionSummaryWithSpendLogsTable(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`, exactQueryRows("count").AddRow("1"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`, exactQueryRows("count").AddRow("4"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 5.0 AND spend <= 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`, exactQueryRows("count").AddRow("3"))
	expectExactQuery(mock, `
		SELECT COUNT(DISTINCT model) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`, exactQueryRows("count").AddRow("5"))
	expectExactQuery(mock, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`, exactQueryRows("count").AddRow("11"))

	summary, err := service.DetectionSummary(context.Background())
	if err != nil {
		t.Fatalf("DetectionSummary() error = %v", err)
	}
	if !summary.SpendLogsTableExists || summary.HighSeverity != 4 || summary.MediumSeverity != 3 || summary.UniqueModels24h != 5 || summary.TotalEntries24h != 11 {
		t.Fatalf("unexpected detection summary: %+v", summary)
	}
}

func TestReadonlyServiceTrafficSummary(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`, exactQueryRows("count").AddRow("1"))
	expectExactQuery(mock, `
		SELECT
			COALESCE(COUNT(*), 0) || '|' ||
			COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0) || '|' ||
			COALESCE(SUM(spend), 0) || '|' ||
			COUNT(*) FILTER (WHERE status != 'success')
		FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`, exactQueryRows("count").AddRow("25|4096|12.3456|3"))

	summary, err := service.TrafficSummary(context.Background())
	if err != nil {
		t.Fatalf("TrafficSummary() error = %v", err)
	}
	if !summary.SpendLogsTableExists || summary.TotalRequests24h != 25 || summary.TotalTokens24h != 4096 || summary.TotalSpend24h != 12.3456 || summary.ErrorRequests24h != 3 {
		t.Fatalf("unexpected traffic summary: %+v", summary)
	}
}

func TestReadonlyServiceTrafficSummaryWithoutSpendLogsTable(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	expectExactQuery(mock, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`, exactQueryRows("count").AddRow("0"))

	summary, err := service.TrafficSummary(context.Background())
	if err != nil {
		t.Fatalf("TrafficSummary() error = %v", err)
	}
	if summary.SpendLogsTableExists {
		t.Fatalf("expected zero-value traffic summary when table is absent: %+v", summary)
	}
}
