// Package chargeback provides canonical report generation helpers.
//
// Purpose:
//   - Implement the typed database-backed chargeback report data store.
//
// Responsibilities:
//   - Query the canonical LiteLLM/PostgreSQL tables through the typed DB client.
//   - Decode JSON-backed report sections into strongly typed structures.
//
// Non-scope:
//   - Does not render reports.
//   - Does not perform notification delivery.
//
// Invariants/Assumptions:
//   - SQL remains read-only.
//   - Query outputs are JSON arrays or scalar values suitable for typed decoding.
//
// Scope:
//   - Database adapter implementation only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package chargeback

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
)

type DBStore struct {
	Reader *db.ChargebackReader
}

func NewDBStore(reader *db.ChargebackReader) *DBStore {
	return &DBStore{Reader: reader}
}

func (s *DBStore) CostCenterAllocations(ctx context.Context, monthStart string, monthEnd string) ([]CostCenterAllocation, error) {
	reader, err := s.requireReader()
	if err != nil {
		return nil, err
	}
	raw, err := reader.CostCenterAllocationsJSON(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	return decodeCostCenters(raw)
}

func (s *DBStore) ModelAllocations(ctx context.Context, monthStart string, monthEnd string) ([]ModelAllocation, error) {
	reader, err := s.requireReader()
	if err != nil {
		return nil, err
	}
	raw, err := reader.ModelAllocationsJSON(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	return decodeModels(raw)
}

func (s *DBStore) TopPrincipals(ctx context.Context, monthStart string, monthEnd string, limit int) ([]PrincipalSpend, error) {
	reader, err := s.requireReader()
	if err != nil {
		return nil, err
	}
	raw, err := reader.TopPrincipalsJSON(ctx, monthStart, monthEnd, limit)
	if err != nil {
		return nil, err
	}
	var values []PrincipalSpend
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode principals: %w", err)
	}
	return values, nil
}

func (s *DBStore) TotalSpend(ctx context.Context, monthStart string, monthEnd string) (float64, error) {
	reader, err := s.requireReader()
	if err != nil {
		return 0, err
	}
	return reader.TotalSpend(ctx, monthStart, monthEnd)
}

func (s *DBStore) Metrics(ctx context.Context, monthStart string, monthEnd string) (Metrics, error) {
	reader, err := s.requireReader()
	if err != nil {
		return Metrics{}, err
	}
	summary, err := reader.MetricsSummary(ctx, monthStart, monthEnd)
	if err != nil {
		return Metrics{}, err
	}
	return Metrics{TotalRequests: summary.TotalRequests, TotalTokens: summary.TotalTokens}, nil
}

func (s *DBStore) HistoricalSpend(ctx context.Context, monthsBack int, monthStart string) ([]HistoricalSpend, error) {
	reader, err := s.requireReader()
	if err != nil {
		return nil, err
	}
	raw, err := reader.HistoricalSpendJSON(ctx, monthsBack, monthStart)
	if err != nil {
		return nil, err
	}
	var values []HistoricalSpend
	if err := decodeJSONArray(raw, &values); err != nil {
		return nil, fmt.Errorf("decode historical spend: %w", err)
	}
	return values, nil
}

func (s *DBStore) TotalBudget(ctx context.Context) (float64, error) {
	reader, err := s.requireReader()
	if err != nil {
		return 0, err
	}
	return reader.TotalBudget(ctx)
}

func (s *DBStore) requireReader() (*db.ChargebackReader, error) {
	if s == nil || s.Reader == nil {
		return nil, fmt.Errorf("chargeback DB store requires a database reader")
	}
	return s.Reader, nil
}

func encodeJSON(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
