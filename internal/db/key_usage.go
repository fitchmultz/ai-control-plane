// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose key-lifecycle usage summaries without adding a raw-SQL escape hatch.
//
// Responsibilities:
//   - Summarize month-scoped spend, request, and token totals for a key alias.
//   - Provide per-model usage breakdowns for inspection and rotation workflows.
//
// Scope:
//   - Key usage reporting queries only.
//
// Usage:
//   - Construct via `NewReadonlyService(connector)` and call `KeyUsage`.
//
// Invariants/Assumptions:
//   - Queries remain read-only and are scoped to the canonical LiteLLM schema.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/keygen"
)

// KeyUsage returns typed month-scoped usage totals for a virtual key alias.
func (s *ReadonlyService) KeyUsage(ctx context.Context, alias string, window keygen.MonthWindow) (keygen.KeyUsage, error) {
	if s == nil || s.connector == nil {
		return keygen.KeyUsage{}, fmt.Errorf("database readonly service requires a connector")
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return keygen.KeyUsage{}, fmt.Errorf("alias is required")
	}

	totalsQuery := fmt.Sprintf(`
SELECT
  COUNT(*) || '|' ||
  COALESCE(SUM(COALESCE(s."prompt_tokens", 0) + COALESCE(s."completion_tokens", 0)), 0) || '|' ||
  COALESCE(SUM(s.spend), 0) || '|' ||
  COALESCE(TO_CHAR(MAX(s."startTime") AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
WHERE v.key_alias = '%s'
  AND s."startTime" >= '%s 00:00:00'
  AND s."startTime" <= '%s 23:59:59';
`, sqlLiteral(alias), window.Start, window.End)
	rawTotals, err := s.connector.scalarString(ctx, totalsQuery)
	if err != nil {
		return keygen.KeyUsage{}, err
	}
	usage, err := parseKeyUsageTotals(alias, window.ReportMonth, rawTotals)
	if err != nil {
		return keygen.KeyUsage{}, err
	}

	modelsQuery := fmt.Sprintf(`
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
FROM (
  SELECT
    COALESCE(s.model, 'unknown') AS model_name,
    COUNT(*) AS request_count,
    COALESCE(SUM(COALESCE(s."prompt_tokens", 0) + COALESCE(s."completion_tokens", 0)), 0) AS token_count,
    ROUND(COALESCE(SUM(s.spend), 0)::numeric, 4) AS spend_amount
  FROM "LiteLLM_SpendLogs" s
  LEFT JOIN "LiteLLM_VerificationToken" v ON s.api_key = v.token
  WHERE v.key_alias = '%s'
    AND s."startTime" >= '%s 00:00:00'
    AND s."startTime" <= '%s 23:59:59'
  GROUP BY model_name
) usage;
`, sqlLiteral(alias), window.Start, window.End)
	rawModels, err := s.connector.scalarString(ctx, modelsQuery)
	if err != nil {
		return keygen.KeyUsage{}, err
	}
	if err := json.Unmarshal([]byte(rawModels), &usage.ByModel); err != nil {
		return keygen.KeyUsage{}, fmt.Errorf("decode key usage models: %w", err)
	}

	return usage, nil
}

func parseKeyUsageTotals(alias string, reportMonth string, raw string) (keygen.KeyUsage, error) {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 4 {
		return keygen.KeyUsage{}, fmt.Errorf("unexpected key usage totals payload: %q", raw)
	}
	requests, err := parseInt64(parts[0])
	if err != nil {
		return keygen.KeyUsage{}, err
	}
	tokens, err := parseInt64(parts[1])
	if err != nil {
		return keygen.KeyUsage{}, err
	}
	spend, err := parseFloat(parts[2])
	if err != nil {
		return keygen.KeyUsage{}, err
	}
	return keygen.KeyUsage{
		Alias:         alias,
		ReportMonth:   reportMonth,
		TotalRequests: requests,
		TotalTokens:   tokens,
		TotalSpend:    spend,
		LastSeen:      strings.TrimSpace(parts[3]),
	}, nil
}

func sqlLiteral(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "'", "''")
}
