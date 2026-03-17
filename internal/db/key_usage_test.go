// key_usage_test.go - Coverage for typed key-usage reporting.
//
// Purpose:
//   - Verify key lifecycle usage queries translate SQL results into typed summaries.
//
// Responsibilities:
//   - Cover nil-guard behavior and totals/model decoding.
//   - Keep parsing deterministic for inspection workflows.
//
// Scope:
//   - Key usage query behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests use sqlmock-backed external connectors.
package db

import (
	"context"
	"regexp"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/keygen"
)

func TestReadonlyServiceKeyUsageRequiresConnector(t *testing.T) {
	service := &ReadonlyService{}
	if _, err := service.KeyUsage(context.Background(), "demo", keygen.MonthWindow{ReportMonth: "2026-03", Start: "2026-03-01", End: "2026-03-31"}); err == nil {
		t.Fatal("expected connector guard error")
	}
}

func TestReadonlyServiceKeyUsage(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	service := NewReadonlyService(connector)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT
  COUNT(*) || '|' ||`)).WillReturnRows(exactQueryRows("value").AddRow("7|99|4.2|2026-03-06T12:00:00Z"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(`)).WillReturnRows(exactQueryRows("value").AddRow(`[{"model":"openai-gpt5.2","request_count":5,"token_count":70,"spend_amount":3.1}]`))

	usage, err := service.KeyUsage(context.Background(), "alice", keygen.MonthWindow{
		ReportMonth: "2026-03",
		Start:       "2026-03-01",
		End:         "2026-03-31",
	})
	if err != nil {
		t.Fatalf("KeyUsage() error = %v", err)
	}
	if usage.Alias != "alice" || usage.ReportMonth != "2026-03" || usage.TotalRequests != 7 || usage.TotalTokens != 99 || usage.TotalSpend != 4.2 {
		t.Fatalf("unexpected usage summary: %+v", usage)
	}
	if len(usage.ByModel) != 1 || usage.ByModel[0].Model != "openai-gpt5.2" {
		t.Fatalf("unexpected model usage: %+v", usage.ByModel)
	}
}

func TestParseKeyUsageTotals(t *testing.T) {
	usage, err := parseKeyUsageTotals("demo", "2026-03", "3|120|1.5|2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("parseKeyUsageTotals() error = %v", err)
	}
	if usage.TotalRequests != 3 || usage.TotalTokens != 120 || usage.TotalSpend != 1.5 {
		t.Fatalf("unexpected parsed usage: %+v", usage)
	}
	if _, err := parseKeyUsageTotals("demo", "2026-03", "broken"); err == nil {
		t.Fatal("expected parseKeyUsageTotals error")
	}
}
