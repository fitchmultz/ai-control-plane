// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose typed readonly inventory and audit summaries.
//
// Responsibilities:
//   - Report key inventory, budget utilization, and recent detection metrics.
//   - Keep readonly reporting separate from runtime and admin workflows.
//
// Scope:
//   - Readonly summary queries only.
//
// Usage:
//   - Construct via `NewReadonlyService(connector)` for collectors and reports.
//
// Invariants/Assumptions:
//   - Queries are read-only and return typed summaries instead of raw SQL output.
package db

import (
	"context"
	"fmt"
)

// ReadonlyService provides typed readonly inventory and audit summaries.
type ReadonlyService struct {
	connector *Connector
}

// NewReadonlyService creates a readonly database service.
func NewReadonlyService(connector *Connector) *ReadonlyService {
	return &ReadonlyService{connector: connector}
}

// KeySummary returns typed virtual key counts.
func (s *ReadonlyService) KeySummary(ctx context.Context) (KeySummary, error) {
	if s == nil || s.connector == nil {
		return KeySummary{}, fmt.Errorf("database readonly service requires a connector")
	}

	total, err := s.connector.scalarInt(ctx, `SELECT COUNT(*) FROM "LiteLLM_VerificationToken";`)
	if err != nil {
		return KeySummary{}, err
	}
	active, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_VerificationToken"
		WHERE expires IS NULL OR expires > NOW();
	`)
	if err != nil {
		return KeySummary{}, err
	}
	return KeySummary{
		Total:   total,
		Active:  active,
		Expired: total - active,
	}, nil
}

// BudgetSummary returns typed budget utilization counts.
func (s *ReadonlyService) BudgetSummary(ctx context.Context) (BudgetSummary, error) {
	if s == nil || s.connector == nil {
		return BudgetSummary{}, fmt.Errorf("database readonly service requires a connector")
	}

	total, err := s.connector.scalarInt(ctx, `SELECT COUNT(*) FROM "LiteLLM_BudgetTable";`)
	if err != nil {
		return BudgetSummary{}, err
	}
	high, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE max_budget > 0 AND (budget::float / max_budget::float * 100) <= 20;
	`)
	if err != nil {
		return BudgetSummary{}, err
	}
	exhausted, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE budget <= 0;
	`)
	if err != nil {
		return BudgetSummary{}, err
	}
	return BudgetSummary{
		Total:           total,
		HighUtilization: high,
		Exhausted:       exhausted,
	}, nil
}

// DetectionSummary returns typed recent detection findings.
func (s *ReadonlyService) DetectionSummary(ctx context.Context) (DetectionSummary, error) {
	if s == nil || s.connector == nil {
		return DetectionSummary{}, fmt.Errorf("database readonly service requires a connector")
	}

	tableCount, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	if tableCount == 0 {
		return DetectionSummary{}, nil
	}

	high, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	medium, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 5.0 AND spend <= 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	uniqueModels, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(DISTINCT model) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	totalEntries, err := s.connector.scalarInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	return DetectionSummary{
		SpendLogsTableExists: true,
		HighSeverity:         high,
		MediumSeverity:       medium,
		UniqueModels24h:      uniqueModels,
		TotalEntries24h:      totalEntries,
	}, nil
}
