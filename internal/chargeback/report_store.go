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
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/db"
)

type DBStore struct {
	Client *db.Client
}

func NewDBStore(client *db.Client) *DBStore {
	return &DBStore{Client: client}
}

func (s *DBStore) CostCenterAllocations(ctx context.Context, monthStart string, monthEnd string) ([]CostCenterAllocation, error) {
	raw, err := s.query(ctx, fmt.Sprintf(`
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
	if err != nil {
		return nil, err
	}
	return DecodeCostCenters(raw)
}

func (s *DBStore) ModelAllocations(ctx context.Context, monthStart string, monthEnd string) ([]ModelAllocation, error) {
	raw, err := s.query(ctx, fmt.Sprintf(`
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
	if err != nil {
		return nil, err
	}
	return DecodeModels(raw)
}

func (s *DBStore) TopPrincipals(ctx context.Context, monthStart string, monthEnd string, limit int) ([]PrincipalSpend, error) {
	raw, err := s.query(ctx, fmt.Sprintf(`
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
	raw, err := s.query(ctx, fmt.Sprintf(`
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

func (s *DBStore) Metrics(ctx context.Context, monthStart string, monthEnd string) (Metrics, error) {
	raw, err := s.query(ctx, fmt.Sprintf(`
SELECT
  COALESCE(COUNT(*), 0) || '|' ||
  COALESCE(SUM("prompt_tokens" + "completion_tokens"), 0)
FROM "LiteLLM_SpendLogs"
WHERE "startTime" >= '%s 00:00:00'
  AND "startTime" <= '%s 23:59:59';
`, monthStart, monthEnd))
	if err != nil {
		return Metrics{}, err
	}
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 2 {
		return Metrics{}, fmt.Errorf("unexpected metrics payload: %q", raw)
	}
	requests, err := parseInt64(parts[0])
	if err != nil {
		return Metrics{}, err
	}
	tokens, err := parseInt64(parts[1])
	if err != nil {
		return Metrics{}, err
	}
	return Metrics{TotalRequests: requests, TotalTokens: tokens}, nil
}

func (s *DBStore) HistoricalSpend(ctx context.Context, monthsBack int, monthStart string) ([]HistoricalSpend, error) {
	raw, err := s.query(ctx, fmt.Sprintf(`
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
	raw, err := s.query(ctx, `
SELECT COALESCE(SUM(max_budget), 0) AS total_max_budget
FROM "LiteLLM_BudgetTable"
WHERE max_budget > 0;
`)
	if err != nil {
		return 0, err
	}
	return parseFloat(raw)
}

func (s *DBStore) query(ctx context.Context, query string) (string, error) {
	if s == nil || s.Client == nil {
		return "", fmt.Errorf("chargeback DB store requires a database client")
	}
	return s.Client.Query(ctx, query)
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

func encodeJSON(value any) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}
