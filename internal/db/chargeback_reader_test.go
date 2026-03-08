// chargeback_reader_test.go - Coverage for chargeback-specific database helpers.
//
// Purpose:
//   - Verify chargeback reader parsing and guard behavior.
//
// Responsibilities:
//   - Exercise typed aggregate parsing and error handling.
//   - Cover nil/config-error guard rails on reader construction and query methods.
//
// Scope:
//   - Chargeback-reader behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests use deterministic sqlmock responses instead of a live database.
package db

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func TestNewChargebackReaderRequiresValidConnector(t *testing.T) {
	if _, err := NewChargebackReader(nil); err == nil {
		t.Fatal("expected nil connector rejection")
	}

	connector := &Connector{
		settings: config.DatabaseSettings{AmbiguousErr: errors.New("ambiguous")},
	}
	if _, err := NewChargebackReader(connector); err == nil {
		t.Fatal("expected config error rejection")
	}
}

func TestChargebackReaderScalarStringRequiresConnector(t *testing.T) {
	reader := &ChargebackReader{}
	if _, err := reader.scalarString(context.Background(), "SELECT 1;"); err == nil {
		t.Fatal("expected connector guard error")
	}
}

func TestChargebackReaderMetricsSummary(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	reader, err := NewChargebackReader(connector)
	if err != nil {
		t.Fatalf("NewChargebackReader() error = %v", err)
	}

	expectExactQuery(mock, `
SELECT
  COALESCE(COUNT(*), 0) || '|' ||
  COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '2026-03-01 00:00:00'
  AND "startTime" <= '2026-03-31 23:59:59';
`, exactQueryRows("value").AddRow("9|1200"))

	summary, err := reader.MetricsSummary(context.Background(), "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("MetricsSummary() error = %v", err)
	}
	if summary.TotalRequests != 9 || summary.TotalTokens != 1200 {
		t.Fatalf("unexpected metrics summary: %+v", summary)
	}
}

func TestChargebackReaderMetricsSummaryRejectsMalformedPayload(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	reader, err := NewChargebackReader(connector)
	if err != nil {
		t.Fatalf("NewChargebackReader() error = %v", err)
	}

	expectExactQuery(mock, `
SELECT
  COALESCE(COUNT(*), 0) || '|' ||
  COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '2026-03-01 00:00:00'
  AND "startTime" <= '2026-03-31 23:59:59';
`, exactQueryRows("value").AddRow("broken"))

	if _, err := reader.MetricsSummary(context.Background(), "2026-03-01", "2026-03-31"); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestChargebackReaderTotals(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	reader, err := NewChargebackReader(connector)
	if err != nil {
		t.Fatalf("NewChargebackReader() error = %v", err)
	}

	expectExactQuery(mock, `
SELECT COALESCE(SUM(spend), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '2026-03-01 00:00:00'
  AND "startTime" <= '2026-03-31 23:59:59';
`, exactQueryRows("value").AddRow("12.5"))
	expectExactQuery(mock, `
SELECT COALESCE(SUM(max_budget), 0) AS total_max_budget
FROM "LiteLLM_BudgetTable"
WHERE max_budget > 0;
`, exactQueryRows("value").AddRow("90.75"))

	totalSpend, err := reader.TotalSpend(context.Background(), "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("TotalSpend() error = %v", err)
	}
	if totalSpend != 12.5 {
		t.Fatalf("TotalSpend() = %v, want 12.5", totalSpend)
	}

	totalBudget, err := reader.TotalBudget(context.Background())
	if err != nil {
		t.Fatalf("TotalBudget() error = %v", err)
	}
	if totalBudget != 90.75 {
		t.Fatalf("TotalBudget() = %v, want 90.75", totalBudget)
	}
}

func TestChargebackReaderJSONAccessors(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	reader, err := NewChargebackReader(connector)
	if err != nil {
		t.Fatalf("NewChargebackReader() error = %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`WITH attribution AS (`)).WillReturnRows(exactQueryRows("value").AddRow(`[{"cost_center":"100"}]`))
	mock.ExpectQuery(regexp.QuoteMeta(`WITH aggregated AS (`)).WillReturnRows(exactQueryRows("value").AddRow(`[{"model":"gpt-5.2"}]`))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(`)).WillReturnRows(exactQueryRows("value").AddRow(`[{"principal":"demo"}]`))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(`)).WillReturnRows(exactQueryRows("value").AddRow(`[{"month":"2026-02","spend":10.0}]`))

	costCenters, err := reader.CostCenterAllocationsJSON(context.Background(), "2026-03-01", "2026-03-31")
	if err != nil || costCenters == "" {
		t.Fatalf("CostCenterAllocationsJSON() = %q, %v", costCenters, err)
	}
	models, err := reader.ModelAllocationsJSON(context.Background(), "2026-03-01", "2026-03-31")
	if err != nil || models == "" {
		t.Fatalf("ModelAllocationsJSON() = %q, %v", models, err)
	}
	principals, err := reader.TopPrincipalsJSON(context.Background(), "2026-03-01", "2026-03-31", 5)
	if err != nil || principals == "" {
		t.Fatalf("TopPrincipalsJSON() = %q, %v", principals, err)
	}
	history, err := reader.HistoricalSpendJSON(context.Background(), 3, "2026-03-01")
	if err != nil || history == "" {
		t.Fatalf("HistoricalSpendJSON() = %q, %v", history, err)
	}
}

func TestChargebackParsingHelpers(t *testing.T) {
	if value, err := parseFloat(" 1.5 "); err != nil || value != 1.5 {
		t.Fatalf("parseFloat() = %v, %v", value, err)
	}
	if _, err := parseFloat("bad"); err == nil {
		t.Fatal("expected parseFloat error")
	}
	if value, err := parseInt64(" 9 "); err != nil || value != 9 {
		t.Fatalf("parseInt64() = %v, %v", value, err)
	}
	if _, err := parseInt64("bad"); err == nil {
		t.Fatal("expected parseInt64 error")
	}
}
