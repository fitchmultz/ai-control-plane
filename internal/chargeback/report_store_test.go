// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Verify the DB-backed chargeback store against a fake typed reader.
//
// Responsibilities:
//   - Cover nil-reader guards and JSON decode failures.
//   - Verify typed decoding for store methods that shape report inputs.
//   - Cover JSON fallback helpers used by report assembly.
//
// Scope:
//   - Unit tests for DBStore behavior only.
//
// Usage:
//   - Run with `go test ./internal/chargeback`.
//
// Invariants/Assumptions:
//   - Tests do not require a live database connection.
//   - The fake reader matches the narrow DB store contract exactly.
package chargeback

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/db"
)

type fakeChargebackReader struct {
	costCentersJSON string
	modelsJSON      string
	principalsJSON  string
	historyJSON     string
	totalSpend      float64
	totalBudget     float64
	metrics         db.ChargebackMetricsSummary
	costCentersErr  error
	modelsErr       error
	principalsErr   error
	totalSpendErr   error
	metricsErr      error
	historyErr      error
	totalBudgetErr  error
}

func (f fakeChargebackReader) CostCenterAllocationsJSON(context.Context, string, string) (string, error) {
	return f.costCentersJSON, f.costCentersErr
}
func (f fakeChargebackReader) ModelAllocationsJSON(context.Context, string, string) (string, error) {
	return f.modelsJSON, f.modelsErr
}
func (f fakeChargebackReader) TopPrincipalsJSON(context.Context, string, string, int) (string, error) {
	return f.principalsJSON, f.principalsErr
}
func (f fakeChargebackReader) TotalSpend(context.Context, string, string) (float64, error) {
	return f.totalSpend, f.totalSpendErr
}
func (f fakeChargebackReader) MetricsSummary(context.Context, string, string) (db.ChargebackMetricsSummary, error) {
	return f.metrics, f.metricsErr
}
func (f fakeChargebackReader) HistoricalSpendJSON(context.Context, int, string) (string, error) {
	return f.historyJSON, f.historyErr
}
func (f fakeChargebackReader) TotalBudget(context.Context) (float64, error) {
	return f.totalBudget, f.totalBudgetErr
}

func TestDBStore_RequireReaderAndEncodeJSON(t *testing.T) {
	t.Parallel()

	var store *DBStore
	if _, err := store.requireReader(); err == nil || !strings.Contains(err.Error(), "requires a database reader") {
		t.Fatalf("expected requireReader guard, got %v", err)
	}
	if got := encodeJSON(make(chan int)); got != "[]" {
		t.Fatalf("expected encodeJSON fallback, got %q", got)
	}
}

func TestDBStore_DecodesTypedValuesAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	reader := fakeChargebackReader{
		costCentersJSON: `[{"cost_center":"1001","team":"platform","request_count":5,"token_count":50,"spend_amount":25.5,"percent_of_total":100}]`,
		modelsJSON:      `[{"model":"gpt-4o-mini","request_count":5,"token_count":50,"spend_amount":25.5}]`,
		principalsJSON:  `[{"principal":"alice","team":"platform","cost_center":"1001","request_count":5,"spend_amount":25.5}]`,
		historyJSON:     `[{"month":"2026-01","spend":15.5}]`,
		totalSpend:      25.5,
		totalBudget:     100,
		metrics:         db.ChargebackMetricsSummary{TotalRequests: 5, TotalTokens: 50},
	}
	store := NewDBStore(reader)
	ctx := context.Background()

	costCenters, err := store.CostCenterAllocations(ctx, "2026-02-01", "2026-02-28")
	if err != nil || len(costCenters) != 1 || costCenters[0].CostCenter != "1001" {
		t.Fatalf("unexpected cost centers: %#v err=%v", costCenters, err)
	}
	models, err := store.ModelAllocations(ctx, "2026-02-01", "2026-02-28")
	if err != nil || len(models) != 1 || models[0].Model != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %#v err=%v", models, err)
	}
	principals, err := store.TopPrincipals(ctx, "2026-02-01", "2026-02-28", 10)
	if err != nil || len(principals) != 1 || principals[0].Principal != "alice" {
		t.Fatalf("unexpected principals: %#v err=%v", principals, err)
	}
	totalSpend, err := store.TotalSpend(ctx, "2026-02-01", "2026-02-28")
	if err != nil || totalSpend != 25.5 {
		t.Fatalf("unexpected total spend: %v err=%v", totalSpend, err)
	}
	metrics, err := store.Metrics(ctx, "2026-02-01", "2026-02-28")
	if err != nil || metrics.TotalRequests != 5 || metrics.TotalTokens != 50 {
		t.Fatalf("unexpected metrics: %#v err=%v", metrics, err)
	}
	history, err := store.HistoricalSpend(ctx, 3, "2026-02-01")
	if err != nil || len(history) != 1 || history[0].Spend != 15.5 {
		t.Fatalf("unexpected history: %#v err=%v", history, err)
	}
	totalBudget, err := store.TotalBudget(ctx)
	if err != nil || totalBudget != 100 {
		t.Fatalf("unexpected total budget: %v err=%v", totalBudget, err)
	}

	_, err = NewDBStore(fakeChargebackReader{costCentersErr: errors.New("db down")}).CostCenterAllocations(ctx, "2026-02-01", "2026-02-28")
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("expected reader error propagation, got %v", err)
	}
	_, err = NewDBStore(fakeChargebackReader{principalsJSON: `[`}).TopPrincipals(ctx, "2026-02-01", "2026-02-28", 10)
	if err == nil || !strings.Contains(err.Error(), "decode principals") {
		t.Fatalf("expected principal decode failure, got %v", err)
	}
	_, err = NewDBStore(fakeChargebackReader{historyJSON: `[`}).HistoricalSpend(ctx, 3, "2026-02-01")
	if err == nil || !strings.Contains(err.Error(), "decode historical spend") {
		t.Fatalf("expected history decode failure, got %v", err)
	}
}
