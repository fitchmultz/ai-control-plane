// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose chargeback-specific readonly queries without a generic raw-SQL API.
//
// Responsibilities:
//   - Serve chargeback JSON payloads and scalar aggregates through named methods.
//   - Keep chargeback query text out of shared runtime and admin services.
//
// Scope:
//   - Chargeback reporting queries only.
//
// Usage:
//   - Construct via `NewChargebackReader(connector)` from chargeback composition.
//
// Invariants/Assumptions:
//   - Public API stays domain-specific and avoids generic query execution.
package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ChargebackReader provides chargeback-specific readonly queries.
type ChargebackReader struct {
	connector *Connector
}

// NewChargebackReader creates a chargeback-specific readonly query service.
func NewChargebackReader(connector *Connector) (*ChargebackReader, error) {
	if connector == nil {
		return nil, fmt.Errorf("database chargeback reader requires a connector")
	}
	if err := connector.ConfigError(); err != nil {
		return nil, err
	}
	return &ChargebackReader{connector: connector}, nil
}

// CostCenterAllocationsJSON returns the canonical cost-center allocation payload.
func (r *ChargebackReader) CostCenterAllocationsJSON(ctx context.Context, monthStart string, monthEnd string) (string, error) {
	return r.scalarString(ctx, fmt.Sprintf(`
WITH attribution AS (
  SELECT
    s.spend,
    s."prompt_tokens",
    s."completion_tokens",
    CASE
      WHEN v.key_alias LIKE '%%__team-%%' THEN substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown-team'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%%__cc-%%' THEN substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center
  FROM "LiteLLM_SpendLogs" s
  LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
  WHERE s."startTime" >= '%s 00:00:00'
    AND s."startTime" <= '%s 23:59:59'
),
aggregated AS (
  SELECT
    cost_center,
    team,
    COUNT(*) AS request_count,
    COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0) AS token_count,
    ROUND(COALESCE(SUM(spend), 0)::numeric, 4) AS spend_amount,
    ROUND((COALESCE(SUM(spend), 0) / NULLIF((SELECT SUM(spend) FROM attribution), 0) * 100)::numeric, 2) AS percent_of_total
  FROM attribution
  GROUP BY cost_center, team
)
SELECT COALESCE(
  json_agg(
    json_build_object(
      'cost_center', cost_center,
      'team', team,
      'request_count', request_count,
      'token_count', token_count,
      'spend_amount', spend_amount,
      'percent_of_total', COALESCE(percent_of_total, 0)
    )
    ORDER BY spend_amount DESC, cost_center ASC, team ASC
  ),
  '[]'::json
)
FROM aggregated;
`, monthStart, monthEnd))
}

// ModelAllocationsJSON returns the canonical model allocation payload.
func (r *ChargebackReader) ModelAllocationsJSON(ctx context.Context, monthStart string, monthEnd string) (string, error) {
	return r.scalarString(ctx, fmt.Sprintf(`
WITH aggregated AS (
  SELECT
    COALESCE(model, 'unknown') AS model_name,
    COUNT(*) AS request_count,
    COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0) AS token_count,
    ROUND(COALESCE(SUM(spend), 0)::numeric, 4) AS spend_amount
  FROM "LiteLLM_SpendLogs"
  WHERE "startTime" >= '%s 00:00:00'
    AND "startTime" <= '%s 23:59:59'
  GROUP BY model_name
)
SELECT COALESCE(
  json_agg(
    json_build_object(
      'model', model_name,
      'request_count', request_count,
      'token_count', token_count,
      'spend_amount', spend_amount
    )
    ORDER BY spend_amount DESC, model_name ASC
  ),
  '[]'::json
)
FROM aggregated;
`, monthStart, monthEnd))
}

// TopPrincipalsJSON returns the top-principals payload.
func (r *ChargebackReader) TopPrincipalsJSON(ctx context.Context, monthStart string, monthEnd string, limit int) (string, error) {
	return r.scalarString(ctx, fmt.Sprintf(`
SELECT COALESCE(
  json_agg(
    json_build_object(
      'principal', key_alias,
      'team', team,
      'cost_center', cost_center,
      'request_count', request_count,
      'spend_amount', total_spend
    )
    ORDER BY total_spend DESC, key_alias ASC
  ),
  '[]'::json
)
FROM (
  SELECT
    COALESCE(v.key_alias, 'unknown') AS key_alias,
    COUNT(*) AS request_count,
    ROUND(COALESCE(SUM(s.spend), 0)::numeric, 4) AS total_spend,
    CASE
      WHEN v.key_alias LIKE '%%__team-%%' THEN substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%%__cc-%%' THEN substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown'
    END AS cost_center
  FROM "LiteLLM_SpendLogs" s
  LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
  WHERE s."startTime" >= '%s 00:00:00'
    AND s."startTime" <= '%s 23:59:59'
  GROUP BY v.key_alias
  ORDER BY SUM(s.spend) DESC
  LIMIT %d
) principals;
`, monthStart, monthEnd, limit))
}

// TotalSpend returns the total spend for the month.
func (r *ChargebackReader) TotalSpend(ctx context.Context, monthStart string, monthEnd string) (float64, error) {
	raw, err := r.scalarString(ctx, fmt.Sprintf(`
SELECT COALESCE(SUM(spend), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '%s 00:00:00'
  AND "startTime" <= '%s 23:59:59';
`, monthStart, monthEnd))
	if err != nil {
		return 0, err
	}
	return parseFloat(raw)
}

// MetricsSummary returns typed aggregate request/token counts for the month.
func (r *ChargebackReader) MetricsSummary(ctx context.Context, monthStart string, monthEnd string) (ChargebackMetricsSummary, error) {
	raw, err := r.scalarString(ctx, fmt.Sprintf(`
SELECT
  COALESCE(COUNT(*), 0) || '|' ||
  COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '%s 00:00:00'
  AND "startTime" <= '%s 23:59:59';
`, monthStart, monthEnd))
	if err != nil {
		return ChargebackMetricsSummary{}, err
	}
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 2 {
		return ChargebackMetricsSummary{}, fmt.Errorf("unexpected metrics payload: %q", raw)
	}
	requests, err := parseInt64(parts[0])
	if err != nil {
		return ChargebackMetricsSummary{}, err
	}
	tokens, err := parseInt64(parts[1])
	if err != nil {
		return ChargebackMetricsSummary{}, err
	}
	return ChargebackMetricsSummary{
		TotalRequests: requests,
		TotalTokens:   tokens,
	}, nil
}

// HistoricalSpendJSON returns the monthly historical spend payload.
func (r *ChargebackReader) HistoricalSpendJSON(ctx context.Context, monthsBack int, monthStart string) (string, error) {
	return r.scalarString(ctx, fmt.Sprintf(`
SELECT COALESCE(
  json_agg(
    json_build_object(
      'month', month,
      'spend', monthly_spend
    )
    ORDER BY month DESC
  ),
  '[]'::json
)
FROM (
  SELECT
    TO_CHAR(DATE_TRUNC('month', "startTime"), 'YYYY-MM') AS month,
    COALESCE(SUM(spend), 0) AS monthly_spend
  FROM "LiteLLM_SpendLogs"
  WHERE "startTime" < '%s 00:00:00'
    AND "startTime" >= '%s 00:00:00'::timestamp - INTERVAL '%d months'
  GROUP BY DATE_TRUNC('month', "startTime")
  ORDER BY month DESC
  LIMIT %d
) history;
`, monthStart, monthStart, monthsBack, monthsBack))
}

// TotalBudget returns the total configured positive budget.
func (r *ChargebackReader) TotalBudget(ctx context.Context) (float64, error) {
	raw, err := r.scalarString(ctx, `
SELECT COALESCE(SUM(max_budget), 0) AS total_max_budget
FROM "LiteLLM_BudgetTable"
WHERE max_budget > 0;
`)
	if err != nil {
		return 0, err
	}
	return parseFloat(raw)
}

func (r *ChargebackReader) scalarString(ctx context.Context, query string) (string, error) {
	if r == nil || r.connector == nil {
		return "", fmt.Errorf("database chargeback reader requires a connector")
	}
	return r.connector.scalarString(ctx, query)
}

func parseFloat(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("parse float %q: %w", raw, err)
	}
	return value, nil
}

func parseInt64(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse integer %q: %w", raw, err)
	}
	return value, nil
}
