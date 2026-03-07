#!/usr/bin/env bash
# demo/scripts/lib/chargeback_db.sh - Database querying library for chargeback reports
#
# Purpose: Encapsulates all SQL queries and database interactions for chargeback
#          reporting, supporting both production (Docker) and test (mock) modes.
#
# Responsibilities:
#   - Execute single-value and table-formatted SQL queries
#   - Provide specific data collection functions for cost centers, models, and principals
#   - Handle testability hooks (CHARGEBACK_PSQL_BIN)
#
# Non-scope:
#   - Does NOT handle report rendering or analysis
#   - Does NOT handle month boundary calculations
#
# Invariants:
#   - Read-only queries only (no modifications to SpendLogs)
#   - Uses docker_sql.sh for production queries
#

if [[ -n "${ACP_CHARGEBACK_DB_LOADED:-}" ]]; then
    return 0
fi
readonly ACP_CHARGEBACK_DB_LOADED=1

resolve_chargeback_psql_bin() {
    local bin="${CHARGEBACK_PSQL_BIN:-}"

    if [[ -z "${bin}" ]]; then
        return 1
    fi
    if [[ "${bin}" != /* || "${bin}" =~ [[:space:]] ]]; then
        printf 'ERROR: CHARGEBACK_PSQL_BIN must be an absolute executable path without arguments\n' >&2
        return 64
    fi
    if [[ ! -x "${bin}" || -d "${bin}" ]]; then
        printf 'ERROR: CHARGEBACK_PSQL_BIN is not executable: %s\n' "${bin}" >&2
        return 2
    fi

    printf '%s\n' "${bin}"
}

trim_chargeback_scalar() {
    local value="$1"
    value="${value//$'\r'/}"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    printf '%s' "${value}"
}

run_chargeback_psql() {
    local tuple_only="$1"
    local sql="$2"
    local bin
    local -a args

    if ! bin="$(resolve_chargeback_psql_bin)"; then
        return $?
    fi

    args=()
    args+=("-X" "-v" "ON_ERROR_STOP=1" "-P" "pager=off")
    if [[ "${tuple_only}" == "true" ]]; then
        args+=("-t" "-A")
    fi
    args+=("-c" "${sql}")

    "${bin}" "${args[@]}" 2>/dev/null
}

# Execute SQL query with error handling
# Uses CHARGEBACK_PSQL_BIN override if set (for testing)
query() {
    local sql="$1"
    local result
    local exit_code=0

    if [[ -n "${CHARGEBACK_PSQL_BIN:-}" ]]; then
        # Test mode: use overridden psql binary with argv-safe execution
        if result="$(run_chargeback_psql true "$sql")"; then
            result="$(trim_chargeback_scalar "$result")"
            if [[ -z "$result" ]]; then
                echo "0"
                return 1
            fi
            echo "$result"
            return 0
        else
            echo "0"
            return 3
        fi
    else
        # Production mode: use docker_sql_query
        # shellcheck disable=SC2154
        result=$(docker_sql_query "$sql" "0") || exit_code=$?
        echo "$result"
        return $exit_code
    fi
}

query_json() {
    local sql="$1"
    local result
    local exit_code=0

    if [[ -n "${CHARGEBACK_PSQL_BIN:-}" ]]; then
        if result="$(run_chargeback_psql true "$sql")"; then
            result="$(trim_chargeback_scalar "$result")"
            if [[ -z "$result" ]]; then
                echo "[]"
                return 1
            fi
            echo "$result"
            return 0
        fi
        echo "[]"
        return 3
    fi

    result=$(docker_sql_query "$sql" "[]") || exit_code=$?
    echo "$result"
    return $exit_code
}

# Execute SQL query and return formatted table results
query_table() {
    local sql="$1"
    local result
    local exit_code=0

    if [[ -n "${CHARGEBACK_PSQL_BIN:-}" ]]; then
        # Test mode
        result="$(run_chargeback_psql false "$sql")" || exit_code=$?
        if [[ $exit_code -eq 0 && "$result" == *"(0 rows)"* ]]; then
            echo "$result"
            return 1
        fi
        echo "$result"
        return $exit_code
    else
        # Production mode
        # shellcheck disable=SC2154
        docker_sql_query_table "$sql"
        return $?
    fi
}

# Get monthly spend by cost center
get_cost_center_spend() {
    local month_start="$1"
    local month_end="$2"

    local sql="
WITH attribution AS (
  SELECT
    s.spend,
    s.\"prompt_tokens\",
    s.\"completion_tokens\",
    COALESCE(v.key_alias, 'unknown') AS key_alias,
    CASE
      WHEN v.key_alias LIKE '%__team-%' THEN
        substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown-team'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN
        substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" >= '${month_start} 00:00:00'
    AND s.\"startTime\" <= '${month_end} 23:59:59'
)
SELECT
  cost_center,
  team,
  COUNT(*) AS request_count,
  SUM(\"prompt_tokens\" + \"completion_tokens\") AS total_tokens,
  ROUND(SUM(spend)::numeric, 4) AS total_spend,
  ROUND((SUM(spend) / NULLIF((SELECT SUM(spend) FROM attribution), 0) * 100)::numeric, 2) AS percent_of_total
FROM attribution
GROUP BY cost_center, team
ORDER BY SUM(spend) DESC;
"
    query_table "$sql"
}

get_cost_center_spend_json() {
    local month_start="$1"
    local month_end="$2"

    local sql="
WITH attribution AS (
  SELECT
    s.spend,
    s.\"prompt_tokens\",
    s.\"completion_tokens\",
    CASE
      WHEN v.key_alias LIKE '%__team-%' THEN
        substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown-team'
    END AS team,
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN
        substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" >= '${month_start} 00:00:00'
    AND s.\"startTime\" <= '${month_end} 23:59:59'
),
aggregated AS (
  SELECT
    cost_center,
    team,
    COUNT(*) AS request_count,
    COALESCE(SUM(\"prompt_tokens\" + \"completion_tokens\"), 0) AS token_count,
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
"
    query_json "$sql"
}

# Get unattributed usage
get_unattributed_usage() {
    local month_start="$1"
    local month_end="$2"

    local sql="
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  COUNT(*) AS request_count,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend
FROM \"LiteLLM_SpendLogs\" s
LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
WHERE s.\"startTime\" >= '${month_start} 00:00:00'
  AND s.\"startTime\" <= '${month_end} 23:59:59'
  AND (v.key_alias IS NULL 
       OR v.key_alias NOT LIKE '%__cc-%'
       OR v.key_alias NOT LIKE '%__team-%')
GROUP BY v.key_alias
ORDER BY SUM(s.spend) DESC;
"
    query_table "$sql"
}

# Get top principals by spend
get_top_principals() {
    local month_start="$1"
    local month_end="$2"
    local limit="${3:-20}"

    local sql="
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  COUNT(*) AS request_count,
  ROUND(SUM(s.spend)::numeric, 4) AS total_spend,
  CASE
    WHEN v.key_alias LIKE '%__team-%' THEN
      substring(v.key_alias from '__team-([^_]+)')
    ELSE 'unknown'
  END AS team,
  CASE
    WHEN v.key_alias LIKE '%__cc-%' THEN
      substring(v.key_alias from '__cc-([0-9]+)')
    ELSE 'unknown'
  END AS cost_center
FROM \"LiteLLM_SpendLogs\" s
LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
WHERE s.\"startTime\" >= '${month_start} 00:00:00'
  AND s.\"startTime\" <= '${month_end} 23:59:59'
GROUP BY v.key_alias
ORDER BY SUM(s.spend) DESC
LIMIT ${limit};
"
    query_table "$sql"
}

# Get spend by model
get_model_spend() {
    local month_start="$1"
    local month_end="$2"

    local sql="
SELECT
  COALESCE(model, 'unknown') AS model_id,
  COUNT(*) AS request_count,
  SUM(\"prompt_tokens\" + \"completion_tokens\") AS total_tokens,
  ROUND(SUM(spend)::numeric, 4) AS total_spend
FROM \"LiteLLM_SpendLogs\"
WHERE \"startTime\" >= '${month_start} 00:00:00'
  AND \"startTime\" <= '${month_end} 23:59:59'
GROUP BY model
ORDER BY SUM(spend) DESC;
"
    query_table "$sql"
}

get_model_spend_json() {
    local month_start="$1"
    local month_end="$2"

    local sql="
WITH aggregated AS (
  SELECT
    COALESCE(model, 'unknown') AS model_name,
    COUNT(*) AS request_count,
    COALESCE(SUM(\"prompt_tokens\" + \"completion_tokens\"), 0) AS token_count,
    ROUND(COALESCE(SUM(spend), 0)::numeric, 4) AS spend_amount
  FROM \"LiteLLM_SpendLogs\"
  WHERE \"startTime\" >= '${month_start} 00:00:00'
    AND \"startTime\" <= '${month_end} 23:59:59'
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
"
    query_json "$sql"
}

get_cost_center_anomalies_json() {
    local month_start="$1"
    local month_end="$2"
    local prev_month_start="$3"
    local prev_month_end="$4"
    local threshold="$5"

    local sql="
WITH current_month AS (
  SELECT
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center,
    CASE
      WHEN v.key_alias LIKE '%__team-%' THEN substring(v.key_alias from '__team-([^_]+)')
      ELSE 'unknown-team'
    END AS team,
    ROUND(COALESCE(SUM(s.spend), 0)::numeric, 4) AS current_spend
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" >= '${month_start} 00:00:00'
    AND s.\"startTime\" <= '${month_end} 23:59:59'
  GROUP BY cost_center, team
),
previous_month AS (
  SELECT
    CASE
      WHEN v.key_alias LIKE '%__cc-%' THEN substring(v.key_alias from '__cc-([0-9]+)')
      ELSE 'unknown-cc'
    END AS cost_center,
    ROUND(COALESCE(SUM(s.spend), 0)::numeric, 4) AS previous_spend
  FROM \"LiteLLM_SpendLogs\" s
  LEFT JOIN \"LiteLLM_VerificationToken\" v ON s.api_key = v.token
  WHERE s.\"startTime\" >= '${prev_month_start} 00:00:00'
    AND s.\"startTime\" <= '${prev_month_end} 23:59:59'
  GROUP BY cost_center
),
ranked AS (
  SELECT
    c.cost_center,
    c.team,
    c.current_spend,
    p.previous_spend,
    ROUND((((c.current_spend - p.previous_spend) / NULLIF(p.previous_spend, 0)) * 100)::numeric, 2) AS spike_percent,
    'spike'::text AS type
  FROM current_month c
  INNER JOIN previous_month p ON p.cost_center = c.cost_center
  WHERE c.cost_center <> 'unknown-cc'
    AND p.previous_spend > 0
    AND (((c.current_spend - p.previous_spend) / NULLIF(p.previous_spend, 0)) * 100) >= ${threshold}
)
SELECT COALESCE(
  json_agg(
    json_build_object(
      'cost_center', cost_center,
      'team', team,
      'current_spend', current_spend,
      'previous_spend', previous_spend,
      'spike_percent', spike_percent,
      'type', type
    )
    ORDER BY spike_percent DESC, cost_center ASC
  ),
  '[]'::json
)
FROM ranked;
"
    query_json "$sql"
}

# Get total monthly spend (single value)
get_total_spend() {
    local month_start="$1"
    local month_end="$2"

    local sql="
SELECT COALESCE(SUM(spend), 0)
FROM \"LiteLLM_SpendLogs\"
WHERE \"startTime\" >= '${month_start} 00:00:00'
  AND \"startTime\" <= '${month_end} 23:59:59';
"
    query "$sql"
}

# Get total monthly requests and tokens
total_monthly_metrics() {
    local month_start="$1"
    local month_end="$2"

    local sql="
SELECT
  COALESCE(COUNT(*), 0) || '|' ||
  COALESCE(SUM(\"prompt_tokens\" + \"completion_tokens\"), 0)
FROM \"LiteLLM_SpendLogs\"
WHERE \"startTime\" >= '${month_start} 00:00:00'
  AND \"startTime\" <= '${month_end} 23:59:59';
"
    query "$sql"
}

# Get historical monthly spend for trend analysis
# Returns: month|spend pairs for the last N months (excluding current)
get_historical_spend() {
    local months_back="${1:-6}"
    local current_month_start="${2:-}"

    # If no current month provided, use current month
    if [[ -z "$current_month_start" ]]; then
        current_month_start=$(date +%Y-%m-01)
    fi

    local sql="
SELECT
  TO_CHAR(DATE_TRUNC('month', \"startTime\"), 'YYYY-MM') AS month,
  COALESCE(SUM(spend), 0) AS monthly_spend
FROM \"LiteLLM_SpendLogs\"
WHERE \"startTime\" < '${current_month_start} 00:00:00'
  AND \"startTime\" >= '${current_month_start} 00:00:00'::timestamp - INTERVAL '${months_back} months'
GROUP BY DATE_TRUNC('month', \"startTime\")
ORDER BY month DESC
LIMIT ${months_back};
"
    query_table "$sql"
}

# Get total active budgets for burn rate calculation
# Returns: total_max_budget (sum of all active budget limits)
get_budget_totals() {
    local sql="
SELECT COALESCE(SUM(max_budget), 0) AS total_max_budget
FROM \"LiteLLM_BudgetTable\"
WHERE max_budget > 0;
"
    query "$sql"
}
