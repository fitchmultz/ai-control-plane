# AI Control Plane - Detection Rules Authoring Guide

This document provides comprehensive guidelines for authoring, validating, and troubleshooting detection rules for the AI Control Plane. Detection rules are SIEM-style SQL queries that identify security anomalies, policy violations, and cost risks in LiteLLM gateway usage logs.

## Table of Contents

1. [Overview](#1-overview)
2. [Rule Structure](#2-rule-structure)
3. [SQL Query Requirements](#3-sql-query-requirements)
4. [Database Schema Reference](#4-database-schema-reference)
5. [SQL Pattern Guidelines](#5-sql-pattern-guidelines)
6. [Parameter Handling](#6-parameter-handling)
7. [Validation](#7-validation)
8. [Testing New Rules](#8-testing-new-rules)
9. [Troubleshooting](#9-troubleshooting)

---

## 1. Overview

Detection rules analyze gateway usage patterns to identify:

- **Security anomalies** - Failed authentication, rapid request rates
- **Policy violations** - Non-approved model access
- **Cost risks** - Budget exhaustion, token usage spikes
- **Availability issues** - High error rates

Rules are defined in `demo/config/detection_rules.yaml` and executed by `make validate-detections`.

### Operational Classification (Required)

Each rule includes explicit maturity metadata:
- `operational_status`: `validated` or `example`
- `coverage_tier`: `decision-grade` or `demo`
- `expected_signal`: concise statement of what the rule is expected to surface

This classification prevents demo placeholders from being treated as decision-grade controls.

### Key Design Principles

1. **SQL-first**: Rules are written as PostgreSQL SQL queries
2. **Read-only**: Rules never modify database state
3. **Time-bounded**: All queries use relative time windows (not hardcoded timestamps)
4. **Attribution-friendly**: Results include human-readable key aliases via JOINs
5. **Secure**: External parameters use psql variable substitution (not string interpolation)

---

## 2. Rule Structure

Detection rules follow a YAML schema with the following fields:

```yaml
- rule_id: DR-XXX                          # Required: Unique identifier (format: DR-###)
  name: "Human-Readable Name"              # Required: Short descriptive name
  description: "What this rule detects"    # Required: Detailed description
  severity: "high|medium|low"              # Required: Alert severity level
  category: "policy_violation|anomaly|availability|cost_management|security"  # Required
  operational_status: "validated|example"  # Required: Rule maturity state
  coverage_tier: "decision-grade|demo"     # Required: Confidence tier
  expected_signal: "What this trigger means" # Required: Concrete signal statement
  enabled: true|false                      # Required: Whether rule is active
  parameters:                              # Optional: Rule-specific parameters
    threshold_value: 100
    window_hours: 24
  sql_query: |                             # Required: The SQL query (MUST use | block scalar)
    SELECT ...
  remediation: "How to respond"            # Required: Remediation guidance
```

### Field Requirements

| Field | Required | Format | Description |
|-------|----------|--------|-------------|
| `rule_id` | Yes | `DR-###` (e.g., DR-001) | Unique identifier |
| `name` | Yes | String (max ~50 chars) | Human-readable name |
| `description` | Yes | String | What the rule detects |
| `severity` | Yes | `high`, `medium`, `low` | Alert priority |
| `category` | Yes | Enum | Classification category |
| `operational_status` | Yes | `validated`, `example` | Rule maturity |
| `coverage_tier` | Yes | `decision-grade`, `demo` | Confidence tier |
| `expected_signal` | Yes | String | Concrete trigger intent |
| `enabled` | Yes | Boolean | Whether to execute rule |
| `parameters` | No | Key-value map | Rule configuration |
| `sql_query` | Yes | SQL (block scalar) | The detection query |
| `remediation` | Yes | String | Response guidance |

### Severity Levels

- **high** - Requires immediate attention, potential security incident
- **medium** - Investigate promptly, may indicate issues
- **low** - Monitor for trends, no immediate action required

### Categories

- **policy_violation** - Governance policy breaches (e.g., non-approved models)
- **anomaly** - Unusual usage patterns (e.g., token spikes, rapid requests)
- **availability** - Service disruption indicators (e.g., high error rates)
- **cost_management** - Budget and spending concerns
- **security** - Security threats (e.g., failed authentication)

---

## 3. SQL Query Requirements

### Required Table References

Detection rules must reference at least one of these LiteLLM tables:

#### 1. LiteLLM_SpendLogs (Audit/Usage Logs)

The primary table for most detection rules. Contains request-level usage data.

**Key Columns:**

| Column | Type | Description |
|--------|------|-------------|
| `request_id` | String | Unique request identifier |
| `model` | String | Model identifier (e.g., `claude-sonnet-4-5`) |
| `api_key` | String | Token identifier (foreign key to VerificationToken) |
| `"user"` | String | User identifier (quoted due to reserved word) |
| `"startTime"` | Timestamp | Request timestamp (quoted due to camelCase) |
| `status` | String | Request status (`success`, `failure`, etc.) |
| `prompt_tokens` | Integer | Input token count |
| `completion_tokens` | Integer | Output token count |

**Use Case:** Most detection rules query this table for usage patterns, error rates, and token consumption.

#### 2. LiteLLM_VerificationToken (Virtual Key Metadata)

Stores virtual key information including human-readable aliases.

**Key Columns:**

| Column | Type | Description |
|--------|------|-------------|
| `token` | String | The API key token (matches `LiteLLM_SpendLogs.api_key`) |
| `key_alias` | String | Human-readable key name (e.g., `dev-key`, `prod-app`) |
| `user_id` | String | Associated user identifier |
| `budget_id` | String | Foreign key to BudgetTable |
| `spend` | Decimal | Current spend amount |

**Use Case:** JOIN to this table for operator-friendly output with `key_alias` attribution.

#### 3. LiteLLM_BudgetTable (Budget Limits)

Stores budget limits for keys.

**Key Columns:**

| Column | Type | Description |
|--------|------|-------------|
| `budget_id` | String | Unique budget identifier |
| `max_budget` | Decimal | Maximum budget limit |

**Use Case:** Budget exhaustion and threshold rules (DR-004, DR-007).

### SQL Pattern Requirements

#### 1. Time Windows (CRITICAL)

**ALWAYS** use `INTERVAL` with `NOW()` for time windows. **NEVER** use hardcoded timestamps.

```sql
-- CORRECT
WHERE s."startTime" > NOW() - INTERVAL '24 hours'

-- INCORRECT - Hardcoded timestamp
WHERE s."startTime" > '2026-01-01'

-- INCORRECT - No time window
WHERE s.status = 'failure'
```

**Standard Time Windows:**

| Window | Use Case |
|--------|----------|
| `INTERVAL '1 hour'` | Rapid request detection, real-time anomalies |
| `INTERVAL '24 hours'` | Daily analysis, most detection rules |

#### 2. Key Alias Attribution (REQUIRED)

Always JOIN to `LiteLLM_VerificationToken` for operator-friendly output:

```sql
-- CORRECT - Includes key_alias for attribution
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  s.model,
  s.status
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v
  ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '24 hours';

-- INCORRECT - No attribution
SELECT api_key, model, status
FROM "LiteLLM_SpendLogs"
WHERE "startTime" > NOW() - INTERVAL '24 hours';
```

Use `COALESCE(v.key_alias, 'unknown')` to handle cases where the key has been deleted.

#### 3. psql Variable Substitution

For external parameters (like approved model lists), use psql colon-quoted syntax:

```sql
-- For JSON arrays (DR-001)
WHERE s.model NOT IN (
  SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb)
)

-- For string values
WHERE s.status = :'TARGET_STATUS'
```

**Security Note:** The `:'VAR_NAME'` syntax safely quotes the value as a SQL literal, preventing SQL injection. Variables are passed via `psql -v VAR_NAME='value'`.

**Contrast with unsafe interpolation:**

```sql
-- NEVER DO THIS - SQL injection vulnerability
WHERE s.model = '$USER_INPUT'

-- CORRECT - psql variable substitution
WHERE s.model = :'MODEL_NAME'
```

#### 4. Required Output Columns

While not strictly enforced, include these columns for consistent output:

- `key_alias` - From `COALESCE(v.key_alias, 'unknown')`
- `timestamp` - Formatted as `TO_CHAR(s."startTime", 'YYYY-MM-DD HH24:MI:SS')`
- `count|total|sum` - Aggregation results for threshold rules

---

## 4. Database Schema Reference

### Complete Column Reference

#### LiteLLM_SpendLogs

```sql
SELECT
  request_id,
  model,
  api_key,           -- Joins to LiteLLM_VerificationToken.token
  "user",            -- Quoted: reserved word
  "startTime",       -- Quoted: camelCase
  status,
  prompt_tokens,
  completion_tokens,
  spend,             -- Cost in USD
  response_time_ms
FROM "LiteLLM_SpendLogs";
```

#### LiteLLM_VerificationToken

```sql
SELECT
  token,             -- Joins to LiteLLM_SpendLogs.api_key
  key_alias,         -- Human-readable name
  user_id,
  budget_id,         -- Joins to LiteLLM_BudgetTable.budget_id
  spend,
  max_budget,
  created_at,
  expires
FROM "LiteLLM_VerificationToken";
```

#### LiteLLM_BudgetTable

```sql
SELECT
  budget_id,
  max_budget,
  created_at,
  updated_at
FROM "LiteLLM_BudgetTable";
```

### Table Relationships

```
LiteLLM_SpendLogs          LiteLLM_VerificationToken        LiteLLM_BudgetTable
├─ api_key ────────────────>├─ token                         ├─ budget_id
├─ model                    ├─ key_alias                     ├─ max_budget
├─ "user"                   ├─ user_id                       └─ ...
├─ "startTime"              └─ budget_id ────────────────────>
├─ status
├─ prompt_tokens
├─ completion_tokens
└─ ...
```

---

## 5. SQL Pattern Guidelines

### Pattern 1: Threshold-Based Detection

Detect when a metric exceeds a threshold:

```sql
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  SUM(s."prompt_tokens" + s."completion_tokens") AS total_tokens,
  COUNT(*) AS request_count,
  TO_CHAR(MAX(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS last_seen
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v
  ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '24 hours'
GROUP BY COALESCE(v.key_alias, 'unknown')
HAVING SUM(s."prompt_tokens" + s."completion_tokens") > 100000
ORDER BY total_tokens DESC;
```

### Pattern 2: Rate/Error Rate Detection

Detect elevated error rates:

```sql
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  COUNT(*) FILTER (WHERE s.status != 'success') AS error_count,
  COUNT(*) AS total_requests,
  ROUND(100.0 * COUNT(*) FILTER (WHERE s.status != 'success')
    / NULLIF(COUNT(*), 0), 2) AS error_rate
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v
  ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '24 hours'
GROUP BY COALESCE(v.key_alias, 'unknown')
HAVING COUNT(*) >= 10
  AND COUNT(*) FILTER (WHERE s.status != 'success')
    * 100.0 / NULLIF(COUNT(*), 0) > 10
ORDER BY error_rate DESC;
```

### Pattern 3: Budget Threshold Detection

Query budget tables (no SpendLogs join needed):

```sql
SELECT
  v.key_alias,
  ROUND(v.spend::numeric, 4) AS spent,
  ROUND(b.max_budget::numeric, 4) AS maximum,
  ROUND(((b.max_budget - v.spend) / NULLIF(b.max_budget, 0)
    * 100)::numeric, 2) AS percent_remaining
FROM "LiteLLM_VerificationToken" v
JOIN "LiteLLM_BudgetTable" b ON v.budget_id = b.budget_id
WHERE b.max_budget > 0
  AND ((b.max_budget - v.spend) / NULLIF(b.max_budget, 0)) < 0.2
ORDER BY percent_remaining ASC;
```

### Pattern 4: Policy Violation (Model List)

Use psql variables for external lists:

```sql
SELECT
  s.request_id,
  s.model AS model_id,
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  TO_CHAR(s."startTime", 'YYYY-MM-DD HH24:MI:SS') AS timestamp
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v
  ON s.api_key = v.token
WHERE s.model NOT IN (
  SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb)
)
  AND s."startTime" > NOW() - INTERVAL '24 hours'
ORDER BY s."startTime" DESC
LIMIT 100;
```

### Pattern 5: Time-Based Rate Detection

Detect rapid requests per time window:

```sql
SELECT
  COALESCE(v.key_alias, 'unknown') AS key_alias,
  COUNT(*) AS request_count,
  TO_CHAR(MIN(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS first_request,
  TO_CHAR(MAX(s."startTime"), 'YYYY-MM-DD HH24:MI:SS') AS last_request
FROM "LiteLLM_SpendLogs" s
LEFT JOIN "LiteLLM_VerificationToken" v
  ON s.api_key = v.token
WHERE s."startTime" > NOW() - INTERVAL '1 hour'
GROUP BY
  COALESCE(v.key_alias, 'unknown'),
  DATE_TRUNC('minute', s."startTime")
HAVING COUNT(*) > 60
ORDER BY request_count DESC
LIMIT 50;
```

---

## 6. Parameter Handling

### Current Approach: Hardcoded in SQL

Most rules hardcode parameter values directly in the SQL:

```yaml
parameters:
  threshold_tokens: 100000
  window_hours: 24
sql_query: |
  SELECT ...
  HAVING SUM(...) > 100000  -- Hardcoded value
  WHERE s."startTime" > NOW() - INTERVAL '24 hours'  -- Hardcoded interval
```

**Note:** The `parameters` section is currently documentation-only. Future enhancement may support psql variable substitution for dynamic parameters.

### psql Variables for External Data

For data that changes at runtime (like approved model lists), use psql variables:

```sql
-- In detection_rules.yaml
WHERE s.model NOT IN (
  SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb)
)
```

```bash
# Validation tooling sets this variable during detection execution
# (invoked via `make validate-detections`):
docker exec ... psql ... -v "APPROVED_MODELS_JSON='[\"model1\",\"model2\"]'" -c "..."
```

---

## 7. Validation

### JSON/Schema Validation

Validate that rules produce well-formed JSON output:

```bash
# Validate all enabled rules
make validate-detections

# Or run directly
make validate-detections

# With verbose output
make validate-detections

# Wait for database to be ready (useful in CI)
make validate-detections
```

This validates:
- JSON syntax is valid
- Required schema fields are present
- Schema version matches expected value
- Summary counts are consistent
- Findings have required fields
- Timestamps are ISO 8601 format

### SQL Syntax Validation

Validate SQL syntax without executing against live data:

```bash
# Validate SQL syntax for all enabled rules
make validate-detections

# Or use the wrapper script
make validate-detections
```

This validates:
- SQL is parseable by PostgreSQL (uses EXPLAIN)
- Required table references are present
- psql variables are properly formatted

**Requirements:** Requires a running PostgreSQL container. Rules are skipped if prerequisites are not ready.

### SIEM Query Sync Validation

Ensure SIEM query mappings stay in sync:

```bash
make validate-siem-queries
```

Validates that:
- Every rule_id in `detection_rules.yaml` has a matching entry in `siem_queries.yaml`
- Enabled rules have required vendor queries (Splunk, ELK, Sentinel, Sigma)

---

## 8. Testing New Rules

### Step 1: Dry-Run to Preview

Preview the rule without execution:

```bash
# Preview all rules
make validate-detections

# Preview specific rule
make validate-detections

# Preview with SQL output
make validate-detections

# Preview as JSON
make validate-detections
```

### Step 2: Test Against Database

Run the rule against live data:

```bash
# Run specific rule
make validate-detections

# Run with verbose output
make validate-detections

# Get JSON output
make validate-detections
```

### Step 3: Validate JSON Output

```bash
make validate-detections
```

### Step 4: Validate SQL Syntax

```bash
make validate-detections
```

### Step 5: Test in CI Pipeline

Ensure the rule passes the full CI gate:

```bash
make ci
```

---

## 9. Troubleshooting

### SQL Query Errors

**Error: "column does not exist"**

```
ERROR: column "startTime" does not exist
```

**Solution:** Column names with uppercase letters must be quoted:

```sql
-- INCORRECT
WHERE s.startTime > ...

-- CORRECT
WHERE s."startTime" > ...
```

**Error: "syntax error at or near ':'"**

```
ERROR: syntax error at or near ":"
```

**Solution:** psql variables require the `:'VARNAME'` syntax with quotes:

```sql
-- INCORRECT
WHERE model = :APPROVED_MODELS

-- CORRECT
WHERE s.model NOT IN (
  SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb)
)
```

**Error: "relation does not exist"**

```
ERROR: relation "LiteLLM_SpendLogs" does not exist
```

**Solution:** Ensure table names are quoted (they contain uppercase letters):

```sql
-- INCORRECT
FROM LiteLLM_SpendLogs

-- CORRECT
FROM "LiteLLM_SpendLogs"
```

Also verify the database is initialized: `make db-status`

### Validation Failures

**"SQL does not reference required tables"**

Ensure your query references at least one of:
- `LiteLLM_SpendLogs`
- `LiteLLM_VerificationToken`
- `LiteLLM_BudgetTable`

**"SQL syntax validation failed"**

Test the query manually:

```bash
# Open database shell
make db-shell

# Run EXPLAIN to check syntax
EXPLAIN (FORMAT TEXT) SELECT ...;
```

### Performance Issues

**Query is slow**

- Ensure time window filters are applied
- Add `LIMIT` for rules that may return many rows
- Check if indexes exist on `startTime` and `api_key` columns

### Rule Not Executing

**"Rule is disabled or not found"**

Check that:
1. `enabled: true` is set in the rule
2. `rule_id` matches exactly (case-sensitive)
3. YAML indentation is correct (2 spaces)

### Getting Help

- Review existing rules in `demo/config/detection_rules.yaml` for examples
- Check logs: `make logs`
- Verify database: `make db-status`
- Run linting: `make lint`

---

## Related Documentation

- [Security Detection Rules](security/DETECTION.md) - Rule descriptions and SIEM integration
- [Database Reference](DATABASE.md) - PostgreSQL schema and operations
- [Deployment Guide](DEPLOYMENT.md) - Full deployment instructions
